import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";

const clack = vi.hoisted(() => {
  return {
    confirm: vi.fn(),
    isCancel: vi.fn(),
    multiline: vi.fn(),
    password: vi.fn(),
    select: vi.fn(),
    text: vi.fn()
  };
});

vi.mock("@clack/prompts", () => clack);

import { createClackConfigPrompts } from "../../src/commands/config-prompts.js";

const CANCELLED = Symbol("cancelled");

describe("config prompts", () => {
  beforeEach(() => {
    clack.isCancel.mockImplementation((value: unknown) => value === CANCELLED);
  });

  afterEach(() => {
    clack.confirm.mockReset();
    clack.isCancel.mockReset();
    clack.password.mockReset();
    clack.select.mockReset();
    clack.text.mockReset();
  });

  it("maps menu cancellation to null", async () => {
    clack.select.mockResolvedValue(CANCELLED);
    const prompts = createClackConfigPrompts();

    const action = await prompts.selectAction({
      effectiveConfig: {
        model: "gpt-5-mini",
        promptStyle: "conventional",
        provider: "openai-responses"
      },
      userConfig: {}
    });

    expect(action).toBeNull();
  });

  it("maps api key prompt cancellation to null", async () => {
    clack.password.mockResolvedValue(CANCELLED);
    const prompts = createClackConfigPrompts();

    const apiKey = await prompts.inputApiKey({ hasExistingValue: false });

    expect(apiKey).toBeNull();
  });

  it("passes required validation into text prompts", async () => {
    let textCalls = 0;
    clack.text.mockImplementation(async (options: { validate?: (value: string) => string | undefined }) => {
      textCalls += 1;

      if (textCalls === 1) {
        expect(options.validate).toBeUndefined();
        return "https://valid.example/v1";
      }

      expect(options.validate).toBeUndefined();
      return "Use concise wording with spaces.";
    });
    const prompts = createClackConfigPrompts();

    const baseURL = await prompts.inputBaseURL({ currentValue: undefined });
    const customPrompt = await prompts.inputCustomPrompt({ currentValue: undefined });

    expect(baseURL).toBe("https://valid.example/v1");
    expect(customPrompt).toBe("Use concise wording with spaces.");
  });

  it("selects a prompt style", async () => {
    clack.select.mockResolvedValue("google");
    const prompts = createClackConfigPrompts();

    const promptStyle = await prompts.selectPromptStyle({ currentPromptStyle: "auto" });

    expect(promptStyle).toBe("google");
    expect(clack.select).toHaveBeenCalledWith(expect.objectContaining({
      message: "Choose the commit message style.",
      options: expect.arrayContaining([
        expect.objectContaining({ value: "auto" }),
        expect.objectContaining({ value: "custom" })
      ])
    }));
  });
});
