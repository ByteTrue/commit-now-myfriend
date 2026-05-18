import { confirm, isCancel } from "@clack/prompts";
import { Command, CommanderError } from "commander";

import { resolveEffectiveConfig } from "../config/index.js";
import {
	runDoctor,
	renderDoctorReport,
	runProviderProbe,
	type DoctorDependencies,
	type DoctorProbeResult,
} from "../doctor/index.js";
import {
	createOutputRouter,
	EXIT_CODES,
	type CliWriteStream,
} from "../output/index.js";

interface GlobalCliOptions {
	json?: boolean;
	probe?: boolean;
}

export interface DoctorCommandRuntime {
	cwd?: string;
	dependencies?: Partial<DoctorDependencies>;
	env?: NodeJS.ProcessEnv;
	nodeEngine?: string;
	nodeVersion?: string;
	stderr?: CliWriteStream;
	stdout?: CliWriteStream;
}

function resolveJsonOption(command: Command): boolean {
	return Boolean(command.optsWithGlobals<GlobalCliOptions>().json);
}

function resolveProbeOption(command: Command): boolean {
	return Boolean(command.optsWithGlobals<GlobalCliOptions>().probe);
}

function renderProbeResult(result: DoctorProbeResult): string {
	const lines: string[] = [];

	lines.push("");
	lines.push("Provider probe:");

	if (result.status === "skipped") {
		lines.push(`  status=skipped (${result.error ?? "No probe run."})`);
	} else if (result.status === "ok") {
		lines.push(`  status=PASS`);
		lines.push(`  provider=${result.provider}`);
		lines.push(`  model=${result.model}`);
		lines.push(`  latency=${result.latencyMs}ms`);
	} else {
		lines.push(`  status=FAIL`);
		lines.push(`  provider=${result.provider}`);
		lines.push(`  model=${result.model}`);

		if (result.latencyMs !== undefined) {
			lines.push(`  latency=${result.latencyMs}ms`);
		}

		lines.push(`  error=${result.error ?? "Unknown error."}`);
	}

	return lines.join("\n");
}

async function maybeRunProbe(
	runtime: DoctorCommandRuntime,
	isTty: boolean,
	probeFlag: boolean,
): Promise<DoctorProbeResult | null> {
	if (!probeFlag && !isTty) {
		return null;
	}

	const effectiveConfig = await resolveEffectiveConfig({
		cwd: runtime.cwd ?? process.cwd(),
		env: runtime.env ?? process.env,
	});
	const provider = effectiveConfig.values.provider;
	const model = effectiveConfig.values.model;
	const baseURL = effectiveConfig.values.baseURL ?? "(default)";

	if (!probeFlag) {
		const userConfirmed = await confirm({
			initialValue: false,
			message: `Send a test request to ${provider} (${baseURL}) to verify connectivity?`,
		});

		if (isCancel(userConfirmed) || !userConfirmed) {
			return { status: "skipped", provider, model };
		}
	}

	return runProviderProbe(effectiveConfig.values);
}

export function createDoctorCommand(
	runtime: DoctorCommandRuntime = {},
): Command {
	return new Command("doctor")
		.description("Run cnm environment diagnostics.")
		.option("--probe", "Send a test request to verify provider connectivity.")
		.action(async function (this: Command) {
			const router = createOutputRouter({
				json: resolveJsonOption(this),
				stderr: runtime.stderr,
				stdout: runtime.stdout,
			});
			const probeFlag = resolveProbeOption(this);
			const isTty =
				process.stdin.isTTY && process.stdout.isTTY && !router.isJson;
			const report = await runDoctor({
				cwd: runtime.cwd ?? process.cwd(),
				dependencies: runtime.dependencies,
				env: runtime.env ?? process.env,
				nodeEngine: runtime.nodeEngine,
				nodeVersion: runtime.nodeVersion ?? process.version,
			});
			const probeResult = await maybeRunProbe(runtime, isTty, probeFlag);

			if (router.isJson) {
				const output: Record<string, unknown> = {
					...(report as unknown as Record<string, unknown>),
				};

				if (probeResult) {
					output.probe = probeResult;
				}

				router.writeJson(output);
			} else {
				router.writeHuman(renderDoctorReport(report), "stdout");

				if (probeResult) {
					router.writeHuman(renderProbeResult(probeResult), "stdout");
				}
			}

			if (!report.ok) {
				throw new CommanderError(
					EXIT_CODES.ERROR,
					"cnm.doctor.failed",
					"Doctor found configuration issues.",
				);
			}
		});
}
