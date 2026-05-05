import { mkdir, mkdtemp, realpath, rm, writeFile } from "node:fs/promises";
import { tmpdir } from "node:os";
import path from "node:path";

import { execa } from "execa";
import { describe, expect, it } from "vitest";

import { inspectGitRepository, stageAllChanges, type GitCommandRunner } from "../../src/git/index.js";
import { createTempGitRepo } from "../helpers/temp-git-repo.js";

function createFailingGitRunner(shouldFail: (args: string[]) => boolean): GitCommandRunner {
  return async (cwd, args, env) => {
    if (shouldFail(args)) {
      return {
        exitCode: 128,
        stderr: `simulated failure for git ${args.join(" ")}`,
        stdout: ""
      };
    }

    const result = await execa("git", args, {
      cwd,
      env,
      reject: false,
      stdin: "ignore",
      stdout: "pipe",
      stderr: "pipe"
    });

    return {
      exitCode: result.exitCode ?? 1,
      stderr: result.stderr,
      stdout: result.stdout
    };
  };
}

describe("inspectGitRepository", () => {
  it("detects a non-git directory without crashing", async () => {
    const outsidePath = await mkdtemp(path.join(tmpdir(), "cnm-non-git-"));

    const inspection = await inspectGitRepository({ cwd: outsidePath });

    expect(inspection.repository.isRepository).toBe(false);
    expect(inspection.blockingIssues.map((issue) => issue.code)).toContain("not_git_repository");
    await rm(outsidePath, { recursive: true, force: true });
  });


  it("blocks bare repositories", async () => {
    const barePath = await mkdtemp(path.join(tmpdir(), "cnm-bare-git-"));
    await execa("git", ["init", "--bare"], { cwd: barePath });

    const inspection = await inspectGitRepository({ cwd: barePath });

    expect(inspection.repository.isRepository).toBe(true);
    expect(inspection.repository.isBare).toBe(true);
    expect(inspection.blockingIssues.map((issue) => issue.code)).toContain("bare_repository");
    await rm(barePath, { recursive: true, force: true });
  });

  it("detects the repo root from a subdirectory", async () => {
    const repo = await createTempGitRepo();
    const subdirectory = path.join(repo.path, "nested", "child");
    await mkdir(subdirectory, { recursive: true });
    await repo.write("nested/child/file.txt", "hello\n");
    await repo.git(["add", "nested/child/file.txt"]);

    const inspection = await inspectGitRepository({ cwd: subdirectory });

    expect(inspection.repository.isRepository).toBe(true);
    expect(await realpath(inspection.repository.rootPath ?? "")).toBe(await realpath(repo.path));
    expect(inspection.stagedFiles.map((file) => file.path)).toEqual(["nested/child/file.txt"]);
    await repo.cleanup();
  });

  it("supports initial commits without HEAD", async () => {
    const repo = await createTempGitRepo();
    await repo.write("first.txt", "first\n");
    await repo.git(["add", "first.txt"]);

    const inspection = await inspectGitRepository({ cwd: repo.path });

    expect(inspection.repository.isInitialCommit).toBe(true);
    expect(inspection.hasStagedChanges).toBe(true);
    expect(inspection.stagedDiff).toContain("first.txt");
    await repo.cleanup();
  });

  it("keeps staged diff scoped to staged files when unstaged files exist", async () => {
    const repo = await createTempGitRepo({ initialCommit: true });
    await repo.write("staged.txt", "staged secret-free content\n");
    await repo.git(["add", "staged.txt"]);
    await repo.write("README.md", "unstaged content must stay out\n");

    const inspection = await inspectGitRepository({ cwd: repo.path });

    expect(inspection.hasStagedChanges).toBe(true);
    expect(inspection.hasUnstagedChanges).toBe(true);
    expect(inspection.stagedFiles.map((file) => file.path)).toEqual(["staged.txt"]);
    expect(inspection.stagedDiff).toContain("staged.txt");
    expect(inspection.stagedDiff).not.toContain("README.md");
    expect(inspection.stagedDiff).not.toContain("unstaged content must stay out");
    expect(inspection.warnings.map((issue) => issue.code)).toContain("unstaged_changes_present");
    await repo.cleanup();
  });

  it("distinguishes staged, unstaged, and untracked files", async () => {
    const repo = await createTempGitRepo({ initialCommit: true });
    await repo.write("staged.txt", "staged\n");
    await repo.git(["add", "staged.txt"]);
    await repo.write("README.md", "initial changed\n");
    await repo.write("untracked.txt", "new\n");

    const inspection = await inspectGitRepository({ cwd: repo.path });

    expect(inspection.stagedFiles.map((file) => file.path)).toEqual(["staged.txt"]);
    expect(inspection.unstagedFiles.map((file) => file.path)).toEqual(["README.md"]);
    expect(inspection.untrackedFiles.map((file) => file.path)).toEqual(["untracked.txt"]);
    expect(inspection.warnings.map((issue) => issue.code)).toEqual(
      expect.arrayContaining(["unstaged_changes_present", "untracked_files_present"])
    );
    await repo.cleanup();
  });

  it("reports untracked-only worktrees without fabricating staged diffs", async () => {
    const repo = await createTempGitRepo({ initialCommit: true });
    await repo.write("untracked-only.txt", "new file only\n");

    const inspection = await inspectGitRepository({ cwd: repo.path });

    expect(inspection.hasStagedChanges).toBe(false);
    expect(inspection.hasUntrackedFiles).toBe(true);
    expect(inspection.stagedFiles).toEqual([]);
    expect(inspection.untrackedFiles.map((file) => file.path)).toEqual(["untracked-only.txt"]);
    expect(inspection.stagedDiff).toBe("");
    expect(inspection.warnings.map((issue) => issue.code)).toContain("untracked_files_present");
    await repo.cleanup();
  });

  it("preserves staged file names containing spaces and unicode", async () => {
    const repo = await createTempGitRepo({ initialCommit: true });
    const filename = "dir with spaces/unicode-文件 name.txt";
    await repo.write(filename, "unicode path content\n");
    await repo.git(["add", filename]);

    const inspection = await inspectGitRepository({ cwd: repo.path });

    expect(inspection.hasStagedChanges).toBe(true);
    expect(inspection.stagedFiles.map((file) => file.path)).toEqual([filename]);
    expect(inspection.stagedDiff).toContain("unicode path content");
    await repo.cleanup();
  });

  it("detects binary, renamed, and deleted staged files", async () => {
    const repo = await createTempGitRepo({ initialCommit: true });
    await repo.write("old-name.txt", "rename me\n");
    await repo.write("remove-me.txt", "delete me\n");
    await repo.git(["add", "old-name.txt", "remove-me.txt"]);
    await repo.git(["commit", "-m", "test: add metadata fixtures"]);
    await repo.git(["mv", "old-name.txt", "new-name.txt"]);
    await repo.git(["rm", "remove-me.txt"]);
    await repo.write("image.bin", new Uint8Array([0, 159, 146, 150, 0, 1, 2, 3]));
    await repo.git(["add", "image.bin"]);

    const inspection = await inspectGitRepository({ cwd: repo.path });
    const renamed = inspection.stagedFiles.find((file) => file.path === "new-name.txt");
    const deleted = inspection.stagedFiles.find((file) => file.path === "remove-me.txt");
    const binary = inspection.stagedFiles.find((file) => file.path === "image.bin");

    expect(renamed).toMatchObject({ staged: "renamed", originalPath: "old-name.txt" });
    expect(deleted).toMatchObject({ staged: "deleted" });
    expect(binary).toMatchObject({ binary: true });
    await repo.cleanup();
  });

  it("reports merge, rebase, and cherry-pick marker states as blocking", async () => {
    const repo = await createTempGitRepo({ initialCommit: true });
    const gitDir = path.join(repo.path, ".git");
    await writeFile(path.join(gitDir, "MERGE_HEAD"), "marker\n");
    await mkdir(path.join(gitDir, "rebase-merge"));
    await writeFile(path.join(gitDir, "CHERRY_PICK_HEAD"), "marker\n");

    const inspection = await inspectGitRepository({ cwd: repo.path });

    expect(inspection.blockingIssues.map((issue) => issue.code)).toEqual(
      expect.arrayContaining(["merge_in_progress", "rebase_in_progress", "cherry_pick_in_progress"])
    );
    await repo.cleanup();
  });

  it("warns on detached HEAD", async () => {
    const repo = await createTempGitRepo({ initialCommit: true });
    const head = await repo.git(["rev-parse", "HEAD"]);
    await repo.git(["checkout", "--detach", head.trim()]);

    const inspection = await inspectGitRepository({ cwd: repo.path });

    expect(inspection.repository.isDetachedHead).toBe(true);
    expect(inspection.warnings.map((issue) => issue.code)).toContain("detached_head");
    await repo.cleanup();
  });

  it("detects missing git identity before commit workflow", async () => {
    const repo = await createTempGitRepo({ identity: false });
    await repo.write("file.txt", "content\n");
    await repo.git(["add", "file.txt"]);

    const isolatedHome = path.join(repo.path, "isolated-home");
    await mkdir(isolatedHome);
    const inspection = await inspectGitRepository({
      cwd: repo.path,
      env: { HOME: isolatedHome, XDG_CONFIG_HOME: path.join(isolatedHome, ".config") }
    });

    expect(inspection.repository.hasGitIdentity).toBe(false);
    expect(inspection.blockingIssues.map((issue) => issue.code)).toContain("git_identity_missing");
    await repo.cleanup();
  });

  it("exposes truncation metadata and scans staged diffs for secrets", async () => {
    const repo = await createTempGitRepo({ initialCommit: true });
    await repo.write("secret.txt", "api_key = 'sk_1234567890abcdef1234567890abcdef'\nlarge=content\n");
    await repo.git(["add", "secret.txt"]);

    const inspection = await inspectGitRepository({ cwd: repo.path, maxDiffBytes: 40 });

    expect(inspection.diff.truncated).toBe(true);
    expect(inspection.diff.omittedBytes).toBeGreaterThan(0);
    expect(inspection.secretScan.findings.length).toBeGreaterThan(0);
    expect(inspection.secretScan.hasBlockingFindings).toBe(false);
    expect(inspection.warnings.map((issue) => issue.code)).toContain("secret_scan_match");
    expect(inspection.blockingIssues.map((issue) => issue.code)).not.toContain("secret_scan_match");
    await repo.cleanup();
  });

  it.each([
    {
      code: "git_status_failed",
      command: "status",
      shouldFail: (args: string[]) => args[0] === "status"
    },
    {
      code: "git_diff_numstat_failed",
      command: "diff --cached --numstat",
      shouldFail: (args: string[]) => args[0] === "diff" && args.includes("--numstat")
    },
    {
      code: "git_diff_patch_failed",
      command: "diff --cached --patch",
      shouldFail: (args: string[]) => args[0] === "diff" && args.includes("--patch")
    }
  ])("blocks when git $command fails during inspection", async ({ code, shouldFail }) => {
    const repo = await createTempGitRepo({ initialCommit: true });
    await repo.write("staged.txt", "content\n");
    await repo.git(["add", "staged.txt"]);

    const inspection = await inspectGitRepository({
      cwd: repo.path,
      gitRunner: createFailingGitRunner(shouldFail)
    });

    expect(inspection.blockingIssues.map((issue) => issue.code)).toContain(code);
    expect(inspection.blockingIssues.find((issue) => issue.code === code)?.message).toContain("simulated failure");
    await repo.cleanup();
  });
});

