import OpenAI from "openai";

import { missingApiKey, missingConfig, ProviderError, providerFailure } from "./errors.js";
import { sanitizeCommitMessage } from "./sanitize.js";
import type {
  CommitMessageProvider,
  CommitMessageResult,
  GenerateCommitMessageInput,
  OpenAiCompatibleClient,
  OpenAiCompatibleProviderConfig,
  ProviderClientFactories
} from "./types.js";
import { buildCommitMessagePrompt, resolveMaxSubjectLength } from "../prompt/commit-message.js";

const DEFAULT_MAX_OUTPUT_TOKENS = 8192;

export function createOpenAiCompatibleProvider(
  config: OpenAiCompatibleProviderConfig,
  clients: ProviderClientFactories = {}
): CommitMessageProvider {
  validateConfig(config);
  const client = clients.openAiCompatible?.(config) ?? createDefaultClient(config);

  return {
    name: "openai-compatible",
    async generateCommitMessage(input: GenerateCommitMessageInput): Promise<CommitMessageResult> {
      const prompt = buildCommitMessagePrompt(input);
      const maxSubjectLength = resolveMaxSubjectLength(input);

      try {
        const response = await client.chat.completions.create({
          model: config.model,
          messages: [
            { role: "system", content: prompt.system },
            { role: "user", content: prompt.user }
          ],
          temperature: 0.2,
          max_tokens: config.maxOutputTokens ?? DEFAULT_MAX_OUTPUT_TOKENS
        });
        const message = sanitizeCommitMessage(response.choices[0]?.message?.content, {
          provider: "openai-compatible",
          messageStyle: input.messageStyle,
          maxSubjectLength
        });

        return {
          message,
          metadata: {
            provider: "openai-compatible",
            model: response.model ?? config.model,
            responseId: response.id,
            finishReason: response.choices[0]?.finish_reason ?? undefined,
            usage: response.usage
          }
        };
      } catch (error) {
        if (error instanceof ProviderError) {
          throw error;
        }

        throw providerFailure("openai-compatible", error);
      }
    }
  };
}

function validateConfig(config: OpenAiCompatibleProviderConfig): void {
  if (config.apiKey === undefined || config.apiKey.trim().length === 0) {
    throw missingApiKey("openai-compatible");
  }

  if (config.model.trim().length === 0) {
    throw missingConfig("openai-compatible", "model");
  }

  if (config.baseURL.trim().length === 0) {
    throw missingConfig("openai-compatible", "baseURL");
  }
}

function createDefaultClient(config: OpenAiCompatibleProviderConfig): OpenAiCompatibleClient {
  return new OpenAI({ apiKey: config.apiKey, baseURL: config.baseURL, maxRetries: 0 });
}
