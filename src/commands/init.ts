import { Command } from "commander";

import {
  ConfigError,
  DEFAULT_PROMPT_STYLE,
  DEFAULT_PROVIDER,
  getConfigPaths,
  getDefaultModel,
  isPromptStyle,
  isProviderType,
  loadUserConfig,
  toHumanConfigLines,
  toJsonConfigView,
  writeUserConfig,
  type ConfigValues,
  type EffectiveConfig
} from "../config/index.js";
import { createOutputRouter } from "../output/index.js";
import { type CommandRuntime } from "./config.js";
import { PLAINTEXT_API_KEY_WARNING } from "./config-shared.js";

interface GlobalCliOptions {
  dryRun?: boolean;
  json?: boolean;
}

interface InitCommandOptions {
  apiKey?: string;
  baseUrl?: string;
  customPrompt?: string;
  model?: string;
  promptStyle?: string;
  provider?: string;
}

function normalizeOptionalValue(value: string | undefined): string | undefined {
  const normalizedValue = value?.trim();
  return normalizedValue ? normalizedValue : undefined;
}

function resolveJsonOption(command: Command): boolean {
  return Boolean(command.optsWithGlobals<GlobalCliOptions>().json);
}

function resolveDryRun(command: Command): boolean {
  return Boolean(command.optsWithGlobals<GlobalCliOptions>().dryRun);
}

function getRuntimeCwd(runtime: CommandRuntime): string {
  return runtime.cwd ?? process.cwd();
}

function getRuntimeEnv(runtime: CommandRuntime): NodeJS.ProcessEnv {
  return runtime.env ?? process.env;
}

function toInitPatch(options: InitCommandOptions): ConfigValues {
  const patch: ConfigValues = {};

  if (options.provider) {
    const provider = normalizeOptionalValue(options.provider);

    if (!provider || !isProviderType(provider)) {
      throw new ConfigError(`Unsupported provider \`${options.provider}\`.`);
    }

    patch.provider = provider;
  }

  if (options.promptStyle) {
    const promptStyle = normalizeOptionalValue(options.promptStyle);

    if (!promptStyle || !isPromptStyle(promptStyle)) {
      throw new ConfigError(`Unsupported prompt style \`${options.promptStyle}\`.`);
    }

    patch.promptStyle = promptStyle;
  }

  const model = normalizeOptionalValue(options.model);
  const baseUrl = normalizeOptionalValue(options.baseUrl);
  const customPrompt = normalizeOptionalValue(options.customPrompt);
  const apiKey = normalizeOptionalValue(options.apiKey);

  if (model) {
    patch.model = model;
  }

  if (baseUrl) {
    patch.baseURL = baseUrl;
  }

  if (customPrompt) {
    patch.customPrompt = customPrompt;
  }

  if (apiKey) {
    patch.apiKey = apiKey;
  }

  return patch;
}

function toInitializedConfig(userConfig: ConfigValues, patch: ConfigValues): ConfigValues {
  const mergedConfig: ConfigValues = { ...userConfig, ...patch };
  const provider = mergedConfig.provider ?? DEFAULT_PROVIDER;

  return {
    ...mergedConfig,
    model: mergedConfig.model ?? getDefaultModel(provider),
    promptStyle: mergedConfig.promptStyle ?? DEFAULT_PROMPT_STYLE,
    provider
  };
}

function toEffectiveConfig(config: ConfigValues): EffectiveConfig {
  const provider = config.provider ?? DEFAULT_PROVIDER;

  return {
    apiKey: config.apiKey,
    baseURL: config.baseURL,
    customPrompt: config.customPrompt,
    model: config.model ?? getDefaultModel(provider),
    promptStyle: config.promptStyle ?? DEFAULT_PROMPT_STYLE,
    provider
  };
}

export function createInitCommand(runtime: CommandRuntime = {}): Command {
  const command = new Command("init");

  command
    .description("Set up cnm onboarding and user configuration.")
    .option("--provider <provider>", "default AI provider")
    .option("--model <model>", "default AI model")
    .option("--base-url <baseUrl>", "OpenAI-compatible base URL")
    .option("--prompt-style <promptStyle>", "commit prompt style")
    .option("--custom-prompt <customPrompt>", "custom prompt instructions")
    .option("--api-key <apiKey>", "API key stored in the user config")
    .action(async function (this: Command, options: InitCommandOptions) {
      const router = createOutputRouter({
        json: resolveJsonOption(this),
        stderr: runtime.stderr,
        stdout: runtime.stdout
      });
      const dryRun = resolveDryRun(this);
      const cwd = getRuntimeCwd(runtime);
      const env = getRuntimeEnv(runtime);
      const patch = toInitPatch(options);
      const userConfig = await loadUserConfig({ cwd, env });
      const initializedConfig = toInitializedConfig(userConfig, patch);
      const effectiveConfig = toEffectiveConfig(initializedConfig);
      const { userConfigPath } = getConfigPaths({ cwd, env });

      if (patch.apiKey) {
        router.writeHuman(PLAINTEXT_API_KEY_WARNING, "stderr");
      }

      if (dryRun) {
        router.writeStructured(
          {
            command: "cnm init",
            config: toJsonConfigView(effectiveConfig),
            dryRun: true,
            ok: true,
            path: userConfigPath,
            status: "dry_run"
          },
          `Dry-run: would initialize user config at ${userConfigPath}.`,
          "stdout"
        );
        return;
      }

      const result = await writeUserConfig(initializedConfig, { cwd, env });

      for (const warning of result.warnings) {
        router.writeHuman(`Warning: ${warning}`, "stderr");
      }

      if (router.isJson) {
        router.writeJson({
          command: "cnm init",
          config: toJsonConfigView(effectiveConfig),
          dryRun: false,
          ok: true,
          path: result.path,
          status: "initialized"
        });
        return;
      }

      router.writeHuman(`Initialized user config at ${result.path}.`, "stdout");

      for (const line of toHumanConfigLines(effectiveConfig)) {
        router.writeHuman(line, "stdout");
      }
    });

  return command;
}
