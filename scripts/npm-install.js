#!/usr/bin/env node

const fs = require("node:fs");
const path = require("node:path");
const os = require("node:os");
const https = require("node:https");
const { spawnSync } = require("node:child_process");

const pkg = require("../package.json");
const {
	archiveNameFor,
	archName,
	createInstallerSessionPaths,
	findBinary,
	parseChecksums,
	platformName,
	targetExecutableName,
	verifyFileChecksum,
} = require("./npm-install-lib");
const version = pkg.version;
const owner = process.env.CNM_RELEASE_OWNER || "ByteTrue";
const repo = process.env.CNM_RELEASE_REPO || "commit-now-myfriend";
const baseUrl =
	process.env.CNM_RELEASE_BASE_URL ||
	`https://github.com/${owner}/${repo}/releases/download/v${version}`;
const binDir =
	process.env.CNM_INSTALL_BIN_DIR || path.join(__dirname, "..", "bin");
const targetName = targetExecutableName(process.platform);
const targetPath = path.join(binDir, targetName);

function download(url, destination) {
	return new Promise((resolve, reject) => {
		const file = fs.createWriteStream(destination);
		const request = https.get(url, (response) => {
			if (
				response.statusCode >= 300 &&
				response.statusCode < 400 &&
				response.headers.location
			) {
				file.close(() => fs.rmSync(destination, { force: true }));
				return resolve(download(response.headers.location, destination));
			}
			if (response.statusCode !== 200) {
				file.close(() => fs.rmSync(destination, { force: true }));
				return reject(
					new Error(
						`Download failed with HTTP ${response.statusCode} for ${url}`,
					),
				);
			}
			response.pipe(file);
			file.on("finish", () => file.close(resolve));
		});
		request.on("error", (error) => {
			file.close(() => fs.rmSync(destination, { force: true }));
			reject(error);
		});
	});
}

function extract(archivePath, destination, osName) {
	if (osName === "windows") {
		const powershell = spawnSync(
			"powershell.exe",
			[
				"-NoProfile",
				"-Command",
				`Expand-Archive -LiteralPath '${archivePath}' -DestinationPath '${destination}' -Force`,
			],
			{ stdio: "inherit" },
		);
		if (powershell.status !== 0) {
			throw new Error(
				"Failed to extract Windows archive with PowerShell Expand-Archive.",
			);
		}
		return;
	}
	const tar = spawnSync("tar", ["-xzf", archivePath, "-C", destination], {
		stdio: "inherit",
	});
	if (tar.status !== 0) {
		throw new Error("Failed to extract release archive with tar.");
	}
}

async function main() {
	const osName = platformName(process.platform);
	const arch = archName(process.arch);
	if (!osName || !arch) {
		console.warn(
			`[cnm] No prebuilt binary is published for ${process.platform}/${process.arch}.`,
		);
		console.warn(
			"[cnm] Install a release binary manually or build from source with `make go-build`.",
		);
		return;
	}

	fs.mkdirSync(binDir, { recursive: true });
	const archiveName = archiveNameFor(version, osName, arch);
	const tmpRoot = process.env.CNM_INSTALL_TMP_DIR || os.tmpdir();
	const { archivePath, checksumsPath, extractDir } =
		createInstallerSessionPaths(tmpRoot, archiveName, version);
	const url = `${baseUrl}/${archiveName}`;
	const checksumsUrl = `${baseUrl}/checksums.txt`;

	console.log(`[cnm] Downloading ${url}`);
	await download(url, archivePath);
	console.log(`[cnm] Downloading ${checksumsUrl}`);
	await download(checksumsUrl, checksumsPath);
	verifyFileChecksum(
		archivePath,
		archiveName,
		parseChecksums(fs.readFileSync(checksumsPath, "utf8")),
	);
	extract(archivePath, extractDir, osName);

	const binary = findBinary(extractDir, targetName);
	if (!binary) {
		throw new Error(`Unable to locate ${targetName} in extracted archive.`);
	}

	fs.copyFileSync(binary, targetPath);
	if (process.platform !== "win32") {
		fs.chmodSync(targetPath, 0o755);
	}
	console.log(`[cnm] Installed ${targetName} to ${targetPath}`);
}

main().catch((error) => {
	console.error(
		`[cnm] ${error instanceof Error ? error.message : String(error)}`,
	);
	process.exitCode = 1;
});
