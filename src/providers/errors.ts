import type { AiProviderName } from "./types.js";

export type ProviderErrorCode =
  | "missing_config"
  | "missing_api_key"
  | "empty_output"
  | "malformed_output"
  | "provider_failure";

export interface ProviderErrorOptions {
  code: ProviderErrorCode;
  provider: AiProviderName;
  message: string;
  cause?: unknown;
}

export class ProviderError extends Error {
  readonly code: ProviderErrorCode;
  readonly provider: AiProviderName;

  constructor({ code, provider, message, cause }: ProviderErrorOptions) {
    super(message, { cause });
    this.name = "ProviderError";
    this.code = code;
    this.provider = provider;
  }
}

export function missingApiKey(provider: AiProviderName): ProviderError {
  return new ProviderError({
    code: "missing_api_key",
    provider,
    message: `Missing API key for ${provider}.`
  });
}

export function missingConfig(provider: AiProviderName, field: string): ProviderError {
  return new ProviderError({
    code: "missing_config",
    provider,
    message: `Missing ${field} for ${provider}.`
  });
}

export function providerFailure(provider: AiProviderName, cause: unknown): ProviderError {
  const details = summarizeErrorCause(cause);

  return new ProviderError({
    code: "provider_failure",
    provider,
    message: details.length === 0
      ? `${provider} failed to generate a commit message.`
      : `${provider} failed to generate a commit message: ${details}.`,
    cause
  });
}

function summarizeErrorCause(cause: unknown): string {
  if (cause instanceof Error) {
    return summarizeErrorObject(cause);
  }

  if (typeof cause === "string") {
    return truncateDetail(cause);
  }

  if (cause === null || typeof cause !== "object") {
    return "";
  }

  return summarizeErrorObject(cause as Record<string, unknown>);
}

function summarizeErrorObject(error: Error | Record<string, unknown>): string {
  const details = [
    readErrorField(error, "status"),
    readErrorField(error, "statusCode"),
    readErrorField(error, "code"),
    readErrorField(error, "message")
  ];
  const uniqueDetails = details.filter((detail, index) => detail.length > 0 && details.indexOf(detail) === index);

  return truncateDetail(uniqueDetails.join(" "));
}

function readErrorField(error: Error | Record<string, unknown>, field: string): string {
  const value = field in error ? error[field as keyof typeof error] : undefined;

  if (typeof value === "number") {
    return String(value);
  }

  return typeof value === "string" ? value.trim() : "";
}

function truncateDetail(value: string): string {
  return value.replace(/\s+/g, " ").trim().slice(0, 300);
}
