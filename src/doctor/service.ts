import { access, constants, stat } from "node:fs/promises";

import { execa } from "execa";

import {
	DEFAULT_PROMPT_STYLE,
	DEFAULT_PROVIDER,
	getConfigPaths,
	getDefaultModel,
	getEnvConfig,
	loadProjectConfig,
	loadUserConfig,
	redactSecret,
	type ConfigKey,
	type ConfigValues,
	type EffectiveConfig,
} from "../config/index.js";
import { inspectGitRepository } from "../git/index.js";
import type { GitIssue } from "../git/types.js";
import type {
	DoctorCheckStatus,
	DoctorConfigDirectoryCheck,
	DoctorConfigFileCheck,
	DoctorConfigSnapshot,
	DoctorConfigSource,
	DoctorDependencies,
	DoctorEffectiveConfigCheck,
	DoctorGitCheck,
	DoctorIssue,
	DoctorIssueSeverity,
	DoctorNodeCheck,
	DoctorProbeResult,
	DoctorReport,
	DoctorRepositoryCheck,
	GitVersionResult,
	RunDoctorOptions,
	SafeConfigSourceResult,
} from "./types.js";

const PACKAGE_NAME = "commit-now-myfriend";
const PACKAGE_BIN = "cnm";

interface DirectoryAccessResult {
	exists: boolean;
	isDirectory: boolean;
	readable: boolean;
	writable: boolean;
}

interface FileMetadata {
	exists: boolean;
	mode: string | null;
}

interface EffectiveConfigSources {
	apiKey: DoctorConfigSource;
	baseURL: DoctorConfigSource;
	model: DoctorConfigSource;
	provider: DoctorConfigSource;
}

function pushIssue(
	issues: DoctorIssue[],
	issue: Omit<DoctorIssue, "severity"> & { severity: DoctorIssueSeverity },
): void {
	issues.push(issue);
}

function readIssueCodes(
	issues: DoctorIssue[],
	check: DoctorIssue["check"],
): string[] {
	return issues
		.filter((issue) => issue.check === check)
		.map((issue) => issue.code);
}

function summarizeCheck(
	issues: DoctorIssue[],
	check: DoctorIssue["check"],
): DoctorCheckStatus {
	const related = issues.filter((issue) => issue.check === check);

	if (related.some((issue) => issue.severity === "error")) {
		return "error";
	}

	if (related.length > 0) {
		return "warning";
	}

	return "pass";
}

function toSnapshot(
	config: ConfigValues | EffectiveConfig | null,
): DoctorConfigSnapshot | null {
	if (!config) {
		return null;
	}

	return {
		apiKey: redactSecret(config.apiKey) ?? null,
		baseURL: config.baseURL ?? null,
		customPrompt: config.customPrompt ?? null,
		model: config.model ?? null,
		promptStyle: config.promptStyle ?? null,
		provider: config.provider ?? null,
	};
}

function createEffectiveConfig(
	userConfig: ConfigValues,
	projectConfig: ConfigValues,
	envConfig: ConfigValues,
): EffectiveConfig {
	const mergedConfig: ConfigValues = {
		...userConfig,
		...projectConfig,
		...envConfig,
	};
	const provider = mergedConfig.provider ?? DEFAULT_PROVIDER;
	const model = mergedConfig.model ?? getDefaultModel(provider);
	const promptStyle = mergedConfig.promptStyle ?? DEFAULT_PROMPT_STYLE;

	return {
		apiKey: mergedConfig.apiKey,
		baseURL: mergedConfig.baseURL,
		customPrompt: mergedConfig.customPrompt,
		model,
		promptStyle,
		provider,
	};
}

