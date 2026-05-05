import OpenAI from "openai";

import { missingApiKey, missingConfig, ProviderError, providerFailure } from "./errors.js";
import { sanitizeCommitMessage } from "./sanitize.js";
import type {
  CommitMessageProvider,
  CommitMessageResult,
  GenerateCommitMessageInput,
  OpenAiResponsesClient,
  OpenAiResponsesProviderConfig,
  ProviderClientFactories
} from "./types.js";
import { buildCommitMessagePrompt, resolveMaxSubjectLength } from "../prompt/commit-message.js";

const DEFAULT_MAX_OUTPUT_TOKENS = 8192;

export function createOpenAiResponsesProvider(
  config: OpenAiResponsesProviderConfig,
  clients: ProviderClientFactories = {}
): CommitMessageProvider {
  validateConfig(config);
  const client = clients.openAiResponses?.(config) ?? createDefaultClient(config);

  return {
    name: "openai-responses",
    async generateCommitMessage(input: GenerateCommitMessageInput): Promise<CommitMessageResult> {
      const prompt = buildCommitMessagePrompt(input);
      const maxSubjectLength = resolveMaxSubjectLength(input);

      try {
        const response = await client.responses.create({
          model: config.model,
          instructions: prompt.system,
          input: prompt.user,
          temperature: 0.2,
          max_output_tokens: config.maxOutputTokens ?? DEFAULT_MAX_OUTPUT_TOKENS
        });
        const message = sanitizeCommitMessage(response.output_text, {
          provider: "openai-responses",
          messageStyle: input.messageStyle,
          maxSubjectLength
        });

        return {
          message,
          metadata: {
            provider: "openai-responses",
            model: response.model ?? config.model,
            responseId: response.id,
            usage: response.usage
          }
        };
      } catch (error) {
        if (error instanceof ProviderError) {
          throw error;
        }

        throw providerFailure("openai-responses", error);
      }
    }
  };
}

function validateConfig(config: OpenAiResponsesProviderConfig): void {
  if (config.apiKey === undefined || config.apiKey.trim().length === 0) {
    throw missingApiKey("openai-responses");
  }

  if (config.model.trim().length === 0) {
    throw missingConfig("openai-responses", "model");
  }
}

function createDefaultClient(config: OpenAiResponsesProviderConfig): OpenAiResponsesClient {
  return new OpenAI({
    apiKey: config.apiKey,
    baseURL: config.baseURL,
    maxRetries: 0
  });
}
