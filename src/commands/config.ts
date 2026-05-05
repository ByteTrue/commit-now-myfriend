import { Command, CommanderError } from "commander";

import {
  assertConfigKey,
  getConfigPaths,
  getConfigValue,
  parseKeyValue,
  resolveEffectiveConfig,
  toHumanConfigLines,
  toJsonConfigView,
  unsetUserConfigKey,
  writeUserConfigPatch
} from "../config/index.js";
import { createOutputRouter, EXIT_CODES, type CliWriteStream } from "../output/index.js";
import { runConfigPanel, type ConfigPanelPrompts } from "./config-panel.js";
import { createClackConfigPrompts } from "./config-prompts.js";
import { PLAINTEXT_API_KEY_WARNING } from "./config-shared.js";

interface GlobalCliOptions {
  dryRun?: boolean;
  json?: boolean;
}

export interface CommandRuntime {
  cwd?: string;
  env?: NodeJS.ProcessEnv;
  isTty?: boolean;
  prompts?: ConfigPanelPrompts;
  stdout?: CliWriteStream;
  stderr?: CliWriteStream;
}

function resolveJsonOption(command: Command, localJson: boolean | undefined): boolean {
  const globalOptions = command.optsWithGlobals<GlobalCliOptions>();
  return localJson ?? Boolean(globalOptions.json);
}

function resolveDryRun(command: Command): boolean {
  return Boolean(command.optsWithGlobals<GlobalCliOptions>().dryRun);
}

function createRouter(command: Command, runtime: CommandRuntime, localJson?: boolean) {
  return createOutputRouter({
    json: resolveJsonOption(command, localJson),
    stderr: runtime.stderr,
    stdout: runtime.stdout
  });
}

function getRuntimeCwd(runtime: CommandRuntime): string {
  return runtime.cwd ?? process.cwd();
}

function getRuntimeEnv(runtime: CommandRuntime): NodeJS.ProcessEnv {
  return runtime.env ?? process.env;
}