function resolveConfigSource(
	key: ConfigKey,
	userConfig: ConfigValues,
	projectConfig: ConfigValues,
	envConfig: ConfigValues,
	effectiveConfig: EffectiveConfig,
): DoctorConfigSource {
	if (envConfig[key] !== undefined) {
		return "env";
	}

	if (projectConfig[key] !== undefined) {
		return "project";
	}

	if (userConfig[key] !== undefined) {
		return "user";
	}

	return effectiveConfig[key] === undefined ? "missing" : "default";
}

function resolveEffectiveConfigSources(
	userConfig: ConfigValues,
	projectConfig: ConfigValues,
	envConfig: ConfigValues,
	effectiveConfig: EffectiveConfig,
): EffectiveConfigSources {
	return {
		apiKey: resolveConfigSource(
			"apiKey",
			userConfig,
			projectConfig,
			envConfig,
			effectiveConfig,
		),
		baseURL: resolveConfigSource(
			"baseURL",
			userConfig,
			projectConfig,
			envConfig,
			effectiveConfig,
		),
		model: resolveConfigSource(
			"model",
			userConfig,
			projectConfig,
			envConfig,
			effectiveConfig,
		),
		provider: resolveConfigSource(
			"provider",
			userConfig,
			projectConfig,
			envConfig,
			effectiveConfig,
		),
	};
}

function normalizeNodeMajor(version: string | null): number | null {
	if (!version) {
		return null;
	}

	const match = version.match(/v?(\d+)/u);
	return match ? Number.parseInt(match[1] ?? "", 10) : null;
}

function buildNodeCheck(
	issues: DoctorIssue[],
	currentVersion: string,
	requiredVersion: string | undefined,
): DoctorNodeCheck {
	const normalizedRequiredVersion = requiredVersion ?? null;

	if (!normalizedRequiredVersion) {
		return {
			currentVersion,
			issueCodes: [],
			message: `Node ${currentVersion} detected. No package engine requirement was supplied to doctor.`,
			requiredVersion: null,
			status: "pass",
			supported: null,
		};
	}

	const requiredMajor = normalizeNodeMajor(normalizedRequiredVersion);
	const currentMajor = normalizeNodeMajor(currentVersion);

	if (requiredMajor === null || currentMajor === null) {
		return {
			currentVersion,
			issueCodes: [],
			message: `Node ${currentVersion} detected. Could not evaluate requirement ${normalizedRequiredVersion}.`,
			requiredVersion: normalizedRequiredVersion,
			status: "pass",
			supported: null,
		};
	}

	if (currentMajor < requiredMajor) {
		pushIssue(issues, {
			check: "node",
			code: "node_version_unsupported",
			message: `Node ${currentVersion} does not satisfy the package requirement ${normalizedRequiredVersion}.`,
			severity: "error",
		});
	}

	const status = summarizeCheck(issues, "node");

	return {
		currentVersion,
		issueCodes: readIssueCodes(issues, "node"),
		message:
			status === "error"
				? `Node ${currentVersion} does not satisfy ${normalizedRequiredVersion}.`
				: `Node ${currentVersion} satisfies ${normalizedRequiredVersion}.`,
		requiredVersion: normalizedRequiredVersion,
		status,
		supported: status !== "error",
	};
}

async function runGitVersion(
	cwd: string,
	env?: NodeJS.ProcessEnv,
): Promise<GitVersionResult> {
	try {
		const result = await execa("git", ["--version"], {
			cwd,
			env,
			reject: false,
			stdin: "ignore",
			stderr: "pipe",
			stdout: "pipe",
		});

		if ((result.exitCode ?? 1) !== 0 || !result.stdout.trim()) {
			return {
				available: false,
				version: null,
			};
		}

		return {
			available: true,
			version: result.stdout.trim(),
		};
	} catch (error) {
		const errorCode = (error as NodeJS.ErrnoException).code;

		if (errorCode === "ENOENT") {
			return {
				available: false,
				version: null,
			};
		}

		return {
			available: false,
			version: null,
		};
	}
}

