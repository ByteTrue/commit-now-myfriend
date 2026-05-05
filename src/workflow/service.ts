import { mkdtemp, rm, writeFile } from "node:fs/promises";
import { tmpdir } from "node:os";
import path from "node:path";

import { execa } from "execa";

import type { EffectiveConfig } from "../config/index.js";
import { ProviderError } from "../providers/index.js";
import { missingApiKey, missingConfig } from "../providers/errors.js";
import type {
  CommitMessageResult,
  GenerateCommitMessageInput,
  ProviderConfig
} from "../providers/index.js";
import { EXIT_CODES } from "../output/index.js";
import { validateEditedMessage } from "./prompts.js";
import type {
  CommitRunnerOptions,
  CommitRunnerResult,
  CommitWorkflowDependencies,
  CommitWorkflowResult,
  RunCommitWorkflowOptions,
  WorkflowFileView
} from "./types.js";
import { toWorkflowFileView } from "./types.js";

function createResult(
  partial: Omit<CommitWorkflowResult, "command" | "previewShown"> & { previewShown?: boolean }
): CommitWorkflowResult {
  return {
    command: "cnm",
    previewShown: partial.previewShown ?? false,
    ...partial
  };
}

function summarizeWarnings(warnings: string[]): string[] {
  return warnings.map((warning) => warning.trim()).filter((warning) => warning.length > 0);
}

function summarizeCommitFailure(result: CommitRunnerResult): string {
  const output = [result.stderr.trim(), result.stdout.trim()].filter((value) => value.length > 0).join("\n");

  if (output.length === 0) {
    return "git commit failed.";
  }

  return `git commit failed.\n${output}`;
}

function resolveProviderConfig(config: EffectiveConfig): ProviderConfig {
  if (!config.apiKey || config.apiKey.trim().length === 0) {
    throw missingApiKey(config.provider);
  }

  if (!config.model || config.model.trim().length === 0) {
    throw missingConfig(config.provider, "model");
  }

  if (config.provider === "openai-compatible") {
    if (!config.baseURL || config.baseURL.trim().length === 0) {
      throw missingConfig(config.provider, "baseURL");
    }

    return {
      apiKey: config.apiKey,
      baseURL: config.baseURL,
      model: config.model,
      provider: config.provider
    };
  }

  return {
    apiKey: config.apiKey,
    model: config.model,
    provider: config.provider
  };
}

function createProviderInput(files: WorkflowFileView[], inspection: Awaited<ReturnType<CommitWorkflowDependencies["inspectGitRepository"]>>): GenerateCommitMessageInput {
  return {
    customPrompt: undefined,
    diff: inspection.stagedDiff,
    files: files.map((file) => ({
      path: file.path,
      status: file.staged ?? (file.untracked ? "untracked" : undefined)
    })),
    messageStyle: undefined,
    repo: {
      branch: inspection.repository.branchName ?? undefined,
      root: inspection.repository.rootPath ?? undefined
    }
  };
}

function applyConfigToProviderInput(input: GenerateCommitMessageInput, config: EffectiveConfig): GenerateCommitMessageInput {
  return {
    ...input,
    customPrompt: config.customPrompt,
    messageStyle: config.promptStyle
  };
}

function buildNoChangeResult(dryRun: boolean, files: WorkflowFileView[], warnings: string[], message: string): CommitWorkflowResult {
  return createResult({
    committed: false,
    dryRun,
    error: null,
    exitCode: EXIT_CODES.NO_CHANGE,
    files,
    message: null,
    ok: true,
    provider: null,
    status: "no_changes",
    warnings: warnings.length === 0 ? [message] : [...warnings, message]
  });
}

function buildBlockedInspectionResult(
  dryRun: boolean,
  files: WorkflowFileView[],
  warnings: string[],
  inspection: Awaited<ReturnType<CommitWorkflowDependencies["inspectGitRepository"]>>
): CommitWorkflowResult {
  return createResult({
    committed: false,
    dryRun,
    error: inspection.blockingIssues.map((issue) => issue.message).join("\n"),
    exitCode: EXIT_CODES.ERROR,
    files,
    message: null,
    ok: false,
    provider: null,
    status: "blocked",
    warnings
  });
}

function buildPreviewResult(options: {
  dryRun: boolean;
  files: WorkflowFileView[];
  message: string;
  provider: NonNullable<CommitWorkflowResult["provider"]>;
  warnings: string[];
}): CommitWorkflowResult {
  return createResult({
    committed: false,
    dryRun: options.dryRun,
    error: null,
    exitCode: options.dryRun ? EXIT_CODES.DRY_RUN : EXIT_CODES.SUCCESS,
    files: options.files,
    message: options.message,
    ok: true,
    provider: options.provider,
    status: options.dryRun ? "dry_run" : "preview",
    warnings: options.warnings
  });
}

