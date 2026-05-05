import { mkdtemp } from "node:fs/promises";
import { tmpdir } from "node:os";
import { join } from "node:path";

import { execa } from "execa";
import { beforeAll, describe, expect, it } from "vitest";

const projectRoot = new URL("../..", import.meta.url).pathname;
const distEntry = join(projectRoot, "dist", "index.js");

describe("built dist CLI subprocess", { timeout: 30000 }, () => {
  beforeAll(async () => {
    await execa("pnpm", ["build"], {
      cwd: projectRoot,
      env: {
        ...process.env,
        CI: "true"
      },
      stderr: "pipe",
      stdout: "pipe"
    });
  });

  it("serves help from dist/index.js without touching user config", async () => {
    const home = await mkdtemp(join(tmpdir(), "cnm-dist-home-"));
    const result = await execa("node", [distEntry, "--help"], {
      cwd: projectRoot,
      env: {
        ...process.env,
        CNM_HOME: home
      },
      reject: false,
      stderr: "pipe",
      stdout: "pipe"
    });

    expect(result.exitCode).toBe(0);
    expect(result.stderr).toBe("");
    expect(result.stdout).toContain("Usage: cnm");
    expect(result.stdout).toContain("doctor");
    expect(result.stdout).toContain("--dry-run");
    expect(result.stdout).not.toContain("--yes");
  });

  it("emits parseable doctor JSON from dist/index.js in a temp CNM_HOME", async () => {
    const cwd = await mkdtemp(join(tmpdir(), "cnm-dist-cwd-"));
    const home = join(cwd, ".cnm-home");
    const result = await execa("node", [distEntry, "doctor", "--json"], {
      cwd,
      env: {
        ...process.env,
        CNM_HOME: home
      },
      reject: false,
      stderr: "pipe",
      stdout: "pipe"
    });
    const payload = JSON.parse(result.stdout) as {
      issues: Array<{ code: string }>;
      command: string;
      status: string;
    };

    expect(result.exitCode).toBe(1);
    expect(result.stderr).toBe("");
    expect(payload.command).toBe("cnm doctor");
    expect(payload.issues.map((issue) => issue.code)).toEqual(
      expect.arrayContaining(["not_git_repository", "provider_config_missing", "api_key_missing"])
    );
  });
});