function resolveIsTty(runtime: CommandRuntime): boolean {
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

function renderConfigKeyValue(key: string, value: string | undefined | null): string {
  return `${key}=${value ?? "(unset)"}`;
}

async function handleGetCommand(
  runtime: CommandRuntime,
  key: string | undefined,
  localJson: boolean | undefined,
  command: Command,
): Promise<void> {
  const router = createRouter(command, runtime, localJson);
  const resolvedConfig = await resolveEffectiveConfig({
    cwd: getRuntimeCwd(runtime),
    env: getRuntimeEnv(runtime)
  });

  writeWarnings(router, resolvedConfig.warnings);

  if (key) {
    const configKey = assertConfigKey(key);
    const value = getConfigValue(resolvedConfig.values, configKey) ?? null;

    if (router.isJson) {
      router.writeJson({ [configKey]: value });
      return;
    }

    router.writeHuman(renderConfigKeyValue(configKey, value), "stdout");
    return;
  }

  const jsonView = toJsonConfigView(resolvedConfig.values);

  if (router.isJson) {
    router.writeJson({ ...jsonView });
    return;
  }

  for (const line of toHumanConfigLines(resolvedConfig.values)) {
    router.writeHuman(line, "stdout");
  }
}

async function handlePanelCommand(runtime: CommandRuntime, command: Command): Promise<void> {
  const router = createRouter(command, runtime);
  const result = await runConfigPanel({
    cwd: getRuntimeCwd(runtime),
    env: getRuntimeEnv(runtime),
    onOutput(output) {
      router.writeHuman(output.message, output.target);
    },
    prompts: runtime.prompts ?? createClackConfigPrompts({ stdout: process.stdout })
  });

  if (result.exitCode === EXIT_CODES.USER_CANCEL) {
    throw new CommanderError(EXIT_CODES.USER_CANCEL, "cnm.cancelled", "User cancelled config panel.");
  }
}

async function handleConfigAction(runtime: CommandRuntime, command: Command): Promise<void> {
  const shouldUsePanel = !resolveJsonOption(command, undefined)
    && !resolveDryRun(command)
    && (Boolean(runtime.prompts) || resolveIsTty(runtime));

  if (!shouldUsePanel) {
    await handleGetCommand(runtime, undefined, undefined, command);
    return;
  }

  await handlePanelCommand(runtime, command);
}

async function handleSetCommand(
  runtime: CommandRuntime,
  key: string,
  value: string,
  command: Command,
): Promise<void> {
  const router = createRouter(command, runtime);
  const configKey = assertConfigKey(key);
  const dryRun = resolveDryRun(command);

  if (configKey === "apiKey") {
    router.writeHuman(PLAINTEXT_API_KEY_WARNING, "stderr");
  }

  const patch = parseKeyValue(configKey, value);

  if (dryRun) {
    const { userConfigPath } = getConfigPaths({ cwd: getRuntimeCwd(runtime), env: getRuntimeEnv(runtime) });
    const maskedValue = configKey === "apiKey" ? "[redacted]" : value.trim();

    router.writeStructured(
      {
        command: "cnm config set",
        dryRun: true,
        key: configKey,
        ok: true,
        path: userConfigPath,
        status: "dry_run",
        value: maskedValue
      },
      `Dry-run: would update ${configKey} in user config at ${userConfigPath}.`,
      "stdout"
    );
    return;
  }

  const result = await writeUserConfigPatch(patch, {
    cwd: getRuntimeCwd(runtime),
    env: getRuntimeEnv(runtime)
  });

  writeWarnings(router, result.warnings);

  const maskedValue = configKey === "apiKey" ? "[redacted]" : value.trim();

  router.writeStructured(
    {
      command: "cnm config set",
      dryRun: false,
      key: configKey,
      ok: true,
      path: result.path,
      status: "updated",
      value: maskedValue
    },
    `Updated user config at ${result.path}.\n${renderConfigKeyValue(configKey, maskedValue)}`,
    "stdout"
  );
}

async function handleUnsetCommand(
  runtime: CommandRuntime,
  key: string,
  command: Command,
): Promise<void> {
  const router = createRouter(command, runtime);
  const configKey = assertConfigKey(key);
  const dryRun = resolveDryRun(command);

  if (dryRun) {
    const { userConfigPath } = getConfigPaths({ cwd: getRuntimeCwd(runtime), env: getRuntimeEnv(runtime) });

    router.writeStructured(
      {
        command: "cnm config unset",
        dryRun: true,
        key: configKey,
        ok: true,
        path: userConfigPath,
        status: "dry_run"
      },
      `Dry-run: would remove ${configKey} from user config at ${userConfigPath}.`,
      "stdout"
    );
    return;
  }

  const result = await unsetUserConfigKey(configKey, {
    cwd: getRuntimeCwd(runtime),
    env: getRuntimeEnv(runtime)
  });

  writeWarnings(router, result.warnings);
  router.writeStructured(
    {
      command: "cnm config unset",
      dryRun: false,
      key: configKey,
      ok: true,
      path: result.path,
      status: "removed"
    },
    `Removed ${configKey} from user config at ${result.path}.`,
    "stdout"
  );
}

export function createConfigCommand(runtime: CommandRuntime = {}): Command {
  const command = new Command("config");

  command
    .description("Inspect and edit cnm configuration.")
    .action(async function (this: Command) {
      await handleConfigAction(runtime, this);
    });

  command
    .command("get")
    .description("Get the effective config or a single key.")
    .argument("[key]", "config key")
    .option("--json", "print JSON output")
    .action(async function (this: Command, key: string | undefined, options: { json?: boolean }) {
      await handleGetCommand(runtime, key, options.json, this);
    });

  command
    .command("list")
    .description("List the effective config.")
    .option("--json", "print JSON output")
    .action(async function (this: Command, options: { json?: boolean }) {
      await handleGetCommand(runtime, undefined, options.json, this);
    });

  command
    .command("set")
    .description("Set a user config value.")
    .argument("<key>", "config key")
    .argument("<value>", "config value")
    .action(async function (this: Command, key: string, value: string) {
      await handleSetCommand(runtime, key, value, this);
    });

  command
    .command("unset")
    .description("Remove a user config value.")
    .argument("<key>", "config key")
    .action(async function (this: Command, key: string) {
      await handleUnsetCommand(runtime, key, this);
    });

  return command;
}