async function executeGitCommit({ cwd, env, message }: CommitRunnerOptions): Promise<CommitRunnerResult> {
  const tempDirectory = await mkdtemp(path.join(tmpdir(), "cnm-commit-"));
  const messagePath = path.join(tempDirectory, "COMMIT_EDITMSG");

  try {
    await writeFile(messagePath, `${message.trim()}\n`, "utf8");

    const result = await execa("git", ["commit", "-F", messagePath], {
      cwd,
      env,
      reject: false,
      stderr: "pipe",
      stdin: "ignore",
      stdout: "pipe"
    });

    return {
      exitCode: result.exitCode ?? 1,
      stderr: result.stderr,
      stdout: result.stdout
    };
  } finally {
    await rm(tempDirectory, { force: true, recursive: true });
  }
}

async function generateMessage(
  dependencies: CommitWorkflowDependencies,
  providerConfig: ProviderConfig,
  providerInput: GenerateCommitMessageInput
): Promise<CommitMessageResult> {
  const provider = dependencies.createCommitMessageProvider(providerConfig);
  return provider.generateCommitMessage(providerInput);
}

function createProviderInfo(result: CommitMessageResult) {
  return {
    model: result.metadata.model,
    name: result.metadata.provider
  };
}

function toWarningMessages(result: Awaited<ReturnType<CommitWorkflowDependencies["resolveEffectiveConfig"]>>, inspection: Awaited<ReturnType<CommitWorkflowDependencies["inspectGitRepository"]>>): string[] {
  return summarizeWarnings([
    ...result.warnings,
    ...inspection.warnings.map((warning) => warning.message)
  ]);
}

