import { chmod, mkdir, readFile, writeFile } from "node:fs/promises";
import { homedir } from "node:os";
import { join, resolve } from "node:path";

import {
  CONFIG_KEYS,
  DEFAULT_PROMPT_STYLE,
  DEFAULT_PROVIDER,
  type ConfigKey,
  type ConfigValues,
  type EffectiveConfig,
  getDefaultModel,
  isConfigKey,
  isPromptStyle,
  isProviderType
} from "./schema.js";

export class ConfigError extends Error {
  readonly exitCode = 1;

  constructor(message: string) {
    super(message);
    this.name = "ConfigError";
  }
}

export interface ConfigEnvironment {
  cwd?: string;
  env?: NodeJS.ProcessEnv;
}

export interface ResolveConfigOptions extends ConfigEnvironment {
  flagOverrides?: ConfigValues;
}

export interface ConfigPaths {
  projectConfigPath: string;
  userConfigHome: string;
  userConfigPath: string;
}

export interface ResolvedConfig {
  paths: ConfigPaths;
  userConfig: ConfigValues;
  projectConfig: ConfigValues;
  envConfig: ConfigValues;
  flagOverrides: ConfigValues;
  warnings: string[];
  values: EffectiveConfig;
}

export interface WriteUserConfigResult {
  path: string;
  stored: ConfigValues;
  warnings: string[];
}

export interface JsonConfigView {
  apiKey: string | null;
  baseURL: string | null;
  customPrompt: string | null;
  model: string;
  promptStyle: string;
  provider: string;
}

const CONFIG_FILE_NAME = "config.json";
const PROJECT_CONFIG_FILE_NAME = ".cnmrc.json";

function getRuntimeEnv(env: NodeJS.ProcessEnv | undefined): NodeJS.ProcessEnv {
  return env ?? process.env;
}

function getRuntimeCwd(cwd: string | undefined): string {
  return cwd ?? process.cwd();
}

export function getUserConfigHome(env?: NodeJS.ProcessEnv): string {
  const runtimeEnv = getRuntimeEnv(env);
  const configuredHome = runtimeEnv.CNM_HOME?.trim();

  if (configuredHome) {
    return resolve(configuredHome);
  }

  return join(homedir(), ".cnm");
}

export function getConfigPaths(options: ConfigEnvironment = {}): ConfigPaths {
  const userConfigHome = getUserConfigHome(options.env);

  return {
    projectConfigPath: join(getRuntimeCwd(options.cwd), PROJECT_CONFIG_FILE_NAME),
    userConfigHome,
    userConfigPath: join(userConfigHome, CONFIG_FILE_NAME)
  };
}

function isObjectRecord(value: unknown): value is Record<string, unknown> {
  return typeof value === "object" && value !== null && !Array.isArray(value);
}

function ensureNonEmptyString(value: unknown, key: ConfigKey, sourceLabel: string): string {
  if (typeof value !== "string") {
    throw new ConfigError(`${sourceLabel} has an invalid \`${key}\` value; expected a string.`);
  }

  const normalizedValue = value.trim();

  if (!normalizedValue) {
    throw new ConfigError(`${sourceLabel} has an empty \`${key}\` value.`);
  }

  return normalizedValue;
}

function parseConfigObject(rawValue: unknown, sourceLabel: string): ConfigValues {
  if (!isObjectRecord(rawValue)) {
    throw new ConfigError(`${sourceLabel} must contain a JSON object.`);
  }

  const parsedConfig: ConfigValues = {};

  if ("provider" in rawValue) {
    const provider = ensureNonEmptyString(rawValue.provider, "provider", sourceLabel);

    if (!isProviderType(provider)) {
      throw new ConfigError(
        `${sourceLabel} has unsupported \`provider\` value \`${provider}\`.`,
      );
    }

    parsedConfig.provider = provider;
  }

  if ("model" in rawValue) {
    parsedConfig.model = ensureNonEmptyString(rawValue.model, "model", sourceLabel);
  }

  if ("baseURL" in rawValue) {
    parsedConfig.baseURL = ensureNonEmptyString(rawValue.baseURL, "baseURL", sourceLabel);
  }

  if ("promptStyle" in rawValue) {
    const promptStyle = ensureNonEmptyString(rawValue.promptStyle, "promptStyle", sourceLabel);

    if (!isPromptStyle(promptStyle)) {
      throw new ConfigError(
        `${sourceLabel} has unsupported \`promptStyle\` value \`${promptStyle}\`.`,
      );
    }

    parsedConfig.promptStyle = promptStyle;
  }

  if ("customPrompt" in rawValue) {
    parsedConfig.customPrompt = ensureNonEmptyString(
      rawValue.customPrompt,
      "customPrompt",
      sourceLabel,
    );
  }

  if ("apiKey" in rawValue) {
    parsedConfig.apiKey = ensureNonEmptyString(rawValue.apiKey, "apiKey", sourceLabel);
  }

  return parsedConfig;
}

