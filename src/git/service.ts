import { access } from "node:fs/promises";
import path from "node:path";

import { execa } from "execa";

import { scanTextForSecrets } from "../security/index.js";
import type {
  GitChangeKind,
  GitCommandRunner,
  GitDiffMetadata,
  GitFileStatus,
  GitInspection,
  GitIssue,
  GitRepositoryState,
  InspectGitRepositoryOptions,
  StageAllChangesOptions,
  StageAllChangesResult
} from "./types.js";

const defaultMaxDiffBytes = 200_000;

const defaultGitRunner: GitCommandRunner = async (cwd, args, env) => {
  const result = await execa("git", args, {
    cwd,
    env,
    reject: false,
    stdin: "ignore",
    stdout: "pipe",
    stderr: "pipe"
  });

  return {
    stdout: result.stdout,
    stderr: result.stderr,
    exitCode: result.exitCode ?? 1
  };
};

function formatGitFailure(command: string, stderr: string, stdout: string): string {
  const output = [stderr.trim(), stdout.trim()].filter(Boolean).join("\n");

  return output ? `${command} failed.\n${output}` : `${command} failed.`;
}

function addBlockingIssue(inspection: GitInspection, issue: GitIssue): GitInspection {
  return {
    ...inspection,
    blockingIssues: [...inspection.blockingIssues, issue],
    issues: [...inspection.issues, issue]
  };
}

function toAbsolutePath(basePath: string, candidatePath: string): string {
  return path.isAbsolute(candidatePath) ? candidatePath : path.resolve(basePath, candidatePath);
}

async function exists(candidatePath: string): Promise<boolean> {
  try {
    await access(candidatePath);
    return true;
  } catch {
    return false;
  }
}

function changeKind(status: string): GitChangeKind | null {
  switch (status) {
    case "A":
      return "added";
    case "C":
      return "copied";
    case "D":
      return "deleted";
    case "M":
      return "modified";
    case "R":
      return "renamed";
    case "T":
      return "typechange";
    case "U":
      return "unmerged";
    case "?":
      return "unknown";
    case " ":
      return null;
    default:
      return status ? "unknown" : null;
  }
}

function mergeFileStatus(files: Map<string, GitFileStatus>, nextFile: GitFileStatus): void {
  const existing = files.get(nextFile.path);
  if (!existing) {
    files.set(nextFile.path, nextFile);
    return;
  }

  files.set(nextFile.path, {
    path: existing.path,
    originalPath: nextFile.originalPath ?? existing.originalPath,
    staged: nextFile.staged ?? existing.staged,
    unstaged: nextFile.unstaged ?? existing.unstaged,
    untracked: existing.untracked || nextFile.untracked,
    binary: existing.binary || nextFile.binary
  });
}

function parsePorcelainStatus(output: string): GitFileStatus[] {
  if (!output) {
    return [];
  }

  const entries = output.split("\0").filter(Boolean);
  const files = new Map<string, GitFileStatus>();

  for (let index = 0; index < entries.length; index += 1) {
    const entry = entries[index] ?? "";
    const x = entry[0] ?? " ";
    const y = entry[1] ?? " ";
    const rawPath = entry.slice(3);

    if (x === "?" && y === "?") {
      mergeFileStatus(files, {
        path: rawPath,
        staged: null,
        unstaged: null,
        untracked: true,
        binary: false
      });
      continue;
    }

    if ((x === "R" || x === "C") && entries[index + 1]) {
      const originalPath = entries[index + 1];
      index += 1;
      mergeFileStatus(files, {
        path: rawPath,
        originalPath,
        staged: changeKind(x),
        unstaged: changeKind(y),
        untracked: false,
        binary: false
      });
      continue;
    }

    mergeFileStatus(files, {
      path: rawPath,
      staged: changeKind(x),
      unstaged: changeKind(y),
      untracked: false,
      binary: false
    });
  }

  return [...files.values()].sort((left, right) => left.path.localeCompare(right.path));
}

