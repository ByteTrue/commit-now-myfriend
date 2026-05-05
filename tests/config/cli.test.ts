import { mkdir, mkdtemp, readFile, writeFile } from "node:fs/promises";
import { tmpdir } from "node:os";
import { join } from "node:path";

import { afterEach, describe, expect, it, vi } from "vitest";

import type { ConfigPanelPrompts } from "../../src/commands/config-panel.js";
import { runCli } from "../../src/cli.js";
import { EXIT_CODES, type CliWriteStream } from "../../src/output/index.js";

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

const originalCwd = process.cwd();
const originalCnmHome = process.env.CNM_HOME;

afterEach(() => {
  process.chdir(originalCwd);

  if (originalCnmHome === undefined) {
    delete process.env.CNM_HOME;
    return;
  }

  process.env.CNM_HOME = originalCnmHome;
});

async function setupCliRuntime() {
  const cwd = await mkdtemp(join(tmpdir(), "cnm-cli-workspace-"));
  const home = join(cwd, ".cnm-home");

  process.chdir(cwd);
  process.env.CNM_HOME = home;

  return { cwd, home };
}

async function writeUserConfigFile(home: string, config: Record<string, unknown>) {
  await mkdir(home, { recursive: true });
  await writeFile(join(home, "config.json"), JSON.stringify(config, null, 2), "utf8");
}

function createConfigPromptStub(overrides: Partial<ConfigPanelPrompts> = {}): ConfigPanelPrompts {
  return {
    confirmReset: vi.fn(async () => false),
    inputApiKey: vi.fn(async () => "test-panel-api-key"),
    inputBaseURL: vi.fn(async () => "https://panel.example/v1"),
    inputCustomPrompt: vi.fn(async () => "Prefer concise summaries."),
    inputModel: vi.fn(async ({ currentValue }) => currentValue),
    selectAction: vi.fn(async () => "exit" as const),
    selectPromptStyle: vi.fn(async () => "auto" as const),
    selectProvider: vi.fn(async ({ currentProvider }) => currentProvider),
    selectResetTarget: vi.fn(async () => "all" as const),
    ...overrides
  };
}

async function runConfigCli(options: {
  argv: string[];
  isTty?: boolean;
  prompts?: ConfigPanelPrompts;
}) {
  const io = createCliIo();
  const exitCode = await runCli({
    argv: options.argv,
    configRuntime: {
      isTty: options.isTty,
      prompts: options.prompts
    },
    stderr: io.stderr.stream,
    stdout: io.stdout.stream,
    version: "0.1.0"
  });

  return { exitCode, io };
}