function buildGitCheck(
	issues: DoctorIssue[],
	gitVersion: GitVersionResult,
): DoctorGitCheck {
	if (!gitVersion.available) {
		pushIssue(issues, {
			check: "git",
			code: "git_executable_missing",
			message:
				"Git is not available on PATH; install git before using cnm workflows.",
			severity: "error",
		});
	}

	const status = summarizeCheck(issues, "git");

	return {
		available: gitVersion.available,
		issueCodes: readIssueCodes(issues, "git"),
		message: gitVersion.available
			? `${gitVersion.version ?? "git detected."}`
			: "Git is not available on PATH.",
		status,
		version: gitVersion.version,
	};
}

async function inspectDirectory(
	candidatePath: string,
): Promise<DirectoryAccessResult> {
	try {
		const metadata = await stat(candidatePath);
		const isDirectory = metadata.isDirectory();

		if (!isDirectory) {
			return {
				exists: true,
				isDirectory: false,
				readable: false,
				writable: false,
			};
		}

		let readable = false;
		let writable = false;

		try {
			await access(candidatePath, constants.R_OK);
			readable = true;
		} catch {
			readable = false;
		}

		try {
			await access(candidatePath, constants.W_OK);
			writable = true;
		} catch {
			writable = false;
		}

		return {
			exists: true,
			isDirectory,
			readable,
			writable,
		};
	} catch (error) {
		if ((error as NodeJS.ErrnoException).code === "ENOENT") {
			return {
				exists: false,
				isDirectory: false,
				readable: false,
				writable: false,
			};
		}

		return {
			exists: true,
			isDirectory: false,
			readable: false,
			writable: false,
		};
	}
}

function formatMode(mode: number): string {
	return `0${(mode & 0o777).toString(8).padStart(3, "0")}`;
}

async function inspectFileMode(filePath: string): Promise<FileMetadata> {
	try {
		const metadata = await stat(filePath);

		if (!metadata.isFile()) {
			return {
				exists: true,
				mode: null,
			};
		}

		return {
			exists: true,
			mode: formatMode(metadata.mode),
		};
	} catch (error) {
		if ((error as NodeJS.ErrnoException).code === "ENOENT") {
			return {
				exists: false,
				mode: null,
			};
		}

		return {
			exists: true,
			mode: null,
		};
	}
}

async function safeLoadUserConfig(
	cwd: string,
	env?: NodeJS.ProcessEnv,
): Promise<SafeConfigSourceResult> {
	const paths = getConfigPaths({ cwd, env });
	const metadata = await inspectFileMode(paths.userConfigPath);

	try {
		const config = await loadUserConfig({ cwd, env });
		return {
			config,
			error: null,
			exists: metadata.exists,
			warningMessages: [],
		};
	} catch (error) {
		return {
			config: {},
			error:
				error instanceof Error
					? error
					: new Error("Unable to load user config."),
			exists: metadata.exists,
			warningMessages: [],
		};
	}
}

async function safeLoadProjectConfig(
	cwd: string,
	env?: NodeJS.ProcessEnv,
): Promise<SafeConfigSourceResult> {
	const paths = getConfigPaths({ cwd, env });
	const metadata = await inspectFileMode(paths.projectConfigPath);

	try {
		const result = await loadProjectConfig({ cwd, env });
		return {
			config: result.config,
			error: null,
			exists: metadata.exists,
			warningMessages: result.warnings,
		};
	} catch (error) {
		return {
			config: {},
			error:
				error instanceof Error
					? error
					: new Error("Unable to load project config."),
			exists: metadata.exists,
			warningMessages: [],
		};
	}
}

function safeLoadEnvConfig(env?: NodeJS.ProcessEnv): SafeConfigSourceResult {
	try {
		return {
			config: getEnvConfig(env),
			error: null,
			exists: true,
			warningMessages: [],
		};
	} catch (error) {
		return {
			config: {},
			error:
				error instanceof Error
					? error
					: new Error("Unable to read environment configuration."),
			exists: true,
			warningMessages: [],
		};
	}
}