function applyBinaryMetadata(files: GitFileStatus[], numstatOutput: string): GitFileStatus[] {
  if (!numstatOutput) {
    return files;
  }

  const binaryPaths = new Set<string>();
  const entries = numstatOutput.split("\0").filter(Boolean);

  for (const entry of entries) {
    const [added, deleted, filePath] = entry.split("\t");
    if (added === "-" && deleted === "-" && filePath) {
      binaryPaths.add(filePath);
    }
  }

  return files.map((file) => ({
    ...file,
    binary: file.binary || binaryPaths.has(file.path)
  }));
}

function truncateDiff(diff: string, maxBytes: number): { diff: string; metadata: GitDiffMetadata } {
  const buffer = Buffer.from(diff, "utf8");

  if (buffer.byteLength <= maxBytes) {
    return {
      diff,
      metadata: {
        bytes: buffer.byteLength,
        originalBytes: buffer.byteLength,
        truncated: false,
        omittedBytes: 0,
        maxBytes
      }
    };
  }

  const truncated = buffer.subarray(0, maxBytes).toString("utf8");
  const suffix = `\n[cnm: diff truncated; omitted ${buffer.byteLength - maxBytes} bytes]\n`;

  return {
    diff: `${truncated}${suffix}`,
    metadata: {
      bytes: Buffer.byteLength(`${truncated}${suffix}`, "utf8"),
      originalBytes: buffer.byteLength,
      truncated: true,
      omittedBytes: buffer.byteLength - maxBytes,
      maxBytes
    }
  };
}

function createIssue(code: string, message: string, severity: "warning" | "blocking"): GitIssue {
  return { code, message, severity };
}

async function detectRepository(cwd: string, env: NodeJS.ProcessEnv | undefined, gitRunner: GitCommandRunner): Promise<GitRepositoryState> {
  const bareResult = await gitRunner(cwd, ["rev-parse", "--is-bare-repository"], env);

  if (bareResult.exitCode !== 0) {
    return {
      isRepository: false,
      rootPath: null,
      gitDirPath: null,
      isBare: false,
      isInitialCommit: false,
      isDetachedHead: false,
      branchName: null,
      mergeInProgress: false,
      rebaseInProgress: false,
      cherryPickInProgress: false,
      hasGitIdentity: false,
      gitIdentity: { name: null, email: null }
    };
  }

  const isBare = bareResult.stdout.trim() === "true";
  const rootResult = await gitRunner(cwd, ["rev-parse", "--show-toplevel"], env);
  const gitDirResult = await gitRunner(cwd, ["rev-parse", "--git-dir"], env);
  const rootPath = rootResult.exitCode === 0 && rootResult.stdout.trim() ? rootResult.stdout.trim() : cwd;
  const gitDirPath = gitDirResult.exitCode === 0 && gitDirResult.stdout.trim() ? toAbsolutePath(rootPath, gitDirResult.stdout.trim()) : null;
  const headResult = await gitRunner(cwd, ["rev-parse", "--verify", "HEAD"], env);
  const branchResult = await gitRunner(cwd, ["symbolic-ref", "--quiet", "--short", "HEAD"], env);
  const nameResult = await gitRunner(cwd, ["config", "--get", "user.name"], env);
  const emailResult = await gitRunner(cwd, ["config", "--get", "user.email"], env);
  const mergeInProgress = gitDirPath ? await exists(path.join(gitDirPath, "MERGE_HEAD")) : false;
  const rebaseInProgress = gitDirPath
    ? (await exists(path.join(gitDirPath, "rebase-merge"))) || (await exists(path.join(gitDirPath, "rebase-apply")))
    : false;
  const cherryPickInProgress = gitDirPath ? await exists(path.join(gitDirPath, "CHERRY_PICK_HEAD")) : false;
  const name = nameResult.exitCode === 0 && nameResult.stdout.trim() ? nameResult.stdout.trim() : null;
  const email = emailResult.exitCode === 0 && emailResult.stdout.trim() ? emailResult.stdout.trim() : null;

  return {
    isRepository: true,
    rootPath,
    gitDirPath,
    isBare,
    isInitialCommit: headResult.exitCode !== 0,
    isDetachedHead: headResult.exitCode === 0 && branchResult.exitCode !== 0,
    branchName: branchResult.exitCode === 0 && branchResult.stdout.trim() ? branchResult.stdout.trim() : null,
    mergeInProgress,
    rebaseInProgress,
    cherryPickInProgress,
    hasGitIdentity: Boolean(name && email),
    gitIdentity: { name, email }
  };
}

