import { ProviderError } from "./errors.js";
import type { AiProviderName } from "./types.js";

const CONVENTIONAL_SUBJECT = /^(build|chore|ci|docs|feat|fix|perf|refactor|revert|style|test)(\([^)\r\n]+\))?!?: .+/;

export interface SanitizeOptions {
  messageStyle?: string;
  provider: AiProviderName;
  maxSubjectLength: number;
}

const STRICT_CONVENTIONAL_STYLES = new Set(["angular", "conventional"]);

export function sanitizeCommitMessage(output: string | null | undefined, options: SanitizeOptions): string {
  const trimmed = stripMarkdownFence((output ?? "").trim());

  if (trimmed.length === 0) {
    throw new ProviderError({
      code: "empty_output",
      provider: options.provider,
      message: `${options.provider} returned an empty commit message.`
    });
  }

  const lines = trimmed
    .split(/\r?\n/)
    .map((line) => line.trimEnd());
  const strictConventional = options.messageStyle === undefined || STRICT_CONVENTIONAL_STYLES.has(options.messageStyle);
  const firstSubjectIndex = strictConventional
    ? lines.findIndex((line) => CONVENTIONAL_SUBJECT.test(line.trim()))
    : lines.findIndex((line) => line.trim().length > 0);

  if (firstSubjectIndex === -1) {
    throw new ProviderError({
      code: "malformed_output",
      provider: options.provider,
      message: `${options.provider} returned a malformed commit message.`
    });
  }

  const commitLines = lines.slice(firstSubjectIndex);
  const message = commitLines.join("\n").trim();
  const subject = message.split(/\r?\n/, 1)[0] ?? "";

  if (subject.length === 0) {
    throw new ProviderError({
      code: "empty_output",
      provider: options.provider,
      message: `${options.provider} returned an empty commit message.`
    });
  }

  if ((strictConventional && !CONVENTIONAL_SUBJECT.test(subject)) || subject.length > options.maxSubjectLength) {
    throw new ProviderError({
      code: "malformed_output",
      provider: options.provider,
      message: `${options.provider} returned an invalid commit subject.`
    });
  }

  return message;
}

function stripMarkdownFence(output: string): string {
  return output.replace(/^```[a-zA-Z0-9_-]*\s*\n([\s\S]*?)\n```$/u, "$1").trim();
}
