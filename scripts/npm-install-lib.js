

const crypto = require('node:crypto');
const fs = require('node:fs');

function archiveExt(osName) {
  return osName === 'windows' ? 'zip' : 'tar.gz';
}

function archiveNameFor(version, osName, arch) {
  return `commit-now-myfriend_${version}_${osName}_${arch}.${archiveExt(osName)}`;
}

function parseChecksums(content) {
  const checksums = new Map();
  for (const rawLine of content.split(/\r?\n/)) {
    const line = rawLine.trim();
    if (!line) continue;
    const match = line.match(/^([a-fA-F0-9]{64})\s+\*?(.+)$/);
    if (!match) continue;
    checksums.set(match[2], match[1].toLowerCase());
  }
  return checksums;
}

function sha256File(filePath) {
  const hash = crypto.createHash('sha256');
  hash.update(fs.readFileSync(filePath));
  return hash.digest('hex');
}

function verifyFileChecksum(filePath, fileName, checksums) {
  const expected = checksums.get(fileName);
  if (!expected) {
    throw new Error(`No checksum entry found for ${fileName}`);
  }
  const actual = sha256File(filePath);
  if (actual !== expected) {
    throw new Error(`Checksum verification failed for ${fileName}`);
  }
}

module.exports = {
  archiveExt,
  archiveNameFor,
  parseChecksums,
  sha256File,
  verifyFileChecksum,
};
