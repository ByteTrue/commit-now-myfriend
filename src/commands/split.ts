import { confirm, isCancel, note, select, text } from "@clack/prompts";
import { Command, CommanderError } from "commander";
import { execa } from "execa";

import {
	resolveEffectiveConfig,
	type EffectiveConfig,
	type PromptStyle,
} from "../config/index.js";
import { EXIT_CODES, type CliWriteStream } from "../output/index.js";
import { ProviderError } from "../providers/index.js";

// ── Types ──────────────────────────────────────────────────────────────────────

interface SplitGroupFile {
	path: string;
}

interface SplitGroup {
	files: SplitGroupFile[];
	message: string;
}

interface SplitRuntime {
	cwd?: string;
	env?: NodeJS.ProcessEnv;
	stderr?: CliWriteStream;
	stdout?: CliWriteStream;
}

interface ProviderCallResult {
	text: string;
}

// ── Prompt ──────────────────────────────────────────────────────────────────────

function buildSplitPrompt(
	files: SplitGroupFile[],
	diff: string,
	style: PromptStyle,
	customPrompt?: string,
): { system: string; user: string } {
	const styleInstruction: string = (() => {
		switch (style) {
			case "auto":
			case "conventional":
				return "Use Conventional Commits: type(scope)?: subject with an optional body.";
			case "angular":
				return "Use Angular commit format: type(scope): subject.";
			case "google":
				return "Use Google-style: a short imperative subject line, optional body after blank line.";
			case "atom":
				return "Use Atom-style: concise imperative subject, body optional.";
			case "plain":
				return "Use plain natural-language commit messages without strict prefixes.";
			case "custom":
				return (
					customPrompt?.trim() ??
					"Use the style implied by the custom instructions below."
				);
			default:
				return `Use "${style}" style.`;
		}
	})();

	const customSection = customPrompt?.trim()
		? `\nAdditional user guidance:\n${customPrompt}`
		: "";

	return {
		system: [
			"You are a git commit message generator. Group the staged changes into logical independent commits.",
			"",
			"Rules:",
			"- Each group must be a self-contained, atomic commit.",
			"- Every staged file must appear in exactly one group.",
			"- Do not group unrelated changes together.",
			"- Return ONLY a valid JSON array. No markdown fences, no labels, no explanation.",
			"",
			"JSON schema for the response:",
			"[",
			'  { "files": ["path/to/file1.ts", "path/to/file2.ts"], "message": "type(scope): subject" }',
			"]",
			"",
			`Commit message style: ${styleInstruction}${customSection}`,
			"",
			"Remember: only output the JSON array, nothing else.",
		].join("\n"),
		user: [
			"Changed files:",
			...files.map((f) => `- ${f.path}`),
			"",
			"Diff:",
			diff,
		].join("\n"),
	};
}

// ── Response parsing ────────────────────────────────────────────────────────────

function parseSplitResponse(raw: string, allFiles: Set<string>): SplitGroup[] {
	const cleaned = raw
		.replace(/^```(?:json)?\s*\n?/u, "")
		.replace(/\n?```\s*$/u, "")
		.trim();

	if (cleaned.length === 0) {
		throw new ProviderError({
			code: "empty_output",
			provider: "openai-compatible",
			message: "The provider returned an empty response.",
		});
	}

	let parsed: unknown;
	try {
		parsed = JSON.parse(cleaned);
	} catch {
		throw new ProviderError({
			code: "malformed_output",
			provider: "openai-compatible",
			message: "The provider did not return valid JSON for the split request.",
		});
	}

	if (!Array.isArray(parsed)) {
		throw new ProviderError({
			code: "malformed_output",
			provider: "openai-compatible",
			message:
				"Expected a JSON array from the provider but got something else.",
		});
	}

	if (parsed.length === 0) {
		throw new ProviderError({
			code: "empty_output",
			provider: "openai-compatible",
			message: "The provider returned an empty grouping (no groups).",
		});
	}

	const groups = parsed.map((entry: unknown, index: number): SplitGroup => {
		if (typeof entry !== "object" || entry === null) {
			throw new ProviderError({
				code: "malformed_output",
				provider: "openai-compatible",
				message: `Group at index ${index} is not an object.`,
			});
		}

		const obj = entry as Record<string, unknown>;
		const messageRaw = obj.message;

		if (typeof messageRaw !== "string" || messageRaw.trim().length === 0) {
			throw new ProviderError({
				code: "malformed_output",
				provider: "openai-compatible",
				message: `Group at index ${index} has no commit message.`,
			});
		}

		const filesRaw = obj.files;

		if (!Array.isArray(filesRaw) || filesRaw.length === 0) {
			throw new ProviderError({
				code: "malformed_output",
				provider: "openai-compatible",
				message: `Group at index ${index} has no files.`,
			});
		}

		return {
			files: filesRaw.map((f: unknown) => {
				if (typeof f !== "string") {
					throw new ProviderError({
						code: "malformed_output",
						provider: "openai-compatible",
						message: `Group at index ${index} has a non-string file entry.`,
					});
				}
				return { path: f };
			}),
			message: messageRaw.trim(),
		};
	});

	// Verify all staged files are covered
	const coveredFiles = new Set<string>();
	for (const group of groups) {
		for (const file of group.files) {
			if (!allFiles.has(file.path)) {
				throw new ProviderError({
					code: "malformed_output",
					provider: "openai-compatible",
					message: `Group references "${file.path}" which is not in the staged changes.`,
				});
			}
			coveredFiles.add(file.path);
		}
	}

	const missingFiles = [...allFiles].filter((f) => !coveredFiles.has(f));
	if (missingFiles.length > 0) {
		throw new ProviderError({
			code: "malformed_output",
			provider: "openai-compatible",
			message: `The following staged files were not assigned to any group: ${missingFiles.join(", ")}`,
		});
	}

	return groups;
}

