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
const RESPONSE_SNIPPET_LENGTH = 500;

interface ChatCompletionChoice {
  finish_reason?: string | null;
  message: {
    content?: string | null;
  };
}

interface ChatCompletionResponse {
  id?: string;
  model?: string;
  choices: ChatCompletionChoice[];
  usage?: unknown;
}

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
        const rawResponse = await client.chat.completions.create({
          model: config.model,
          messages: [
            { role: "system", content: prompt.system },
            { role: "user", content: prompt.user }
          ],
          temperature: 0.2,
          max_tokens: config.maxOutputTokens ?? DEFAULT_MAX_OUTPUT_TOKENS
        }) as unknown;
        const response = ensureChatCompletionResponse(rawResponse);
        const firstChoice = response.choices[0];
        const message = sanitizeCommitMessage(firstChoice?.message?.content, {
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
            finishReason: firstChoice?.finish_reason ?? undefined,
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

function ensureChatCompletionResponse(response: unknown): ChatCompletionResponse {
  if (!isRecord(response)) {
    throw createUnexpectedResponseError(response, "the API did not return a JSON object");
  }

  if (!Array.isArray(response.choices) || response.choices.length === 0) {
    throw createUnexpectedResponseError(response, "the API response did not include choices[0]");
  }

  const firstChoice = response.choices[0];
  if (!isRecord(firstChoice)) {
    throw createUnexpectedResponseError(response, "choices[0] is not an object");
  }

  if (!isRecord(firstChoice.message)) {
    throw createUnexpectedResponseError(response, "choices[0].message is missing or invalid");
  }

  const content = firstChoice.message.content;
  if (content !== undefined && content !== null && typeof content !== "string") {
    throw createUnexpectedResponseError(response, "choices[0].message.content is not a string");
  }

  return response as unknown as ChatCompletionResponse;
}

function createUnexpectedResponseError(response: unknown, reason: string): ProviderError {
  const hint = typeof response === "string" && looksLikeHtmlResponse(response)
    ? "The response looks like HTML, which usually means baseURL points to a web page or login page instead of the API endpoint."
    : "The response did not match the expected chat completion shape.";

  return new ProviderError({
    code: "provider_failure",
    provider: "openai-compatible",
    message: `openai-compatible returned an unexpected response from the API (${reason}). ${hint} Check that baseURL points to the OpenAI-compatible API root (often ending in /v1). Response snippet: ${formatResponseSnippet(response)}`,
    cause: response
  });
}

function formatResponseSnippet(response: unknown): string {
  const serialized = serializeResponse(response).replace(/\s+/g, " ").trim();

  if (serialized.length === 0) {
    return "<empty>";
  }

  return serialized.length > RESPONSE_SNIPPET_LENGTH ? `${serialized.slice(0, RESPONSE_SNIPPET_LENGTH)}…` : serialized;
}

function serializeResponse(response: unknown): string {
  if (typeof response === "string") {
    return response;
  }

  if (response === undefined) {
    return "undefined";
  }

  if (response === null) {
    return "null";
  }

  try {
    return JSON.stringify(response) ?? String(response);
  } catch {
    return String(response);
  }
}

function looksLikeHtmlResponse(response: string): boolean {
  const normalized = response.trimStart().toLowerCase();

  return normalized.startsWith("<!doctype html") || normalized.startsWith("<html") || normalized.startsWith("<head") || normalized.startsWith("<body");
}

function isRecord(value: unknown): value is Record<string, unknown> {
  return typeof value === "object" && value !== null;
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
