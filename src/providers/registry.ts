import { createAnthropicMessagesProvider } from "./anthropic-messages.js";
import { createGoogleGeminiProvider } from "./google-gemini.js";
import { createOpenAiCompatibleProvider } from "./openai-compatible.js";
import { createOpenAiResponsesProvider } from "./openai-responses.js";
import { ProviderError } from "./errors.js";
import type {
  AnthropicMessagesProviderConfig,
  CommitMessageProvider,
  GoogleGeminiProviderConfig,
  OpenAiCompatibleProviderConfig,
  OpenAiResponsesProviderConfig,
  ProviderClientFactories,
  ProviderConfig
} from "./types.js";

export type ProviderFactory<TConfig extends ProviderConfig = ProviderConfig> = (
  config: TConfig,
  clients?: ProviderClientFactories
) => CommitMessageProvider;

interface ProviderRegistry {
  "openai-compatible": ProviderFactory<OpenAiCompatibleProviderConfig>;
  "openai-responses": ProviderFactory<OpenAiResponsesProviderConfig>;
  "anthropic-messages": ProviderFactory<AnthropicMessagesProviderConfig>;
  "google-gemini": ProviderFactory<GoogleGeminiProviderConfig>;
}

export const providerRegistry = {
  "openai-compatible": createOpenAiCompatibleProvider,
  "openai-responses": createOpenAiResponsesProvider,
  "anthropic-messages": createAnthropicMessagesProvider,
  "google-gemini": createGoogleGeminiProvider
} satisfies ProviderRegistry;

export function createCommitMessageProvider(
  config: ProviderConfig,
  clients?: ProviderClientFactories
): CommitMessageProvider {
  switch (config.provider) {
    case "openai-compatible":
      return providerRegistry[config.provider](config, clients);
    case "openai-responses":
      return providerRegistry[config.provider](config, clients);
    case "anthropic-messages":
      return providerRegistry[config.provider](config, clients);
    case "google-gemini":
      return providerRegistry[config.provider](config, clients);
    default:
      return assertUnsupportedProvider(config);
  }
}

function assertUnsupportedProvider(config: never): never {
  throw new ProviderError({
    code: "missing_config",
    provider: "openai-compatible",
    message: `Unsupported provider: ${String(config)}.`
  });
}
