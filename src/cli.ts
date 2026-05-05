import { Command, CommanderError } from "commander";

import { createCommitAction, type CommitCommandRuntime } from "./commands/commit.js";
import { createConfigCommand, type CommandRuntime as ConfigCommandRuntime } from "./commands/config.js";
import { createDoctorCommand } from "./commands/doctor.js";
import { createInitCommand } from "./commands/init.js";
import {
  EXIT_CODES,
  type CliWriteStream,
  type ExitCode
} from "./output/index.js";

export interface BuildCliOptions {
  commitRuntime?: CommitCommandRuntime;
  configRuntime?: ConfigCommandRuntime;
  nodeEngine?: string;
  version: string;
  stdout?: CliWriteStream;
  stderr?: CliWriteStream;
}

export interface RunCliOptions extends BuildCliOptions {
  argv: string[];
}

const COMMAND_NAMES = new Set(["init", "config", "doctor"]);
const OPTIONS_WITH_VALUES = new Set([
  "--provider",
  "--model",
  "--base-url",
  "--prompt-style",
  "--custom-prompt"
]);

function applyExecutionOptions(command: Command): Command {
  return command
    .option("--dry-run", "Preview command execution without side effects.")
    .option("--json", "Emit JSON for supported command handlers.");
}

function applyRootOptions(command: Command): Command {
  return applyExecutionOptions(command)
    .option("--provider <provider>", "Override the AI provider for this commit workflow.")
    .option("--model <model>", "Override the AI model for this commit workflow.")
    .option("--base-url <baseUrl>", "Override the OpenAI-compatible base URL for this commit workflow.")
    .option("--prompt-style <promptStyle>", "Override the commit prompt style for this commit workflow.")
    .option("--custom-prompt <customPrompt>", "Override custom prompt instructions for this commit workflow.");
}

function writeError(stderr: CliWriteStream | undefined, message: string): void {
  const stream = stderr ?? process.stderr;

  stream.write(`${message}\n`);
}

function normalizeExitCode(error: CommanderError): ExitCode {
  if (error.exitCode === 0) {
    return EXIT_CODES.SUCCESS;
  }

  if (error.exitCode === EXIT_CODES.USER_CANCEL) {
    return EXIT_CODES.USER_CANCEL;
  }

  return EXIT_CODES.ERROR;
}

function isJsonHelpRequest(argv: string[]): boolean {
  const usesJson = argv.includes("--json");
  const usesHelp = argv.includes("--help") || argv.includes("-h");

  return usesJson && usesHelp;
}

function resolveRequestedCommand(argv: string[]): string | undefined {
  for (let index = 0; index < argv.length; index += 1) {
    const token = argv[index] ?? "";

    if (token === "--") {
      return undefined;
    }

    if (OPTIONS_WITH_VALUES.has(token)) {
      index += 1;
      continue;
    }

    if (!token.startsWith("-")) {
      return token;
    }
  }

  return undefined;
}

function isUnknownCommandRequest(argv: string[]): boolean {
  const requestedCommand = resolveRequestedCommand(argv);

  if (!requestedCommand) {
    return false;
  }

  return !COMMAND_NAMES.has(requestedCommand);
}

export function buildCli({ commitRuntime, configRuntime, nodeEngine, version, stdout, stderr }: BuildCliOptions): Command {
  const program = new Command();
  const streams = { stdout, stderr };
  const initCommand = createInitCommand({ ...configRuntime, ...streams });
  const configCommand = createConfigCommand({ ...configRuntime, ...streams });
  const doctorCommand = createDoctorCommand({ ...streams, nodeEngine });

  applyRootOptions(program)
    .name("cnm")
    .description("AI-assisted commit workflow CLI.")
    .version(version)
    .showHelpAfterError("\nRun cnm --help to inspect available commands.")
    .showSuggestionAfterError()
    .enablePositionalOptions()
    .addHelpText(
      "after",
      "\nExamples:\n  $ cnm\n  $ cnm init\n  $ cnm config\n  $ cnm doctor"
    )
    .configureOutput({
      writeOut: (message) => {
        (stdout ?? process.stdout).write(message);
      },
      writeErr: (message) => {
        (stderr ?? process.stderr).write(message);
      },
      outputError: (message, write) => {
        write(message);
      }
     })
     .exitOverride()
     .action(createCommitAction(commitRuntime, streams));

  applyExecutionOptions(initCommand);
  applyExecutionOptions(configCommand);
  applyExecutionOptions(doctorCommand);
  program.addCommand(initCommand);
  program.addCommand(configCommand);
  program.addCommand(doctorCommand);

  return program;
}

export async function runCli({ argv, commitRuntime, configRuntime, nodeEngine, version, stdout, stderr }: RunCliOptions): Promise<ExitCode> {
  if (isJsonHelpRequest(argv)) {
    writeError(stderr, "error: --json cannot be combined with --help.");
    return EXIT_CODES.ERROR;
  }

  if (isUnknownCommandRequest(argv)) {
    const requestedCommand = resolveRequestedCommand(argv);

    writeError(stderr, `error: unknown command '${requestedCommand ?? ""}'`);
    writeError(stderr, "Run cnm --help to inspect available commands.");
    return EXIT_CODES.ERROR;
  }

  const program = buildCli({ commitRuntime, configRuntime, nodeEngine, version, stdout, stderr });

  try {
    await program.parseAsync(argv, { from: "user" });
    return EXIT_CODES.SUCCESS;
  } catch (error) {
    if (error instanceof CommanderError) {
      return normalizeExitCode(error);
    }

    if (error instanceof Error) {
      writeError(stderr, `error: ${error.message}`);
      return EXIT_CODES.ERROR;
    }

    writeError(stderr, "error: Unexpected CLI failure.");
    return EXIT_CODES.ERROR;
  }
}