function buildIssues(repository: GitRepositoryState, files: GitFileStatus[], diff: GitDiffMetadata): GitIssue[] {
  const issues: GitIssue[] = [];

  if (!repository.isRepository) {
    issues.push(createIssue("not_git_repository", "Current directory is not inside a git repository.", "blocking"));
    return issues;
  }

  if (repository.isBare) {
    issues.push(createIssue("bare_repository", "Bare git repositories are not supported by this workflow.", "blocking"));
  }

  if (repository.mergeInProgress) {
    issues.push(createIssue("merge_in_progress", "A merge is in progress; resolve it before generating a commit.", "blocking"));
  }

  if (repository.rebaseInProgress) {
    issues.push(createIssue("rebase_in_progress", "A rebase is in progress; resolve it before generating a commit.", "blocking"));
  }

  if (repository.cherryPickInProgress) {
    issues.push(createIssue("cherry_pick_in_progress", "A cherry-pick is in progress; resolve it before generating a commit.", "blocking"));
  }

  if (repository.isDetachedHead) {
    issues.push(createIssue("detached_head", "Repository is on a detached HEAD; committing here may be hard to find later.", "warning"));
  }

  if (!repository.hasGitIdentity) {
    issues.push(createIssue("git_identity_missing", "Git user.name and user.email must be configured before committing.", "blocking"));
  }

  if (files.some((file) => file.unstaged)) {
    issues.push(createIssue("unstaged_changes_present", "Unstaged changes are present and are not included in the staged diff.", "warning"));
  }

  if (files.some((file) => file.untracked)) {
    issues.push(createIssue("untracked_files_present", "Untracked files are present and are not included unless explicitly staged.", "warning"));
  }

  if (diff.truncated) {
    issues.push(createIssue("diff_truncated", "The staged diff exceeded the configured size limit and was truncated.", "warning"));
  }

  return issues;
}

