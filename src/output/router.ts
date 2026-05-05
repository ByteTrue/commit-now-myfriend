export interface CliWriteStream {
  write(chunk: string | Uint8Array): unknown;
}

export interface OutputRouterOptions {
  json: boolean;
  stdout?: CliWriteStream;
  stderr?: CliWriteStream;
}

export interface RoutedJsonResult {
  ok: boolean;
  status: string;
  command: string;
  message: string;
  dryRun: boolean;
}

export type JsonPayload = RoutedJsonResult | Record<string, unknown>;

type HumanTarget = "stdout" | "stderr";

const encoder = new TextEncoder();

function writeChunk(stream: CliWriteStream, message: string): void {
  stream.write(message);
}

function normalizeMessage(message: string): string {
  return message.endsWith("\n") ? message : `${message}\n`;
}

export function createOutputRouter({
  json,
  stdout = process.stdout,
  stderr = process.stderr
}: OutputRouterOptions) {
  return {
    isJson: json,
    writeHuman(message: string, target: HumanTarget = "stderr"): void {
      const stream = target === "stdout" ? stdout : stderr;
      writeChunk(stream, normalizeMessage(message));
    },
    writeJson(payload: JsonPayload): void {
      writeChunk(stdout, `${JSON.stringify(payload)}\n`);
    },
    writeStructured(
      payload: JsonPayload,
      message: string,
      target: HumanTarget = "stderr"
    ): void {
      if (json) {
        this.writeJson(payload);
        return;
      }

      this.writeHuman(message, target);
    },
    writeEncodedHuman(message: string, target: HumanTarget = "stderr"): void {
      const stream = target === "stdout" ? stdout : stderr;
      stream.write(encoder.encode(normalizeMessage(message)));
    }
  };
}
