#!/usr/bin/env python3
"""Mechanically validate #35 teacher records and evidence against frozen diffs."""
from __future__ import annotations

import argparse
import hashlib
import json
import re
import sys
from collections import Counter
from pathlib import Path

ARTIFACT = Path(__file__).resolve().parents[1]
CS34 = ARTIFACT.parent / "034"
sys.path.insert(0, str(CS34 / "scripts"))
sys.path.insert(0, str(ARTIFACT / "scripts"))
from build_dataset import sensitive_categories  # noqa: E402
from leakage_v2 import evidence_lines  # noqa: E402

ROOT = Path("/tmp/cnm-pierce35")
TYPES = {"build", "chore", "ci", "docs", "feat", "fix", "perf", "refactor", "revert", "style", "test"}
SCOPE = re.compile(r"^[a-z0-9][a-z0-9._/-]{0,31}$")
PREFIX = re.compile(r"^(?:build|chore|ci|docs|feat|fix|perf|refactor|revert|style|test)(?:\([^)]+\))?!?:\s", re.I)


def digest(data: bytes) -> str:
    return hashlib.sha256(data).hexdigest()


def load_jsonl(path: Path) -> list[dict]:
    return [json.loads(line) for line in path.read_text().splitlines() if line]


def changed_lines(diff: str) -> list[str]:
    return evidence_lines(diff)


def validate_record(row: dict, saved: dict) -> tuple[dict | None, list[str]]:
    errors: list[str] = []
    try:
        record = json.loads(saved["content"])
    except (KeyError, json.JSONDecodeError, TypeError) as error:
        return None, [f"invalid_json:{error}"]
    required = {"type", "scope", "subject", "body", "subject_evidence", "body_evidence"}
    if set(record) != required:
        errors.append("schema_keys")
    type_value = record.get("type")
    scope = record.get("scope")
    subject = record.get("subject")
    body = record.get("body")
    subject_evidence = record.get("subject_evidence")
    body_evidence = record.get("body_evidence")
    if type_value is not None and type_value not in TYPES:
        errors.append("type")
    if scope is not None and (not isinstance(scope, str) or not SCOPE.fullmatch(scope)):
        errors.append("scope")
    if not isinstance(subject, str) or not subject.strip() or len(subject) > 72:
        errors.append("subject")
    elif subject.endswith(".") or PREFIX.match(subject):
        errors.append("subject_shape")
    if not isinstance(body, str) or len(body) > 1200:
        errors.append("body")
    elif row["body_policy"] == "required" and not body.strip():
        errors.append("body_required")
    if not isinstance(subject_evidence, list) or not 1 <= len(subject_evidence) <= 3 or not all(isinstance(value, str) and 1 <= len(value) <= 240 for value in subject_evidence):
        errors.append("subject_evidence_shape")
    if not isinstance(body_evidence, list) or len(body_evidence) > 6 or not all(isinstance(value, str) and 1 <= len(value) <= 240 for value in body_evidence):
        errors.append("body_evidence_shape")
    if isinstance(body, str) and body.strip() and not body_evidence:
        errors.append("body_evidence_missing")
    if isinstance(body, str) and not body.strip() and body_evidence:
        errors.append("body_evidence_without_body")
    evidence = (subject_evidence if isinstance(subject_evidence, list) else []) + (body_evidence if isinstance(body_evidence, list) else [])
    lines = changed_lines(row["diff"])
    for value in evidence:
        if isinstance(value, str) and not any(value in line for line in lines):
            errors.append("evidence_not_exact_changed_line")
            break
    if len(evidence) != len(set(evidence)):
        errors.append("duplicate_evidence")
    if isinstance(subject, str) and isinstance(body, str) and sensitive_categories(subject + "\n" + body + "\n" + "\n".join(evidence)):
        errors.append("secret_or_pii")
    if saved.get("family") != row["family"] or saved.get("diff_sha256") != row["diff_sha256"] or saved.get("body_policy") != row["body_policy"]:
        errors.append("identity")
    if not isinstance(saved.get("server_pid"), int) or saved["server_pid"] <= 0:
        errors.append("server_pid")
    if saved.get("finish_reason") != "stop":
        errors.append("finish_reason")
    return record, sorted(set(errors))


def main() -> None:
    parser = argparse.ArgumentParser()
    parser.add_argument("--pilot", type=Path, default=ROOT / "data/pilot-200.jsonl")
    parser.add_argument("--manifest", type=Path, default=ARTIFACT / "pilot-manifest.json")
    parser.add_argument("--labels", type=Path, default=ROOT / "labels")
    parser.add_argument("--out", type=Path, default=ROOT / "labels/validation.json")
    parser.add_argument("--indices", help="comma-separated indices; default all 200")
    args = parser.parse_args()
    manifest = json.loads(args.manifest.read_text())
    if digest(args.pilot.read_bytes()) != manifest["outputs"]["pilot_200_sha256"]:
        raise SystemExit("pilot hash mismatch")
    rows = load_jsonl(args.pilot)
    chosen = set(range(len(rows))) if not args.indices else {int(value) for value in args.indices.split(",")}
    results = []
    counts: Counter[str] = Counter()
    server_pids: Counter[int] = Counter()
    for row in rows:
        index = row["index"]
        if index not in chosen:
            continue
        path = args.labels / f"{index:03d}.json"
        if not path.is_file():
            results.append({"index": index, "family": row["family"], "errors": ["missing_output"]})
            counts["missing_output"] += 1
            continue
        saved = json.loads(path.read_text())
        record, errors = validate_record(row, saved)
        if isinstance(saved.get("server_pid"), int):
            server_pids[saved["server_pid"]] += 1
        results.append({"index": index, "family": row["family"], "record": record, "errors": errors})
        for error in errors:
            counts[error] += 1
        counts["valid" if not errors else "invalid"] += 1
    report = {
        "status": "pass" if len(results) == len(chosen) and not counts["invalid"] and not counts["missing_output"] else "fail",
        "cases": len(results),
        "counts": counts,
        "server_pid_counts": {str(key): value for key, value in sorted(server_pids.items())},
        "results": results,
    }
    args.out.parent.mkdir(parents=True, exist_ok=True)
    args.out.write_text(json.dumps(report, ensure_ascii=False, indent=2, default=dict) + "\n")
    print(json.dumps({"status": report["status"], "cases": report["cases"], "counts": counts}, default=dict))
    if report["status"] != "pass":
        raise SystemExit(1)


if __name__ == "__main__":
    main()
