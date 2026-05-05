import type { ConfigValues, PromptStyle, ResolvedConfig, ResolveConfigOptions } from "../config/index.js";
import type {
  GitChangeKind,
  GitFileStatus,
  GitInspection,
  InspectGitRepositoryOptions,
  StageAllChangesOptions,
  StageAllChangesResult
} from "../git/index.js";
import type { ExitCode } from "../output/index.js";
import type {
  CommitMessageProvider,
  ProviderConfig,
  AiProviderName
} from "../providers/index.js";

export type StageAllDecision = "stage" | "skip" | "cancel";
export type PreviewAction = "confirm" | "edit" | "regenerate" | "cancel";

export interface WorkflowFileView {
  path: string;
  staged: GitChangeKind | null;
  unstaged: GitChangeKind | null;
  untracked: boolean;
  binary: boolean;
}

export interface StageAllPromptInput {
  files: WorkflowFileView[];
}

export interface PreviewPromptInput {
  attempt: number;
  dryRun: boolean;
  files: WorkflowFileView[];
  message: string;
  operation: string;
  warnings: string[];
}

export interface EditMessagePromptInput {
  currentMessage: string;
  promptStyle?: PromptStyle | string;
  validationMessage?: string;
}

export interface CommitWorkflowPrompts {
  confirmStageAll(input: StageAllPromptInput): Promise<StageAllDecision>;
  editMessage(input: EditMessagePromptInput): Promise<string | null>;
  selectPreviewAction(input: PreviewPromptInput): Promise<PreviewAction>;
}

export interface CommitRunnerOptions {
  cwd: string;
  env?: NodeJS.ProcessEnv;
  message: string;
}

export interface CommitRunnerResult {
  exitCode: number;
  stderr: string;
  stdout: string;
}

export interface CommitWorkflowDependencies {
  commitRunner(options: CommitRunnerOptions): Promise<CommitRunnerResult>;
  createCommitMessageProvider(config: ProviderConfig): CommitMessageProvider;
  inspectGitRepository(options: InspectGitRepositoryOptions): Promise<GitInspection>;
  prompts: CommitWorkflowPrompts;
  resolveEffectiveConfig(options: ResolveConfigOptions): Promise<ResolvedConfig>;
  stageAllChanges(options: StageAllChangesOptions): Promise<StageAllChangesResult>;
}

export interface RunCommitWorkflowOptions {
  cwd: string;
  dependencies: CommitWorkflowDependencies;
  dryRun: boolean;
  env?: NodeJS.ProcessEnv;
  flagOverrides?: ConfigValues;
  isTty: boolean;
  json: boolean;
}

export interface CommitWorkflowJsonResult {
  command: string;
  committed: boolean;
  dryRun: boolean;
  error: string | null;
  files: WorkflowFileView[];
  message: string | null;
  ok: boolean;
  provider:
    | {
        model: string;
        name: AiProviderName;
      }
    | null;
  status: string;
  warnings: string[];
}

export interface CommitWorkflowResult extends CommitWorkflowJsonResult {
  exitCode: ExitCode;
  previewShown: boolean;
}

export function toWorkflowFileView(file: GitFileStatus): WorkflowFileView {
  return {
    path: file.path,
    staged: file.staged,
    unstaged: file.unstaged,
    untracked: file.untracked,
    binary: file.binary
  };
}
