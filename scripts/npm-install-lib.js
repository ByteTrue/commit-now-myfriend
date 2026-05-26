const path = require("node:path");
const crypto = require("node:crypto");
const fs = require("node:fs");

function archiveExt(osName) {
	return osName === "windows" ? "zip" : "tar.gz";
}

function archiveNameFor(version, osName, arch) {
	return `commit-now-myfriend_${version}_${osName}_${arch}.${archiveExt(osName)}`;
}

function platformName(platform = process.platform) {
	switch (platform) {
		case "darwin":
			return "darwin";
		case "linux":
			return "linux";
		case "win32":
			return "windows";
		default:
			return null;
	}
}

function archName(arch = process.arch) {
	switch (arch) {
		case "x64":
			return "amd64";
		case "arm64":
			return "arm64";
		default:
			return null;
	}
}

function targetExecutableName(platform = process.platform) {
	return platform === "win32" ? "cnm.exe" : "cnm";
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
	const hash = crypto.createHash("sha256");
	hash.update(fs.readFileSync(filePath));
	return hash.digest("hex");
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

function resolveWrapperBinaryPath({
	platform = process.platform,
	installedBinaryPath,
	builtBinaryPath,
	existsSync = fs.existsSync,
}) {
	const executable = targetExecutableName(platform);
	const installedPath = installedBinaryPath ?? path.join("bin", executable);
	const builtPath = builtBinaryPath ?? path.join("dist", "go", executable);

	if (existsSync(installedPath)) {
		return installedPath;
	}
	if (existsSync(builtPath)) {
		return builtPath;
	}
	return null;
}

function createInstallerSessionPaths(tmpRoot, archiveName, version) {
	const sessionRoot = fs.mkdtempSync(path.join(tmpRoot, "cnm-install-"));
	const extractDir = path.join(sessionRoot, "extract");
	fs.mkdirSync(extractDir, { recursive: true });
	return {
		sessionRoot,
		archivePath: path.join(sessionRoot, archiveName),
		checksumsPath: path.join(sessionRoot, `checksums-${version}.txt`),
		extractDir,
	};
}

module.exports = {
	archiveExt,
	archiveNameFor,
	platformName,
	archName,
	targetExecutableName,
	parseChecksums,
	sha256File,
	verifyFileChecksum,
	findBinary,
	resolveWrapperBinaryPath,
	createInstallerSessionPaths,
};
