import type { GenerateCommitMessageInput } from "../providers/types.js";

export interface CommitPromptParts {
  system: string;
  user: string;
}

const DEFAULT_MAX_SUBJECT_LENGTH = 72;

function styleInstruction(style: string, maxSubjectLength: number): string | null {
  switch (style) {
    case "auto":
      return [
        "Infer the commit message style from repository context when possible.",
        "If the input does not provide enough history/context to infer a style, use Conventional Commits: type(scope)?: subject with an optional body."
      ].join(" ");
    case "conventional":
      return "Use Conventional Commits: type(scope)?: subject with an optional body separated by one blank line.";
    case "angular":
      return [
        "Use Angular commit format: type(scope): subject.",
        "Use a lowercase type such as build, chore, ci, docs, feat, fix, perf, refactor, revert, style, or test.",
        "Use an imperative, lowercase subject without a trailing period."
      ].join(" ");
    case "google":
      return [
        "Use Google-style commit message guidance: a short, specific, imperative subject line with no trailing period.",
        "After a blank line, include a body only when useful to explain what changed and why."
      ].join(" ");
    case "atom":
      return [
        "Use Atom-style commit messages: a concise imperative subject line, optionally followed by a body with supporting details.",
        "Do not require a Conventional Commit type prefix unless it is clearly natural for the repository."
      ].join(" ");
    case "plain":
      return "Use a plain, concise natural-language commit message. Do not require a type prefix or strict format.";
    case "custom":
      return null;
    default:
      return `Use ${style} style. Keep the subject line at or below ${maxSubjectLength} characters when possible.`;
  }
}

export function resolveMaxSubjectLength(input: GenerateCommitMessageInput): number {
  return input.maxSubjectLength ?? DEFAULT_MAX_SUBJECT_LENGTH;
}

export function buildCommitMessagePrompt(input: GenerateCommitMessageInput): CommitPromptParts {
  const maxSubjectLength = resolveMaxSubjectLength(input);
  const style = input.messageStyle ?? "conventional";
  const instruction = styleInstruction(style, maxSubjectLength);

  const customPrompt = input.customPrompt?.trim();
  const customPromptSection = customPrompt === undefined || customPrompt.length === 0
    ? ""
    : `Additional user guidance:\n${customPrompt}`;
  const styleInstructions = [
    instruction,
    customPromptSection.length > 0 ? customPromptSection : null
  ].filter((part): part is string => part !== null && part.length > 0);

  return {
    system: [
      "You write git commit messages from staged changes.",
      "Return only the commit message, with no markdown, labels, explanation, or surrounding quotes.",
      "Output the final commit message directly; do not include reasoning or analysis.",
      `Keep the subject line at or below ${maxSubjectLength} characters when possible.`,
      "Do not invent details that are not supported by the diff.",
      ...styleInstructions
    ].join("\n"),
    user: [
      formatRepo(input),
      formatRecentCommits(input),
      formatFiles(input.files),
      "Diff:",
      input.diff
    ].filter((part) => part.length > 0).join("\n\n")
  };
}

function formatRepo(input: GenerateCommitMessageInput): string {
  if (input.repo === undefined) {
    return "";
  }

  const lines = [
    input.repo.root === undefined ? undefined : `root: ${input.repo.root}`,
    input.repo.branch === undefined ? undefined : `branch: ${input.repo.branch}`,
    input.repo.remote === undefined ? undefined : `remote: ${input.repo.remote}`
  ].filter((line) => line !== undefined);

  return lines.length === 0 ? "" : `Repository metadata:\n${lines.join("\n")}`;
}

function formatRecentCommits(input: GenerateCommitMessageInput): string {
  if (!input.recentCommits || input.recentCommits.length === 0) {
    return "";
  }

  const commits = input.recentCommits
    .map((commit, index) => `${index + 1}. ${commit}`)
    .join("\n");

  return `Recent commit messages (for style reference):\n${commits}`;
}

function formatFiles(files: GenerateCommitMessageInput["files"]): string {
  if (files.length === 0) {
    return "Changed files: none provided";
  }

  return [
    "Changed files:",
    ...files.map((file) => `- ${file.status === undefined ? file.path : `${file.status} ${file.path}`}`)
  ].join("\n");
}
