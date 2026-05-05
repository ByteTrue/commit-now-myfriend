import { mkdir, mkdtemp, readFile, stat, writeFile } from "node:fs/promises";
import { tmpdir } from "node:os";
import { dirname, join } from "node:path";

import { describe, expect, it } from "vitest";

import {
  getConfigPaths,
  resolveEffectiveConfig,
  toJsonConfigView,
  writeUserConfigPatch
} from "../../src/config/index.js";

async function createTempWorkspace(): Promise<{ cwd: string; env: NodeJS.ProcessEnv; home: string }> {
  const cwd = await mkdtemp(join(tmpdir(), "cnm-config-workspace-"));
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

describe("config service", () => {
  it("writes user config under CNM_HOME only", async () => {
    const runtime = await createTempWorkspace();
    const result = await writeUserConfigPatch(
      { provider: "openai-responses" },
      { cwd: runtime.cwd, env: runtime.env },
    );
    const fileContent = JSON.parse(await readFile(result.path, "utf8")) as { provider: string };

    expect(result.path).toBe(join(runtime.home, "config.json"));
    expect(fileContent).toEqual({ provider: "openai-responses" });
    expect(result.path.startsWith(runtime.cwd)).toBe(true);
  });

  it("resolves precedence as flags over env over project over user over defaults", async () => {
    const runtime = await createTempWorkspace();

    await writeUserConfigPatch(
      {
        apiKey: "test-user-api-key",
        baseURL: "https://user.example",
        customPrompt: "user prompt",
        model: "user-model",
        provider: "openai-compatible"
      },
      { cwd: runtime.cwd, env: runtime.env },
    );

    await writeFile(
      join(runtime.cwd, ".cnmrc.json"),
      JSON.stringify(
        {
          baseURL: "https://project.example",
          model: "project-model",
          provider: "google-gemini"
        },
        null,
        2,
      ),
      "utf8",
    );

    const resolvedConfig = await resolveEffectiveConfig({
      cwd: runtime.cwd,
      env: {
        ...runtime.env,
        CNM_API_KEY: "test-env-api-key",
        CNM_MODEL: "env-model",
        CNM_PROVIDER: "openai-responses"
      },
      flagOverrides: {
        customPrompt: "flag prompt",
        provider: "anthropic-messages"
      }
    });

    expect(resolvedConfig.values.provider).toBe("anthropic-messages");
    expect(resolvedConfig.values.model).toBe("env-model");
    expect(resolvedConfig.values.baseURL).toBe("https://project.example");
    expect(resolvedConfig.values.customPrompt).toBe("flag prompt");
    expect(resolvedConfig.values.apiKey).toBe("test-env-api-key");
    expect(resolvedConfig.values.promptStyle).toBe("auto");
  });

  it("warns and ignores project api keys while keeping non-secret project values", async () => {
    const runtime = await createTempWorkspace();

    await writeFile(
      join(runtime.cwd, ".cnmrc.json"),
      JSON.stringify(
        {
          apiKey: "test-project-api-key",
          model: "project-model"
        },
        null,
        2,
      ),
      "utf8",
    );

    const resolvedConfig = await resolveEffectiveConfig({ cwd: runtime.cwd, env: runtime.env });
    const jsonView = toJsonConfigView(resolvedConfig.values);

    expect(resolvedConfig.warnings).toHaveLength(1);
    expect(resolvedConfig.warnings[0]).toContain("project-level secrets are ignored");
    expect(resolvedConfig.values.apiKey).toBeUndefined();
    expect(resolvedConfig.values.model).toBe("project-model");
    expect(JSON.stringify(jsonView)).not.toContain("test-project-api-key");
  });

  it("fails clearly for corrupted user config", async () => {
    const runtime = await createTempWorkspace();
    const { userConfigPath } = getConfigPaths({ cwd: runtime.cwd, env: runtime.env });

    await mkdir(dirname(userConfigPath), { recursive: true });
    await writeFile(userConfigPath, "{not-json", "utf8");

    await expect(resolveEffectiveConfig({ cwd: runtime.cwd, env: runtime.env })).rejects.toThrow(
      /not valid JSON/,
    );
  });

  it("applies 0600 permissions on supported platforms", async () => {
    const runtime = await createTempWorkspace();
    const result = await writeUserConfigPatch(
      { provider: "openai-responses" },
      { cwd: runtime.cwd, env: runtime.env },
    );

    if (process.platform === "win32") {
      expect(result.warnings).toEqual([]);
      return;
    }

    const fileStat = await stat(result.path);
    expect(fileStat.mode & 0o777).toBe(0o600);
  });
});
