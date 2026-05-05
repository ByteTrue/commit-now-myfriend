export type AiProviderName =
  | "openai-compatible"
  | "openai-responses"
  | "anthropic-messages"
  | "google-gemini";

export interface CommitFileInput {
  path: string;
  status?: string;
}

export interface CommitRepoMetadata {
  root?: string;
  branch?: string;
  remote?: string;
}

export interface GenerateCommitMessageInput {
  files: CommitFileInput[];
  diff: string;
  repo?: CommitRepoMetadata;
  recentCommits?: string[];
  messageStyle?: "conventional" | string;
  customPrompt?: string;
  maxSubjectLength?: number;
}

export interface CommitMessageResult {
  message: string;
  metadata: {
    provider: AiProviderName;
    model: string;
    responseId?: string;
    finishReason?: string;
    usage?: unknown;
  };
}

export interface CommitMessageProvider {
  readonly name: AiProviderName;
  generateCommitMessage(input: GenerateCommitMessageInput): Promise<CommitMessageResult>;
}

export interface BaseProviderConfig {
  provider: AiProviderName;
  apiKey?: string;
  model: string;
  maxOutputTokens?: number;
}

export interface OpenAiCompatibleProviderConfig extends BaseProviderConfig {
  provider: "openai-compatible";
  baseURL: string;
}

export interface OpenAiResponsesProviderConfig extends BaseProviderConfig {
  provider: "openai-responses";
}

export interface AnthropicMessagesProviderConfig extends BaseProviderConfig {
  provider: "anthropic-messages";
}

export interface GoogleGeminiProviderConfig extends BaseProviderConfig {
  provider: "google-gemini";
}

export type ProviderConfig =
  | OpenAiCompatibleProviderConfig
  | OpenAiResponsesProviderConfig
  | AnthropicMessagesProviderConfig
  | GoogleGeminiProviderConfig;

export interface OpenAiCompatibleClient {
  chat: {
    completions: {
      create(input: {
        model: string;
        messages: Array<{ role: "system" | "user"; content: string }>;
        temperature: number;
        max_tokens: number;
      }): Promise<{
        id?: string;
        model?: string;
        choices: Array<{
          finish_reason?: string | null;
          message?: { content?: string | null };
        }>;
        usage?: unknown;
      }>;
    };
  };
}

export interface OpenAiResponsesClient {
  responses: {
    create(input: {
      model: string;
      instructions: string;
      input: string;
      temperature: number;
      max_output_tokens: number;
    }): Promise<{
      id?: string;
      model?: string;
      output_text?: string;
      usage?: unknown;
    }>;
  };
}

export interface AnthropicMessagesClient {
  messages: {
    create(input: {
      model: string;
      system: string;
      max_tokens: number;
      messages: Array<{ role: "user"; content: string }>;
    }): Promise<{
      id?: string;
      model?: string;
      stop_reason?: string | null;
      content: Array<{ type: string; text?: string }>;
      usage?: unknown;
    }>;
  };
}

export interface GoogleGeminiClient {
  models: {
    generateContent(input: {
      model: string;
      contents: string;
      config: {
        systemInstruction: string;
        temperature: number;
        maxOutputTokens: number;
      };
    }): Promise<{
      responseId?: string;
      modelVersion?: string;
      text?: string;
      usageMetadata?: unknown;
    }>;
  };
}

export interface ProviderClientFactories {
  openAiCompatible?: (config: OpenAiCompatibleProviderConfig) => OpenAiCompatibleClient;
  openAiResponses?: (config: OpenAiResponsesProviderConfig) => OpenAiResponsesClient;
  anthropicMessages?: (config: AnthropicMessagesProviderConfig) => AnthropicMessagesClient;
  googleGemini?: (config: GoogleGeminiProviderConfig) => GoogleGeminiClient;
}