// ── Provider HTTP call ──────────────────────────────────────────────────────────

const SPLIT_TIMEOUT_MS = 60_000;

async function callSplitProvider(
	config: EffectiveConfig,
	system: string,
	userContent: string,
): Promise<ProviderCallResult> {
	const controller = new AbortController();
	const timer = setTimeout(() => controller.abort(), SPLIT_TIMEOUT_MS);

	try {
		const base = (config.baseURL ?? "").replace(/\/+$/u, "");
		let url: string;
		const headers: Record<string, string> = {
			"Content-Type": "application/json",
		};
		let body: Record<string, unknown>;

		switch (config.provider) {
			case "openai-compatible":
			case "openai-responses": {
				url = `${base}/chat/completions`;
				headers["Authorization"] = `Bearer ${config.apiKey}`;
				body = {
					model: config.model,
					messages: [
						{ role: "system", content: system },
						{ role: "user", content: userContent },
					],
					temperature: 0.2,
					max_tokens: 8192,
				};
				break;
			}
			case "anthropic-messages": {
				url = `${base}/messages`;
				headers["x-api-key"] = config.apiKey ?? "";
				headers["anthropic-version"] = "2023-06-01";
				body = {
					model: config.model,
					system,
					messages: [{ role: "user", content: userContent }],
					max_tokens: 8192,
				};
				break;
			}
			case "google-gemini": {
				const model = encodeURIComponent(config.model);
				const keyParam = config.apiKey
					? `?key=${encodeURIComponent(config.apiKey)}`
					: "";
				url = `${base}/models/${model}:generateContent${keyParam}`;
				body = {
					contents: [
						{
							role: "user",
							parts: [{ text: `${system}\n\n${userContent}` }],
						},
					],
					generationConfig: {
						temperature: 0.2,
						maxOutputTokens: 8192,
					},
				};
				break;
			}
			default:
				throw new Error(`Unsupported provider: ${config.provider}`);
		}

		const response = await fetch(url, {
			method: "POST",
			headers,
			body: JSON.stringify(body),
			signal: controller.signal,
		});

		const text = await response.text().catch(() => "");

		if (!response.ok) {
			const snippet = text.replace(/\s+/gu, " ").trim().slice(0, 300);
			throw new ProviderError({
				code: "provider_failure",
				provider: config.provider,
				message: `Provider returned HTTP ${response.status}: ${snippet || "<empty>"}`,
			});
		}

		const resultText = extractProviderText(text, config.provider);

		if (resultText.length === 0) {
			throw new ProviderError({
				code: "empty_output",
				provider: config.provider,
				message: "Provider returned an empty response.",
			});
		}

		return { text: resultText };
	} catch (error) {
		if (error instanceof ProviderError) {
			throw error;
		}
		if (error instanceof Error && error.name === "AbortError") {
			throw new ProviderError({
				code: "provider_failure",
				provider: config.provider,
				message: "Request timed out after 60 seconds.",
			});
		}
		throw new ProviderError({
			code: "provider_failure",
			provider: config.provider,
			message: error instanceof Error ? error.message : "Unknown error.",
		});
	} finally {
		clearTimeout(timer);
	}
}

