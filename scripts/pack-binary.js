#!/usr/bin/env node
'use strict';

const fs = require('node:fs');
const path = require('node:path');
const { spawnSync } = require('node:child_process');

const pkg = require('../package.json');
const version = pkg.version;
const outputDir = path.join(__dirname, '..', 'dist', 'release-local');
const binaryName = process.platform === 'win32' ? 'cnm.exe' : 'cnm';
const builtBinary = path.join(__dirname, '..', 'dist', 'go', binaryName);

if (!fs.existsSync(builtBinary)) {
  console.error(`[cnm] ${builtBinary} does not exist. Run \`npm run build:go\` first.`);
  process.exit(1);
}

fs.mkdirSync(outputDir, { recursive: true });
const osName = process.platform === 'darwin' ? 'darwin' : process.platform === 'linux' ? 'linux' : process.platform === 'win32' ? 'windows' : process.platform;
const arch = process.arch === 'x64' ? 'amd64' : process.arch;
const archiveBase = `commit-now-myfriend_${version}_${osName}_${arch}`;
const stagingDir = path.join(outputDir, archiveBase);
fs.rmSync(stagingDir, { recursive: true, force: true });
fs.mkdirSync(stagingDir, { recursive: true });
fs.copyFileSync(builtBinary, path.join(stagingDir, binaryName));
for (const file of ['README.md', 'LICENSE']) {
  const source = path.join(__dirname, '..', file);
  if (fs.existsSync(source)) {
    fs.copyFileSync(source, path.join(stagingDir, file));
  }
}

if (process.platform === 'win32') {
  const archivePath = path.join(outputDir, `${archiveBase}.zip`);
  const command = `Compress-Archive -Path '${stagingDir}\\*' -DestinationPath '${archivePath}' -Force`;
  const result = spawnSync('powershell.exe', ['-NoProfile', '-Command', command], { stdio: 'inherit' });
  process.exit(result.status ?? 1);
}

const archivePath = path.join(outputDir, `${archiveBase}.tar.gz`);
const result = spawnSync('tar', ['-czf', archivePath, '-C', outputDir, archiveBase], { stdio: 'inherit' });
process.exit(result.status ?? 1);
