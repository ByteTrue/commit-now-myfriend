import { chmod, mkdtemp, writeFile } from "node:fs/promises";
import { tmpdir } from "node:os";
import { join } from "node:path";

import { describe, expect, it } from "vitest";

import { writeUserConfigPatch } from "../../src/config/index.js";
import { runDoctor } from "../../src/doctor/index.js";
import { createTempGitRepo } from "../helpers/temp-git-repo.js";

async function createTempWorkspace(): Promise<{ cwd: string; env: NodeJS.ProcessEnv; home: string }> {
  const cwd = await mkdtemp(join(tmpdir(), "cnm-doctor-workspace-"));
  const home = join(cwd, ".cnm-home");

  return {
    cwd,
    env: {
      ...process.env,
      CNM_HOME: home
    },
    home
  };
}

describe("doctor service", () => {
  it("reports missing provider setup outside git repos with parseable JSON data", async () => {
    const runtime = await createTempWorkspace();
    const report = await runDoctor({
      cwd: runtime.cwd,
      env: runtime.env,
      nodeEngine: ">=20",
      nodeVersion: "v20.12.2"
    });
    const roundTripped = JSON.parse(JSON.stringify(report)) as typeof report;
    const issueCodes = roundTripped.issues.map((issue) => issue.code);

    expect(roundTripped.command).toBe("cnm doctor");
    expect(issueCodes).toEqual(
      expect.arrayContaining(["not_git_repository", "provider_config_missing", "api_key_missing"])
    );
    expect(roundTripped.checks.repository.isRepository).toBe(false);
    expect(roundTripped.checks.effectiveConfig.config.apiKey).toBeNull();
    expect(JSON.stringify(roundTripped)).not.toContain("sk_");
  }, 20000);

  it("redacts user and project api keys in the report", async () => {
    const repo = await createTempGitRepo({ initialCommit: true });
    const home = join(repo.path, ".cnm-home");
    const env = {
      ...process.env,
      CNM_HOME: home
    };

    await writeUserConfigPatch(
      {
        apiKey: "sk_live_user_secret_1234567890",
        model: "gpt-5-mini",
        provider: "openai-responses"
      },
      { cwd: repo.path, env }
    );
    await writeFile(
      join(repo.path, ".cnmrc.json"),
      JSON.stringify(
        {
          apiKey: "sk_project_secret_1234567890",
          model: "project-model"
        },
        null,
        2
      ),
      "utf8"
    );

    const report = await runDoctor({ cwd: repo.path, env, nodeEngine: ">=20", nodeVersion: "v20.12.2" });
    const serialized = JSON.stringify(report);

    expect(report.checks.userConfig.config?.apiKey).toBe("[redacted]");
    expect(report.checks.effectiveConfig.config.apiKey).toBe("[redacted]");
    expect(report.issues.map((issue) => issue.code)).toContain("project_api_key_ignored");
    expect(serialized).not.toContain("sk_live_user_secret_1234567890");
    expect(serialized).not.toContain("sk_project_secret_1234567890");
    await repo.cleanup();
  }, 20000);

  it("warns when user config permissions are broader than 0600", async () => {
    const runtime = await createTempWorkspace();
    const writeResult = await writeUserConfigPatch(
      {
        apiKey: "sk_permissions_secret_1234567890",
        provider: "openai-responses"
      },
      { cwd: runtime.cwd, env: runtime.env }
    );

    if (process.platform !== "win32") {
      await chmod(writeResult.path, 0o644);
    }

    const report = await runDoctor({ cwd: runtime.cwd, env: runtime.env, nodeEngine: ">=20", nodeVersion: "v20.12.2" });

    if (process.platform === "win32") {
      expect(report.issues.map((issue) => issue.code)).not.toContain("user_config_permissions_insecure");
      return;
    }

    expect(report.issues.map((issue) => issue.code)).toContain("user_config_permissions_insecure");
  }, 20000);

  it("surfaces missing git identity from repository inspection", async () => {
    const repo = await createTempGitRepo();
    const isolatedHome = join(repo.path, "isolated-home");

    await repo.git(["config", "--unset", "user.name"]);
    await repo.git(["config", "--unset", "user.email"]);
    await repo.write("file.txt", "content\n");
    await repo.git(["add", "file.txt"]);

    const report = await runDoctor({
      cwd: repo.path,
      env: {
        CNM_HOME: join(isolatedHome, ".cnm-home"),
        HOME: isolatedHome,
        XDG_CONFIG_HOME: join(isolatedHome, ".config")
      },
      nodeEngine: ">=20",
      nodeVersion: "v20.12.2"
    });

    expect(report.issues.map((issue) => issue.code)).toContain("git_identity_missing");
    expect(report.paths.userConfigHome).toBe(join(isolatedHome, ".cnm-home"));
    expect(report.checks.effectiveConfig.sources.apiKey).toBe("missing");
    await repo.cleanup();
  }, 20000);
});
