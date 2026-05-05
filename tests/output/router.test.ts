import { describe, expect, it } from "vitest";

import { createOutputRouter, type CliWriteStream } from "../../src/output/index.js";

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

describe("createOutputRouter", () => {
  it("writes exactly one parseable JSON object to stdout in json mode", () => {
    const stdout = createMemoryStream();
    const stderr = createMemoryStream();
    const router = createOutputRouter({ json: true, stderr: stderr.stream, stdout: stdout.stream });

    router.writeStructured(
      {
        committed: false,
        dryRun: false,
        error: null,
        files: [],
        message: "feat: preview output",
        ok: true,
        warnings: []
      },
      "human text must not leak",
      "stderr"
    );

    const output = stdout.read();

    expect(stderr.read()).toBe("");
    expect(output.trim().split(/\r?\n/)).toHaveLength(1);
    expect(JSON.parse(output)).toMatchObject({
      committed: false,
      dryRun: false,
      error: null,
      files: [],
      message: "feat: preview output",
      ok: true,
      warnings: []
    });
  });
});
