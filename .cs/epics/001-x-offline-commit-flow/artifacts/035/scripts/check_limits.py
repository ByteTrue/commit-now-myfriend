#!/usr/bin/env python3
"""Prove static and executable #35 teacher input-length boundaries."""
from __future__ import annotations

import argparse
import hashlib
import json
import subprocess
import sys
from pathlib import Path

ARTIFACT = Path(__file__).resolve().parents[1]
ROOT = Path("/tmp/cnm-pierce35")


def digest(path: Path) -> str:
    value = hashlib.sha256()
    with path.open("rb") as stream:
        for chunk in iter(lambda: stream.read(8 * 1024 * 1024), b""):
            value.update(chunk)
    return value.hexdigest()


def rows(path: Path) -> list[dict]:
    return [json.loads(line) for line in path.read_text().splitlines() if line]


def static_check() -> dict:
    pilot = rows(ROOT / "data/pilot-200.jsonl")
    over = rows(ROOT / "data/over-limit.jsonl")
    manifest = json.loads((ARTIFACT / "pilot-manifest.json").read_text())
    checks = {
        "pilot_hash_matches": digest(ROOT / "data/pilot-200.jsonl") == manifest["outputs"]["pilot_200_sha256"],
        "over_limit_hash_matches": digest(ROOT / "data/over-limit.jsonl") == manifest["outputs"]["over_limit_sha256"],
        "all_inference_cases_within_limit": len(pilot) == 200 and all(row["input_tokens"] <= 8192 for row in pilot),
        "near_limit_case_present": any(7169 <= row["input_tokens"] <= 8192 for row in pilot),
        "one_over_limit_case_declared": len(over) == 1 and over[0]["input_tokens"] > 8192 and over[0]["expected"] == "reject_before_inference",
        "over_limit_case_not_in_inference_set": bool(over) and over[0]["family"] not in {row["family"] for row in pilot},
    }
    report = {"status": "pass" if all(checks.values()) else "fail", "checks": checks, "pilot_token_max": max(row["input_tokens"] for row in pilot), "over_limit_tokens": over[0]["input_tokens"] if over else None}
    (ARTIFACT / "limit-check.json").write_text(json.dumps(report, indent=2) + "\n")
    return report


def executable_check() -> dict:
    before = subprocess.run(["lsof", "-nP", "-iTCP:63286", "-sTCP:LISTEN", "-t"], text=True, capture_output=True).stdout.split()
    completed = subprocess.run([sys.executable, str(ARTIFACT / "scripts/label_pilot.py"), "--over-limit-check"], text=True, capture_output=True)
    after = subprocess.run(["lsof", "-nP", "-iTCP:63286", "-sTCP:LISTEN", "-t"], text=True, capture_output=True).stdout.split()
    checks = {
        "entrypoint_returned_frozen_rejection_code_42": completed.returncode == 42,
        "entrypoint_reported_rejection_before_inference": '"status": "rejected_before_inference"' in completed.stdout,
        "no_teacher_listener_before": before == [],
        "no_teacher_listener_after": after == [],
        "no_over_limit_output_created": not any((ROOT / "labels").glob("over-limit-*.json")),
    }
    report = {
        "status": "pass" if all(checks.values()) else "fail",
        "checks": checks,
        "returncode": completed.returncode,
        "stdout": completed.stdout,
        "stderr": completed.stderr,
    }
    out = ROOT / "logs/over-limit-entrypoint.json"
    out.parent.mkdir(parents=True, exist_ok=True)
    out.write_text(json.dumps(report, indent=2) + "\n")
    return report


def main() -> None:
    parser = argparse.ArgumentParser()
    parser.add_argument("--entrypoint", action="store_true", help="run after freeze, before server startup")
    args = parser.parse_args()
    report = executable_check() if args.entrypoint else static_check()
    print(json.dumps(report))
    if report["status"] != "pass":
        raise SystemExit(1)


if __name__ == "__main__":
    main()