function extractProviderText(text: string, provider: string): string {
	try {
		const data = JSON.parse(text) as Record<string, unknown>;

		switch (provider) {
			case "openai-compatible":
			case "openai-responses": {
				const choices = data.choices as
					| Array<Record<string, unknown>>
					| undefined;
				const firstChoice = choices?.[0];
				const message =
					firstChoice && typeof firstChoice === "object"
						? (firstChoice as Record<string, unknown>).message
						: undefined;
				const content =
					message && typeof message === "object"
						? (message as Record<string, unknown>).content
						: undefined;
				return typeof content === "string" ? content : "";
			}
			case "anthropic-messages": {
				const contentBlocks = data.content as
					| Array<Record<string, unknown>>
					| undefined;
				const textBlock = contentBlocks?.find((block) => block.type === "text");
				return typeof textBlock?.text === "string" ? textBlock.text : "";
			}
			case "google-gemini": {
				const candidates = data.candidates as
					| Array<Record<string, unknown>>
					| undefined;
				const firstCandidate = candidates?.[0];
				const candidateContent =
					firstCandidate && typeof firstCandidate === "object"
						? (firstCandidate as Record<string, unknown>).content
						: undefined;
				const parts =
					candidateContent && typeof candidateContent === "object"
						? (candidateContent as Record<string, unknown>).parts
						: undefined;
				const textParts = Array.isArray(parts)
					? parts.filter(
							(p): p is Record<string, unknown> =>
								typeof p === "object" && p !== null,
						)
					: [];
				const textPart = textParts.find((p) => "text" in p);
				return typeof textPart?.text === "string" ? textPart.text : "";
			}
			default:
				return "";
		}
	} catch {
		return "";
	}
}

// ── Git operations ──────────────────────────────────────────────────────────────

async function stageFiles(
	cwd: string,
	files: string[],
	env?: NodeJS.ProcessEnv,
): Promise<void> {
	const result = await execa("git", ["add", "--", ...files], {
		cwd,
		env,
		reject: false,
		stdin: "ignore",
		stderr: "pipe",
		stdout: "pipe",
	});

	if (result.exitCode !== 0) {
		throw new Error(
			result.stderr.trim() || `git add failed (exit ${result.exitCode}).`,
		);
	}
}

async function runGitCommit(
	cwd: string,
	message: string,
	env?: NodeJS.ProcessEnv,
): Promise<void> {
	const result = await execa("git", ["commit", "-F", "-"], {
		cwd,
		env,
		input: message,
		reject: false,
		stdin: "pipe",
		stderr: "pipe",
		stdout: "pipe",
	});

	if (result.exitCode !== 0) {
		throw new Error(
			result.stderr.trim() || `git commit failed (exit ${result.exitCode}).`,
		);
	}
}

async function unstageAll(cwd: string, env?: NodeJS.ProcessEnv): Promise<void> {
	const result = await execa("git", ["reset", "HEAD", "--quiet"], {
		cwd,
		env,
		reject: false,
		stdin: "ignore",
		stderr: "pipe",
		stdout: "pipe",
	});

	if (result.exitCode !== 0) {
		throw new Error(
			result.stderr.trim() || `git reset failed (exit ${result.exitCode}).`,
		);
	}
}

// ── UI ──────────────────────────────────────────────────────────────────────────

function renderGroups(groups: SplitGroup[]): string {
	const lines: string[] = [];

	lines.push(
		`Split into ${groups.length} group${groups.length > 1 ? "s" : ""}:`,
	);
	lines.push("");

	for (let index = 0; index < groups.length; index += 1) {
		const group = groups[index]!;
		lines.push(`[${index + 1}/${groups.length}] ${group.message}`);
		for (const file of group.files) {
			lines.push(`      ${file.path}`);
		}
		lines.push("");
	}

	return lines.join("\n").trimEnd();
}

async function confirmGroupAction(
	group: SplitGroup,
	index: number,
	total: number,
): Promise<"confirm" | "edit" | "skip"> {
	const action = await select({
		message: `Commit group ${index + 1}/${total}: ${group.message}`,
		options: [
			{ label: "Confirm and commit", value: "confirm" },
			{ label: "Edit message", value: "edit" },
			{ label: "Skip this group", value: "skip" },
		],
	});

	if (isCancel(action)) {
		return "skip";
	}

	return action as "confirm" | "edit" | "skip";
}

