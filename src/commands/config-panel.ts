import {
  CONFIG_KEYS,
  getDefaultModel,
  resolveEffectiveConfig,
  toHumanConfigLines,
  unsetUserConfigKey,
  writeUserConfig,
  writeUserConfigPatch,
  type ConfigEnvironment,
  type ConfigKey,
  type ConfigValues,
  type EffectiveConfig,
  type PromptStyle,
  type ProviderType,
  type ResolvedConfig,
  type WriteUserConfigResult
} from "../config/index.js";
import { EXIT_CODES, type ExitCode } from "../output/index.js";
import { PLAINTEXT_API_KEY_WARNING } from "./config-shared.js";

export type ConfigPanelAction =
  | "configureProviderModel"
  | "setApiKey"
  | "setBaseURL"
  | "setPromptStyle"
  | "setCustomPrompt"
  | "viewEffectiveConfig"
  | "testCurrentConfig"
  | "resetUnset"
  | "exit";

export type ConfigPanelResetTarget = ConfigKey | "all";

export interface ConfigPanelPrompts {
  confirmReset(input: { target: ConfigPanelResetTarget }): Promise<boolean | null>;
  confirmSetOptionalConfig?(input: { field: "baseURL" | "customPrompt" }): Promise<boolean | null>;
  inputApiKey(input: { hasExistingValue: boolean }): Promise<string | null>;
  inputBaseURL(input: { currentValue?: string }): Promise<string | null>;
  inputCustomPrompt(input: { currentValue?: string }): Promise<string | null>;
  inputModel(input: { currentValue: string; provider: ProviderType }): Promise<string | null>;
  selectAction(input: { effectiveConfig: EffectiveConfig; userConfig: ConfigValues }): Promise<ConfigPanelAction | null>;
  selectPromptStyle(input: { currentPromptStyle: PromptStyle }): Promise<PromptStyle | null>;
  selectProvider(input: { currentProvider: ProviderType }): Promise<ProviderType | null>;
  selectResetTarget(input: { userConfig: ConfigValues }): Promise<ConfigPanelResetTarget | null>;
}

export interface ConfigPanelOutput {
  message: string;
  target: "stdout" | "stderr";
}

export interface ConfigPanelDependencies {
  resolveEffectiveConfig(options: ConfigEnvironment): Promise<ResolvedConfig>;
  unsetUserConfigKey(key: ConfigKey, options: ConfigEnvironment): Promise<WriteUserConfigResult>;
  writeUserConfig(config: ConfigValues, options: ConfigEnvironment): Promise<WriteUserConfigResult>;
  writeUserConfigPatch(patch: ConfigValues, options: ConfigEnvironment): Promise<WriteUserConfigResult>;
}

export interface RunConfigPanelOptions extends ConfigEnvironment {
  dependencies?: ConfigPanelDependencies;
  onOutput?(output: ConfigPanelOutput): void;
  prompts: ConfigPanelPrompts;
}

export interface ConfigPanelResult {
  exitCode: ExitCode;
  status: "cancelled" | "exited";
}

const defaultDependencies: ConfigPanelDependencies = {
  resolveEffectiveConfig,
  unsetUserConfigKey,
  writeUserConfig,
  writeUserConfigPatch
};

function emit(
  onOutput: RunConfigPanelOptions["onOutput"],
  message: string,
  target: ConfigPanelOutput["target"] = "stdout",
): void {
  onOutput?.({ message, target });
}

function emitWarnings(onOutput: RunConfigPanelOptions["onOutput"], warnings: string[]): void {
  for (const warning of warnings) {
    emit(onOutput, `Warning: ${warning}`, "stderr");
  }
}

function normalizeRequiredValue(value: string | null): string | null | undefined {
  if (value === null) {
    return null;
  }

  const normalizedValue = value.trim();
  return normalizedValue.length === 0 ? undefined : normalizedValue;
}

function resolveModelInitialValue(resolvedConfig: ResolvedConfig, provider: ProviderType): string {
  if (resolvedConfig.values.provider === provider && resolvedConfig.values.model.trim().length > 0) {
    return resolvedConfig.values.model;
  }

  return getDefaultModel(provider);
}

function summarizeConfigCheckFailures(config: EffectiveConfig): string[] {
  const failures: string[] = [];

  if (!config.apiKey || config.apiKey.trim().length === 0) {
    failures.push(`Config check failed: No API key is configured for ${config.provider}.`);
  }

  if (config.provider === "openai-compatible" && (!config.baseURL || config.baseURL.trim().length === 0)) {
    failures.push("Config check failed: The openai-compatible provider requires `baseURL` to be configured.");
  }

  return failures;
}

