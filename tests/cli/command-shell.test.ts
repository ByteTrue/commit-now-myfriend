import { mkdtemp } from "node:fs/promises";
import { tmpdir } from "node:os";
import { join } from "node:path";

import { afterEach, describe, expect, it } from "vitest";

import { buildCli, runCli } from "../../src/cli.js";
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
    stream,
    read: () => buffer
  };
}

function createCliIo() {
  const stdout = createMemoryStream();
  const stderr = createMemoryStream();

  return { stdout, stderr };
}

const originalCwd = process.cwd();
const originalCnmHome = process.env.CNM_HOME;

afterEach(() => {
  process.chdir(originalCwd);

  if (originalCnmHome === undefined) {
    delete process.env.CNM_HOME;
    return;
  }

  process.env.CNM_HOME = originalCnmHome;
});

async function setupCliRuntime() {
  const cwd = await mkdtemp(join(tmpdir(), "cnm-command-shell-"));

  process.chdir(cwd);
  process.env.CNM_HOME = join(cwd, ".cnm-home");
}

describe("buildCli", () => {
  it("renders discoverable help without a yes flag", () => {
    const help = buildCli({ version: "0.1.0" }).helpInformation();

    expect(help).toContain("Usage: cnm");
    expect(help).toContain("init");
    expect(help).toContain("config");
    expect(help).toContain("doctor");
    expect(help).toContain("--dry-run");
    expect(help).toContain("--json");
    expect(help).toContain("--provider");
    expect(help).toContain("--model");
    expect(help).toContain("--base-url");
    expect(help).toContain("--prompt-style");
    expect(help).toContain("--custom-prompt");
    expect(help).not.toContain("--api-key");
    expect(help).not.toContain("--yes");
  });
});

describe("runCli", () => {
  it("prints the version to stdout", async () => {
    const io = createCliIo();

    const exitCode = await runCli({
      argv: ["--version"],
      version: "0.1.0",
      stdout: io.stdout.stream,
      stderr: io.stderr.stream
    });

    expect(exitCode).toBe(EXIT_CODES.SUCCESS);
    expect(io.stdout.read()).toBe("0.1.0\n");
    expect(io.stderr.read()).toBe("");
  });

  it("fails an unknown command on stderr", async () => {
    const io = createCliIo();

    const exitCode = await runCli({
      argv: ["nope"],
      version: "0.1.0",
      stdout: io.stdout.stream,
      stderr: io.stderr.stream
    });

    expect(exitCode).toBe(EXIT_CODES.ERROR);
    expect(io.stdout.read()).toBe("");
    expect(io.stderr.read()).toContain("error: unknown command 'nope'");
    expect(io.stderr.read()).toContain("Run cnm --help to inspect available commands.");
  });

  it("routes human init output to stdout", async () => {
    await setupCliRuntime();
    const io = createCliIo();

    const exitCode = await runCli({
      argv: ["init"],
      version: "0.1.0",
      stdout: io.stdout.stream,
      stderr: io.stderr.stream
    });

    expect(exitCode).toBe(EXIT_CODES.SUCCESS);
    expect(io.stderr.read()).toBe("");
    expect(io.stdout.read()).toContain("Initialized user config at");
  });

  it("routes json init output to stdout only, even after the subcommand", async () => {
    await setupCliRuntime();
    const io = createCliIo();

    const exitCode = await runCli({
      argv: ["init", "--json", "--dry-run"],
      version: "0.1.0",
      stdout: io.stdout.stream,
      stderr: io.stderr.stream
    });

    const payload = JSON.parse(io.stdout.read()) as {
      path: string;
      ok: boolean;
      status: string;
      command: string;
      message: string;
      dryRun: boolean;
    };

    expect(exitCode).toBe(EXIT_CODES.SUCCESS);
    expect(io.stderr.read()).toBe("");
    expect(payload).toMatchObject({
      ok: true,
      status: "dry_run",
      command: "cnm init",
      dryRun: true
    });
    expect(payload.path).toContain("config.json");
  });

  it("rejects help in json mode", async () => {
    const io = createCliIo();

    const exitCode = await runCli({
      argv: ["--json", "--help"],
      version: "0.1.0",
      stdout: io.stdout.stream,
      stderr: io.stderr.stream
    });

    expect(exitCode).toBe(EXIT_CODES.ERROR);
    expect(io.stdout.read()).toBe("");
    expect(io.stderr.read()).toContain("error: --json cannot be combined with --help.");
  });
});
