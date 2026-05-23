#!/usr/bin/env node
'use strict';

const path = require('node:path');
const fs = require('node:fs');
const { spawnSync } = require('node:child_process');

const executable = process.platform === 'win32' ? 'cnm.exe' : 'cnm';
const installedBinaryPath = path.join(__dirname, '..', 'bin', executable);
const builtBinaryPath = path.join(__dirname, '..', 'dist', 'go', executable);
const binaryPath = fs.existsSync(installedBinaryPath) ? installedBinaryPath : builtBinaryPath;

if (!fs.existsSync(binaryPath)) {
  console.error('[cnm] Native binary is not installed. Reinstall the package or run `npm run build:go` first.');
  process.exit(1);
}

const result = spawnSync(binaryPath, process.argv.slice(2), {
  stdio: 'inherit'
});

if (result.error) {
  console.error(`[cnm] Failed to launch native binary: ${result.error.message}`);
  process.exit(1);
}

process.exit(result.status ?? 1);
