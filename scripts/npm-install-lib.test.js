const test = require('node:test');
const assert = require('node:assert/strict');
const fs = require('node:fs');
const os = require('node:os');
const path = require('node:path');
const crypto = require('node:crypto');

const {
  archiveExt,
  archiveNameFor,
  parseChecksums,
  verifyFileChecksum,
} = require('./npm-install-lib');

test('archive naming matches release conventions', () => {
  assert.equal(archiveExt('darwin'), 'tar.gz');
  assert.equal(archiveExt('linux'), 'tar.gz');
  assert.equal(archiveExt('windows'), 'zip');
  assert.equal(
    archiveNameFor('0.1.4', 'windows', 'amd64'),
    'commit-now-myfriend_0.1.4_windows_amd64.zip',
  );
});

test('parseChecksums reads goreleaser-style checksum files', () => {
  const content = [
    'aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa  commit-now-myfriend_0.1.4_linux_amd64.tar.gz',
    'bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb *commit-now-myfriend_0.1.4_windows_amd64.zip',
  ].join('\n');
  const checksums = parseChecksums(content);

  assert.equal(
    checksums.get('commit-now-myfriend_0.1.4_linux_amd64.tar.gz'),
    'aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa',
  );
  assert.equal(
    checksums.get('commit-now-myfriend_0.1.4_windows_amd64.zip'),
    'bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb',
  );
});

test('verifyFileChecksum accepts matching checksum and rejects mismatches', () => {
  const tmpDir = fs.mkdtempSync(path.join(os.tmpdir(), 'cnm-checksum-test-'));
  const fileName = 'commit-now-myfriend_0.1.4_linux_amd64.tar.gz';
  const filePath = path.join(tmpDir, fileName);
  fs.writeFileSync(filePath, 'fixture archive');

  const checksum = crypto
    .createHash('sha256')
    .update(fs.readFileSync(filePath))
    .digest('hex');
  const checksums = new Map([[fileName, checksum]]);

  assert.doesNotThrow(() => verifyFileChecksum(filePath, fileName, checksums));

  const badChecksums = new Map([[fileName, '0'.repeat(64)]]);
  assert.throws(
    () => verifyFileChecksum(filePath, fileName, badChecksums),
    /Checksum verification failed/,
  );

  assert.throws(
    () => verifyFileChecksum(filePath, 'missing.tar.gz', checksums),
    /No checksum entry found/,
  );
});