async function readOptionalConfigFile(
  filePath: string,
  sourceLabel: string,
): Promise<ConfigValues | null> {
  let fileContent: string;

  try {
    fileContent = await readFile(filePath, "utf8");
  } catch (error) {
    const errorCode = (error as NodeJS.ErrnoException).code;

    if (errorCode === "ENOENT") {
      return null;
    }

    throw new ConfigError(`Unable to read ${sourceLabel} at ${filePath}.`);
  }

  let parsedJson: unknown;

  try {
    parsedJson = JSON.parse(fileContent) as unknown;
  } catch {
    throw new ConfigError(`${sourceLabel} at ${filePath} is not valid JSON.`);
  }

  return parseConfigObject(parsedJson, `${sourceLabel} at ${filePath}`);
}

export async function loadUserConfig(options: ConfigEnvironment = {}): Promise<ConfigValues> {
  const { userConfigPath } = getConfigPaths(options);

  return (await readOptionalConfigFile(userConfigPath, "User config")) ?? {};
}

export async function loadProjectConfig(
  options: ConfigEnvironment = {},
): Promise<{ config: ConfigValues; warnings: string[] }> {
  const { projectConfigPath } = getConfigPaths(options);
  const projectConfig = await readOptionalConfigFile(projectConfigPath, "Project config");

  if (!projectConfig) {
    return {
      config: {},
      warnings: []
    };
  }

  const warnings: string[] = [];
  const sanitizedProjectConfig = { ...projectConfig };

  if (sanitizedProjectConfig.apiKey) {
    warnings.push(
      `Project config at ${projectConfigPath} contains \`apiKey\`; project-level secrets are ignored.`,
    );
    delete sanitizedProjectConfig.apiKey;
  }

  return {
    config: sanitizedProjectConfig,
    warnings
  };
}

export function getEnvConfig(env?: NodeJS.ProcessEnv): ConfigValues {
  const runtimeEnv = getRuntimeEnv(env);
  const envConfig: ConfigValues = {};

  if (runtimeEnv.CNM_PROVIDER?.trim()) {
    const provider = runtimeEnv.CNM_PROVIDER.trim();

    if (!isProviderType(provider)) {
      throw new ConfigError(`Environment variable \`CNM_PROVIDER\` has unsupported value \`${provider}\`.`);
    }

    envConfig.provider = provider;
  }

  if (runtimeEnv.CNM_MODEL?.trim()) {
    envConfig.model = runtimeEnv.CNM_MODEL.trim();
  }

  if (runtimeEnv.CNM_BASE_URL?.trim()) {
    envConfig.baseURL = runtimeEnv.CNM_BASE_URL.trim();
  }

  if (runtimeEnv.CNM_PROMPT_STYLE?.trim()) {
    const promptStyle = runtimeEnv.CNM_PROMPT_STYLE.trim();

    if (!isPromptStyle(promptStyle)) {
      throw new ConfigError(
        `Environment variable \`CNM_PROMPT_STYLE\` has unsupported value \`${promptStyle}\`.`,
      );
    }

    envConfig.promptStyle = promptStyle;
  }

  if (runtimeEnv.CNM_CUSTOM_PROMPT?.trim()) {
    envConfig.customPrompt = runtimeEnv.CNM_CUSTOM_PROMPT.trim();
  }

  if (runtimeEnv.CNM_API_KEY?.trim()) {
    envConfig.apiKey = runtimeEnv.CNM_API_KEY.trim();
  }

  return envConfig;
}

function normalizeFlagOverrides(flagOverrides: ConfigValues | undefined): ConfigValues {
  if (!flagOverrides) {
    return {};
  }

  const normalizedEntries = Object.entries(flagOverrides).filter(([, value]) => value !== undefined);

  return Object.fromEntries(normalizedEntries) as ConfigValues;
}

export async function resolveEffectiveConfig(
  options: ResolveConfigOptions = {},
): Promise<ResolvedConfig> {
  const paths = getConfigPaths(options);
  const [userConfig, projectConfigResult] = await Promise.all([
    loadUserConfig(options),
    loadProjectConfig(options)
  ]);
  const envConfig = getEnvConfig(options.env);
  const flagOverrides = normalizeFlagOverrides(options.flagOverrides);

  const mergedConfig: ConfigValues = {
    ...userConfig,
    ...projectConfigResult.config,
    ...envConfig,
    ...flagOverrides
  };

  const provider = mergedConfig.provider ?? DEFAULT_PROVIDER;
  const promptStyle = mergedConfig.promptStyle ?? DEFAULT_PROMPT_STYLE;

  return {
    paths,
    userConfig,
    projectConfig: projectConfigResult.config,
    envConfig,
    flagOverrides,
    warnings: projectConfigResult.warnings,
    values: {
      apiKey: mergedConfig.apiKey,
      baseURL: mergedConfig.baseURL,
      customPrompt: mergedConfig.customPrompt,
      model: mergedConfig.model ?? getDefaultModel(provider),
      promptStyle,
      provider
    }
  };
}