export async function inspectGitRepository(options: InspectGitRepositoryOptions): Promise<GitInspection> {
  const maxDiffBytes = options.maxDiffBytes ?? defaultMaxDiffBytes;
  const gitRunner = options.gitRunner ?? defaultGitRunner;
  const repository = await detectRepository(options.cwd, options.env, gitRunner);

  if (!repository.isRepository || repository.isBare) {
    const emptyDiff = truncateDiff("", maxDiffBytes);
    const files: GitFileStatus[] = [];
    const issues = buildIssues(repository, files, emptyDiff.metadata);
    const warnings = issues.filter((issue) => issue.severity === "warning");
    const blockingIssues = issues.filter((issue) => issue.severity === "blocking");

    return {
      repository,
      files,
      stagedFiles: [],
      unstagedFiles: [],
      untrackedFiles: [],
      stagedDiff: emptyDiff.diff,
      diff: emptyDiff.metadata,
      secretScan: scanTextForSecrets(emptyDiff.diff),
      issues,
      warnings,
      blockingIssues,
      hasStagedChanges: false,
      hasUnstagedChanges: false,
      hasUntrackedFiles: false
    };
  }

  const statusResult = await gitRunner(options.cwd, ["status", "--porcelain=v1", "-z", "--untracked-files=all"], options.env);
  const numstatResult = await gitRunner(options.cwd, ["diff", "--cached", "--numstat", "-z", "--find-renames"], options.env);
  const diffResult = await gitRunner(options.cwd, ["diff", "--cached", "--patch", "--binary", "--find-renames"], options.env);
  const commandIssues = [
    statusResult.exitCode === 0
      ? null
      : createIssue("git_status_failed", formatGitFailure("git status --porcelain", statusResult.stderr, statusResult.stdout), "blocking"),
    numstatResult.exitCode === 0
      ? null
      : createIssue("git_diff_numstat_failed", formatGitFailure("git diff --cached --numstat", numstatResult.stderr, numstatResult.stdout), "blocking"),
    diffResult.exitCode === 0
      ? null
      : createIssue("git_diff_patch_failed", formatGitFailure("git diff --cached --patch", diffResult.stderr, diffResult.stdout), "blocking")
  ].filter((issue): issue is GitIssue => issue !== null);
  const files = applyBinaryMetadata(parsePorcelainStatus(statusResult.stdout), numstatResult.stdout);
  const stagedFiles = files.filter((file) => file.staged);
  const unstagedFiles = files.filter((file) => file.unstaged);
  const untrackedFiles = files.filter((file) => file.untracked);
  const truncatedDiff = truncateDiff(diffResult.stdout, maxDiffBytes);
  const secretScan = scanTextForSecrets(diffResult.stdout);
  const issues = [...buildIssues(repository, files, truncatedDiff.metadata), ...commandIssues];

  if (secretScan.findings.length > 0) {
    issues.push(createIssue("secret_scan_match", "Potential secrets were found in the staged diff.", "warning"));
  }

  const warnings = issues.filter((issue) => issue.severity === "warning");
  const blockingIssues = issues.filter((issue) => issue.severity === "blocking");

  return {
    repository,
    files,
    stagedFiles,
    unstagedFiles,
    untrackedFiles,
    stagedDiff: truncatedDiff.diff,
    diff: truncatedDiff.metadata,
    secretScan,
    issues,
    warnings,
    blockingIssues,
    hasStagedChanges: stagedFiles.length > 0,
    hasUnstagedChanges: unstagedFiles.length > 0,
    hasUntrackedFiles: untrackedFiles.length > 0
  };
}

export async function stageAllChanges(options: StageAllChangesOptions): Promise<StageAllChangesResult> {
  if (!options.confirmed) {
    return {
      staged: false,
      reason: "not_confirmed",
      inspection: await inspectGitRepository(options)
    };
  }

  if (!options.isTty) {
    return {
      staged: false,
      reason: "non_tty",
      inspection: await inspectGitRepository(options)
    };
  }

  const gitRunner = options.gitRunner ?? defaultGitRunner;
  const addResult = await gitRunner(options.cwd, ["add", "-A"], options.env);

  if (addResult.exitCode !== 0) {
    const inspection = await inspectGitRepository(options);

    return {
      staged: false,
      reason: "git_add_failed",
      inspection: addBlockingIssue(
        inspection,
        createIssue("git_add_failed", formatGitFailure("git add -A", addResult.stderr, addResult.stdout), "blocking")
      )
    };
  }

  return {
    staged: true,
    reason: "confirmed",
    inspection: await inspectGitRepository(options)
  };
}

export async function getRecentCommits(options: {
  cwd: string;
  env?: NodeJS.ProcessEnv;
  limit?: number;
  gitRunner?: GitCommandRunner;
}): Promise<string[]> {
  const gitRunner = options.gitRunner ?? defaultGitRunner;
  const limit = options.limit ?? 10;

  const result = await gitRunner(
    options.cwd,
    ["log", `--max-count=${limit}`, "--pretty=format:%s%n%b", "--no-merges"],
    options.env
  );

  if (result.exitCode !== 0) {
    return [];
  }

  const commits = result.stdout
    .split("\n\n")
    .map((commit) => commit.trim())
    .filter((commit) => commit.length > 0);

  return commits.slice(0, limit);
}
