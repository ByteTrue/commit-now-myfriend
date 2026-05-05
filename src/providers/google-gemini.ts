import { GoogleGenAI } from "@google/genai";

import { missingApiKey, missingConfig, ProviderError, providerFailure } from "./errors.js";
import { sanitizeCommitMessage } from "./sanitize.js";
import type {
  CommitMessageProvider,
  CommitMessageResult,
  GenerateCommitMessageInput,
  GoogleGeminiClient,
  GoogleGeminiProviderConfig,
  ProviderClientFactories
} from "./types.js";
import { buildCommitMessagePrompt, resolveMaxSubjectLength } from "../prompt/commit-message.js";

const DEFAULT_MAX_OUTPUT_TOKENS = 8192;

export function createGoogleGeminiProvider(
  config: GoogleGeminiProviderConfig,
  clients: ProviderClientFactories = {}
): CommitMessageProvider {
  validateConfig(config);
  const client = clients.googleGemini?.(config) ?? createDefaultClient(config);

  return {
    name: "google-gemini",
    async generateCommitMessage(input: GenerateCommitMessageInput): Promise<CommitMessageResult> {
      const prompt = buildCommitMessagePrompt(input);
      const maxSubjectLength = resolveMaxSubjectLength(input);

      try {
        const response = await client.models.generateContent({
          model: config.model,
          contents: prompt.user,
          config: {
            systemInstruction: prompt.system,
            temperature: 0.2,
            maxOutputTokens: config.maxOutputTokens ?? DEFAULT_MAX_OUTPUT_TOKENS
          }
        });
        const message = sanitizeCommitMessage(response.text, {
          provider: "google-gemini",
          messageStyle: input.messageStyle,
          maxSubjectLength
        });

        return {
          message,
          metadata: {
            provider: "google-gemini",
            model: response.modelVersion ?? config.model,
            responseId: response.responseId,
            usage: response.usageMetadata
          }
        };
      } catch (error) {
        if (error instanceof ProviderError) {
          throw error;
        }

        throw providerFailure("google-gemini", error);
      }
    }
  };
}

function validateConfig(config: GoogleGeminiProviderConfig): void {
  if (config.apiKey === undefined || config.apiKey.trim().length === 0) {
    throw missingApiKey("google-gemini");
  }

  if (config.model.trim().length === 0) {
    throw missingConfig("google-gemini", "model");
  }
}

function createDefaultClient(config: GoogleGeminiProviderConfig): GoogleGeminiClient {
  return new GoogleGenAI({
    apiKey: config.apiKey,
    baseUrl: config.baseURL
  });
}