export function parseKeyValue(key: string, rawValue: string): ConfigValues {
  if (!isConfigKey(key)) {
    throw new ConfigError(`Unsupported config key \`${key}\`.`);
  }

  const sourceLabel = "CLI input";

  switch (key) {
    case "provider": {
      const provider = ensureNonEmptyString(rawValue, key, sourceLabel);

      if (!isProviderType(provider)) {
        throw new ConfigError(`Unsupported provider \`${provider}\`.`);
      }

      return { provider };
    }
    case "promptStyle": {
      const promptStyle = ensureNonEmptyString(rawValue, key, sourceLabel);

      if (!isPromptStyle(promptStyle)) {
        throw new ConfigError(`Unsupported prompt style \`${promptStyle}\`.`);
      }

      return { promptStyle };
    }
    case "apiKey":
      return { apiKey: ensureNonEmptyString(rawValue, key, sourceLabel) };
    case "baseURL":
      return { baseURL: ensureNonEmptyString(rawValue, key, sourceLabel) };
    case "customPrompt":
      return { customPrompt: ensureNonEmptyString(rawValue, key, sourceLabel) };
    case "model":
      return { model: ensureNonEmptyString(rawValue, key, sourceLabel) };
    default:
      throw new ConfigError(`Unsupported config key \`${key satisfies never}\`.`);
  }
}

function stringifyConfig(config: ConfigValues): string {
  return `${JSON.stringify(config, null, 2)}\n`;
}

async function applyUserConfigPermissions(filePath: string): Promise<string | null> {
  if (process.platform === "win32") {
    return null;
  }

  try {
    await chmod(filePath, 0o600);
    return null;
  } catch {
    return `Unable to set 0600 permissions on user config at ${filePath}.`;
  }
}

export async function writeUserConfig(
  config: ConfigValues,
  options: ConfigEnvironment = {},
): Promise<WriteUserConfigResult> {
  const { userConfigHome, userConfigPath } = getConfigPaths(options);
  const warnings: string[] = [];

  await mkdir(userConfigHome, { recursive: true });
  await writeFile(userConfigPath, stringifyConfig(config), "utf8");

  const permissionWarning = await applyUserConfigPermissions(userConfigPath);

  if (permissionWarning) {
    warnings.push(permissionWarning);
  }

  return {
    path: userConfigPath,
    stored: config,
    warnings
  };
}

export async function writeUserConfigPatch(
  patch: ConfigValues,
  options: ConfigEnvironment = {},
): Promise<WriteUserConfigResult> {
  const userConfig = await loadUserConfig(options);
  const nextConfig = { ...userConfig, ...patch };

  return writeUserConfig(nextConfig, options);
}

export async function unsetUserConfigKey(
  key: ConfigKey,
  options: ConfigEnvironment = {},
): Promise<WriteUserConfigResult> {
  const userConfig = await loadUserConfig(options);
  const nextConfig = { ...userConfig };

  delete nextConfig[key];

  return writeUserConfig(nextConfig, options);
}

export function redactSecret(secret: string | undefined): string | undefined {
  if (!secret) {
    return undefined;
  }

  return "[redacted]";
}

export function redactConfig(config: EffectiveConfig): EffectiveConfig;
export function redactConfig(config: ConfigValues): ConfigValues;
export function redactConfig(config: ConfigValues | EffectiveConfig): ConfigValues | EffectiveConfig {
  return {
    ...config,
    apiKey: redactSecret(config.apiKey)
  };
}

export function toJsonConfigView(config: EffectiveConfig): JsonConfigView {
  return {
    apiKey: redactSecret(config.apiKey) ?? null,
    baseURL: config.baseURL ?? null,
    customPrompt: config.customPrompt ?? null,
    model: config.model,
    promptStyle: config.promptStyle,
    provider: config.provider
  };
}

export function toHumanConfigLines(config: EffectiveConfig): string[] {
  const jsonView = toJsonConfigView(config);

  return CONFIG_KEYS.map((key) => {
    const value = jsonView[key as keyof JsonConfigView];
    return `${key}=${value ?? "(unset)"}`;
  });
}

export function getConfigValue(
  config: EffectiveConfig,
  key: ConfigKey,
): string | undefined {
  if (key === "apiKey") {
    return redactSecret(config.apiKey);
  }

  return config[key];
}

export function assertConfigKey(key: string): ConfigKey {
  if (!isConfigKey(key)) {
    throw new ConfigError(`Unsupported config key \`${key}\`.`);
  }

  return key;
}

export const CONFIG_FILE_NAMES = {
  project: PROJECT_CONFIG_FILE_NAME,
  user: CONFIG_FILE_NAME
};
