import type { SecretScanResult } from "../security/index.js";

export type GitIssueSeverity = "warning" | "blocking";

export interface GitIssue {
  code: string;
  message: string;
  severity: GitIssueSeverity;
}

export type GitChangeKind =
  | "added"
  | "copied"
  | "deleted"
  | "modified"
  | "renamed"
  | "typechange"
  | "unmerged"
  | "unknown";

export interface GitFileStatus {
  path: string;
  originalPath?: string;
  staged: GitChangeKind | null;
  unstaged: GitChangeKind | null;
  untracked: boolean;
  binary: boolean;
}

export interface GitDiffMetadata {
  bytes: number;
  originalBytes: number;
  truncated: boolean;
  omittedBytes: number;
  maxBytes: number;
}

export interface GitRepositoryState {
  isRepository: boolean;
  rootPath: string | null;
  gitDirPath: string | null;
  isBare: boolean;
  isInitialCommit: boolean;
  isDetachedHead: boolean;
  branchName: string | null;
  mergeInProgress: boolean;
  rebaseInProgress: boolean;
  cherryPickInProgress: boolean;
  hasGitIdentity: boolean;
  gitIdentity: {
    name: string | null;
    email: string | null;
  };
}

export interface GitInspection {
  repository: GitRepositoryState;
  files: GitFileStatus[];
  stagedFiles: GitFileStatus[];
  unstagedFiles: GitFileStatus[];
  untrackedFiles: GitFileStatus[];
  stagedDiff: string;
  diff: GitDiffMetadata;
  secretScan: SecretScanResult;
  issues: GitIssue[];
  warnings: GitIssue[];
  blockingIssues: GitIssue[];
  hasStagedChanges: boolean;
  hasUnstagedChanges: boolean;
  hasUntrackedFiles: boolean;
}

export interface GitCommandResult {
  stdout: string;
  stderr: string;
  exitCode: number;
}

export type GitCommandRunner = (
  cwd: string,
  args: string[],
  env?: NodeJS.ProcessEnv
) => Promise<GitCommandResult>;

export interface InspectGitRepositoryOptions {
  cwd: string;
  maxDiffBytes?: number;
  env?: NodeJS.ProcessEnv;
  gitRunner?: GitCommandRunner;
}

export interface StageAllChangesOptions {
  cwd: string;
  confirmed: boolean;
  isTty?: boolean;
  maxDiffBytes?: number;
  env?: NodeJS.ProcessEnv;
  gitRunner?: GitCommandRunner;
}

export interface StageAllChangesResult {
  staged: boolean;
  reason: "confirmed" | "not_confirmed" | "non_tty" | "git_add_failed";
  inspection: GitInspection;
}