async function handleConfigureProviderModel(
  resolvedConfig: ResolvedConfig,
  options: RunConfigPanelOptions,
  dependencies: ConfigPanelDependencies,
): Promise<ConfigPanelResult | null> {
  const provider = await options.prompts.selectProvider({ currentProvider: resolvedConfig.values.provider });

  if (provider === null) {
    return { exitCode: EXIT_CODES.USER_CANCEL, status: "cancelled" };
  }

  const model = await options.prompts.inputModel({
    currentValue: resolveModelInitialValue(resolvedConfig, provider),
    provider
  });
  const normalizedModel = normalizeRequiredValue(model);

  if (normalizedModel === null) {
    return { exitCode: EXIT_CODES.USER_CANCEL, status: "cancelled" };
  }

  if (normalizedModel === undefined) {
    emit(options.onOutput, "Model cannot be empty.", "stderr");
    return null;
  }

  const result = await dependencies.writeUserConfigPatch(
    { model: normalizedModel, provider },
    { cwd: options.cwd, env: options.env },
  );

  emitWarnings(options.onOutput, result.warnings);
  emit(options.onOutput, `Updated user config at ${result.path}.`);
  emit(options.onOutput, `provider=${provider}`);
  emit(options.onOutput, `model=${normalizedModel}`);
  return null;
}

async function handleSetApiKey(
  resolvedConfig: ResolvedConfig,
  options: RunConfigPanelOptions,
  dependencies: ConfigPanelDependencies,
): Promise<ConfigPanelResult | null> {
  const apiKey = await options.prompts.inputApiKey({ hasExistingValue: Boolean(resolvedConfig.values.apiKey) });
  const normalizedApiKey = normalizeRequiredValue(apiKey);

  if (normalizedApiKey === null) {
    return { exitCode: EXIT_CODES.USER_CANCEL, status: "cancelled" };
  }

  if (normalizedApiKey === undefined) {
    emit(options.onOutput, "API key cannot be empty.", "stderr");
    return null;
  }

  emit(options.onOutput, PLAINTEXT_API_KEY_WARNING, "stderr");
  const result = await dependencies.writeUserConfigPatch(
    { apiKey: normalizedApiKey },
    { cwd: options.cwd, env: options.env },
  );

  emitWarnings(options.onOutput, result.warnings);
  emit(options.onOutput, `Updated user config at ${result.path}.`);
  emit(options.onOutput, "apiKey=[redacted]");
  return null;
}

async function handleSetBaseURL(
  resolvedConfig: ResolvedConfig,
  options: RunConfigPanelOptions,
  dependencies: ConfigPanelDependencies,
): Promise<ConfigPanelResult | null> {
  const baseURL = await options.prompts.inputBaseURL({ currentValue: resolvedConfig.values.baseURL });
  const normalizedBaseURL = normalizeRequiredValue(baseURL);

  if (normalizedBaseURL === null) {
    return { exitCode: EXIT_CODES.USER_CANCEL, status: "cancelled" };
  }

  if (normalizedBaseURL === undefined) {
    emit(options.onOutput, "baseURL cannot be empty. Use reset/unset to remove it.", "stderr");
    return null;
  }

  const result = await dependencies.writeUserConfigPatch(
    { baseURL: normalizedBaseURL },
    { cwd: options.cwd, env: options.env },
  );

  emitWarnings(options.onOutput, result.warnings);
  emit(options.onOutput, `Updated user config at ${result.path}.`);
  emit(options.onOutput, `baseURL=${normalizedBaseURL}`);
  return null;
}

async function handleSetPromptStyle(
  resolvedConfig: ResolvedConfig,
  options: RunConfigPanelOptions,
  dependencies: ConfigPanelDependencies,
): Promise<ConfigPanelResult | null> {
  const promptStyle = await options.prompts.selectPromptStyle({ currentPromptStyle: resolvedConfig.values.promptStyle });

  if (promptStyle === null) {
    return { exitCode: EXIT_CODES.USER_CANCEL, status: "cancelled" };
  }

  const result = await dependencies.writeUserConfigPatch(
    { promptStyle },
    { cwd: options.cwd, env: options.env },
  );

  emitWarnings(options.onOutput, result.warnings);
  emit(options.onOutput, `Updated user config at ${result.path}.`);
  emit(options.onOutput, `promptStyle=${promptStyle}`);

  if (promptStyle === "custom") {
    emit(options.onOutput, "Style prompt disabled. The model will rely on your customPrompt plus the basic no-markdown/no-explanation guardrails.");
  }

  return null;
}

