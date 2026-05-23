#!/usr/bin/env node
'use strict';

const fs = require('node:fs');
const path = require('node:path');
const os = require('node:os');
const https = require('node:https');
const { spawnSync } = require('node:child_process');

const pkg = require('../package.json');
const version = pkg.version;
const owner = process.env.CNM_RELEASE_OWNER || 'ByteTrue';
const repo = process.env.CNM_RELEASE_REPO || 'commit-now-myfriend';
const baseUrl = process.env.CNM_RELEASE_BASE_URL || `https://github.com/${owner}/${repo}/releases/download/v${version}`;
const binDir = path.join(__dirname, '..', 'bin');
const targetName = process.platform === 'win32' ? 'cnm.exe' : 'cnm';
const targetPath = path.join(binDir, targetName);

function platformName() {
  switch (process.platform) {
    case 'darwin': return 'darwin';
    case 'linux': return 'linux';
    case 'win32': return 'windows';
    default: return null;
  }
}

function archName() {
  switch (process.arch) {
    case 'x64': return 'amd64';
    case 'arm64': return 'arm64';
    default: return null;
  }
}

function archiveExt(osName) {
  return osName === 'windows' ? 'zip' : 'tar.gz';
}

function download(url, destination) {
  return new Promise((resolve, reject) => {
    const file = fs.createWriteStream(destination);
    const request = https.get(url, (response) => {
      if (response.statusCode >= 300 && response.statusCode < 400 && response.headers.location) {
        file.close(() => fs.rmSync(destination, { force: true }));
        return resolve(download(response.headers.location, destination));
      }
      if (response.statusCode !== 200) {
        file.close(() => fs.rmSync(destination, { force: true }));
        return reject(new Error(`Download failed with HTTP ${response.statusCode} for ${url}`));
      }
      response.pipe(file);
      file.on('finish', () => file.close(resolve));
    });
    request.on('error', (error) => {
      file.close(() => fs.rmSync(destination, { force: true }));
      reject(error);
    });
  });
}

function extract(archivePath, destination, osName) {
  if (osName === 'windows') {
    const powershell = spawnSync('powershell.exe', ['-NoProfile', '-Command', `Expand-Archive -LiteralPath '${archivePath}' -DestinationPath '${destination}' -Force`], { stdio: 'inherit' });
    if (powershell.status !== 0) {
      throw new Error('Failed to extract Windows archive with PowerShell Expand-Archive.');
    }
    return;
  }
  const tar = spawnSync('tar', ['-xzf', archivePath, '-C', destination], { stdio: 'inherit' });
  if (tar.status !== 0) {
    throw new Error('Failed to extract release archive with tar.');
  }
}

function findBinary(root, fileName) {
  const queue = [root];
  while (queue.length > 0) {
    const current = queue.shift();
    const entries = fs.readdirSync(current, { withFileTypes: true });
    for (const entry of entries) {
      const fullPath = path.join(current, entry.name);
      if (entry.isDirectory()) {
        queue.push(fullPath);
      } else if (entry.isFile() && entry.name === fileName) {
        return fullPath;
      }
    }
  }
  return null;
}

async function main() {
  const osName = platformName();
  const arch = archName();
  if (!osName || !arch) {
    console.warn(`[cnm] No prebuilt binary is published for ${process.platform}/${process.arch}.`);
    console.warn('[cnm] Install a release binary manually or build from source with `make go-build`.');
    return;
  }

  fs.mkdirSync(binDir, { recursive: true });
  const ext = archiveExt(osName);
  const archiveName = `commit-now-myfriend_${version}_${osName}_${arch}.${ext}`;
  const archivePath = path.join(os.tmpdir(), archiveName);
  const extractDir = fs.mkdtempSync(path.join(os.tmpdir(), 'cnm-install-'));
  const url = `${baseUrl}/${archiveName}`;

  console.log(`[cnm] Downloading ${url}`);
  await download(url, archivePath);
  extract(archivePath, extractDir, osName);

  const binary = findBinary(extractDir, targetName);
  if (!binary) {
    throw new Error(`Unable to locate ${targetName} in extracted archive.`);
  }

  fs.copyFileSync(binary, targetPath);
  if (process.platform !== 'win32') {
    fs.chmodSync(targetPath, 0o755);
  }
  console.log(`[cnm] Installed ${targetName} to ${targetPath}`);
}

main().catch((error) => {
  console.error(`[cnm] ${error instanceof Error ? error.message : String(error)}`);
  process.exitCode = 1;
});