function appendConfigSourceIssues(
	issues: DoctorIssue[],
	check: "userConfig" | "projectConfig" | "effectiveConfig",
	result: SafeConfigSourceResult,
	invalidCode: string,
	invalidLabel: string,
): void {
	if (result.error) {
		pushIssue(issues, {
			check,
			code: invalidCode,
			message: result.error.message || invalidLabel,
			severity: "error",
		});
	}

	for (const warningMessage of result.warningMessages) {
		pushIssue(issues, {
			check,
			code: "project_api_key_ignored",
			message: warningMessage,
			severity: "warning",
		});
	}
}

function severityForGitIssue(issue: GitIssue): DoctorIssueSeverity {
	if (issue.code === "not_git_repository") {
		return "warning";
	}

	return issue.severity === "blocking" ? "error" : "warning";
}

function mergeGitIssues(issues: DoctorIssue[], gitIssues: GitIssue[]): void {
	for (const issue of gitIssues) {
		pushIssue(issues, {
			check: "repository",
			code: issue.code,
			message: issue.message,
			severity: severityForGitIssue(issue),
		});
	}
}

function buildRepositoryCheck(
	issues: DoctorIssue[],
	inspection: Awaited<ReturnType<typeof inspectGitRepository>> | null,
): DoctorRepositoryCheck {
	if (!inspection) {
		return {
			branchName: null,
			gitIdentity: { email: null, name: null },
			isBare: false,
			isRepository: false,
			issueCodes: readIssueCodes(issues, "repository"),
			message: "Repository inspection was skipped because git is unavailable.",
			rootPath: null,
			status:
				summarizeCheck(issues, "repository") === "pass"
					? "warning"
					: summarizeCheck(issues, "repository"),
		};
	}

	const repository = inspection.repository;
	const status = summarizeCheck(issues, "repository");

	return {
		branchName: repository.branchName,
		gitIdentity: repository.gitIdentity,
		isBare: repository.isBare,
		isRepository: repository.isRepository,
		issueCodes: readIssueCodes(issues, "repository"),
		message: !repository.isRepository
			? "Current directory is not inside a git repository."
			: repository.rootPath
				? `Git repository detected at ${repository.rootPath}.`
				: "Git repository detected.",
		rootPath: repository.rootPath,
		status,
	};
}

function buildConfigDirectoryCheck(
	issues: DoctorIssue[],
	configHomePath: string,
	accessResult: DirectoryAccessResult,
): DoctorConfigDirectoryCheck {
	if (!accessResult.exists) {
		pushIssue(issues, {
			check: "configDirectory",
			code: "config_dir_missing",
			message: `Config directory ${configHomePath} does not exist yet.`,
			severity: "warning",
		});
	} else if (!accessResult.isDirectory) {
		pushIssue(issues, {
			check: "configDirectory",
			code: "config_dir_invalid",
			message: `Config path ${configHomePath} exists but is not a directory.`,
			severity: "error",
		});
	} else if (!accessResult.readable || !accessResult.writable) {
		pushIssue(issues, {
			check: "configDirectory",
			code: "config_dir_inaccessible",
			message: `Config directory ${configHomePath} is not both readable and writable.`,
			severity: "error",
		});
	}

	const status = summarizeCheck(issues, "configDirectory");
	let message = `Config directory ${configHomePath} is ready.`;

	if (status === "warning") {
		message = `Config directory ${configHomePath} does not exist yet.`;
	}

	if (status === "error") {
		message = `Config directory ${configHomePath} is not usable.`;
	}

	return {
		exists: accessResult.exists,
		isDirectory: accessResult.isDirectory,
		issueCodes: readIssueCodes(issues, "configDirectory"),
		message,
		path: configHomePath,
		readable: accessResult.readable,
		status,
		writable: accessResult.writable,
	};
}

