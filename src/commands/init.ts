import { Command, CommanderError } from "commander";

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
  type EffectiveConfig,
  type PromptStyle,
  type ProviderType
} from "../config/index.js";
import { createOutputRouter, EXIT_CODES, type CliWriteStream } from "../output/index.js";
import { type ConfigPanelPrompts } from "./config-panel.js";
import { createClackConfigPrompts } from "./config-prompts.js";
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

export interface InitCommandRuntime {
  cwd?: string;
  env?: NodeJS.ProcessEnv;
  isTty?: boolean;
  prompts?: ConfigPanelPrompts;
  stdout?: CliWriteStream;
  stderr?: CliWriteStream;
}

function normalizeOptionalValue(value: string | undefined): string | undefined {
  const normalizedValue = value?.trim();
  return normalizedValue ? normalizedValue : undefined;
}

function normalizeRequiredPromptValue(value: string | null, message: string): string {
  if (value === null) {
    throw new CommanderError(EXIT_CODES.USER_CANCEL, "cnm.cancelled", "User cancelled initialization.");
  }

  const normalizedValue = value.trim();

  if (!normalizedValue) {
    throw new ConfigError(message);
  }

  return normalizedValue;
}

function requirePromptSelection<T>(value: T | null): T {
  if (value === null) {
    throw new CommanderError(EXIT_CODES.USER_CANCEL, "cnm.cancelled", "User cancelled initialization.");
  }

  return value;
}

function resolveJsonOption(command: Command): boolean {
  return Boolean(command.optsWithGlobals<GlobalCliOptions>().json);
}

function resolveDryRun(command: Command): boolean {
  return Boolean(command.optsWithGlobals<GlobalCliOptions>().dryRun);
}

function resolveIsTty(runtime: InitCommandRuntime): boolean {
  if (runtime.isTty !== undefined) {
    return runtime.isTty;
  }

  return Boolean(process.stdin.isTTY && process.stdout.isTTY);
}

function getRuntimeCwd(runtime: InitCommandRuntime): string {
  return runtime.cwd ?? process.cwd();
}

function getRuntimeEnv(runtime: InitCommandRuntime): NodeJS.ProcessEnv {
  return runtime.env ?? process.env;
}

function hasExplicitInitOptions(options: InitCommandOptions): boolean {
  return Boolean(
    normalizeOptionalValue(options.provider)
    || normalizeOptionalValue(options.model)
    || normalizeOptionalValue(options.baseUrl)
    || normalizeOptionalValue(options.promptStyle)
    || normalizeOptionalValue(options.customPrompt)
    || normalizeOptionalValue(options.apiKey)
  );
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

function resolveModelInitialValue(currentConfig: EffectiveConfig, provider: ProviderType): string {
  return currentConfig.provider === provider ? currentConfig.model : getDefaultModel(provider);
}

async function createInteractiveInitPatch(input: {
  currentConfig: EffectiveConfig;
  prompts: ConfigPanelPrompts;
}): Promise<ConfigValues> {
  const provider = requirePromptSelection(
    await input.prompts.selectProvider({ currentProvider: input.currentConfig.provider })
  );
  const model = normalizeRequiredPromptValue(
    await input.prompts.inputModel({
      currentValue: resolveModelInitialValue(input.currentConfig, provider),
      provider
    }),
    "Model cannot be empty."
  );
  const patch: ConfigValues = { model, provider };

  if (input.prompts.confirmSetOptionalConfig) {
    const shouldConfigureBaseURL = requirePromptSelection(
      await input.prompts.confirmSetOptionalConfig({ field: "baseURL" })
    );

    if (shouldConfigureBaseURL) {
      const baseURL = await input.prompts.inputBaseURL({ currentValue: input.currentConfig.baseURL });
      const normalizedBaseURL = normalizeOptionalValue(baseURL ?? undefined);

      if (normalizedBaseURL) {
        patch.baseURL = normalizedBaseURL;
      }
    }
  }

  patch.apiKey = normalizeRequiredPromptValue(
    await input.prompts.inputApiKey({ hasExistingValue: Boolean(input.currentConfig.apiKey) }),
    "API key cannot be empty."
  );

  const promptStyle = requirePromptSelection(
    await input.prompts.selectPromptStyle({ currentPromptStyle: input.currentConfig.promptStyle })
  );

  patch.promptStyle = promptStyle;

  if (input.prompts.confirmSetOptionalConfig) {
    const shouldConfigureCustomPrompt = requirePromptSelection(
      await input.prompts.confirmSetOptionalConfig({ field: "customPrompt" })
    );

    if (shouldConfigureCustomPrompt) {
      const customPrompt = await input.prompts.inputCustomPrompt({ currentValue: input.currentConfig.customPrompt });
      const normalizedCustomPrompt = normalizeOptionalValue(customPrompt ?? undefined);

      if (normalizedCustomPrompt) {
        patch.customPrompt = normalizedCustomPrompt;
      }
    }
  }

  return patch;
}

async function runInteractiveInit(input: {
  cwd: string;
  env: NodeJS.ProcessEnv;
  prompts: ConfigPanelPrompts;
  router: ReturnType<typeof createOutputRouter>;
}): Promise<void> {
  const userConfig = await loadUserConfig({ cwd: input.cwd, env: input.env });
  const currentConfig = toEffectiveConfig(toInitializedConfig(userConfig, {}));

  input.router.writeHuman("Let's set up cnm for AI-assisted commits.", "stdout");
  const patch = await createInteractiveInitPatch({ currentConfig, prompts: input.prompts });
  const initializedConfig = toInitializedConfig(userConfig, patch);
  const effectiveConfig = toEffectiveConfig(initializedConfig);

  input.router.writeHuman(PLAINTEXT_API_KEY_WARNING, "stderr");
  const result = await writeUserConfig(initializedConfig, { cwd: input.cwd, env: input.env });

  for (const warning of result.warnings) {
    input.router.writeHuman(`Warning: ${warning}`, "stderr");
  }

  input.router.writeHuman(`Initialized user config at ${result.path}.`, "stdout");

  for (const line of toHumanConfigLines(effectiveConfig)) {
    input.router.writeHuman(line, "stdout");
  }

  input.router.writeHuman("Next: stage changes and run `cnm`, or run `cnm doctor` to check your setup.", "stdout");
}

export function createInitCommand(runtime: InitCommandRuntime = {}): Command {
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
      const shouldUseInteractiveInit = !router.isJson
        && !dryRun
        && !hasExplicitInitOptions(options)
        && (Boolean(runtime.prompts) || resolveIsTty(runtime));

      if (shouldUseInteractiveInit) {
        await runInteractiveInit({
          cwd,
          env,
          prompts: runtime.prompts ?? createClackConfigPrompts({ stdout: process.stdout }),
          router
        });
        return;
      }

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