describe("stageAllChanges", () => {
  it("does not stage without explicit confirmed TTY intent", async () => {
    const repo = await createTempGitRepo({ initialCommit: true });
    await repo.write("new-file.txt", "new\n");

    const notConfirmed = await stageAllChanges({ cwd: repo.path, confirmed: false, isTty: true });
    const nonTty = await stageAllChanges({ cwd: repo.path, confirmed: true, isTty: false });

    expect(notConfirmed).toMatchObject({ staged: false, reason: "not_confirmed" });
    expect(nonTty).toMatchObject({ staged: false, reason: "non_tty" });
    expect(nonTty.inspection.untrackedFiles.map((file) => file.path)).toEqual(["new-file.txt"]);
    await repo.cleanup();
  });

  it("stages all changes only after explicit confirmed TTY intent", async () => {
    const repo = await createTempGitRepo({ initialCommit: true });
    await repo.write("new-file.txt", "new\n");
    await repo.write("README.md", "changed\n");

    const result = await stageAllChanges({ cwd: repo.path, confirmed: true, isTty: true });

    expect(result).toMatchObject({ staged: true, reason: "confirmed" });
    expect(result.inspection.stagedFiles.map((file) => file.path).sort()).toEqual(["README.md", "new-file.txt"]);
    expect(result.inspection.untrackedFiles).toEqual([]);
    await repo.cleanup();
  });

  it("returns a failed result when git add -A fails", async () => {
    const repo = await createTempGitRepo({ initialCommit: true });
    await repo.write("new-file.txt", "new\n");

    const result = await stageAllChanges({
      cwd: repo.path,
      confirmed: true,
      gitRunner: createFailingGitRunner((args) => args[0] === "add" && args[1] === "-A"),
      isTty: true
    });

    expect(result).toMatchObject({ staged: false, reason: "git_add_failed" });
    expect(result.inspection.blockingIssues.map((issue) => issue.code)).toContain("git_add_failed");
    expect(await repo.git(["diff", "--cached", "--name-only"])).toBe("");
    await repo.cleanup();
  });
});