function maybeWarnOnPermissions(
	issues: DoctorIssue[],
	check: "userConfig",
	filePath: string,
	mode: string | null,
): void {
	if (process.platform === "win32" || !mode) {
		return;
	}

	const parsedMode = Number.parseInt(mode, 8);

	if (Number.isNaN(parsedMode)) {
		return;
	}

	if ((parsedMode & 0o077) !== 0) {
		pushIssue(issues, {
			check,
			code: "user_config_permissions_insecure",
			message: `User config at ${filePath} should use 0600 permissions; found ${mode}.`,
			severity: "warning",
		});
	}
}

function buildConfigFileCheck(
	issues: DoctorIssue[],
	check: "userConfig" | "projectConfig",
	filePath: string,
	result: SafeConfigSourceResult,
	mode: string | null,
	messageWhenMissing: string,
	messageWhenPresent: string,
): DoctorConfigFileCheck {
	const status = summarizeCheck(issues, check);
	let message = result.exists ? messageWhenPresent : messageWhenMissing;

	if (status === "warning") {
		const firstWarning = issues.find(
			(issue) => issue.check === check && issue.severity === "warning",
		);
		message = firstWarning?.message ?? message;
	}

	if (status === "error") {
		const firstError = issues.find(
			(issue) => issue.check === check && issue.severity === "error",
		);
		message = firstError?.message ?? message;
	}

	return {
		config: result.error ? null : toSnapshot(result.config),
		exists: result.exists,
		issueCodes: readIssueCodes(issues, check),
		message,
		mode,
		path: filePath,
		status,
		valid: result.error === null,
	};
}

function appendEffectiveConfigIssues(
	issues: DoctorIssue[],
	effectiveConfig: EffectiveConfig,
	sources: EffectiveConfigSources,
	envConfig: SafeConfigSourceResult,
): void {
	if (envConfig.error) {
		pushIssue(issues, {
			check: "effectiveConfig",
			code: "env_config_invalid",
			message: envConfig.error.message,
			severity: "error",
		});
	}

	const missingExplicitProviderConfig =
		sources.provider === "default" &&
		sources.model === "default" &&
		sources.apiKey === "missing";

	if (missingExplicitProviderConfig) {
		pushIssue(issues, {
			check: "effectiveConfig",
			code: "provider_config_missing",
			message:
				"Provider configuration is still using built-in defaults; run `cnm init` or configure cnm before generating commits.",
			severity: "error",
		});
	}

	if (!effectiveConfig.apiKey) {
		pushIssue(issues, {
			check: "effectiveConfig",
			code: "api_key_missing",
			message: `No API key is configured for ${effectiveConfig.provider}.`,
			severity: "error",
		});
	}

	if (
		effectiveConfig.provider === "openai-compatible" &&
		!effectiveConfig.baseURL
	) {
		pushIssue(issues, {
			check: "effectiveConfig",
			code: "provider_config_missing",
			message:
				"The openai-compatible provider requires `baseURL` to be configured.",
			severity: "error",
		});
	}
}

function buildEffectiveConfigCheck(
	issues: DoctorIssue[],
	effectiveConfig: EffectiveConfig,
	sources: EffectiveConfigSources,
): DoctorEffectiveConfigCheck {
	const status = summarizeCheck(issues, "effectiveConfig");
	let message = `Provider ${effectiveConfig.provider} is configured with model ${effectiveConfig.model}.`;

	if (status === "warning") {
		const warning = issues.find(
			(issue) =>
				issue.check === "effectiveConfig" && issue.severity === "warning",
		);
		message = warning?.message ?? message;
	}

	if (status === "error") {
		const error = issues.find(
			(issue) =>
				issue.check === "effectiveConfig" && issue.severity === "error",
		);
		message = error?.message ?? message;
	}

	return {
		config: toSnapshot(effectiveConfig) ?? {
			apiKey: null,
			baseURL: null,
			customPrompt: null,
			model: null,
			promptStyle: null,
			provider: null,
		},
		issueCodes: readIssueCodes(issues, "effectiveConfig"),
		message,
		sources,
		status,
	};
}

