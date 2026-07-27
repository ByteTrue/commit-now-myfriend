#!/usr/bin/env python3
"""Validate and merge independent #34 audit score slices."""
from __future__ import annotations

import argparse
import json
from pathlib import Path


def lines(path: Path) -> list[dict]:
    return [json.loads(line) for line in path.read_text().splitlines() if line.strip()]


def strict_bool(row: dict, key: str) -> None:
    if type(row.get(key)) is not bool:
        raise SystemExit(f"{key} must be JSON boolean")


def main() -> None:
    parser = argparse.ArgumentParser()
    parser.add_argument("--kind", choices=("base", "guidance"), required=True)
    parser.add_argument("--slices", type=Path, default=Path("/tmp/cnm-pierce34/audit/slices"))
    parser.add_argument("--scores", type=Path, default=Path("/tmp/cnm-pierce34/audit/scores"))
    parser.add_argument("--out", type=Path, required=True)
    args = parser.parse_args()
    inputs = sorted(args.slices.glob(f"{args.kind}-*.jsonl"))
    score_paths = sorted(args.scores.glob(f"{args.kind}-*.jsonl"))
    if len(inputs) != len(score_paths) or not inputs:
        raise SystemExit("audit input/score slice count differs")
    merged = []
    seen = set()
    id_key = "family" if args.kind == "base" else "id"
    for input_path, score_path in zip(inputs, score_paths):
        source = lines(input_path)
        expected = {row["family"] if args.kind == "base" else row["meta"]["id"] for row in source}
        scores = lines(score_path)
        actual = {row.get(id_key) for row in scores}
        if len(scores) != len(expected) or actual != expected or len(actual) != len(scores):
            raise SystemExit(f"audit IDs differ or duplicate: {score_path}")
        for row in scores:
            strict_bool(row, "critical_error")
            strict_bool(row, "fully_grounded")
            if args.kind == "base":
                if row.get("subject_quality") not in (0, 1, 2) or row.get("body_quality") not in (0, 1, 2):
                    raise SystemExit("base quality must be integer 0/1/2")
            else:
                strict_bool(row, "label_correct")
            if not isinstance(row.get("reason"), str):
                raise SystemExit("audit reason must be a string")
            if row[id_key] in seen:
                raise SystemExit("cross-slice duplicate score")
            seen.add(row[id_key])
            merged.append(row)
    args.out.parent.mkdir(parents=True, exist_ok=True)
    with args.out.open("w") as stream:
        for row in merged:
            stream.write(json.dumps(row, ensure_ascii=False, separators=(",", ":")) + "\n")
    print(json.dumps({"kind": args.kind, "rows": len(merged), "out": str(args.out)}))


if __name__ == "__main__":
    main()
