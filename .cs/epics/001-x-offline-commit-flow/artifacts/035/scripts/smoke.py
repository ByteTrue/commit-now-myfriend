#!/usr/bin/env python3
"""Run the frozen near-limit teacher case while recording bounded Mac resource evidence."""
from __future__ import annotations

import argparse
import hashlib
import json
import re
import subprocess
import sys
import threading
import time
from pathlib import Path

ARTIFACT = Path(__file__).resolve().parents[1]
ROOT = Path("/tmp/cnm-pierce35")


def vm_counts() -> dict[str, int]:
    output = subprocess.check_output(["vm_stat"], text=True)
    result = {}
    for line in output.splitlines():
        match = re.match(r"([^:]+):\s+(\d+)\.", line)
        if match:
            result[match.group(1).strip().lower().replace(" ", "_")] = int(match.group(2))
    return result


def free_percent() -> int:
    output = subprocess.check_output(["memory_pressure", "-Q"], text=True, stderr=subprocess.STDOUT)
    match = re.search(r"free percentage:\s*(\d+)%", output)
    if not match:
        raise RuntimeError("cannot parse memory pressure")
    return int(match.group(1))


def rss_bytes(pid: int) -> int:
    output = subprocess.check_output(["ps", "-o", "rss=", "-p", str(pid)], text=True).strip()
    return int(output) * 1024


def digest(path: Path) -> str:
    return hashlib.sha256(path.read_bytes()).hexdigest()


def main() -> None:
    parser = argparse.ArgumentParser()
    parser.add_argument("--server-pid", type=int, required=True)
    parser.add_argument("--index", type=int, required=True)
    parser.add_argument("--out", type=Path, default=ROOT / "logs/teacher-smoke.json")
    args = parser.parse_args()
    pilot = [json.loads(line) for line in (ROOT / "data/pilot-200.jsonl").read_text().splitlines() if line]
    if args.index not in range(len(pilot)) or not 7169 <= pilot[args.index]["input_tokens"] <= 8192:
        raise SystemExit("smoke index is not a frozen near-limit case")

    before = vm_counts()
    samples: list[dict] = []
    done = threading.Event()

    def monitor() -> None:
        while not done.is_set():
            try:
                samples.append({"at": time.time(), "server_rss_bytes": rss_bytes(args.server_pid), "free_percent": free_percent()})
            except (subprocess.CalledProcessError, ValueError, RuntimeError):
                samples.append({"at": time.time(), "sample_error": True})
            done.wait(1)

    thread = threading.Thread(target=monitor, daemon=True)
    thread.start()
    started = time.monotonic()
    command = [
        sys.executable,
        str(ARTIFACT / "scripts/label_pilot.py"),
        "--indices",
        str(args.index),
        "--expected-server-pid",
        str(args.server_pid),
    ]
    completed = subprocess.run(command, text=True, capture_output=True)
    validation_command = [
        sys.executable,
        str(ARTIFACT / "scripts/validate_labels.py"),
        "--indices",
        str(args.index),
        "--out",
        str(ROOT / "logs/smoke-validation.json"),
    ]
    validation = subprocess.run(validation_command, text=True, capture_output=True) if completed.returncode == 0 else None
    elapsed = time.monotonic() - started
    done.set()
    thread.join()
    after = vm_counts()
    pageout_delta = after.get("pageouts", 0) - before.get("pageouts", 0)
    swapout_delta = after.get("swapouts", 0) - before.get("swapouts", 0)
    rss_max = max((sample.get("server_rss_bytes", 0) for sample in samples), default=0)
    free_min = min((sample["free_percent"] for sample in samples if "free_percent" in sample), default=-1)
    checks = {
        "request_exit_zero": completed.returncode == 0,
        "mechanical_validation_exit_zero": validation is not None and validation.returncode == 0,
        "case_is_near_limit": 7169 <= pilot[args.index]["input_tokens"] <= 8192,
        "server_rss_below_40gb": rss_max < 40_000_000_000,
        "system_free_at_least_10_percent": free_min >= 10,
        "pageout_delta_zero": pageout_delta == 0,
        "swapout_delta_zero": swapout_delta == 0,
        "near_limit_latency_below_300_seconds": elapsed < 300,
    }
    report = {
        "status": "pass" if all(checks.values()) else "fail",
        "case_index": args.index,
        "server_pid": args.server_pid,
        "elapsed_seconds": round(elapsed, 6),
        "server_rss_max_bytes": rss_max,
        "system_free_percent_min": free_min,
        "pageout_delta": pageout_delta,
        "swapout_delta": swapout_delta,
        "checks": checks,
        "request_stdout": completed.stdout,
        "request_stderr": completed.stderr,
        "validation_stdout": validation.stdout if validation else None,
        "validation_stderr": validation.stderr if validation else None,
        "label_sha256": digest(ROOT / "labels" / f"{args.index:03d}.json") if completed.returncode == 0 else None,
        "samples": samples,
    }
    args.out.parent.mkdir(parents=True, exist_ok=True)
    args.out.write_text(json.dumps(report, ensure_ascii=False, indent=2) + "\n")
    print(json.dumps({key: report[key] for key in ("status", "case_index", "elapsed_seconds", "server_rss_max_bytes", "system_free_percent_min", "pageout_delta", "swapout_delta")}))
    if report["status"] != "pass":
        raise SystemExit(1)


if __name__ == "__main__":
    main()