function buildGuidance(
	reportIssues: DoctorIssue[],
	userConfigPath: string,
): string[] {
	const guidance = new Set<string>();
	const issueCodes = new Set(reportIssues.map((issue) => issue.code));

	if (
		issueCodes.has("provider_config_missing") ||
		issueCodes.has("api_key_missing")
	) {
		guidance.add(
			"Run `cnm init` or `cnm config set apiKey <value>` to finish provider setup. Prefer the `CNM_API_KEY` environment variable when possible.",
		);
	}

	if (issueCodes.has("not_git_repository")) {
		guidance.add(
			"Run `cnm doctor` from inside the repository you want to use with `cnm`.",
		);
	}

	if (issueCodes.has("project_api_key_ignored")) {
		guidance.add(
			"Remove `apiKey` from project config; project-level secrets are ignored by design.",
		);
	}

	if (issueCodes.has("user_config_permissions_insecure")) {
		guidance.add(
			`On Unix-like systems, run \`chmod 600 ${userConfigPath}\` to tighten user config permissions.`,
		);
	}

	guidance.add(
		"This package installs the `cnm` binary from the `commit-now-myfriend` package name.",
	);
	guidance.add(
		"If `cnm` resolves to another executable, use `npx commit-now-myfriend`, `pnpm exec cnm`, or remove the conflicting global package.",
	);

	return [...guidance];
}

export async function runDoctor({
	cwd,
	dependencies,
	env,
	nodeEngine,
	nodeVersion = process.version,
}: RunDoctorOptions): Promise<DoctorReport> {
	const doctorDependencies: DoctorDependencies = {
		inspectGitRepository,
		runGitVersion,
		...dependencies,
	};
	const paths = getConfigPaths({ cwd, env });
	const issues: DoctorIssue[] = [];
	const nodeCheck = buildNodeCheck(issues, nodeVersion, nodeEngine);
	const gitVersion = await doctorDependencies.runGitVersion(cwd, env);
	const gitCheck = buildGitCheck(issues, gitVersion);
	const inspection = gitVersion.available
		? await doctorDependencies.inspectGitRepository({ cwd, env })
		: null;

	if (inspection) {
		mergeGitIssues(issues, inspection.issues);
	}

	const repositoryCheck = buildRepositoryCheck(issues, inspection);
	const configDirectoryAccess = await inspectDirectory(paths.userConfigHome);
	const configDirectoryCheck = buildConfigDirectoryCheck(
		issues,
		paths.userConfigHome,
		configDirectoryAccess,
	);
	const userConfigResult = await safeLoadUserConfig(cwd, env);
	const projectConfigResult = await safeLoadProjectConfig(cwd, env);
	const envConfigResult = safeLoadEnvConfig(env);
	const userConfigMetadata = await inspectFileMode(paths.userConfigPath);
	const projectConfigMetadata = await inspectFileMode(paths.projectConfigPath);

	appendConfigSourceIssues(
		issues,
		"userConfig",
		userConfigResult,
		"user_config_invalid",
		"User config is invalid.",
	);
	appendConfigSourceIssues(
		issues,
		"projectConfig",
		projectConfigResult,
		"project_config_invalid",
		"Project config is invalid.",
	);
	maybeWarnOnPermissions(
		issues,
		"userConfig",
		paths.userConfigPath,
		userConfigMetadata.mode,
	);

	const effectiveConfig = createEffectiveConfig(
		userConfigResult.config,
		projectConfigResult.config,
		envConfigResult.config,
	);
	const effectiveSources = resolveEffectiveConfigSources(
		userConfigResult.config,
		projectConfigResult.config,
		envConfigResult.config,
		effectiveConfig,
	);

	appendEffectiveConfigIssues(
		issues,
		effectiveConfig,
		effectiveSources,
		envConfigResult,
	);

	const userConfigCheck = buildConfigFileCheck(
		issues,
		"userConfig",
		paths.userConfigPath,
		userConfigResult,
		userConfigMetadata.mode,
		`No user config found at ${paths.userConfigPath}.`,
		`User config loaded from ${paths.userConfigPath}.`,
	);
	const projectConfigCheck = buildConfigFileCheck(
		issues,
		"projectConfig",
		paths.projectConfigPath,
		projectConfigResult,
		projectConfigMetadata.mode,
		`No project config found at ${paths.projectConfigPath}.`,
		`Project config loaded from ${paths.projectConfigPath}.`,
	);
	const effectiveConfigCheck = buildEffectiveConfigCheck(
		issues,
		effectiveConfig,
		effectiveSources,
	);
	const summary = {
		errors: issues.filter((issue) => issue.severity === "error").length,
		warnings: issues.filter((issue) => issue.severity === "warning").length,
	};

	return {
		bin: {
			command: PACKAGE_BIN,
			packageName: PACKAGE_NAME,
		},
		checks: {
			configDirectory: configDirectoryCheck,
			effectiveConfig: effectiveConfigCheck,
			git: gitCheck,
			node: nodeCheck,
			projectConfig: projectConfigCheck,
			repository: repositoryCheck,
			userConfig: userConfigCheck,
		},
		command: "cnm doctor",
		guidance: buildGuidance(issues, paths.userConfigPath),
		issues,
		ok: summary.errors === 0,
		paths: {
			cwd,
			projectConfigPath: paths.projectConfigPath,
			userConfigHome: paths.userConfigHome,
			userConfigPath: paths.userConfigPath,
		},
		readOnly: true,
		status: summary.errors === 0 ? "ok" : "issues_found",
		summary,
	};
}

