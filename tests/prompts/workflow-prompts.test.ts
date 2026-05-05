import { afterEach, describe, expect, it, vi } from "vitest";

const clack = vi.hoisted(() => {
  return {
    confirm: vi.fn(),
    isCancel: vi.fn(),
    note: vi.fn(),
    select: vi.fn(),
    text: vi.fn()
  };
});

vi.mock("@clack/prompts", () => clack);

import {
  createClackWorkflowPrompts,
  renderPreview,
  validateEditedMessage
} from "../../src/workflow/prompts.js";

const CANCELLED = Symbol("cancelled");

describe("workflow prompts", () => {
  afterEach(() => {
    clack.confirm.mockReset();
    clack.isCancel.mockReset();
    clack.note.mockReset();
    clack.select.mockReset();
    clack.text.mockReset();
    clack.isCancel.mockImplementation((value: unknown) => value === CANCELLED);
  });

  it("renders preview details with files, warnings, operation, and attempt count", () => {
    const preview = renderPreview({
      attempt: 2,
      dryRun: false,
      files: [
        {
          path: "src/workflow/service.ts",
          staged: "modified",
          unstaged: "modified",
          untracked: false,
          binary: false
        },
        {
          path: "notes.md",
          staged: null,
          unstaged: null,
          untracked: true,
          binary: false
        }
      ],
      message: "fix(workflow): keep invalid edits out of commits",
      operation: "git commit",
      warnings: ["Diff truncated to staged hunks."]
    });

    expect(preview).toContain("Files");
    expect(preview).toContain("staged:modified, unstaged:modified src/workflow/service.ts");
    expect(preview).toContain("untracked notes.md");
    expect(preview).toContain("Warnings");
    expect(preview).toContain("Diff truncated to staged hunks.");
    expect(preview).toContain("Commit message");
    expect(preview).toContain("fix(workflow): keep invalid edits out of commits");
    expect(preview).toContain("Operation\n- git commit");
    expect(preview).toContain("Attempt\n- 2");
  });

  it("validates conventional commit edits by default", () => {
    expect(validateEditedMessage("   ")).toBe("Commit message cannot be empty.");
    expect(validateEditedMessage("updated files")).toContain("Conventional Commit");
    expect(validateEditedMessage("feat(workflow): validate edited message")).toBeUndefined();
    expect(validateEditedMessage("docs: add workflow note\n\nExplain preview behavior.")).toBeUndefined();
  });

  it("does not enforce conventional commit edits for google, plain, or custom styles", () => {
    expect(validateEditedMessage("Update parser behavior", "google")).toBeUndefined();
    expect(validateEditedMessage("Update parser behavior", "plain")).toBeUndefined();
    expect(validateEditedMessage("Update parser behavior", "custom")).toBeUndefined();
  });

  it("maps preview Ctrl-C to cancel", async () => {
    clack.select.mockResolvedValue(CANCELLED);
    const prompts = createClackWorkflowPrompts();

    const action = await prompts.selectPreviewAction({
      attempt: 1,
      dryRun: false,
      files: [],
      message: "feat: add test",
      operation: "git commit",
      warnings: []
    });

    expect(action).toBe("cancel");
  });

  it("returns null when edit is cancelled and shows validation notes on retry", async () => {
    clack.text.mockResolvedValue(CANCELLED);
    const prompts = createClackWorkflowPrompts();

    const result = await prompts.editMessage({
      currentMessage: "feat(workflow): add preview",
      promptStyle: "conventional",
      validationMessage: "Use a Conventional Commit subject like feat(scope): summary (72 chars max)."
    });

    expect(result).toBeNull();
    expect(clack.note).toHaveBeenCalledWith(
      "Use a Conventional Commit subject like feat(scope): summary (72 chars max).",
      "cnm validation",
      { output: undefined }
    );
  });

  it("passes conventional validation into the inline text editor", async () => {
    clack.text.mockImplementation(async (options: { validate?: (value: string) => string | undefined }) => {
      expect(options.validate?.("updated files")).toContain("Conventional Commit");
      expect(options.validate?.("feat(workflow): accept edited subject")).toBeUndefined();
      return "feat(workflow): accept edited subject";
    });
    const prompts = createClackWorkflowPrompts();

    const result = await prompts.editMessage({
      currentMessage: "feat(workflow): existing subject",
      promptStyle: "conventional"
    });

    expect(result).toBe("feat(workflow): accept edited subject");
  });
});
