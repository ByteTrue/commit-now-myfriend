export type {
  GitChangeKind,
  GitCommandResult,
  GitCommandRunner,
  GitDiffMetadata,
  GitFileStatus,
  GitInspection,
  GitIssue,
  GitIssueSeverity,
  GitRepositoryState,
  InspectGitRepositoryOptions,
  StageAllChangesOptions,
  StageAllChangesResult
} from "./types.js";
export { getRecentCommits, inspectGitRepository, stageAllChanges } from "./service.js";