const PROBE_TIMEOUT_MS = 15_000;

export async function runProviderProbe(
	config: EffectiveConfig,
): Promise<DoctorProbeResult> {
	if (!config.apiKey || config.apiKey.trim().length === 0) {
		return {
			error: "No API key is configured.",
			model: config.model,
			provider: config.provider,
			status: "skipped",
		};
	}

	if (
		config.provider === "openai-compatible" &&
		(!config.baseURL || config.baseURL.trim().length === 0)
	) {
		return {
			error: "The openai-compatible provider requires baseURL.",
			model: config.model,
			provider: config.provider,
			status: "skipped",
		};
	}

	const controller = new AbortController();
	const timer = setTimeout(() => controller.abort(), PROBE_TIMEOUT_MS);
	const start = Date.now();

	try {
		const headers: Record<string, string> = {
			"Content-Type": "application/json",
		};
		const body = buildProbeBody(config);
		const url = buildProbeUrl(config, config.apiKey);

		applyProbeAuth(headers, config);

		const response = await fetch(url, {
			body: JSON.stringify(body),
			headers,
			method: "POST",
			signal: controller.signal,
		});
		const elapsed = Date.now() - start;
		const text = await response.text().catch(() => "");

		if (!response.ok) {
			const snippet = formatProbeSnippet(text);
			const statusInfo =
				response.status === 401 || response.status === 403
					? "Authentication failed"
					: `HTTP ${response.status}`;

			return {
				error: `${statusInfo}: ${snippet}`,
				latencyMs: elapsed,
				model: config.model,
				provider: config.provider,
				status: "fail",
			};
		}

		const result = parseProbeResponse(text, config.provider);

		if (!result.ok) {
			return {
				error: `Unexpected response: ${result.snippet ?? "<unknown>"}`,
				latencyMs: elapsed,
				model: config.model,
				provider: config.provider,
				status: "fail",
			};
		}

		return {
			latencyMs: elapsed,
			model: result.modelName ?? config.model,
			provider: config.provider,
			status: "ok",
		};
	} catch (error) {
		const elapsed = Date.now() - start;
		const message =
			error instanceof Error
				? error.name === "AbortError"
					? "Request timed out after 15 seconds."
					: error.message
				: "Unknown error.";

		return {
			error: message,
			latencyMs: elapsed,
			model: config.model,
			provider: config.provider,
			status: "fail",
		};
	} finally {
		clearTimeout(timer);
	}
}

