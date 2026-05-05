import { Command, CommanderError } from "commander";

import { parseKeyValue, resolveEffectiveConfig, type ConfigValues } from "../config/index.js";
import { inspectGitRepository, stageAllChanges } from "../git/index.js";
import { createOutputRouter, EXIT_CODES, type CliWriteStream } from "../output/index.js";
import { createCommitMessageProvider } from "../providers/index.js";
import {
  createClackWorkflowPrompts,
  createNonInteractiveWorkflowPrompts,
  executeGitCommit,
  runCommitWorkflow,
  type CommitWorkflowDependencies,
  type CommitWorkflowResult
} from "../workflow/index.js";

interface GlobalCliOptions {
  baseUrl?: string;
  customPrompt?: string;
  dryRun?: boolean;
  json?: boolean;
  model?: string;
  promptStyle?: string;
  provider?: string;
}

export interface CommitCommandRuntime {
  cwd?: string;
  env?: NodeJS.ProcessEnv;
  isTty?: boolean;
  workflow?: Partial<CommitWorkflowDependencies>;
}

function resolveGlobalOptions(command: Command): Required<GlobalCliOptions> {
  const options = command.optsWithGlobals<GlobalCliOptions>();

  return {
    baseUrl: options.baseUrl ?? "",
    customPrompt: options.customPrompt ?? "",
    dryRun: Boolean(options.dryRun),
    json: Boolean(options.json),
    model: options.model ?? "",
    promptStyle: options.promptStyle ?? "",
    provider: options.provider ?? ""
  };
}

function resolveFlagOverrides(options: Required<GlobalCliOptions>): ConfigValues {
  return {
    ...parseOptionalFlag("provider", options.provider),
    ...parseOptionalFlag("model", options.model),
    ...parseOptionalFlag("baseURL", options.baseUrl),
    ...parseOptionalFlag("promptStyle", options.promptStyle),
    ...parseOptionalFlag("customPrompt", options.customPrompt)
  };
}

function parseOptionalFlag(key: Parameters<typeof parseKeyValue>[0], value: string): ConfigValues {
  return value.trim().length > 0 ? parseKeyValue(key, value) : {};
}

function resolveIsTty(runtime: CommitCommandRuntime): boolean {
  if (runtime.isTty !== undefined) {
    return runtime.isTty;
  }

  return Boolean(process.stdin.isTTY && process.stdout.isTTY);
}

function writeWarnings(router: ReturnType<typeof createOutputRouter>, warnings: string[]): void {
  for (const warning of warnings) {
    router.writeHuman(`Warning: ${warning}`, "stderr");
  }
}

function writeHumanResult(router: ReturnType<typeof createOutputRouter>, result: Awaited<ReturnType<typeof runCommitWorkflow>>): void {
  if (!result.previewShown) {
    writeWarnings(router, result.warnings);
  }

  switch (result.status) {
    case "committed":
      router.writeHuman(`Committed staged changes with message:\n${result.message ?? ""}`, "stdout");
      return;
    case "dry_run":
      router.writeHuman(`Dry-run preview complete. No commit was created.\n${result.message ?? ""}`, "stdout");
      return;
    case "cancelled":
      router.writeHuman("Cancelled. No commit was created.", "stderr");
      return;
    case "no_changes": {
      const message = result.warnings.at(-1) ?? "No changes were committed.";
      router.writeHuman(message, "stdout");
      return;
    }
    default:
      router.writeHuman(result.error ?? "Unable to complete the commit workflow.", "stderr");
  }
}

function toWorkflowJsonResult(result: CommitWorkflowResult) {
  return {
    command: result.command,
    committed: result.committed,
    dryRun: result.dryRun,
    error: result.error,
    files: result.files,
    message: result.message,
    ok: result.ok,
    provider: result.provider,
    status: result.status,
    warnings: result.warnings
  };
}

export function createCommitAction(
  runtime: CommitCommandRuntime = {},
  streams: { stderr?: CliWriteStream; stdout?: CliWriteStream } = {}
) {
  return async function (this: Command): Promise<void> {
    const globalOptions = resolveGlobalOptions(this);
    const flagOverrides = resolveFlagOverrides(globalOptions);
    const router = createOutputRouter({
      json: globalOptions.json,
      stderr: streams.stderr,
      stdout: streams.stdout
    });
    const dependencies: CommitWorkflowDependencies = {
      commitRunner: runtime.workflow?.commitRunner ?? executeGitCommit,
      createCommitMessageProvider: runtime.workflow?.createCommitMessageProvider ?? createCommitMessageProvider,
      inspectGitRepository: runtime.workflow?.inspectGitRepository ?? inspectGitRepository,
      prompts:
        runtime.workflow?.prompts
        ?? (globalOptions.json
          ? createNonInteractiveWorkflowPrompts()
          : createClackWorkflowPrompts({ stdout: process.stdout })),
      resolveEffectiveConfig: runtime.workflow?.resolveEffectiveConfig ?? resolveEffectiveConfig,
      stageAllChanges: runtime.workflow?.stageAllChanges ?? stageAllChanges
    };
    const result = await runCommitWorkflow({
      cwd: runtime.cwd ?? process.cwd(),
      dependencies,
      dryRun: globalOptions.dryRun,
      env: runtime.env ?? process.env,
      flagOverrides,
      isTty: resolveIsTty(runtime),
      json: globalOptions.json
    });

    if (router.isJson) {
      router.writeJson(toWorkflowJsonResult(result));
    } else {
      writeHumanResult(router, result);
    }

    if (result.exitCode === EXIT_CODES.USER_CANCEL) {
      throw new CommanderError(EXIT_CODES.USER_CANCEL, "cnm.cancelled", "User cancelled the workflow.");
    }

    if (result.exitCode !== EXIT_CODES.SUCCESS) {
      throw new CommanderError(result.exitCode, "cnm.failed", result.error ?? "Commit workflow failed.");
    }
  };
}
