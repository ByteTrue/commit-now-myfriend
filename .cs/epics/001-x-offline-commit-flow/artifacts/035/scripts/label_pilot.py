#!/usr/bin/env python3
"""Generate exactly one frozen local-teacher output per #35 pilot case."""
from __future__ import annotations

import argparse
import hashlib
import json
import subprocess
import time
import urllib.request
from pathlib import Path

ARTIFACT = Path(__file__).resolve().parents[1]
ROOT = Path("/tmp/cnm-pierce35")
SERVER_ROOT = "http://127.0.0.1:63286"
SERVER_URL = f"{SERVER_ROOT}/v1/chat/completions"
MODEL = ROOT / "model/qwen2.5-coder-14b-instruct-q6_k.gguf"
RUNTIME = Path("/opt/homebrew/bin/llama-server")
OPENER = urllib.request.build_opener(urllib.request.ProxyHandler({}))


def digest(path: Path) -> str:
    value = hashlib.sha256()
    with path.open("rb") as stream:
        for chunk in iter(lambda: stream.read(8 * 1024 * 1024), b""):
            value.update(chunk)
    return value.hexdigest()


def load_jsonl(path: Path) -> list[dict]:
    return [json.loads(line) for line in path.read_text().splitlines() if line]


def main() -> None:
    parser = argparse.ArgumentParser()
    parser.add_argument("--pilot", type=Path, default=ROOT / "data/pilot-200.jsonl")
    parser.add_argument("--manifest", type=Path, default=ARTIFACT / "pilot-manifest.json")
    parser.add_argument("--teacher-manifest", type=Path, default=ARTIFACT / "teacher-manifest.json")
    parser.add_argument("--freeze-manifest", type=Path, default=ARTIFACT / "freeze-manifest.json")
    parser.add_argument("--prompt", type=Path, default=ARTIFACT / "teacher-prompt.txt")
    parser.add_argument("--schema", type=Path, default=ARTIFACT / "teacher-schema.json")
    parser.add_argument("--out", type=Path, default=ROOT / "labels")
    parser.add_argument("--expected-server-pid", type=int)
    parser.add_argument("--indices", help="comma-separated frozen case indices; default all")
    parser.add_argument("--over-limit-check", action="store_true", help="exercise the frozen pre-inference rejection path")
    args = parser.parse_args()

    manifest = json.loads(args.manifest.read_text())
    teacher = json.loads(args.teacher_manifest.read_text())
    frozen = json.loads(args.freeze_manifest.read_text())
    if manifest["status"] != "frozen_pre_output" or teacher["status"] != "frozen_pre_output" or frozen["status"] != "frozen_pre_output":
        raise SystemExit("manifests are not frozen pre-output")
    for name, expected in frozen["files"].items():
        path = Path(name)
        if not path.is_file() or path.stat().st_size != expected["bytes"] or digest(path) != expected["sha256"]:
            raise SystemExit(f"frozen input mismatch: {path}")
    if digest(args.pilot) != manifest["outputs"]["pilot_200_sha256"]:
        raise SystemExit("pilot hash mismatch")
    if digest(args.prompt) != manifest["inputs"]["prompt_sha256"]:
        raise SystemExit("prompt hash mismatch")
    if digest(args.schema) != manifest["inputs"]["schema_sha256"]:
        raise SystemExit("schema hash mismatch")
    if args.over_limit_check:
        if args.expected_server_pid is not None or args.indices:
            raise SystemExit("over-limit check cannot select inference options")
        over_path = ROOT / "data/over-limit.jsonl"
        if digest(over_path) != manifest["outputs"]["over_limit_sha256"]:
            raise SystemExit("over-limit input hash mismatch")
        over = load_jsonl(over_path)
        pilot_families = {row["family"] for row in load_jsonl(args.pilot)}
        if (
            len(over) != 1
            or over[0]["input_tokens"] <= manifest["limits"]["max_input_tokens"]
            or over[0]["family"] in pilot_families
            or over[0].get("expected") != "reject_before_inference"
        ):
            raise SystemExit("frozen over-limit case is invalid")
        if any(args.out.glob("over-limit-*.json")):
            raise SystemExit("over-limit output slot unexpectedly exists")
        print(json.dumps({"status": "rejected_before_inference", "input_tokens": over[0]["input_tokens"]}))
        raise SystemExit(42)
    if args.expected_server_pid is None:
        raise SystemExit("inference requires the owned teacher PID")
    listener = subprocess.check_output(
        ["lsof", "-nP", "-iTCP:63286", "-sTCP:LISTEN", "-t"], text=True
    ).split()
    if listener != [str(args.expected_server_pid)]:
        raise SystemExit(f"unexpected teacher listener owner: {listener}")
    command = subprocess.check_output(["ps", "-o", "command=", "-p", str(args.expected_server_pid)], text=True).strip()
    if str(RUNTIME) not in command or str(MODEL) not in command or "--host 127.0.0.1" not in command:
        raise SystemExit("teacher server command does not bind the frozen local runtime and model")
    with OPENER.open(f"{SERVER_ROOT}/v1/models", timeout=5) as response:
        models = json.loads(response.read())
    if [item.get("id") for item in models.get("data", [])] != ["local-teacher"]:
        raise SystemExit(f"unexpected teacher model identity: {models}")

    rows = load_jsonl(args.pilot)
    chosen = set(range(len(rows))) if not args.indices else {int(value) for value in args.indices.split(",")}
    if not chosen <= set(range(len(rows))):
        raise SystemExit("unknown case index")
    schema = json.loads(args.schema.read_text())
    prompt = args.prompt.read_text().strip()
    decode = teacher["decode"]
    args.out.mkdir(parents=True, exist_ok=True)

    for row in rows:
        index = row["index"]
        if index not in chosen:
            continue
        target = args.out / f"{index:03d}.json"
        if target.exists():
            saved = json.loads(target.read_text())
            if saved.get("family") != row["family"] or saved.get("diff_sha256") != row["diff_sha256"]:
                raise SystemExit(f"existing output identity mismatch: {target}")
            print(f"skip frozen output {index:03d}", flush=True)
            continue
        user = f"BODY_POLICY: {row['body_policy']}\n\nCOMPLETE_DIFF:\n{row['diff']}"
        payload = {
            "model": "local-teacher",
            "messages": [{"role": "system", "content": prompt}, {"role": "user", "content": user}],
            "temperature": decode["temperature"],
            "top_p": decode["top_p"],
            "top_k": decode["top_k"],
            "seed": decode["seed"],
            "max_tokens": decode["max_tokens"],
            "stream": False,
            "response_format": {
                "type": "json_schema",
                "json_schema": {"name": "teacher_record", "strict": True, "schema": schema},
            },
        }
        request = urllib.request.Request(
            SERVER_URL,
            data=json.dumps(payload, ensure_ascii=False).encode(),
            headers={"Content-Type": "application/json"},
        )
        started = time.monotonic()
        with OPENER.open(request, timeout=600) as response:
            raw = response.read()
        latency = time.monotonic() - started
        envelope = json.loads(raw)
        content = envelope["choices"][0]["message"]["content"]
        result = {
            "index": index,
            "family": row["family"],
            "diff_sha256": row["diff_sha256"],
            "body_policy": row["body_policy"],
            "server_pid": args.expected_server_pid,
            "content": content,
            "latency_seconds": round(latency, 6),
            "usage": envelope.get("usage"),
            "finish_reason": envelope["choices"][0].get("finish_reason"),
        }
        temporary = target.with_suffix(".tmp")
        temporary.write_text(json.dumps(result, ensure_ascii=False, indent=2) + "\n")
        temporary.replace(target)
        print(json.dumps({"index": index, "latency_seconds": result["latency_seconds"], "finish_reason": result["finish_reason"]}), flush=True)


if __name__ == "__main__":
    main()
