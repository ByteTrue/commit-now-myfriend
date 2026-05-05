#!/usr/bin/env node

import { readFile } from "node:fs/promises";

import { runCli } from "./cli.js";
import { EXIT_CODES } from "./output/index.js";

interface PackageMetadata {
  engines?: {
    node?: string;
  };
  version?: string;
}

async function readPackageMetadata(): Promise<PackageMetadata> {
  const packageJsonUrl = new URL("../package.json", import.meta.url);
  const packageJsonContent = await readFile(packageJsonUrl, "utf8");

  return JSON.parse(packageJsonContent) as PackageMetadata;
}

try {
  const { engines, version = "0.0.0" } = await readPackageMetadata();
  process.exitCode = await runCli({
    argv: process.argv.slice(2),
    nodeEngine: engines?.node,
    version
  });
} catch (error) {
  const message = error instanceof Error ? error.message : "Unexpected startup failure.";

  process.stderr.write(`error: ${message}\n`);
  process.exitCode = EXIT_CODES.ERROR;
}