async function handleSetCustomPrompt(
  resolvedConfig: ResolvedConfig,
  options: RunConfigPanelOptions,
  dependencies: ConfigPanelDependencies,
): Promise<ConfigPanelResult | null> {
  const customPrompt = await options.prompts.inputCustomPrompt({ currentValue: resolvedConfig.values.customPrompt });
  const normalizedCustomPrompt = normalizeRequiredValue(customPrompt);

  if (normalizedCustomPrompt === null) {
    return { exitCode: EXIT_CODES.USER_CANCEL, status: "cancelled" };
  }

  if (normalizedCustomPrompt === undefined) {
    emit(options.onOutput, "Custom prompt cannot be empty. Use reset/unset to remove it.", "stderr");
    return null;
  }

  const result = await dependencies.writeUserConfigPatch(
    { customPrompt: normalizedCustomPrompt },
    { cwd: options.cwd, env: options.env },
  );

  emitWarnings(options.onOutput, result.warnings);
  emit(options.onOutput, `Updated user config at ${result.path}.`);
  emit(options.onOutput, "customPrompt=(updated)");
  return null;
}

function handleViewEffectiveConfig(resolvedConfig: ResolvedConfig, options: RunConfigPanelOptions): void {
  emitWarnings(options.onOutput, resolvedConfig.warnings);

  for (const line of toHumanConfigLines(resolvedConfig.values)) {
    emit(options.onOutput, line);
  }
}

function handleTestCurrentConfig(resolvedConfig: ResolvedConfig, options: RunConfigPanelOptions): void {
  emitWarnings(options.onOutput, resolvedConfig.warnings);
  const failures = summarizeConfigCheckFailures(resolvedConfig.values);

  if (failures.length > 0) {
    for (const failure of failures) {
      emit(options.onOutput, failure, "stderr");
    }

    return;
  }

  emit(options.onOutput, "Config check passed. No provider request was sent.");
}

async function handleResetUnset(
  resolvedConfig: ResolvedConfig,
  options: RunConfigPanelOptions,
  dependencies: ConfigPanelDependencies,
): Promise<ConfigPanelResult | null> {
  const target = await options.prompts.selectResetTarget({ userConfig: resolvedConfig.userConfig });

  if (target === null) {
    return { exitCode: EXIT_CODES.USER_CANCEL, status: "cancelled" };
  }

  const confirmed = await options.prompts.confirmReset({ target });

  if (confirmed === null) {
    return { exitCode: EXIT_CODES.USER_CANCEL, status: "cancelled" };
  }

  if (!confirmed) {
    emit(options.onOutput, "Reset cancelled. User config unchanged.");
    return null;
  }

  const result = target === "all"
    ? await dependencies.writeUserConfig({}, { cwd: options.cwd, env: options.env })
    : await dependencies.unsetUserConfigKey(target, { cwd: options.cwd, env: options.env });

  emitWarnings(options.onOutput, result.warnings);

  if (target === "all") {
    emit(options.onOutput, `Cleared user config at ${result.path}.`);
    return null;
  }

  emit(options.onOutput, `Removed ${target} from user config at ${result.path}.`);
  return null;
}

export function getResetTargets(userConfig: ConfigValues): ConfigPanelResetTarget[] {
  const presentKeys = CONFIG_KEYS.filter((key) => userConfig[key] !== undefined);
  return ["all", ...presentKeys];
}

export async function runConfigPanel(options: RunConfigPanelOptions): Promise<ConfigPanelResult> {
  const dependencies = options.dependencies ?? defaultDependencies;

  while (true) {
    const resolvedConfig = await dependencies.resolveEffectiveConfig({ cwd: options.cwd, env: options.env });
    const action = await options.prompts.selectAction({
      effectiveConfig: resolvedConfig.values,
      userConfig: resolvedConfig.userConfig
    });

    if (action === null) {
      return { exitCode: EXIT_CODES.USER_CANCEL, status: "cancelled" };
    }

    switch (action) {
      case "configureProviderModel": {
        const result = await handleConfigureProviderModel(resolvedConfig, options, dependencies);
        if (result) {
          return result;
        }
        break;
      }
      case "setApiKey": {
        const result = await handleSetApiKey(resolvedConfig, options, dependencies);
        if (result) {
          return result;
        }
        break;
      }
      case "setBaseURL": {
        const result = await handleSetBaseURL(resolvedConfig, options, dependencies);
        if (result) {
          return result;
        }
        break;
      }
      case "setPromptStyle": {
        const result = await handleSetPromptStyle(resolvedConfig, options, dependencies);
        if (result) {
          return result;
        }
        break;
      }
      case "setCustomPrompt": {
        const result = await handleSetCustomPrompt(resolvedConfig, options, dependencies);
        if (result) {
          return result;
        }
        break;
      }
      case "viewEffectiveConfig":
        handleViewEffectiveConfig(resolvedConfig, options);
        break;
      case "testCurrentConfig":
        handleTestCurrentConfig(resolvedConfig, options);
        break;
      case "resetUnset": {
        const result = await handleResetUnset(resolvedConfig, options, dependencies);
        if (result) {
          return result;
        }
        break;
      }
      case "exit":
        return { exitCode: EXIT_CODES.SUCCESS, status: "exited" };
      default:
        action satisfies never;
    }
  }
}
