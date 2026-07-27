#!/usr/bin/env python3
"""Prepare model-identity/source-message-blind review slices after mechanical validation passes."""
from __future__ import annotations

import hashlib
import json
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


def verify_freeze() -> None:
    frozen = json.loads((ARTIFACT / "freeze-manifest.json").read_text())
    for name, expected in frozen["files"].items():
        path = Path(name)
        if not path.is_file() or path.stat().st_size != expected["bytes"] or digest(path) != expected["sha256"]:
            raise SystemExit(f"frozen input mismatch: {path}")


def main() -> None:
    verify_freeze()
    pilot = rows(ROOT / "data/pilot-200.jsonl")
    validation = json.loads((ROOT / "labels/validation.json").read_text())
    if validation["status"] != "pass" or validation["cases"] != 200:
        raise SystemExit("mechanical validation has not passed all 200 cases")
    by_index = {row["index"]: row for row in pilot}
    validated = {row["index"]: row["record"] for row in validation["results"]}
    plan = json.loads((ROOT / "data/audit-plan.json").read_text())
    out = ROOT / "audit"
    out.mkdir(parents=True, exist_ok=True)
    hashes = {}
    for reviewer in plan["reviewers"]:
        name = reviewer["reviewer"]
        for number, indices in enumerate(reviewer["slices"]):
            path = out / f"reviewer-{name.lower()}-slice-{number:02d}.jsonl"
            with path.open("w") as stream:
                for index in indices:
                    source = by_index[index]
                    record = validated[index]
                    stream.write(json.dumps({
                        "index": index,
                        "body_policy": source["body_policy"],
                        "complete_diff": source["diff"],
                        "teacher_target": {key: record[key] for key in ("type", "scope", "subject", "body")},
                        "teacher_evidence": {
                            "subject": record["subject_evidence"],
                            "body": record["body_evidence"],
                        },
                    }, ensure_ascii=False, separators=(",", ":")) + "\n")
            hashes[path.name] = digest(path)
    manifest = {
        "status": "frozen_pre_review",
        "instructions_sha256": digest(ARTIFACT / "audit-instructions.md"),
        "rubric_sha256": digest(ARTIFACT / "audit-rubric.json"),
        "validation_sha256": digest(ROOT / "labels/validation.json"),
        "slices": hashes,
        "reviewer_inputs": ["opaque_index", "complete_diff", "body_policy", "teacher_target", "teacher_evidence"],
        "withheld": ["repository_identity", "commit_sha", "source_commit_message", "other_scores", "teacher_model_identity", "aggregate_progress"],
    }
    path = out / "review-input-manifest.json"
    path.write_text(json.dumps(manifest, ensure_ascii=False, indent=2) + "\n")
    print(json.dumps({"status": manifest["status"], "slices": len(hashes), "sha256": digest(path)}))


if __name__ == "__main__":
    main()
