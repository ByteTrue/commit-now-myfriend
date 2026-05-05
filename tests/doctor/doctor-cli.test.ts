import { mkdtemp } from "node:fs/promises";
import { tmpdir } from "node:os";
import { join } from "node:path";

import { afterEach, describe, expect, it, vi } from "vitest";

import { runCli } from "../../src/cli.js";
import { EXIT_CODES, type CliWriteStream } from "../../src/output/index.js";

function createMemoryStream() {
  let buffer = "";

  const stream: CliWriteStream = {
    write(chunk) {
      buffer += typeof chunk === "string" ? chunk : Buffer.from(chunk).toString("utf8");
      return true;
    }
  };

  return {
    read() {
      return buffer;
    },
    stream
  };
}

function createCliIo() {
  return {
    stderr: createMemoryStream(),
    stdout: createMemoryStream()
  };
}

const originalCwd = process.cwd();
const originalCnmHome = process.env.CNM_HOME;

afterEach(() => {
  process.chdir(originalCwd);

  if (originalCnmHome === undefined) {
    delete process.env.CNM_HOME;
  } else {
    process.env.CNM_HOME = originalCnmHome;
  }

  vi.resetModules();
  vi.clearAllMocks();
});

async function setupCliRuntime() {
  const cwd = await mkdtemp(join(tmpdir(), "cnm-doctor-cli-"));
  const home = join(cwd, ".cnm-home");

  process.chdir(cwd);
  process.env.CNM_HOME = home;

  return { cwd, home };
}

describe("doctor CLI", () => {
  it("writes parseable json to stdout even when doctor finds issues", async () => {
    await setupCliRuntime();
    const io = createCliIo();

    const exitCode = await runCli({
      argv: ["doctor", "--json"],
      nodeEngine: ">=20",
      stderr: io.stderr.stream,
      stdout: io.stdout.stream,
      version: "0.1.0"
    });

    const payload = JSON.parse(io.stdout.read()) as {
      issues: Array<{ code: string }>;
      summary: { errors: number; warnings: number };
    };

    expect(exitCode).toBe(EXIT_CODES.ERROR);
    expect(io.stderr.read()).toBe("");
    expect(payload.summary.errors).toBeGreaterThan(0);
    expect(payload.issues.map((issue) => issue.code)).toEqual(
      expect.arrayContaining(["provider_config_missing", "api_key_missing"])
    );
  });

  it("does not invoke provider creation when running doctor by default", async () => {
    await setupCliRuntime();
    const io = createCliIo();
    const createCommitMessageProvider = vi.fn(() => {
      throw new Error("provider should not be created during doctor");
    });

    vi.doMock("../../src/providers/index.js", async () => {
      const actual = await vi.importActual<typeof import("../../src/providers/index.js")>(
        "../../src/providers/index.js"
      );

      return {
        ...actual,
        createCommitMessageProvider
      };
    });

    const { runCli: isolatedRunCli } = await import("../../src/cli.js");
    const exitCode = await isolatedRunCli({
      argv: ["doctor", "--json"],
      nodeEngine: ">=20",
      stderr: io.stderr.stream,
      stdout: io.stdout.stream,
      version: "0.1.0"
    });

    expect(exitCode).toBe(EXIT_CODES.ERROR);
    expect(createCommitMessageProvider).not.toHaveBeenCalled();
  });
});