function buildProbeUrl(config: EffectiveConfig, apiKey?: string): string {
	const base = (config.baseURL ?? "").replace(/\/+$/u, "");

	switch (config.provider) {
		case "openai-compatible":
			return `${base}/chat/completions`;
		case "openai-responses":
			return `${base}/responses`;
		case "anthropic-messages":
			return `${base}/messages`;
		case "google-gemini": {
			const model = encodeURIComponent(config.model);
			const keyParam = apiKey ? `?key=${encodeURIComponent(apiKey)}` : "";
			return `${base}/models/${model}:generateContent${keyParam}`;
		}
		default:
			return `${base}/chat/completions`;
	}
}

function buildProbeBody(config: EffectiveConfig): Record<string, unknown> {
	switch (config.provider) {
		case "openai-compatible":
		case "openai-responses":
			return {
				max_tokens: 1,
				messages: [{ content: "ok", role: "user" }],
				model: config.model,
				temperature: 0,
			};
		case "anthropic-messages":
			return {
				max_tokens: 1,
				messages: [{ content: "ok", role: "user" }],
				model: config.model,
			};
		case "google-gemini":
			return {
				contents: [{ parts: [{ text: "ok" }] }],
				generationConfig: { maxOutputTokens: 1, temperature: 0 },
			};
		default:
			return {
				max_tokens: 1,
				messages: [{ content: "ok", role: "user" }],
				model: config.model,
			};
	}
}

function applyProbeAuth(
	headers: Record<string, string>,
	config: EffectiveConfig,
): void {
	switch (config.provider) {
		case "openai-compatible":
		case "openai-responses":
			headers["Authorization"] = `Bearer ${config.apiKey}`;
			break;
		case "anthropic-messages":
			headers["x-api-key"] = config.apiKey ?? "";
			headers["anthropic-version"] = "2023-06-01";
			break;
		case "google-gemini":
			// API key is appended as query parameter
			break;
	}
}

function parseProbeResponse(
	text: string,
	provider: string,
): { ok: boolean; modelName?: string; snippet?: string } {
	const snippet = formatProbeSnippet(text);

	try {
		const data = JSON.parse(text) as Record<string, unknown>;

		switch (provider) {
			case "openai-compatible":
			case "openai-responses": {
				const choices = data.choices as
					| Array<Record<string, unknown>>
					| undefined;
				if (!Array.isArray(choices) || choices.length === 0) {
					return { ok: false, snippet };
				}
				return { modelName: String(data.model ?? ""), ok: true };
			}
			case "anthropic-messages": {
				const content = data.content as
					| Array<Record<string, unknown>>
					| undefined;
				if (!Array.isArray(content) || content.length === 0) {
					return { ok: false, snippet };
				}
				return { modelName: String(data.model ?? ""), ok: true };
			}
			case "google-gemini": {
				const candidates = data.candidates as
					| Array<Record<string, unknown>>
					| undefined;
				if (!Array.isArray(candidates) || candidates.length === 0) {
					return { ok: false, snippet };
				}
				return {
					modelName: String(
						(data as Record<string, unknown>).modelVersion ?? "",
					),
					ok: true,
				};
			}
			default:
				return { ok: false, snippet };
		}
	} catch {
		return { ok: false, snippet };
	}
}

function formatProbeSnippet(text: string): string {
	const trimmed = text.replace(/\s+/gu, " ").trim();

	if (trimmed.length === 0) {
		return "<empty>";
	}

	return trimmed.length > 200 ? `${trimmed.slice(0, 200)}…` : trimmed;
}
