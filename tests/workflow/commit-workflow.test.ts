import { chmod, mkdtemp, mkdir, writeFile } from "node:fs/promises";
import { tmpdir } from "node:os";
import { join } from "node:path";

import { afterEach, describe, expect, it, vi } from "vitest";

import { runCli } from "../../src/cli.js";
import { writeUserConfigPatch } from "../../src/config/index.js";
import { EXIT_CODES, type CliWriteStream } from "../../src/output/index.js";
import type { GenerateCommitMessageInput, ProviderConfig } from "../../src/providers/index.js";
import type {
  CommitRunnerOptions,
  CommitRunnerResult,
  CommitWorkflowDependencies,
  CommitWorkflowPrompts
} from "../../src/workflow/index.js";
import { createTempGitRepo } from "../helpers/temp-git-repo.js";

function createMemoryStream() {
  let buffer = "";

  const stream: CliWriteStream = {
    write(chunk) {
      buffer += typeof chunk === "string" ? chunk : Buffer.from(chunk).toString("utf8");
      return true;
    }
  };

  return {
    read() {
      return buffer;
    },
    stream
  };
}

function createCliIo() {
  return {
    stderr: createMemoryStream(),
    stdout: createMemoryStream()
  };
}

function createPromptStub(overrides: Partial<CommitWorkflowPrompts> = {}): CommitWorkflowPrompts {
  return {
    async confirmStageAll() {
      return "skip";
    },
    async editMessage(input) {
      return input.currentMessage;
    },
    async selectPreviewAction() {
      return "confirm";
    },
    ...overrides
  };
}

function createProviderFactory(messages: string[], calls: Array<{ diff: string; files: string[] }>) {
  return vi.fn<CommitWorkflowDependencies["createCommitMessageProvider"]>().mockImplementation((config) => {
    return {
      async generateCommitMessage(input) {
        calls.push({ diff: input.diff, files: input.files.map((file) => file.path) });

        return {
          message: messages.shift() ?? "feat(test): fallback message",
          metadata: {
            model: config.model,
            provider: config.provider
          }
        };
      },
      name: config.provider
    };
  });
}

const originalCwd = process.cwd();
const originalCnmHome = process.env.CNM_HOME;
const originalEnv = { ...process.env };

afterEach(() => {
  process.chdir(originalCwd);
  process.env = { ...originalEnv };

  if (originalCnmHome === undefined) {
    delete process.env.CNM_HOME;
  } else {
    process.env.CNM_HOME = originalCnmHome;
  }
});

async function setupCliRuntime(repoPath: string) {
  const home = await mkdtemp(join(tmpdir(), "cnm-workflow-home-"));

  process.chdir(repoPath);
  process.env.CNM_HOME = home;
  process.env.CNM_PROVIDER = "openai-responses";
  process.env.CNM_MODEL = "test-model";
  process.env.CNM_API_KEY = "test-api-key";

  return home;
}

async function runRootCommand(options: {
  argv?: string[];
  cwd: string;
  io: ReturnType<typeof createCliIo>;
  isTty?: boolean;
  workflow?: Partial<CommitWorkflowDependencies>;
}) {
  return runCli({
    argv: options.argv ?? [],
    commitRuntime: {
      cwd: options.cwd,
      env: { ...process.env },
      isTty: options.isTty ?? true,
      workflow: options.workflow
    },
    stderr: options.io.stderr.stream,
    stdout: options.io.stdout.stream,
    version: "0.1.0"
  });
}

