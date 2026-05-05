import { mkdir, mkdtemp, rm, writeFile } from "node:fs/promises";
import { tmpdir } from "node:os";
import path from "node:path";
import { setTimeout as delay } from "node:timers/promises";

import { execa } from "execa";

export interface TempGitRepo {
  path: string;
  git(args: string[]): Promise<string>;
  write(relativePath: string, content: string | Uint8Array): Promise<void>;
  cleanup(): Promise<void>;
}

const maxGitAttempts = 3;
const retryDelayMs = 25;

interface GitFailureDetails {
  exitCode?: number;
  signal?: string;
  stderr: string;
  stdout: string;
}

function hasOutput(details: GitFailureDetails): boolean {
  return details.stderr.trim().length > 0 || details.stdout.trim().length > 0;
}

function hasResourcePressureSymptom(details: GitFailureDetails): boolean {
  const output = `${details.stderr}\n${details.stdout}`.toLowerCase();

  return output.includes("eagain") || output.includes("resource temporarily unavailable");
}

function isTransientGitFailure(details: GitFailureDetails): boolean {
  if (hasResourcePressureSymptom(details)) {
    return true;
  }

  return !hasOutput(details) && (details.exitCode === undefined || details.exitCode !== 0 || details.signal !== undefined);
}

function formatGitFailure(command: string, details: GitFailureDetails): string {
  return [
    `${command} failed.`,
    `exitCode: ${details.exitCode ?? "unknown"}`,
    `signal: ${details.signal ?? "none"}`,
    `stderr: ${details.stderr.trim() || "<empty>"}`,
    `stdout: ${details.stdout.trim() || "<empty>"}`
  ].join("\n");
}

function isRecord(value: unknown): value is Record<string, unknown> {
  return typeof value === "object" && value !== null;
}

function readStringProperty(value: unknown, key: string): string {
  if (!isRecord(value)) {
    return "";
  }

  const property = value[key];
  return typeof property === "string" ? property : "";
}

function readNumberProperty(value: unknown, key: string): number | undefined {
  if (!isRecord(value)) {
    return undefined;
  }

  const property = value[key];
  return typeof property === "number" ? property : undefined;
}

function outputToString(value: unknown): string {
  if (typeof value === "string") {
    return value;
  }

  if (value instanceof Uint8Array) {
    return Buffer.from(value).toString("utf8");
  }

  if (Array.isArray(value)) {
    return value.map((entry) => outputToString(entry)).join("");
  }

  return "";
}

export async function createTempGitRepo(options: { identity?: boolean; initialCommit?: boolean } = {}): Promise<TempGitRepo> {
  const repoPath = await mkdtemp(path.join(tmpdir(), "cnm-git-"));

  async function git(args: string[]): Promise<string> {
    const command = `git ${args.join(" ")}`;

    for (let attempt = 1; attempt <= maxGitAttempts; attempt += 1) {
      let result: Awaited<ReturnType<typeof execa>>;

      try {
        result = await execa("git", args, {
          cwd: repoPath,
          reject: false,
          stdin: "ignore",
          stdout: "pipe",
          stderr: "pipe"
        });
      } catch (error) {
        const details: GitFailureDetails = {
          exitCode: readNumberProperty(error, "exitCode"),
          signal: readStringProperty(error, "signal") || undefined,
          stderr: readStringProperty(error, "stderr"),
          stdout: readStringProperty(error, "stdout")
        };

        if (attempt < maxGitAttempts && isTransientGitFailure(details)) {
          await delay(retryDelayMs * attempt);
          continue;
        }

        if (error instanceof Error && hasOutput(details)) {
          throw error;
        }

        throw new Error(formatGitFailure(command, details));
      }

      if (!result.failed) {
        return outputToString(result.stdout);
      }

      const details: GitFailureDetails = {
        exitCode: result.exitCode ?? undefined,
        signal: result.signal ?? undefined,
        stderr: outputToString(result.stderr),
        stdout: outputToString(result.stdout)
      };

      if (attempt < maxGitAttempts && isTransientGitFailure(details)) {
        await delay(retryDelayMs * attempt);
        continue;
      }

      throw new Error(formatGitFailure(command, details));
    }

    throw new Error(`${command} failed after ${maxGitAttempts} attempts.`);
  }

  async function write(relativePath: string, content: string | Uint8Array): Promise<void> {
    const targetPath = path.join(repoPath, relativePath);
    await mkdir(path.dirname(targetPath), { recursive: true });
    await writeFile(targetPath, content);
  }

  await git(["init"]);

  if (options.identity !== false) {
    await git(["config", "user.name", "CNM Test"]);
    await git(["config", "user.email", "cnm@example.test"]);
  }

  const repo: TempGitRepo = {
    path: repoPath,
    git,
    write,
    cleanup: () => rm(repoPath, { recursive: true, force: true })
  };

  if (options.initialCommit) {
    await repo.write("README.md", "initial\n");
    await repo.git(["add", "README.md"]);
    await repo.git(["commit", "-m", "chore: initial"]);
  }

  return repo;
}
