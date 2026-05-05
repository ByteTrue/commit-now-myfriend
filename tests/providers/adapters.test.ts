import { describe, expect, it, vi } from "vitest";

import {
  createAnthropicMessagesProvider,
  createCommitMessageProvider,
  createGoogleGeminiProvider,
  createOpenAiCompatibleProvider,
  createOpenAiResponsesProvider,
  ProviderError
} from "../../src/providers/index.js";
import type {
  AnthropicMessagesClient,
  GenerateCommitMessageInput,
  GoogleGeminiClient,
  OpenAiCompatibleClient,
  OpenAiResponsesClient
} from "../../src/providers/index.js";

const input: GenerateCommitMessageInput = {
  files: [{ path: "src/providers/index.ts", status: "M" }],
  diff: "diff --git a/src/providers/index.ts b/src/providers/index.ts\n+export const value = true;",
  repo: { branch: "main" },
  customPrompt: "Prefer a provider scope.",
  maxSubjectLength: 72
};

describe("provider adapters", () => {
  it("calls OpenAI-compatible Chat Completions with baseURL-aware config", async () => {
    const create = vi.fn<OpenAiCompatibleClient["chat"]["completions"]["create"]>().mockResolvedValue({
      id: "chat-response-id",
      model: "chat-model",
      choices: [{ finish_reason: "stop", message: { content: "feat(provider): add chat adapter" } }],
      usage: { total_tokens: 42 }
    });
    const provider = createOpenAiCompatibleProvider(
      { provider: "openai-compatible", apiKey: "test-key", model: "chat-model", baseURL: "https://example.invalid/v1" },
      { openAiCompatible: () => ({ chat: { completions: { create } } }) }
    );

    const result = await provider.generateCommitMessage(input);

    expect(result.message).toBe("feat(provider): add chat adapter");
    expect(result.metadata).toMatchObject({ provider: "openai-compatible", model: "chat-model", responseId: "chat-response-id" });
    expect(create).toHaveBeenCalledWith(expect.objectContaining({
      model: "chat-model",
      temperature: 0.2,
      max_tokens: 8192
    }));
    expect(create.mock.calls[0]?.[0].messages[0]?.content).toContain("Conventional Commits");
    expect(create.mock.calls[0]?.[0].messages[0]?.content).toContain("Prefer a provider scope.");
  });

  it("calls OpenAI Responses separately from Chat Completions", async () => {
    const create = vi.fn<OpenAiResponsesClient["responses"]["create"]>().mockResolvedValue({
      id: "responses-id",
      model: "responses-model",
      output_text: "fix(provider): parse response text",
      usage: { input_tokens: 12 }
    });
    const provider = createOpenAiResponsesProvider(
      { provider: "openai-responses", apiKey: "test-key", model: "responses-model" },
      { openAiResponses: () => ({ responses: { create } }) }
    );

    const result = await provider.generateCommitMessage(input);

    expect(result.message).toBe("fix(provider): parse response text");
    expect(result.metadata).toMatchObject({ provider: "openai-responses", model: "responses-model", responseId: "responses-id" });
    expect(create).toHaveBeenCalledWith(expect.objectContaining({
      model: "responses-model",
      input: expect.stringContaining("Diff:"),
      instructions: expect.stringContaining("Return only the commit message"),
      max_output_tokens: 8192
    }));
  });

  it("calls Anthropic Messages and joins text blocks", async () => {
    const create = vi.fn<AnthropicMessagesClient["messages"]["create"]>().mockResolvedValue({
      id: "anthropic-id",
      model: "claude-test",
      stop_reason: "end_turn",
      content: [{ type: "text", text: "docs(provider): document adapter contract" }],
      usage: { input_tokens: 10, output_tokens: 8 }
    });
    const provider = createAnthropicMessagesProvider(
      { provider: "anthropic-messages", apiKey: "test-key", model: "claude-test" },
      { anthropicMessages: () => ({ messages: { create } }) }
    );

    const result = await provider.generateCommitMessage(input);

    expect(result.message).toBe("docs(provider): document adapter contract");
    expect(result.metadata).toMatchObject({ provider: "anthropic-messages", model: "claude-test", responseId: "anthropic-id" });
    expect(create).toHaveBeenCalledWith(expect.objectContaining({
      model: "claude-test",
      system: expect.stringContaining("Conventional Commits"),
      max_tokens: 8192,
      messages: [{ role: "user", content: expect.stringContaining("src/providers/index.ts") }]
    }));
  });

  it("calls Google Gemini API without Vertex options", async () => {
    const generateContent = vi.fn<GoogleGeminiClient["models"]["generateContent"]>().mockResolvedValue({
      responseId: "gemini-id",
      modelVersion: "gemini-test-version",
      text: "test(provider): cover gemini adapter",
      usageMetadata: { totalTokenCount: 20 }
    });
    const provider = createGoogleGeminiProvider(
      { provider: "google-gemini", apiKey: "test-key", model: "gemini-test" },
      { googleGemini: () => ({ models: { generateContent } }) }
    );

    const result = await provider.generateCommitMessage(input);

    expect(result.message).toBe("test(provider): cover gemini adapter");
    expect(result.metadata).toMatchObject({ provider: "google-gemini", model: "gemini-test-version", responseId: "gemini-id" });
    expect(generateContent).toHaveBeenCalledWith(expect.objectContaining({
      model: "gemini-test",
      contents: expect.stringContaining("Diff:"),
      config: expect.objectContaining({
        systemInstruction: expect.stringContaining("Return only the commit message"),
      maxOutputTokens: 8192
      })
    }));
  });

  it("allows custom-only style output without Conventional Commit format", async () => {
    const create = vi.fn<OpenAiCompatibleClient["chat"]["completions"]["create"]>().mockResolvedValue({
      choices: [{ message: { content: "Update provider adapter behavior" } }]
    });
    const provider = createOpenAiCompatibleProvider(
      { provider: "openai-compatible", apiKey: "test-key", model: "chat-model", baseURL: "https://example.invalid/v1" },
      { openAiCompatible: () => ({ chat: { completions: { create } } }) }
    );

    const result = await provider.generateCommitMessage({
      ...input,
      customPrompt: "Write a plain English commit message.",
      messageStyle: "custom"
    });

    expect(result.message).toBe("Update provider adapter behavior");
    expect(create.mock.calls[0]?.[0].messages[0]?.content).not.toContain("Conventional Commits");
    expect(create.mock.calls[0]?.[0].messages[0]?.content).toContain("Write a plain English commit message.");
  });

  it("creates providers through the registry factory", async () => {
    const create = vi.fn<OpenAiResponsesClient["responses"]["create"]>().mockResolvedValue({
      output_text: "chore(provider): route registry factory"
    });
    const provider = createCommitMessageProvider(
      { provider: "openai-responses", apiKey: "test-key", model: "responses-model" },
      { openAiResponses: () => ({ responses: { create } }) }
    );

    const result = await provider.generateCommitMessage(input);

    expect(provider.name).toBe("openai-responses");
    expect(result.message).toBe("chore(provider): route registry factory");
  });

  it("rejects missing API keys before creating default SDK clients", () => {
    expect(() => createGoogleGeminiProvider({ provider: "google-gemini", model: "gemini-test" }))
      .toThrow(ProviderError);

    try {
      createGoogleGeminiProvider({ provider: "google-gemini", model: "gemini-test" });
    } catch (error) {
      expect(error).toBeInstanceOf(ProviderError);
      expect((error as ProviderError).code).toBe("missing_api_key");
    }
  });

  it("normalizes provider failures into typed errors", async () => {
    const create = vi.fn<OpenAiResponsesClient["responses"]["create"]>().mockRejectedValue(new Error("network disabled"));
    const provider = createOpenAiResponsesProvider(
      { provider: "openai-responses", apiKey: "test-key", model: "responses-model" },
      { openAiResponses: () => ({ responses: { create } }) }
    );

    await expect(provider.generateCommitMessage(input)).rejects.toMatchObject({
      code: "provider_failure",
      provider: "openai-responses"
    });
  });
});
