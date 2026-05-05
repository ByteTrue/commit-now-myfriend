import { Command, CommanderError } from "commander";

import { runDoctor, renderDoctorReport, type DoctorDependencies } from "../doctor/index.js";
import { createOutputRouter, EXIT_CODES, type CliWriteStream } from "../output/index.js";

interface GlobalCliOptions {
  json?: boolean;
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

export function createDoctorCommand(runtime: DoctorCommandRuntime = {}): Command {
  return new Command("doctor")
    .description("Run cnm environment diagnostics.")
    .action(async function (this: Command) {
      const router = createOutputRouter({
        json: resolveJsonOption(this),
        stderr: runtime.stderr,
        stdout: runtime.stdout
      });
      const report = await runDoctor({
        cwd: runtime.cwd ?? process.cwd(),
        dependencies: runtime.dependencies,
        env: runtime.env ?? process.env,
        nodeEngine: runtime.nodeEngine,
        nodeVersion: runtime.nodeVersion ?? process.version
      });

      if (router.isJson) {
        router.writeJson(report as unknown as Record<string, unknown>);
      } else {
        router.writeHuman(renderDoctorReport(report), "stdout");
      }

      if (!report.ok) {
        throw new CommanderError(EXIT_CODES.ERROR, "cnm.doctor.failed", "Doctor found configuration issues.");
      }
    });
}
