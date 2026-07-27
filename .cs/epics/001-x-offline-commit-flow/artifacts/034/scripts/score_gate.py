#!/usr/bin/env python3
"""Join independent semantic scores; this is the only script allowed to emit a quality-gate result."""
from __future__ import annotations

import argparse
import hashlib
import json
from pathlib import Path


def read_jsonl(path: Path) -> list[dict]:
    return [json.loads(line) for line in path.read_text().splitlines() if line.strip()]


def sha(path: Path) -> str:
    return hashlib.sha256(path.read_bytes()).hexdigest()


def strict_bool(item: dict, key: str, required: bool, default: bool = True) -> bool:
    if key not in item:
        if required:
            raise SystemExit(f"missing boolean score field: {key}")
        return default
    if type(item[key]) is not bool:
        raise SystemExit(f"score field must be JSON boolean: {key}")
    return item[key]


def main() -> int:
    parser = argparse.ArgumentParser()
    parser.add_argument("--results", type=Path, required=True)
    parser.add_argument("--scores", type=Path, required=True)
    parser.add_argument("--mode", choices=("shadow", "historical", "public"), required=True)
    parser.add_argument("--out", type=Path, required=True)
    args = parser.parse_args()
    results = read_jsonl(args.results)
    scores_list = read_jsonl(args.scores)
    scores = {item["id"]: item for item in scores_list}
    if len(scores) != len(scores_list) or set(scores) != {item["id"] for item in results}:
        raise SystemExit("score/result IDs differ or duplicate")

    joined = []
    for result in results:
        score = scores[result["id"]]
        if result["status"] == "rejected_context_limit" and result.get("expected_rejection") == "context_limit":
            full = bool(result["mechanical_pass"] and strict_bool(score, "disposition_pass", required=True))
            zero = not full
        else:
            raw_score = score.get("raw_score")
            rendered_score = score.get("rendered_score")
            if raw_score not in (0, 1, 2) or rendered_score not in (0, 1, 2):
                raise SystemExit(f"invalid independent score: {result['id']}")
            full = bool(result["mechanical_pass"] and raw_score == 2 and rendered_score == 2)
            zero = raw_score == 0 or rendered_score == 0
        requirements = result.get("requirements") or {}
        guidance_required = any(value not in (None, False, "", 0) for value in requirements.values())
        body_required = bool(requirements.get("body_required") or requirements.get("required_body"))
        guidance_pass = strict_bool(score, "guidance_pass", required=guidance_required)
        body_pass = strict_bool(score, "body_pass", required=body_required)
        joined.append({
            "id": result["id"],
            "full": full,
            "zero": zero,
            "guidance_required": guidance_required,
            "guidance_pass": guidance_pass,
            "body_required": body_required,
            "body_pass": body_pass,
            "score": score,
        })

    count = len(joined)
    full_count = sum(item["full"] for item in joined)
    zero_count = sum(item["zero"] for item in joined)
    guidance_ok = all(item["guidance_pass"] for item in joined if item["guidance_required"])
    body_ok = all(item["body_pass"] for item in joined if item["body_required"])
    mechanical_ok = all(item["mechanical_pass"] for item in results)
    if args.mode == "shadow":
        passed = count >= 30 and full_count >= 28 and zero_count == 0 and guidance_ok and body_ok and mechanical_ok
    elif args.mode == "historical":
        passed = count >= 50 and full_count / count >= 0.90 and zero_count == 0 and mechanical_ok
    else:
        passed = False  # public corpus is diagnostic and can never authorize GO
    summary = {
        "mode": args.mode,
        "cases": count,
        "full": full_count,
        "zero": zero_count,
        "guidance_pass": guidance_ok,
        "body_pass": body_ok,
        "mechanical_pass": mechanical_ok,
        "quality_gate_pass": passed,
        "public_is_diagnostic_only": args.mode == "public",
        "results_sha256": sha(args.results),
        "scores_sha256": sha(args.scores),
        "joined": joined,
    }
    args.out.parent.mkdir(parents=True, exist_ok=True)
    args.out.write_text(json.dumps(summary, ensure_ascii=False, indent=2) + "\n")
    print(json.dumps({key: value for key, value in summary.items() if key != "joined"}, indent=2))
    return 0 if passed or args.mode == "public" else 2


if __name__ == "__main__":
    raise SystemExit(main())
