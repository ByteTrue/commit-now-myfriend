import Anthropic from "@anthropic-ai/sdk";

import { missingApiKey, missingConfig, ProviderError, providerFailure } from "./errors.js";
import { sanitizeCommitMessage } from "./sanitize.js";
import type {
  AnthropicMessagesClient,
  AnthropicMessagesProviderConfig,
  CommitMessageProvider,
  CommitMessageResult,
  GenerateCommitMessageInput,
  ProviderClientFactories
} from "./types.js";
import { buildCommitMessagePrompt, resolveMaxSubjectLength } from "../prompt/commit-message.js";

const DEFAULT_MAX_OUTPUT_TOKENS = 8192;

export function createAnthropicMessagesProvider(
  config: AnthropicMessagesProviderConfig,
  clients: ProviderClientFactories = {}
): CommitMessageProvider {
  validateConfig(config);
  const client = clients.anthropicMessages?.(config) ?? createDefaultClient(config);

  return {
    name: "anthropic-messages",
    async generateCommitMessage(input: GenerateCommitMessageInput): Promise<CommitMessageResult> {
      const prompt = buildCommitMessagePrompt(input);
      const maxSubjectLength = resolveMaxSubjectLength(input);

      try {
        const response = await client.messages.create({
          model: config.model,
          system: prompt.system,
          max_tokens: config.maxOutputTokens ?? DEFAULT_MAX_OUTPUT_TOKENS,
          messages: [{ role: "user", content: prompt.user }]
        });
        const text = response.content
          .filter((block) => block.type === "text")
          .map((block) => block.text ?? "")
          .join("\n");
        const message = sanitizeCommitMessage(text, {
          provider: "anthropic-messages",
          messageStyle: input.messageStyle,
          maxSubjectLength
        });

        return {
          message,
          metadata: {
            provider: "anthropic-messages",
            model: response.model ?? config.model,
            responseId: response.id,
            finishReason: response.stop_reason ?? undefined,
            usage: response.usage
          }
        };
      } catch (error) {
        if (error instanceof ProviderError) {
          throw error;
        }

        throw providerFailure("anthropic-messages", error);
      }
    }
  };
}

function validateConfig(config: AnthropicMessagesProviderConfig): void {
  if (config.apiKey === undefined || config.apiKey.trim().length === 0) {
    throw missingApiKey("anthropic-messages");
  }

  if (config.model.trim().length === 0) {
    throw missingConfig("anthropic-messages", "model");
  }
}

function createDefaultClient(config: AnthropicMessagesProviderConfig): AnthropicMessagesClient {
  return new Anthropic({
    apiKey: config.apiKey,
    baseURL: config.baseURL,
    maxRetries: 0
  });
}
