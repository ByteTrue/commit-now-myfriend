export const PROVIDER_TYPES = [
  "openai-compatible",
  "openai-responses",
  "anthropic-messages",
  "google-gemini"
] as const;

export type ProviderType = (typeof PROVIDER_TYPES)[number];

export const PROMPT_STYLES = [
  "auto",
  "conventional",
  "angular",
  "google",
  "atom",
  "plain",
  "custom"
] as const;

export type PromptStyle = (typeof PROMPT_STYLES)[number];

export const CONFIG_KEYS = [
  "provider",
  "model",
  "baseURL",
  "promptStyle",
  "customPrompt",
  "apiKey"
] as const;

export type ConfigKey = (typeof CONFIG_KEYS)[number];

export interface ConfigValues {
  provider?: ProviderType;
  model?: string;
  baseURL?: string;
  promptStyle?: PromptStyle;
  customPrompt?: string;
  apiKey?: string;
}

export interface EffectiveConfig {
  provider: ProviderType;
  model: string;
  baseURL?: string;
  promptStyle: PromptStyle;
  customPrompt?: string;
  apiKey?: string;
}

export const DEFAULT_PROVIDER: ProviderType = "openai-responses";
export const DEFAULT_PROMPT_STYLE: PromptStyle = "auto";

const DEFAULT_MODELS: Record<ProviderType, string> = {
  "anthropic-messages": "claude-sonnet-4-20250514",
  "google-gemini": "gemini-2.5-flash",
  "openai-compatible": "gpt-5-mini",
  "openai-responses": "gpt-5-mini"
};

export function getDefaultModel(provider: ProviderType): string {
  return DEFAULT_MODELS[provider];
}

export function isConfigKey(value: string): value is ConfigKey {
  return CONFIG_KEYS.includes(value as ConfigKey);
}

export function isProviderType(value: string): value is ProviderType {
  return PROVIDER_TYPES.includes(value as ProviderType);
}

export function isPromptStyle(value: string): value is PromptStyle {
  return PROMPT_STYLES.includes(value as PromptStyle);
}