describe("cnm main workflow", { timeout: 20000 }, () => {
  it("prompts first-time users to run init before inspecting repository changes", async () => {
    const repo = await createTempGitRepo({ initialCommit: true });
    await setupCliRuntime(repo.path);
    delete process.env.CNM_API_KEY;
    const io = createCliIo();
    const providerCalls: Array<{ diff: string; files: string[] }> = [];
    const createProvider = createProviderFactory(["feat: should not generate"], providerCalls);

    const exitCode = await runRootCommand({
      cwd: repo.path,
      io,
      workflow: {
        createCommitMessageProvider: createProvider,
        prompts: createPromptStub()
      }
    });

    expect(exitCode).toBe(EXIT_CODES.ERROR);
    expect(providerCalls).toEqual([]);
    expect(io.stdout.read()).toBe("");
    expect(io.stderr.read()).toContain("cnm is not configured yet. Run `cnm init` to choose an AI provider, model, and API key.");
    await repo.cleanup();
  });

  it("creates a real commit only after confirmed preview and preserves unstaged changes", async () => {
    const repo = await createTempGitRepo({ initialCommit: true });
    await setupCliRuntime(repo.path);
    await repo.write("staged.txt", "staged content\n");
    await repo.git(["add", "staged.txt"]);
    await repo.write("README.md", "unstaged update\n");
    const io = createCliIo();
    const providerCalls: Array<{ diff: string; files: string[] }> = [];
    const createProvider = createProviderFactory(["feat: add staged file"], providerCalls);

    const exitCode = await runRootCommand({
      cwd: repo.path,
      io,
      workflow: {
        createCommitMessageProvider: createProvider,
        prompts: createPromptStub({
          async selectPreviewAction() {
            return "confirm";
          }
        })
      }
    });

    expect(exitCode).toBe(EXIT_CODES.SUCCESS);
    expect(io.stderr.read()).toBe("");
    expect(io.stdout.read()).toContain("Committed staged changes with message:");
    expect(io.stdout.read()).toContain("feat: add staged file");
    expect(providerCalls).toHaveLength(1);
    expect(providerCalls[0]?.files).toEqual(["staged.txt"]);
    expect(providerCalls[0]?.diff).toContain("staged.txt");
    expect(providerCalls[0]?.diff).not.toContain("README.md");
    expect(await repo.git(["log", "-1", "--pretty=%s"])).toBe("feat: add staged file");
    expect(await repo.git(["show", "--name-only", "--pretty=format:", "HEAD"])).toContain("staged.txt");
    expect(await repo.git(["status", "--porcelain"])).toContain(" M README.md");
    await repo.cleanup();
  });

  it("exits 130 and skips git commit when the preview is cancelled", async () => {
    const repo = await createTempGitRepo({ initialCommit: true });
    await setupCliRuntime(repo.path);
    await repo.write("staged.txt", "staged content\n");
    await repo.git(["add", "staged.txt"]);
    const io = createCliIo();
    const providerCalls: Array<{ diff: string; files: string[] }> = [];
    const createProvider = createProviderFactory(["feat: add staged file"], providerCalls);
    const commitRunner = vi.fn<(options: CommitRunnerOptions) => Promise<CommitRunnerResult>>().mockResolvedValue({
      exitCode: 0,
      stderr: "",
      stdout: ""
    });

    const exitCode = await runRootCommand({
      cwd: repo.path,
      io,
      workflow: {
        commitRunner,
        createCommitMessageProvider: createProvider,
        prompts: createPromptStub({
          async selectPreviewAction() {
            return "cancel";
          }
        })
      }
    });

    expect(exitCode).toBe(EXIT_CODES.USER_CANCEL);
    expect(providerCalls).toHaveLength(1);
    expect(commitRunner).not.toHaveBeenCalled();
    expect(io.stderr.read()).toContain("Cancelled. No commit was created.");
    expect(await repo.git(["rev-list", "--count", "HEAD"])).toBe("1");
    await repo.cleanup();
  });

  it("re-prompts invalid edited messages and commits only the valid edit", async () => {
    const repo = await createTempGitRepo({ initialCommit: true });
    await setupCliRuntime(repo.path);
    process.env.CNM_PROMPT_STYLE = "conventional";
    await repo.write("staged.txt", "staged content\n");
    await repo.git(["add", "staged.txt"]);
    const io = createCliIo();
    const providerCalls: Array<{ diff: string; files: string[] }> = [];
    const createProvider = createProviderFactory(["feat: generated message"], providerCalls);
    const editMessage = vi.fn<CommitWorkflowPrompts["editMessage"]>()
      .mockResolvedValueOnce("updated files")
      .mockResolvedValueOnce("fix(workflow): validate edited message");
    const selectPreviewAction = vi.fn<CommitWorkflowPrompts["selectPreviewAction"]>()
      .mockResolvedValueOnce("edit")
      .mockResolvedValueOnce("confirm");

    const exitCode = await runRootCommand({
      cwd: repo.path,
      io,
      workflow: {
        createCommitMessageProvider: createProvider,
        prompts: createPromptStub({
          editMessage,
          selectPreviewAction
        })
      }
    });

    expect(exitCode).toBe(EXIT_CODES.SUCCESS);
    expect(providerCalls).toHaveLength(1);
    expect(editMessage).toHaveBeenCalledTimes(2);
    expect(editMessage.mock.calls[0]?.[0]).toMatchObject({
      currentMessage: "feat: generated message",
      promptStyle: "conventional"
    });
    expect(editMessage.mock.calls[1]?.[0]).toMatchObject({
      currentMessage: "updated files"
    });
    expect(editMessage.mock.calls[1]?.[0].validationMessage).toContain("Conventional Commit");
    expect(selectPreviewAction).toHaveBeenCalledTimes(2);
    expect(await repo.git(["log", "-1", "--pretty=%s"])).toBe("fix(workflow): validate edited message");
    await repo.cleanup();
  });

  it("regenerates exactly once per selection and does not commit before final confirmation", async () => {
    const repo = await createTempGitRepo({ initialCommit: true });
    await setupCliRuntime(repo.path);
    await repo.write("staged.txt", "staged content\n");
    await repo.git(["add", "staged.txt"]);
    const io = createCliIo();
    const providerCalls: Array<{ diff: string; files: string[] }> = [];
    const createProvider = createProviderFactory([
      "feat(workflow): initial message",
      "fix(workflow): regenerated message"
    ], providerCalls);
    const commitRunner = vi.fn<(options: CommitRunnerOptions) => Promise<CommitRunnerResult>>().mockResolvedValue({
      exitCode: 0,
      stderr: "",
      stdout: ""
    });
    const selectPreviewAction = vi.fn<CommitWorkflowPrompts["selectPreviewAction"]>()
      .mockResolvedValueOnce("regenerate")
      .mockResolvedValueOnce("cancel");

    const exitCode = await runRootCommand({
      cwd: repo.path,
      io,
      workflow: {
        commitRunner,
        createCommitMessageProvider: createProvider,
        prompts: createPromptStub({
          selectPreviewAction
        })
      }
    });

    expect(exitCode).toBe(EXIT_CODES.USER_CANCEL);
    expect(providerCalls).toHaveLength(2);
    expect(selectPreviewAction).toHaveBeenCalledTimes(2);
    expect(selectPreviewAction.mock.calls[0]?.[0]).toMatchObject({
      attempt: 1,
      message: "feat(workflow): initial message",
      operation: "git commit"
    });
    expect(selectPreviewAction.mock.calls[1]?.[0]).toMatchObject({
      attempt: 2,
      message: "fix(workflow): regenerated message"
    });
    expect(commitRunner).not.toHaveBeenCalled();
    expect(await repo.git(["rev-list", "--count", "HEAD"])).toBe("1");
    await repo.cleanup();
  });

  it("exits 130 when the edit prompt is cancelled and does not create a commit", async () => {
    const repo = await createTempGitRepo({ initialCommit: true });
    await setupCliRuntime(repo.path);
    await repo.write("staged.txt", "staged content\n");
    await repo.git(["add", "staged.txt"]);
    const io = createCliIo();
    const providerCalls: Array<{ diff: string; files: string[] }> = [];
    const createProvider = createProviderFactory(["feat(workflow): generated message"], providerCalls);
    const commitRunner = vi.fn<(options: CommitRunnerOptions) => Promise<CommitRunnerResult>>().mockResolvedValue({
      exitCode: 0,
      stderr: "",
      stdout: ""
    });

    const exitCode = await runRootCommand({
      cwd: repo.path,
      io,
      workflow: {
        commitRunner,
        createCommitMessageProvider: createProvider,
        prompts: createPromptStub({
          async editMessage() {
            return null;
          },
          async selectPreviewAction() {
            return "edit";
          }
        })
      }
    });

    expect(exitCode).toBe(EXIT_CODES.USER_CANCEL);
    expect(providerCalls).toHaveLength(1);
    expect(commitRunner).not.toHaveBeenCalled();
    expect(io.stderr.read()).toContain("Cancelled. No commit was created.");
    expect(await repo.git(["rev-list", "--count", "HEAD"])).toBe("1");
    await repo.cleanup();
  });

  it("defaults to no staging when nothing is staged and does not call the provider", async () => {
    const repo = await createTempGitRepo({ initialCommit: true });
    await setupCliRuntime(repo.path);
    await repo.write("README.md", "unstaged only\n");
    const io = createCliIo();
    const providerCalls: Array<{ diff: string; files: string[] }> = [];
    const createProvider = createProviderFactory(["feat: should not run"], providerCalls);
    const commitRunner = vi.fn<(options: CommitRunnerOptions) => Promise<CommitRunnerResult>>().mockResolvedValue({
      exitCode: 0,
      stderr: "",
      stdout: ""
    });

    const exitCode = await runRootCommand({
      cwd: repo.path,
      io,
      workflow: {
        commitRunner,
        createCommitMessageProvider: createProvider,
        prompts: createPromptStub({
          async confirmStageAll() {
            return "skip";
          }
        })
      }
    });

    expect(exitCode).toBe(EXIT_CODES.NO_CHANGE);
    expect(providerCalls).toEqual([]);
    expect(commitRunner).not.toHaveBeenCalled();
    expect(io.stdout.read()).toContain("Skipped staging current changes. No commit was created.");
    expect(await repo.git(["diff", "--cached", "--name-only"])).toBe("");
    await repo.cleanup();
  });

  it("generates a dry-run preview without creating a commit", async () => {
    const repo = await createTempGitRepo({ initialCommit: true });
    await setupCliRuntime(repo.path);
    await repo.write("staged.txt", "preview only\n");
    await repo.git(["add", "staged.txt"]);
    const io = createCliIo();
    const providerCalls: Array<{ diff: string; files: string[] }> = [];
    const createProvider = createProviderFactory(["feat: preview staged file"], providerCalls);
    const commitRunner = vi.fn<(options: CommitRunnerOptions) => Promise<CommitRunnerResult>>().mockResolvedValue({
      exitCode: 0,
      stderr: "",
      stdout: ""
    });
    const prompts = {
      confirmStageAll: vi.fn(async () => "skip" as const),
      editMessage: vi.fn(async () => "feat: preview staged file"),
      selectPreviewAction: vi.fn(async () => "confirm" as const)
    };

    const exitCode = await runRootCommand({
      argv: ["--dry-run"],
      cwd: repo.path,
      io,
      workflow: {
        commitRunner,
        createCommitMessageProvider: createProvider,
        prompts
      }
    });

    expect(exitCode).toBe(EXIT_CODES.SUCCESS);
    expect(providerCalls).toHaveLength(1);
    expect(commitRunner).not.toHaveBeenCalled();
    expect(prompts.confirmStageAll).not.toHaveBeenCalled();
    expect(prompts.editMessage).not.toHaveBeenCalled();
    expect(prompts.selectPreviewAction).not.toHaveBeenCalled();
    expect(io.stderr.read()).toBe("");
    expect(io.stdout.read()).toContain("Dry-run preview complete. No commit was created.");
    expect(io.stdout.read()).toContain("feat: preview staged file");
    expect(await repo.git(["rev-list", "--count", "HEAD"])).toBe("1");
    await repo.cleanup();
  });

  it("returns one JSON preview object without prompts or git commit", async () => {
    const repo = await createTempGitRepo({ initialCommit: true });
    await setupCliRuntime(repo.path);
    await repo.write("staged.txt", "json preview\n");
    await repo.git(["add", "staged.txt"]);
    const io = createCliIo();
    const providerCalls: Array<{ diff: string; files: string[] }> = [];
    const createProvider = createProviderFactory(["feat: json preview staged file"], providerCalls);
    const commitRunner = vi.fn<(options: CommitRunnerOptions) => Promise<CommitRunnerResult>>().mockResolvedValue({
      exitCode: 0,
      stderr: "",
      stdout: ""
    });
    const prompts = {
      confirmStageAll: vi.fn(async () => "skip" as const),
      editMessage: vi.fn(async () => "feat: json preview staged file"),
      selectPreviewAction: vi.fn(async () => "confirm" as const)
    };

    const exitCode = await runRootCommand({
      argv: ["--json"],
      cwd: repo.path,
      io,
      workflow: {
        commitRunner,
        createCommitMessageProvider: createProvider,
        prompts
      }
    });

    const stdout = io.stdout.read();
    const payload = JSON.parse(stdout) as {
      committed: boolean;
      dryRun: boolean;
      error: string | null;
      files: Array<{ path: string }>;
      message: string | null;
      ok: boolean;
      status: string;
      warnings: string[];
    };

    expect(exitCode).toBe(EXIT_CODES.SUCCESS);
    expect(providerCalls).toHaveLength(1);
    expect(commitRunner).not.toHaveBeenCalled();
    expect(prompts.confirmStageAll).not.toHaveBeenCalled();
    expect(prompts.editMessage).not.toHaveBeenCalled();
    expect(prompts.selectPreviewAction).not.toHaveBeenCalled();
    expect(io.stderr.read()).toBe("");
    expect(stdout.trim().split(/\r?\n/)).toHaveLength(1);
    expect(payload).toMatchObject({
      committed: false,
      dryRun: false,
      error: null,
      message: "feat: json preview staged file",
      ok: true,
      status: "preview"
    });
    expect(payload.files.map((file) => file.path)).toEqual(["staged.txt"]);
    expect(Array.isArray(payload.warnings)).toBe(true);
    expect(await repo.git(["rev-list", "--count", "HEAD"])).toBe("1");
    await repo.cleanup();
  });

  it("applies root CLI config flags over environment config in the main workflow", async () => {
    const repo = await createTempGitRepo({ initialCommit: true });
    const home = await setupCliRuntime(repo.path);
    await writeUserConfigPatch(
      {
        apiKey: "user-api-key",
        baseURL: "https://user.example/v1",
        customPrompt: "Use user prompt.",
        model: "user-model",
        provider: "anthropic-messages"
      },
      { cwd: repo.path, env: { ...process.env, CNM_HOME: home } }
    );
    await writeFile(
      join(repo.path, ".cnmrc.json"),
      JSON.stringify(
        {
          baseURL: "https://project.example/v1",
          customPrompt: "Use project prompt.",
          model: "project-model",
          provider: "google-gemini"
        },
        null,
        2
      ),
      "utf8"
    );
    process.env.CNM_PROVIDER = "openai-responses";
    process.env.CNM_MODEL = "env-model";
    process.env.CNM_BASE_URL = "https://env.example/v1";
    process.env.CNM_CUSTOM_PROMPT = "Use env prompt.";
    await repo.write("staged.txt", "flag override preview\n");
    await repo.git(["add", "staged.txt"]);
    const io = createCliIo();
    const providerConfigs: ProviderConfig[] = [];
    const providerInputs: GenerateCommitMessageInput[] = [];
    const createProvider = vi.fn<CommitWorkflowDependencies["createCommitMessageProvider"]>().mockImplementation((config) => {
      providerConfigs.push(config);

      return {
        async generateCommitMessage(input) {
          providerInputs.push(input);

          return {
            message: "feat: preview flag overrides",
            metadata: {
              model: config.model,
              provider: config.provider
            }
          };
        },
        name: config.provider
      };
    });
    const commitRunner = vi.fn<(options: CommitRunnerOptions) => Promise<CommitRunnerResult>>().mockResolvedValue({
      exitCode: 0,
      stderr: "",
      stdout: ""
    });

    const exitCode = await runRootCommand({
      argv: [
        "--dry-run",
        "--provider",
        "openai-compatible",
        "--model",
        "flag-model",
        "--base-url",
        "https://flag.example/v1",
        "--prompt-style",
        "conventional",
        "--custom-prompt",
        "Prefer user-facing wording."
      ],
      cwd: repo.path,
      io,
      workflow: {
        commitRunner,
        createCommitMessageProvider: createProvider,
        prompts: createPromptStub({
          async selectPreviewAction() {
            return "confirm";
          }
        })
      }
    });

    expect(exitCode).toBe(EXIT_CODES.SUCCESS);
    expect(providerConfigs).toEqual([
      {
        apiKey: "test-api-key",
        baseURL: "https://flag.example/v1",
        model: "flag-model",
        provider: "openai-compatible"
      }
    ]);
    expect(providerInputs).toHaveLength(1);
    expect(providerInputs[0]).toMatchObject({
      customPrompt: "Prefer user-facing wording.",
      messageStyle: "conventional"
    });
    expect(commitRunner).not.toHaveBeenCalled();
    await repo.cleanup();
  });

  it("blocks non-TTY commits before provider generation when confirmation is required", async () => {
    const repo = await createTempGitRepo({ initialCommit: true });
    await setupCliRuntime(repo.path);
    await repo.write("staged.txt", "non tty commit requires confirmation\n");
    await repo.git(["add", "staged.txt"]);
    const io = createCliIo();
    const providerCalls: Array<{ diff: string; files: string[] }> = [];
    const createProvider = createProviderFactory(["feat: should not generate"], providerCalls);
    const commitRunner = vi.fn<(options: CommitRunnerOptions) => Promise<CommitRunnerResult>>().mockResolvedValue({
      exitCode: 0,
      stderr: "",
      stdout: ""
    });

    const exitCode = await runRootCommand({
      cwd: repo.path,
      io,
      isTty: false,
      workflow: {
        commitRunner,
        createCommitMessageProvider: createProvider,
        prompts: createPromptStub({
          async selectPreviewAction() {
            return "confirm";
          }
        })
      }
    });

    expect(exitCode).toBe(EXIT_CODES.ERROR);
    expect(providerCalls).toEqual([]);
    expect(commitRunner).not.toHaveBeenCalled();
    expect(io.stderr.read()).toContain("Interactive confirmation is required before creating a commit.");
    expect(await repo.git(["rev-list", "--count", "HEAD"])).toBe("1");
    await repo.cleanup();
  });

  it("warns about staged secrets without blocking provider generation or commit execution", async () => {
    const repo = await createTempGitRepo({ initialCommit: true });
    await setupCliRuntime(repo.path);
    await repo.write("secret.txt", "api_key = 'sk_1234567890abcdef1234567890abcdef'\n");
    await repo.git(["add", "secret.txt"]);
    const io = createCliIo();
    const providerCalls: Array<{ diff: string; files: string[] }> = [];
    const createProvider = createProviderFactory(["feat: commit secret fixture"], providerCalls);
    const commitRunner = vi.fn<(options: CommitRunnerOptions) => Promise<CommitRunnerResult>>().mockResolvedValue({
      exitCode: 0,
      stderr: "",
      stdout: ""
    });
    const selectPreviewAction = vi.fn<CommitWorkflowPrompts["selectPreviewAction"]>().mockResolvedValue("confirm");
    const prompts = createPromptStub({ selectPreviewAction });

    const exitCode = await runRootCommand({
      cwd: repo.path,
      io,
      workflow: {
        commitRunner,
        createCommitMessageProvider: createProvider,
        prompts
      }
    });

    expect(exitCode).toBe(EXIT_CODES.SUCCESS);
    expect(providerCalls).toHaveLength(1);
    expect(providerCalls[0]?.diff).toContain("secret.txt");
    expect(commitRunner).toHaveBeenCalledWith(expect.objectContaining({
      message: "feat: commit secret fixture"
    }));
    expect(io.stderr.read()).toBe("");
    expect(selectPreviewAction).toHaveBeenCalledWith(expect.objectContaining({
      warnings: expect.arrayContaining(["Potential secrets were found in the staged diff."])
    }));
    expect(await repo.git(["rev-list", "--count", "HEAD"])).toBe("1");
    await repo.cleanup();
  });

  it("surfaces pre-commit hook failures without retrying", async () => {
    const repo = await createTempGitRepo({ initialCommit: true });
    await setupCliRuntime(repo.path);
    await repo.write("staged.txt", "hook test\n");
    await repo.git(["add", "staged.txt"]);
    const hookPath = join(repo.path, ".git", "hooks", "pre-commit");
    await mkdir(join(repo.path, ".git", "hooks"), { recursive: true });
    await writeFile(hookPath, "#!/bin/sh\necho 'blocked by hook' >&2\nexit 1\n", "utf8");
    await chmod(hookPath, 0o755);
    const io = createCliIo();
    const providerCalls: Array<{ diff: string; files: string[] }> = [];
    const createProvider = createProviderFactory(["feat: hook failure"], providerCalls);

    const exitCode = await runRootCommand({
      cwd: repo.path,
      io,
      workflow: {
        createCommitMessageProvider: createProvider,
        prompts: createPromptStub({
          async selectPreviewAction() {
            return "confirm";
          }
        })
      }
    });

    expect(exitCode).toBe(EXIT_CODES.ERROR);
    expect(providerCalls).toHaveLength(1);
    expect(io.stderr.read()).toContain("git commit failed");
    expect(io.stderr.read()).toContain("blocked by hook");
    expect(await repo.git(["rev-list", "--count", "HEAD"])).toBe("1");
    await repo.cleanup();
  });
});
