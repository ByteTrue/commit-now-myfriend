const test = require("node:test");
const assert = require("node:assert/strict");
const fs = require("node:fs");
const os = require("node:os");
const path = require("node:path");
const crypto = require("node:crypto");

const {
	archiveNameFor,
	findBinary,
	parseChecksums,
	resolveWrapperBinaryPath,
	verifyFileChecksum,
} = require("./npm-install-lib");

test("windows archive contract matches zip naming and checksum lookup", () => {
	const archiveName = archiveNameFor("0.1.4", "windows", "amd64");
	assert.equal(archiveName, "commit-now-myfriend_0.1.4_windows_amd64.zip");

	const checksum = "a".repeat(64);
	const checksums = parseChecksums(`${checksum} *${archiveName}`);
	assert.equal(checksums.get(archiveName), checksum);
});

test("windows installer smoke finds cnm.exe inside extracted archive tree", () => {
	const extractRoot = fs.mkdtempSync(
		path.join(os.tmpdir(), "cnm-windows-extract-"),
	);
	const nestedDir = path.join(
		extractRoot,
		"commit-now-myfriend_0.1.4_windows_amd64",
	);
	fs.mkdirSync(nestedDir, { recursive: true });
	const binaryPath = path.join(nestedDir, "cnm.exe");
	fs.writeFileSync(binaryPath, "binary");

	assert.equal(findBinary(extractRoot, "cnm.exe"), binaryPath);
});

test("windows wrapper smoke prefers installed cnm.exe and falls back to built binary", () => {
	const installedPath = "C:/cnm/bin/cnm.exe";
	const builtPath = "C:/cnm/dist/go/cnm.exe";

	assert.equal(
		resolveWrapperBinaryPath({
			platform: "win32",
			installedBinaryPath: installedPath,
			builtBinaryPath: builtPath,
			existsSync: (candidate) => candidate === installedPath,
		}),
		installedPath,
	);

	assert.equal(
		resolveWrapperBinaryPath({
			platform: "win32",
			installedBinaryPath: installedPath,
			builtBinaryPath: builtPath,
			existsSync: (candidate) => candidate === builtPath,
		}),
		builtPath,
	);
});

test("windows installer smoke verifies downloaded zip before extraction", () => {
	const tmpDir = fs.mkdtempSync(
		path.join(os.tmpdir(), "cnm-windows-checksum-"),
	);
	const archiveName = "commit-now-myfriend_0.1.4_windows_amd64.zip";
	const archivePath = path.join(tmpDir, archiveName);
	fs.writeFileSync(archivePath, "windows archive fixture");

	const checksum = crypto
		.createHash("sha256")
		.update(fs.readFileSync(archivePath))
		.digest("hex");
	const checksums = new Map([[archiveName, checksum]]);

	assert.doesNotThrow(() =>
		verifyFileChecksum(archivePath, archiveName, checksums),
	);
});
