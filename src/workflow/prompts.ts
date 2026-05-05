import { confirm, isCancel, note, select, text } from "@clack/prompts";

import type {
  CommitWorkflowPrompts,
  EditMessagePromptInput,
  PreviewAction,
  PreviewPromptInput,
  StageAllPromptInput,
  StageAllDecision,
  WorkflowFileView
} from "./types.js";

const CONVENTIONAL_SUBJECT = /^(build|chore|ci|docs|feat|fix|perf|refactor|revert|style|test)(\([^)\r\n]+\))?!?: .+/;
const MAX_CONVENTIONAL_SUBJECT_LENGTH = 72;

export interface CreateWorkflowPromptsOptions {
  stdout?: NodeJS.WriteStream;
}

function unexpectedPromptCall(name: keyof CommitWorkflowPrompts): never {
  throw new Error(`Internal error: workflow prompt '${name}' should not run in non-interactive mode.`);
}

function formatStage(file: WorkflowFileView): string {
  const labels: string[] = [];

  if (file.staged) {
    labels.push(`staged:${file.staged}`);
  }

  if (file.unstaged) {
    labels.push(`unstaged:${file.unstaged}`);
  }

  if (file.untracked) {
    labels.push("untracked");
  }

  if (file.binary) {
    labels.push("binary");
  }

  if (labels.length === 0) {
    labels.push("unchanged");
  }

  return labels.join(", ");
}

function formatFiles(title: string, files: WorkflowFileView[]): string {
  if (files.length === 0) {
    return `${title}\n- none`;
  }

  return [title, ...files.map((file) => `- ${formatStage(file)} ${file.path}`)].join("\n");
}

function formatWarnings(warnings: string[]): string {
  if (warnings.length === 0) {
    return "Warnings\n- none";
  }

  return ["Warnings", ...warnings.map((warning) => `- ${warning}`)].join("\n");
}

export function validateEditedMessage(message: string, promptStyle: string = "conventional"): string | undefined {
  const trimmedMessage = message.trim();

  if (trimmedMessage.length === 0) {
    return "Commit message cannot be empty.";
  }

  if (promptStyle !== "conventional" && promptStyle !== "angular") {
    return undefined;
  }

  const subject = trimmedMessage.split(/\r?\n/, 1)[0] ?? "";

  if (!CONVENTIONAL_SUBJECT.test(subject) || subject.length > MAX_CONVENTIONAL_SUBJECT_LENGTH) {
    return "Use a Conventional Commit subject like feat(scope): summary (72 chars max).";
  }

  return undefined;
}

export function renderPreview(input: PreviewPromptInput): string {
  return [
    formatFiles("Files", input.files),
    formatWarnings(input.warnings),
    "Commit message",
    input.message,
    `Operation\n- ${input.operation}`,
    `Mode\n- ${input.dryRun ? "dry-run preview only" : "interactive confirm before commit"}`,
    `Attempt\n- ${input.attempt}`
  ].join("\n\n");
}

async function promptStageAll(input: StageAllPromptInput, output?: NodeJS.WriteStream): Promise<StageAllDecision> {
  note(formatFiles("Current changes", input.files), "cnm", { output });

  const result = await confirm({
    initialValue: false,
    message: "No staged changes found. Stage all current changes?",
    output
  });

  if (isCancel(result)) {
    return "cancel";
  }

  return result ? "stage" : "skip";
}

async function promptPreview(input: PreviewPromptInput, output?: NodeJS.WriteStream): Promise<PreviewAction> {
  note(renderPreview(input), "cnm preview", { output });

  const action = await select({
    initialValue: "confirm",
    message: input.dryRun ? "Review the dry-run preview." : "Choose the next action.",
    options: [
      { label: input.dryRun ? "Finish dry-run" : "Confirm commit", value: "confirm" },
      { label: "Edit commit message", value: "edit" },
      { label: "Regenerate message", value: "regenerate" },
      { label: "Cancel", value: "cancel" }
    ],
    output
  });

  if (isCancel(action)) {
    return "cancel";
  }

  return action as PreviewAction;
}

async function promptEditMessage(input: EditMessagePromptInput, output?: NodeJS.WriteStream): Promise<string | null> {
  if (input.validationMessage) {
    note(input.validationMessage, "cnm validation", { output });
  }

  const value = await text({
    initialValue: input.currentMessage,
    message: "Edit the commit message.",
    output,
    validate(nextValue) {
      return validateEditedMessage(nextValue ?? "", input.promptStyle ?? "conventional");
    }
  });

  if (isCancel(value)) {
    return null;
  }

  return value.trim();
}

export function createClackWorkflowPrompts({ stdout }: CreateWorkflowPromptsOptions = {}): CommitWorkflowPrompts {
  return {
    confirmStageAll(input) {
      return promptStageAll(input, stdout);
    },
    editMessage(input) {
      return promptEditMessage(input, stdout);
    },
    selectPreviewAction(input) {
      return promptPreview(input, stdout);
    }
  };
}

export function createNonInteractiveWorkflowPrompts(): CommitWorkflowPrompts {
  return {
    async confirmStageAll() {
      return unexpectedPromptCall("confirmStageAll");
    },
    async editMessage() {
      return unexpectedPromptCall("editMessage");
    },
    async selectPreviewAction() {
      return unexpectedPromptCall("selectPreviewAction");
    }
  };
}
