#!/usr/bin/env python3
"""Own the exact frozen local teacher process for #35 smoke or full labeling."""
from __future__ import annotations

import argparse
import hashlib
import json
import subprocess
import sys
import time
import urllib.error
import urllib.request
from collections import Counter
from pathlib import Path

ARTIFACT = Path(__file__).resolve().parents[1]
ROOT = Path("/tmp/cnm-pierce35")
RUNTIME = Path("/opt/homebrew/bin/llama-server")
MODEL = ROOT / "model/qwen2.5-coder-14b-instruct-q6_k.gguf"
SERVER_ROOT = "http://127.0.0.1:63286"
OPENER = urllib.request.build_opener(urllib.request.ProxyHandler({}))


def digest(path: Path) -> str:
    value = hashlib.sha256()
    with path.open("rb") as stream:
        for chunk in iter(lambda: stream.read(8 * 1024 * 1024), b""):
            value.update(chunk)
    return value.hexdigest()


def frozen() -> dict:
    manifest = json.loads((ARTIFACT / "freeze-manifest.json").read_text())
    if manifest["status"] != "frozen_pre_output":
        raise SystemExit("inputs not frozen")
    for name, expected in manifest["files"].items():
        path = Path(name)
        if not path.is_file() or path.stat().st_size != expected["bytes"] or digest(path) != expected["sha256"]:
            raise SystemExit(f"frozen input mismatch: {path}")
    return manifest


def listener_pids() -> list[int]:
    result = subprocess.run(["lsof", "-nP", "-iTCP:63286", "-sTCP:LISTEN", "-t"], text=True, capture_output=True)
    return sorted({int(value) for value in result.stdout.split()})


def get_json(path: str, timeout: float = 5) -> dict:
    with OPENER.open(f"{SERVER_ROOT}{path}", timeout=timeout) as response:
        return json.loads(response.read())


def main() -> None:
    parser = argparse.ArgumentParser()
    parser.add_argument("mode", choices=("smoke", "full"))
    parser.add_argument("--index", type=int)
    args = parser.parse_args()
    if (args.mode == "smoke") != (args.index is not None):
        raise SystemExit("smoke requires exactly one --index; full accepts none")
    subprocess.check_call([sys.executable, str(ARTIFACT / "scripts/verify_environment.py")])
    frozen_manifest = frozen()
    if listener_pids():
        raise SystemExit("port 63286 already has a listener; refusing an unowned teacher")

    command = [
        str(RUNTIME), "--model", str(MODEL), "--alias", "local-teacher",
        "--host", "127.0.0.1", "--port", "63286", "--ctx-size", "9216",
        "--parallel", "1", "--n-gpu-layers", "99", "--threads", "12",
        "--flash-attn", "on", "--no-ui",
    ]
    logs = ROOT / "logs"
    logs.mkdir(parents=True, exist_ok=True)
    server_log = logs / f"teacher-server-{args.mode}.log"
    provenance_path = logs / f"teacher-server-{args.mode}-provenance.json"
    started = time.time()
    status = "fail"
    error = None
    child = None
    with server_log.open("w") as log:
        process = subprocess.Popen(command, stdout=log, stderr=subprocess.STDOUT, text=True)
        try:
            deadline = time.monotonic() + 240
            while time.monotonic() < deadline:
                if process.poll() is not None:
                    raise RuntimeError(f"teacher server exited during startup: {process.returncode}")
                try:
                    health = get_json("/health")
                    if health.get("status") == "ok":
                        break
                except (urllib.error.URLError, json.JSONDecodeError):
                    pass
                time.sleep(1)
            else:
                raise RuntimeError("teacher server did not become healthy")
            if listener_pids() != [process.pid]:
                raise RuntimeError(f"teacher listener is not owned by started process: {listener_pids()}")
            models = get_json("/v1/models")
            if [item.get("id") for item in models.get("data", [])] != ["local-teacher"]:
                raise RuntimeError(f"unexpected served model: {models}")
            props = get_json("/props")
            served_path = props.get("model_path") or props.get("model", {}).get("path")
            if served_path is not None and Path(served_path).resolve() != MODEL.resolve():
                raise RuntimeError(f"unexpected served model path: {served_path}")
            if args.mode == "smoke":
                child_command = [sys.executable, str(ARTIFACT / "scripts/smoke.py"), "--server-pid", str(process.pid), "--index", str(args.index)]
            else:
                child_command = [sys.executable, str(ARTIFACT / "scripts/label_pilot.py"), "--expected-server-pid", str(process.pid)]
            child = subprocess.run(child_command, text=True, capture_output=True)
            if child.returncode != 0:
                raise RuntimeError(f"{args.mode} command failed: {child.stderr}")
            if args.mode == "full":
                validation = subprocess.run([sys.executable, str(ARTIFACT / "scripts/validate_labels.py")], text=True, capture_output=True)
                if validation.returncode != 0:
                    raise RuntimeError(f"full mechanical validation failed: {validation.stderr}")
            label_server_pids = Counter()
            for label_path in sorted((ROOT / "labels").glob("[0-9][0-9][0-9].json")):
                label = json.loads(label_path.read_text())
                if isinstance(label.get("server_pid"), int):
                    label_server_pids[label["server_pid"]] += 1
            status = "pass"
        except Exception as caught:  # preserve evidence before returning the failure
            error = repr(caught)
        finally:
            process.terminate()
            try:
                process.wait(timeout=20)
            except subprocess.TimeoutExpired:
                process.kill()
                process.wait()
    provenance = {
        "status": status,
        "mode": args.mode,
        "smoke_index": args.index,
        "local_only": True,
        "server_url": SERVER_ROOT,
        "pid": process.pid,
        "command": command,
        "listener_owner_verified": True if status == "pass" else listener_pids() == [process.pid],
        "runtime": frozen_manifest["files"][str(RUNTIME)],
        "model": frozen_manifest["files"][str(MODEL)],
        "models_response": models if "models" in locals() else None,
        "props_model_path": props.get("model_path") if "props" in locals() else None,
        "started_at": started,
        "finished_at": time.time(),
        "server_log_sha256": digest(server_log),
        "child_stdout": child.stdout if child else None,
        "child_stderr": child.stderr if child else None,
        "label_server_pid_counts": {str(key): value for key, value in sorted(label_server_pids.items())} if "label_server_pids" in locals() else None,
        "error": error,
    }
    provenance_path.write_text(json.dumps(provenance, ensure_ascii=False, indent=2) + "\n")
    print(json.dumps({"status": status, "mode": args.mode, "pid": process.pid, "provenance": str(provenance_path)}))
    if status != "pass":
        raise SystemExit(1)


if __name__ == "__main__":
    main()
