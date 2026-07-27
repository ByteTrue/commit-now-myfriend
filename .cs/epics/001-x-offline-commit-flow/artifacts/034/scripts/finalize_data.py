#!/usr/bin/env python3
"""Finalize #34 data only after independent blind audit scores meet frozen thresholds."""
from __future__ import annotations

import argparse
import hashlib
import json
from pathlib import Path


def read_jsonl(path: Path) -> list[dict]:
    return [json.loads(line) for line in path.read_text().splitlines() if line.strip()]


def sha(path: Path) -> str:
    return hashlib.sha256(path.read_bytes()).hexdigest()


def main() -> int:
    parser = argparse.ArgumentParser()
    parser.add_argument("--base-manifest", type=Path, required=True)
    parser.add_argument("--guidance-manifest", type=Path, required=True)
    parser.add_argument("--base-audit", type=Path, required=True)
    parser.add_argument("--base-scores", type=Path, required=True)
    parser.add_argument("--guidance-audit", type=Path, required=True)
    parser.add_argument("--guidance-scores", type=Path, required=True)
    parser.add_argument("--out", type=Path, required=True)
    args = parser.parse_args()
    base_manifest = json.loads(args.base_manifest.read_text())
    guidance_manifest = json.loads(args.guidance_manifest.read_text())
    base_rows = read_jsonl(args.base_audit)
    base_scores_list = read_jsonl(args.base_scores)
    guidance_rows = read_jsonl(args.guidance_audit)
    guidance_scores_list = read_jsonl(args.guidance_scores)
    base_ids = {row["family"] for row in base_rows}
    base_scores = {row["family"]: row for row in base_scores_list}
    guidance_ids = {row["meta"]["id"] for row in guidance_rows}
    guidance_scores = {row["id"]: row for row in guidance_scores_list}
    if len(base_ids) != len(base_rows) or set(base_scores) != base_ids or len(base_scores) != len(base_scores_list):
        raise SystemExit("base audit IDs differ or duplicate")
    if len(guidance_ids) != len(guidance_rows) or set(guidance_scores) != guidance_ids or len(guidance_scores) != len(guidance_scores_list):
        raise SystemExit("guidance audit IDs differ or duplicate")
    base_critical = sum(bool(row.get("critical_error")) for row in base_scores.values())
    base_grounded = sum(bool(row.get("fully_grounded")) for row in base_scores.values())
    guidance_critical = sum(bool(row.get("critical_error")) for row in guidance_scores.values())
    guidance_correct = sum(bool(row.get("label_correct")) for row in guidance_scores.values())
    guidance_grounded = sum(bool(row.get("fully_grounded")) for row in guidance_scores.values())
    passed = (
        len(base_rows) == 200
        and base_critical == 0
        and base_grounded / len(base_rows) >= 0.95
        and len(guidance_rows) == 100
        and guidance_critical == 0
        and guidance_correct == len(guidance_rows)
        and guidance_grounded / len(guidance_rows) >= 0.95
    )
    result = {
        "status": "accepted" if passed else "rejected",
        "base": {"rows": len(base_rows), "critical": base_critical, "fully_grounded": base_grounded},
        "guidance": {
            "rows": len(guidance_rows),
            "critical": guidance_critical,
            "label_correct": guidance_correct,
            "fully_grounded": guidance_grounded,
        },
        "hashes": {
            "base_manifest": sha(args.base_manifest),
            "guidance_manifest": sha(args.guidance_manifest),
            "base_audit": sha(args.base_audit),
            "base_scores": sha(args.base_scores),
            "guidance_audit": sha(args.guidance_audit),
            "guidance_scores": sha(args.guidance_scores),
        },
        "training_hashes": guidance_manifest["output_hashes"] if passed else None,
    }
    args.out.parent.mkdir(parents=True, exist_ok=True)
    args.out.write_text(json.dumps(result, indent=2) + "\n")
    print(json.dumps(result, indent=2))
    return 0 if passed else 2


if __name__ == "__main__":
    raise SystemExit(main())
