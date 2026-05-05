import { describe, expect, it } from "vitest";

import { ProviderError, sanitizeCommitMessage } from "../../src/providers/index.js";

describe("sanitizeCommitMessage", () => {
  it("removes a markdown fence around a commit message", () => {
    const message = sanitizeCommitMessage("```text\nfeat: add x\n```", {
      provider: "openai-compatible",
      maxSubjectLength: 72
    });

    expect(message).toBe("feat: add x");
  });

  it("removes explanatory text before the commit message", () => {
    const message = sanitizeCommitMessage(
      "Here is a commit message:\n\nfix(parser): handle empty diffs",
      { provider: "anthropic-messages", maxSubjectLength: 72 }
    );

    expect(message).toBe("fix(parser): handle empty diffs");
  });

  it("preserves Why body lines after a valid Conventional subject", () => {
    const message = sanitizeCommitMessage(
      "feat(parser): add support\n\nWhy: needed for users",
      { provider: "anthropic-messages", maxSubjectLength: 72 }
    );

    expect(message).toBe("feat(parser): add support\n\nWhy: needed for users");
  });

  it("preserves Explanation body lines after a valid Conventional subject", () => {
    const message = sanitizeCommitMessage(
      "fix(parser): handle empty diffs\n\nExplanation: this fixes parsing.",
      { provider: "anthropic-messages", maxSubjectLength: 72 }
    );

    expect(message).toBe("fix(parser): handle empty diffs\n\nExplanation: this fixes parsing.");
  });

  it("preserves an optional body after a valid subject", () => {
    const message = sanitizeCommitMessage(
      "feat(provider): normalize output\n\nAdd shared metadata for diagnostics.",
      { provider: "google-gemini", maxSubjectLength: 72 }
    );

    expect(message).toBe("feat(provider): normalize output\n\nAdd shared metadata for diagnostics.");
  });

  it("allows non-conventional output for custom style", () => {
    const message = sanitizeCommitMessage("Update provider adapter behavior", {
      provider: "openai-compatible",
      maxSubjectLength: 72,
      messageStyle: "custom"
    });

    expect(message).toBe("Update provider adapter behavior");
  });

  it("allows non-conventional output for google style", () => {
    const message = sanitizeCommitMessage("Update provider adapter behavior", {
      provider: "openai-compatible",
      maxSubjectLength: 72,
      messageStyle: "google"
    });

    expect(message).toBe("Update provider adapter behavior");
  });

  it("rejects empty output with a typed provider error", () => {
    expect(() => sanitizeCommitMessage("   ", { provider: "openai-responses", maxSubjectLength: 72 }))
      .toThrow(ProviderError);

    try {
      sanitizeCommitMessage("   ", { provider: "openai-responses", maxSubjectLength: 72 });
    } catch (error) {
      expect(error).toBeInstanceOf(ProviderError);
      expect((error as ProviderError).code).toBe("empty_output");
    }
  });

  it("rejects malformed output with a typed provider error", () => {
    expect(() => sanitizeCommitMessage("updated files", { provider: "google-gemini", maxSubjectLength: 72 }))
      .toThrow(ProviderError);

    try {
      sanitizeCommitMessage("updated files", { provider: "google-gemini", maxSubjectLength: 72 });
    } catch (error) {
      expect(error).toBeInstanceOf(ProviderError);
      expect((error as ProviderError).code).toBe("malformed_output");
    }
  });
});