describe("config CLI", () => {
  it("initializes config in temp CNM_HOME and warns about plaintext api keys", async () => {
    const runtime = await setupCliRuntime();
    const io = createCliIo();

    const exitCode = await runCli({
      argv: ["init", "--provider", "google-gemini", "--api-key", "test-init-api-key"],
      version: "0.1.0",
      stderr: io.stderr.stream,
      stdout: io.stdout.stream
    });

    const storedConfig = JSON.parse(await readFile(join(runtime.home, "config.json"), "utf8")) as {
      apiKey: string;
      provider: string;
    };

    expect(exitCode).toBe(EXIT_CODES.SUCCESS);
    expect(io.stderr.read()).toContain("stored in plaintext");
    expect(io.stdout.read()).toContain(`Initialized user config at ${join(runtime.home, "config.json")}.`);
    expect(storedConfig.provider).toBe("google-gemini");
    expect(storedConfig.apiKey).toBe("test-init-api-key");
  });

  it("walks interactive init through required AI configuration", async () => {
    const runtime = await setupCliRuntime();
    const rawApiKey = "test-interactive-init-api-key";
    const prompts = createConfigPromptStub({
      confirmSetOptionalConfig: vi.fn<ConfigPanelPrompts["confirmSetOptionalConfig"]>().mockResolvedValue(true),
      inputApiKey: vi.fn<ConfigPanelPrompts["inputApiKey"]>().mockResolvedValue(rawApiKey),
      inputBaseURL: vi.fn<ConfigPanelPrompts["inputBaseURL"]>().mockResolvedValue("https://interactive.example/v1"),
      inputCustomPrompt: vi.fn<ConfigPanelPrompts["inputCustomPrompt"]>().mockResolvedValue("Prefer short imperative messages."),
      inputModel: vi.fn<ConfigPanelPrompts["inputModel"]>().mockResolvedValue("local-model"),
      selectPromptStyle: vi.fn<ConfigPanelPrompts["selectPromptStyle"]>().mockResolvedValue("custom"),
      selectProvider: vi.fn<ConfigPanelPrompts["selectProvider"]>().mockResolvedValue("openai-compatible")
    });
    const io = createCliIo();

    const exitCode = await runCli({
      argv: ["init"],
      configRuntime: {
        isTty: true,
        prompts
      },
      version: "0.1.0",
      stderr: io.stderr.stream,
      stdout: io.stdout.stream
    });

    const storedConfig = JSON.parse(await readFile(join(runtime.home, "config.json"), "utf8")) as Record<string, string>;

    expect(exitCode).toBe(EXIT_CODES.SUCCESS);
    expect(storedConfig).toEqual({
      apiKey: rawApiKey,
      baseURL: "https://interactive.example/v1",
      customPrompt: "Prefer short imperative messages.",
      model: "local-model",
      promptStyle: "custom",
      provider: "openai-compatible"
    });
    expect(io.stderr.read()).toContain("stored in plaintext");
    expect(io.stderr.read()).not.toContain(rawApiKey);
    expect(io.stdout.read()).toContain("Let's set up cnm for AI-assisted commits.");
    expect(io.stdout.read()).toContain("apiKey=[redacted]");
    expect(io.stdout.read()).toContain("Next: stage changes and run `cnm`");
    expect(io.stdout.read()).not.toContain(rawApiKey);
  });

  it("redacts api keys from config get json output", async () => {
    await setupCliRuntime();
    const setIo = createCliIo();
    const getIo = createCliIo();

    const setExitCode = await runCli({
      argv: ["config", "set", "apiKey", "test-live-api-key"],
      version: "0.1.0",
      stderr: setIo.stderr.stream,
      stdout: setIo.stdout.stream
    });

    const getExitCode = await runCli({
      argv: ["config", "get", "--json"],
      version: "0.1.0",
      stderr: getIo.stderr.stream,
      stdout: getIo.stdout.stream
    });

    const payload = JSON.parse(getIo.stdout.read()) as {
      apiKey: string | null;
      model: string;
      provider: string;
    };

    expect(setExitCode).toBe(EXIT_CODES.SUCCESS);
    expect(setIo.stderr.read()).toContain("stored in plaintext");
    expect(setIo.stdout.read()).not.toContain("test-live-api-key");
    expect(getExitCode).toBe(EXIT_CODES.SUCCESS);
    expect(getIo.stderr.read()).toBe("");
    expect(payload.apiKey).toBe("[redacted]");
    expect(getIo.stdout.read()).not.toContain("test-live-api-key");
  });

  it("warns and ignores project config secrets in effective output", async () => {
    const runtime = await setupCliRuntime();
    const io = createCliIo();

    await writeFile(
      join(runtime.cwd, ".cnmrc.json"),
      JSON.stringify(
        {
          apiKey: "test-project-api-key",
          model: "project-model"
        },
        null,
        2,
      ),
      "utf8",
    );

    const exitCode = await runCli({
      argv: ["config", "get", "--json"],
      version: "0.1.0",
      stderr: io.stderr.stream,
      stdout: io.stdout.stream
    });

    const payload = JSON.parse(io.stdout.read()) as { apiKey: string | null; model: string };

    expect(exitCode).toBe(EXIT_CODES.SUCCESS);
    expect(io.stderr.read()).toContain("project-level secrets are ignored");
    expect(io.stderr.read()).not.toContain("test-project-api-key");
    expect(payload.model).toBe("project-model");
    expect(payload.apiKey).toBeNull();
    expect(io.stdout.read()).not.toContain("test-project-api-key");
  });

  it("keeps bare json output scriptable and never calls panel prompts", async () => {
    await setupCliRuntime();
    const prompts = createConfigPromptStub();
    const setResult = await runConfigCli({
      argv: ["config", "set", "apiKey", "test-json-api-key"],
      isTty: true,
      prompts
    });
    const configJsonResult = await runConfigCli({ argv: ["config", "--json"], isTty: true, prompts });
    const rootJsonResult = await runConfigCli({ argv: ["--json", "config"], isTty: true, prompts });
    const configPayload = JSON.parse(configJsonResult.io.stdout.read()) as { apiKey: string | null };
    const rootPayload = JSON.parse(rootJsonResult.io.stdout.read()) as { apiKey: string | null };

    expect(setResult.exitCode).toBe(EXIT_CODES.SUCCESS);
    expect(configJsonResult.exitCode).toBe(EXIT_CODES.SUCCESS);
    expect(rootJsonResult.exitCode).toBe(EXIT_CODES.SUCCESS);
    expect(configPayload.apiKey).toBe("[redacted]");
    expect(rootPayload.apiKey).toBe("[redacted]");
    expect(configJsonResult.io.stdout.read()).not.toContain("test-json-api-key");
    expect(rootJsonResult.io.stdout.read()).not.toContain("test-json-api-key");
    expect(prompts.selectAction).not.toHaveBeenCalled();
  });

  it("keeps get list set unset and bare dry-run away from panel prompts", async () => {
    await setupCliRuntime();
    const prompts = createConfigPromptStub();

    const setResult = await runConfigCli({
      argv: ["config", "set", "provider", "google-gemini"],
      isTty: true,
      prompts
    });
    const getResult = await runConfigCli({ argv: ["config", "get", "provider"], isTty: true, prompts });
    const listResult = await runConfigCli({ argv: ["config", "list", "--json"], isTty: true, prompts });
    const unsetResult = await runConfigCli({ argv: ["config", "unset", "provider"], isTty: true, prompts });
    const dryRunResult = await runConfigCli({ argv: ["config", "--dry-run"], isTty: true, prompts });

    expect(setResult.exitCode).toBe(EXIT_CODES.SUCCESS);
    expect(getResult.exitCode).toBe(EXIT_CODES.SUCCESS);
    expect(getResult.io.stdout.read()).toContain("provider=google-gemini");
    expect(listResult.exitCode).toBe(EXIT_CODES.SUCCESS);
    expect(JSON.parse(listResult.io.stdout.read())).toMatchObject({ provider: "google-gemini" });
    expect(unsetResult.exitCode).toBe(EXIT_CODES.SUCCESS);
    expect(dryRunResult.exitCode).toBe(EXIT_CODES.SUCCESS);
    expect(dryRunResult.io.stdout.read()).toContain("provider=openai-responses");
    expect(prompts.selectAction).not.toHaveBeenCalled();
    expect(prompts.inputApiKey).not.toHaveBeenCalled();
  });

  it("returns success and creates no config file when the panel exits immediately", async () => {
    const runtime = await setupCliRuntime();
    const prompts = createConfigPromptStub({
      selectAction: vi.fn<ConfigPanelPrompts["selectAction"]>().mockResolvedValue("exit")
    });

    const result = await runConfigCli({ argv: ["config"], isTty: true, prompts });

    expect(result.exitCode).toBe(EXIT_CODES.SUCCESS);
    await expect(readFile(join(runtime.home, "config.json"), "utf8")).rejects.toMatchObject({ code: "ENOENT" });
  });

  it("writes provider and model from the interactive panel", async () => {
    const runtime = await setupCliRuntime();
    const prompts = createConfigPromptStub({
      inputModel: vi.fn<ConfigPanelPrompts["inputModel"]>().mockResolvedValue("gemini-2.5-pro"),
      selectAction: vi.fn<ConfigPanelPrompts["selectAction"]>()
        .mockResolvedValueOnce("configureProviderModel")
        .mockResolvedValueOnce("exit"),
      selectProvider: vi.fn<ConfigPanelPrompts["selectProvider"]>().mockResolvedValue("google-gemini")
    });

    const result = await runConfigCli({ argv: ["config"], isTty: true, prompts });
    const storedConfig = JSON.parse(await readFile(join(runtime.home, "config.json"), "utf8")) as {
      model: string;
      provider: string;
    };

    expect(result.exitCode).toBe(EXIT_CODES.SUCCESS);
    expect(storedConfig).toEqual({ model: "gemini-2.5-pro", provider: "google-gemini" });
  });

  it("writes api keys from the panel, warns about plaintext storage, and never prints the raw key", async () => {
    const runtime = await setupCliRuntime();
    const rawApiKey = "test-panel-live-api-key";
    const prompts = createConfigPromptStub({
      inputApiKey: vi.fn<ConfigPanelPrompts["inputApiKey"]>().mockResolvedValue(rawApiKey),
      selectAction: vi.fn<ConfigPanelPrompts["selectAction"]>()
        .mockResolvedValueOnce("setApiKey")
        .mockResolvedValueOnce("exit")
    });

    const result = await runConfigCli({ argv: ["config"], isTty: true, prompts });
    const storedConfig = JSON.parse(await readFile(join(runtime.home, "config.json"), "utf8")) as { apiKey: string };

    expect(result.exitCode).toBe(EXIT_CODES.SUCCESS);
    expect(storedConfig.apiKey).toBe(rawApiKey);
    expect(result.io.stderr.read()).toContain("stored in plaintext");
    expect(result.io.stdout.read()).toContain("apiKey=[redacted]");
    expect(result.io.stdout.read()).not.toContain(rawApiKey);
    expect(result.io.stderr.read()).not.toContain(rawApiKey);
  });

  it("writes baseURL and customPrompt from the panel", async () => {
    const runtime = await setupCliRuntime();
    const prompts = createConfigPromptStub({
      inputBaseURL: vi.fn<ConfigPanelPrompts["inputBaseURL"]>().mockResolvedValue("https://local.example/v1"),
      inputCustomPrompt: vi.fn<ConfigPanelPrompts["inputCustomPrompt"]>().mockResolvedValue("Use a direct tone."),
      selectAction: vi.fn<ConfigPanelPrompts["selectAction"]>()
        .mockResolvedValueOnce("setBaseURL")
        .mockResolvedValueOnce("setCustomPrompt")
        .mockResolvedValueOnce("exit")
    });

    const result = await runConfigCli({ argv: ["config"], isTty: true, prompts });
    const storedConfig = JSON.parse(await readFile(join(runtime.home, "config.json"), "utf8")) as {
      baseURL: string;
      customPrompt: string;
    };

    expect(result.exitCode).toBe(EXIT_CODES.SUCCESS);
    expect(storedConfig).toEqual({
      baseURL: "https://local.example/v1",
      customPrompt: "Use a direct tone."
    });
  });

  it("writes promptStyle from the panel", async () => {
    const runtime = await setupCliRuntime();
    const prompts = createConfigPromptStub({
      selectAction: vi.fn<ConfigPanelPrompts["selectAction"]>()
        .mockResolvedValueOnce("setPromptStyle")
        .mockResolvedValueOnce("exit"),
      selectPromptStyle: vi.fn<ConfigPanelPrompts["selectPromptStyle"]>().mockResolvedValue("google")
    });

    const result = await runConfigCli({ argv: ["config"], isTty: true, prompts });
    const storedConfig = JSON.parse(await readFile(join(runtime.home, "config.json"), "utf8")) as {
      promptStyle: string;
    };

    expect(result.exitCode).toBe(EXIT_CODES.SUCCESS);
    expect(storedConfig.promptStyle).toBe("google");
    expect(result.io.stdout.read()).toContain("promptStyle=google");
  });

  it("shows redacted effective config from the panel", async () => {
    const runtime = await setupCliRuntime();

    await writeUserConfigFile(runtime.home, { apiKey: "test-view-api-key", provider: "openai-responses" });

    const prompts = createConfigPromptStub({
      selectAction: vi.fn<ConfigPanelPrompts["selectAction"]>()
        .mockResolvedValueOnce("viewEffectiveConfig")
        .mockResolvedValueOnce("exit")
    });
    const result = await runConfigCli({ argv: ["config"], isTty: true, prompts });

    expect(result.exitCode).toBe(EXIT_CODES.SUCCESS);
    expect(result.io.stdout.read()).toContain("apiKey=[redacted]");
    expect(result.io.stdout.read()).not.toContain("test-view-api-key");
  });

  it("reports missing api keys during local config tests without making network calls", async () => {
    await setupCliRuntime();
    const prompts = createConfigPromptStub({
      selectAction: vi.fn<ConfigPanelPrompts["selectAction"]>()
        .mockResolvedValueOnce("testCurrentConfig")
        .mockResolvedValueOnce("exit")
    });

    const result = await runConfigCli({ argv: ["config"], isTty: true, prompts });

    expect(result.exitCode).toBe(EXIT_CODES.SUCCESS);
    expect(result.io.stderr.read()).toContain("No API key is configured for openai-responses.");
    expect(result.io.stdout.read()).not.toContain("provider request was sent");
  });

  it("reports missing baseURL for openai-compatible during local config tests", async () => {
    const runtime = await setupCliRuntime();

    await writeUserConfigFile(runtime.home, {
      apiKey: "test-openai-compatible-key",
      provider: "openai-compatible"
    });

    const prompts = createConfigPromptStub({
      selectAction: vi.fn<ConfigPanelPrompts["selectAction"]>()
        .mockResolvedValueOnce("testCurrentConfig")
        .mockResolvedValueOnce("exit")
    });
    const result = await runConfigCli({ argv: ["config"], isTty: true, prompts });

    expect(result.exitCode).toBe(EXIT_CODES.SUCCESS);
    expect(result.io.stderr.read()).toContain("The openai-compatible provider requires `baseURL` to be configured.");
  });

  it("passes local config tests when the current config is complete", async () => {
    const runtime = await setupCliRuntime();

    await writeUserConfigFile(runtime.home, {
      apiKey: "test-complete-config-key",
      baseURL: "https://complete.example/v1",
      model: "gpt-5-mini",
      provider: "openai-compatible"
    });

    const prompts = createConfigPromptStub({
      selectAction: vi.fn<ConfigPanelPrompts["selectAction"]>()
        .mockResolvedValueOnce("testCurrentConfig")
        .mockResolvedValueOnce("exit")
    });
    const result = await runConfigCli({ argv: ["config"], isTty: true, prompts });

    expect(result.exitCode).toBe(EXIT_CODES.SUCCESS);
    expect(result.io.stdout.read()).toContain("Config check passed. No provider request was sent.");
    expect(result.io.stderr.read()).not.toContain("Config check failed");
  });

  it("leaves user config unchanged when reset confirmation is declined", async () => {
    const runtime = await setupCliRuntime();

    await writeUserConfigFile(runtime.home, { provider: "google-gemini" });

    const prompts = createConfigPromptStub({
      confirmReset: vi.fn<ConfigPanelPrompts["confirmReset"]>().mockResolvedValue(false),
      selectAction: vi.fn<ConfigPanelPrompts["selectAction"]>()
        .mockResolvedValueOnce("resetUnset")
        .mockResolvedValueOnce("exit"),
      selectResetTarget: vi.fn<ConfigPanelPrompts["selectResetTarget"]>().mockResolvedValue("provider")
    });

    const result = await runConfigCli({ argv: ["config"], isTty: true, prompts });
    const storedConfig = JSON.parse(await readFile(join(runtime.home, "config.json"), "utf8")) as { provider: string };

    expect(result.exitCode).toBe(EXIT_CODES.SUCCESS);
    expect(result.io.stdout.read()).toContain("Reset cancelled. User config unchanged.");
    expect(storedConfig).toEqual({ provider: "google-gemini" });
  });

  it("unsets a selected key and can clear the full user config", async () => {
    const runtime = await setupCliRuntime();

    await writeUserConfigFile(runtime.home, {
      apiKey: "test-reset-key",
      baseURL: "https://before-reset.example/v1",
      provider: "openai-compatible"
    });

    const unsetPrompts = createConfigPromptStub({
      confirmReset: vi.fn<ConfigPanelPrompts["confirmReset"]>().mockResolvedValue(true),
      selectAction: vi.fn<ConfigPanelPrompts["selectAction"]>()
        .mockResolvedValueOnce("resetUnset")
        .mockResolvedValueOnce("exit"),
      selectResetTarget: vi.fn<ConfigPanelPrompts["selectResetTarget"]>().mockResolvedValue("baseURL")
    });
    const unsetResult = await runConfigCli({ argv: ["config"], isTty: true, prompts: unsetPrompts });
    const afterUnset = JSON.parse(await readFile(join(runtime.home, "config.json"), "utf8")) as {
      apiKey: string;
      provider: string;
    };

    expect(unsetResult.exitCode).toBe(EXIT_CODES.SUCCESS);
    expect(afterUnset).toEqual({ apiKey: "test-reset-key", provider: "openai-compatible" });

    const clearPrompts = createConfigPromptStub({
      confirmReset: vi.fn<ConfigPanelPrompts["confirmReset"]>().mockResolvedValue(true),
      selectAction: vi.fn<ConfigPanelPrompts["selectAction"]>()
        .mockResolvedValueOnce("resetUnset")
        .mockResolvedValueOnce("exit"),
      selectResetTarget: vi.fn<ConfigPanelPrompts["selectResetTarget"]>().mockResolvedValue("all")
    });
    const clearResult = await runConfigCli({ argv: ["config"], isTty: true, prompts: clearPrompts });
    const afterClear = JSON.parse(await readFile(join(runtime.home, "config.json"), "utf8")) as Record<string, unknown>;

    expect(clearResult.exitCode).toBe(EXIT_CODES.SUCCESS);
    expect(afterClear).toEqual({});
  });

  it("falls back to human effective config output for bare non-tty config", async () => {
    const runtime = await setupCliRuntime();

    await writeUserConfigFile(runtime.home, { provider: "google-gemini" });

    const result = await runConfigCli({ argv: ["config"], isTty: false });

    expect(result.exitCode).toBe(EXIT_CODES.SUCCESS);
    expect(result.io.stdout.read()).toContain("provider=google-gemini");
  });

  it("returns exit code 130 when the panel is cancelled", async () => {
    await setupCliRuntime();
    const prompts = createConfigPromptStub({
      selectAction: vi.fn<ConfigPanelPrompts["selectAction"]>().mockResolvedValue(null)
    });

    const result = await runConfigCli({ argv: ["config"], isTty: true, prompts });

    expect(result.exitCode).toBe(EXIT_CODES.USER_CANCEL);
  });
});