async function editGroupMessage(current: string): Promise<string | null> {
	const value = await text({
		initialValue: current,
		message: "Edit the commit message for this group.",
	});

	if (isCancel(value)) {
		return null;
	}

	return value.trim();
}

// ── Main action ─────────────────────────────────────────────────────────────────

async function runSplit(
	cwd: string,
	env: NodeJS.ProcessEnv,
	config: EffectiveConfig,
	files: SplitGroupFile[],
	diff: string,
): Promise<void> {
	// Build and send the split prompt
	const prompt = buildSplitPrompt(
		files,
		diff,
		config.promptStyle ?? "conventional",
		config.customPrompt,
	);

	note("Sending staged diff to provider for grouping…", "cnm split");

	const rawResponse = await callSplitProvider(
		config,
		prompt.system,
		prompt.user,
	);
	const allFilePaths = new Set(files.map((f) => f.path));
	const groups = parseSplitResponse(rawResponse.text, allFilePaths);

	// Show groups
	note(renderGroups(groups), "cnm split");

	if (groups.length <= 1) {
		note(
			"AI detected only one logical change. Use `cnm` directly instead of split.",
			"cnm split",
		);
		return;
	}

	const startConfirmed = await confirm({
		initialValue: true,
		message: `Start committing these ${groups.length} groups one by one?`,
	});

	if (isCancel(startConfirmed) || !startConfirmed) {
		note("Split cancelled. No commits were created.", "cnm split");
		return;
	}

	// Unstage everything first
	await unstageAll(cwd, env);

	let committed = 0;

	for (let index = 0; index < groups.length; index += 1) {
		const group = groups[index]!;
		let message = group.message;

		for (;;) {
			const action = await confirmGroupAction(group, index, groups.length);

			if (action === "skip") {
				break;
			}

			if (action === "edit") {
				const edited = await editGroupMessage(message);
				if (edited === null) {
					continue;
				}
				message = edited;
			}

			try {
				await stageFiles(
					cwd,
					group.files.map((f) => f.path),
					env,
				);
				await runGitCommit(cwd, message, env);
				committed += 1;
				break;
			} catch (error) {
				const errMessage =
					error instanceof Error ? error.message : "Unknown error.";
				const retry = await confirm({
					initialValue: false,
					message: `Commit failed: ${errMessage}. Retry?`,
				});

				if (isCancel(retry) || !retry) {
					note("Aborting split. Remaining files are unstaged.", "cnm split");
					return;
				}
			}
		}
	}

	note(
		`Done. ${committed} commit${committed !== 1 ? "s" : ""} created.`,
		"cnm split",
	);
}

// ── Command ─────────────────────────────────────────────────────────────────────

export function createSplitCommand(runtime: SplitRuntime = {}): Command {
	return new Command("split")
		.description("Split staged changes into multiple logical commits.")
		.action(async () => {
			const cwd = runtime.cwd ?? process.cwd();
			const env = runtime.env ?? process.env;

			// Inspect git repo
			const { inspectGitRepository } = await import("../git/index.js");
			const inspection = await inspectGitRepository({ cwd, env });

			if (!inspection.repository.isRepository) {
				throw new CommanderError(
					EXIT_CODES.ERROR,
					"cnm.split.not_git",
					"Not a git repository.",
				);
			}

			if (!inspection.hasStagedChanges) {
				throw new CommanderError(
					EXIT_CODES.ERROR,
					"cnm.split.no_staged",
					"No staged changes found. Stage files first with `git add`.",
				);
			}

			const stagedFileViews = inspection.stagedFiles.map((f) => ({
				path: f.path,
			}));
			const stagedDiff = inspection.stagedDiff;

			// Resolve config
			const resolvedConfig = await resolveEffectiveConfig({ cwd, env });
			const config = resolvedConfig.values;

			if (!config.apiKey || config.apiKey.trim().length === 0) {
				throw new CommanderError(
					EXIT_CODES.ERROR,
					"cnm.split.no_api_key",
					"No API key configured. Run `cnm init` first.",
				);
			}

			try {
				await runSplit(cwd, env, config, stagedFileViews, stagedDiff);
			} catch (error) {
				if (error instanceof CommanderError) {
					throw error;
				}

				const message =
					error instanceof ProviderError
						? error.message
						: error instanceof Error
							? error.message
							: "Split failed.";

				throw new CommanderError(EXIT_CODES.ERROR, "cnm.split.failed", message);
			}
		});
}