export async function runCommitWorkflow({
  cwd,
  dependencies,
  dryRun,
  env,
  flagOverrides,
  isTty,
  json
}: RunCommitWorkflowOptions): Promise<CommitWorkflowResult> {
  const resolvedConfig = await dependencies.resolveEffectiveConfig({ cwd, env, flagOverrides });
  let inspection = await dependencies.inspectGitRepository({ cwd, env });
  let warnings = toWarningMessages(resolvedConfig, inspection);
  const initialFiles = inspection.files.map(toWorkflowFileView);

  if (inspection.blockingIssues.length > 0) {
    return buildBlockedInspectionResult(dryRun, initialFiles, warnings, inspection);
  }

  if (!inspection.hasStagedChanges) {
    if (inspection.files.length === 0) {
      return buildNoChangeResult(dryRun, initialFiles, warnings, "No staged or working tree changes were found.");
    }

    if (json || !isTty) {
      return buildNoChangeResult(
        dryRun,
        initialFiles,
        warnings,
        json
          ? "No staged changes found. Stage files manually first to preview a commit message in JSON mode."
          : "No staged changes found. Re-run in a TTY to stage all changes, or stage files manually first."
      );
    }

    const stageDecision = await dependencies.prompts.confirmStageAll({ files: initialFiles });

    if (stageDecision === "cancel") {
      return createResult({
        committed: false,
        dryRun,
        error: null,
        exitCode: EXIT_CODES.USER_CANCEL,
        files: initialFiles,
        message: null,
        ok: true,
        previewShown: false,
        provider: null,
        status: "cancelled",
        warnings
      });
    }

    if (stageDecision === "skip") {
      return buildNoChangeResult(dryRun, initialFiles, warnings, "Skipped staging current changes. No commit was created.");
    }

    const stagedAll = await dependencies.stageAllChanges({ cwd, confirmed: true, env, isTty: true });
    inspection = stagedAll.inspection;
    warnings = toWarningMessages(resolvedConfig, inspection);

    if (inspection.blockingIssues.length > 0) {
      return buildBlockedInspectionResult(dryRun, inspection.files.map(toWorkflowFileView), warnings, inspection);
    }
  }

  const stagedFiles = inspection.stagedFiles.map(toWorkflowFileView);

  if (!inspection.hasStagedChanges) {
    return buildNoChangeResult(dryRun, stagedFiles, warnings, "No staged changes are available to commit.");
  }

  if (!dryRun && !json && !isTty) {
    return createResult({
      committed: false,
      dryRun,
      error: "Interactive confirmation is required before creating a commit.",
      exitCode: EXIT_CODES.ERROR,
      files: stagedFiles,
      message: null,
      ok: false,
      provider: null,
      status: "blocked",
      warnings
    });
  }

  let providerConfig: ProviderConfig;

  try {
    providerConfig = resolveProviderConfig(resolvedConfig.values);
  } catch (error) {
    const message = error instanceof ProviderError ? error.message : "Provider configuration is invalid.";

    return createResult({
      committed: false,
      dryRun,
      error: message,
      exitCode: EXIT_CODES.ERROR,
      files: stagedFiles,
      message: null,
      ok: false,
      provider: null,
      status: "blocked",
      warnings
    });
  }

  const providerInput = applyConfigToProviderInput(
    createProviderInput(stagedFiles, inspection),
    resolvedConfig.values
  );
  const promptStyle = resolvedConfig.values.promptStyle;
  let attempt = 1;
  let generated: CommitMessageResult;

  try {
    generated = await generateMessage(dependencies, providerConfig, providerInput);
  } catch (error) {
    const message = error instanceof ProviderError ? error.message : "Provider failed to generate a commit message.";

    return createResult({
      committed: false,
      dryRun,
      error: message,
      exitCode: EXIT_CODES.ERROR,
      files: stagedFiles,
      message: null,
      ok: false,
      provider: null,
      status: "blocked",
      warnings
    });
  }

  let currentMessage = generated.message;
  let providerInfo = createProviderInfo(generated);

  if (json) {
    return buildPreviewResult({
      dryRun,
      files: stagedFiles,
      message: currentMessage,
      provider: providerInfo,
      warnings
    });
  }

  while (true) {
    const previewInput = {
      attempt,
      dryRun,
      files: stagedFiles,
      message: currentMessage,
      operation: "git commit",
      warnings
    };

    if (dryRun) {
      return buildPreviewResult({
        dryRun: true,
        files: stagedFiles,
        message: currentMessage,
        provider: providerInfo,
        warnings
      });
    }

    const action = await dependencies.prompts.selectPreviewAction(previewInput);

    if (action === "cancel") {
      return createResult({
        committed: false,
        dryRun,
        error: null,
        exitCode: EXIT_CODES.USER_CANCEL,
        files: stagedFiles,
        message: currentMessage,
        ok: true,
        previewShown: true,
        provider: providerInfo,
        status: "cancelled",
        warnings
      });
    }

    if (action === "edit") {
      let draftMessage = currentMessage;
      let validationMessage: string | undefined;

      while (true) {
        const editedMessage = await dependencies.prompts.editMessage({
          currentMessage: draftMessage,
          promptStyle,
          validationMessage
        });

        if (editedMessage === null) {
          return createResult({
            committed: false,
            dryRun,
            error: null,
            exitCode: EXIT_CODES.USER_CANCEL,
            files: stagedFiles,
            message: currentMessage,
            ok: true,
            previewShown: true,
            provider: providerInfo,
            status: "cancelled",
            warnings
          });
        }

        draftMessage = editedMessage.trim();
        validationMessage = validateEditedMessage(draftMessage, promptStyle);

        if (!validationMessage) {
          currentMessage = draftMessage;
          break;
        }
      }

      continue;
    }

    if (action === "regenerate") {
      attempt += 1;

      try {
        generated = await generateMessage(dependencies, providerConfig, providerInput);
      } catch (error) {
        const message = error instanceof ProviderError ? error.message : "Provider failed to regenerate a commit message.";

        return createResult({
          committed: false,
          dryRun,
          error: message,
          exitCode: EXIT_CODES.ERROR,
          files: stagedFiles,
          message: currentMessage,
          ok: false,
          previewShown: true,
          provider: providerInfo,
          status: "blocked",
          warnings
        });
      }

      currentMessage = generated.message;
      providerInfo = createProviderInfo(generated);
      continue;
    }

    const commitResult = await dependencies.commitRunner({ cwd, env, message: currentMessage });

    if (commitResult.exitCode !== 0) {
      return createResult({
        committed: false,
        dryRun,
        error: summarizeCommitFailure(commitResult),
        exitCode: EXIT_CODES.ERROR,
        files: stagedFiles,
        message: currentMessage,
        ok: false,
        previewShown: true,
        provider: providerInfo,
        status: "blocked",
        warnings
      });
    }

    return createResult({
      committed: true,
      dryRun,
      error: null,
      exitCode: EXIT_CODES.SUCCESS,
      files: stagedFiles,
      message: currentMessage,
      ok: true,
      previewShown: true,
      provider: providerInfo,
      status: "committed",
      warnings
    });
  }
}

export { executeGitCommit };
