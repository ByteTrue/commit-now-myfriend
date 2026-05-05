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
  return new ProviderError({
    code: "provider_failure",
    provider,
    message: `${provider} failed to generate a commit message.`,
    cause
  });
}
