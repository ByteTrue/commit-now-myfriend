#!/usr/bin/env python3
"""Run frozen cnm #34 cases through the JSON semantic-record boundary."""
from __future__ import annotations

import argparse
import hashlib
import json
import time
import urllib.request
from pathlib import Path

from transformers import AutoTokenizer

from pipeline import JSON_SCHEMA, build_messages, check_requirements, parse_record, render, resolve_auto

MAX_PROMPT_TOKENS = 16_000
MAX_GENERATED_TOKENS = 256
ENDPOINT = "http://127.0.0.1:63286/v1/chat/completions"


def digest(path: Path) -> str:
    return hashlib.sha256(path.read_bytes()).hexdigest()


def load_cases(path: Path) -> list[dict]:
    return [json.loads(line) for line in path.read_text().splitlines() if line.strip()]


def main() -> int:
    parser = argparse.ArgumentParser()
    parser.add_argument("--cases", type=Path, required=True)
    parser.add_argument("--tokenizer", type=Path, default=Path("/tmp/cnm-pierce34/base"))
    parser.add_argument("--out", type=Path, required=True)
    parser.add_argument("--model", default="cnm")
    parser.add_argument("--endpoint", default=ENDPOINT)
    parser.add_argument("--only")
    args = parser.parse_args()
    tokenizer = AutoTokenizer.from_pretrained(args.tokenizer, local_files_only=True)
    all_cases = load_cases(args.cases)
    cases = [
        case for case in all_cases
        if not args.only or (case.get("id") or case.get("case_id")) == args.only
    ]
    if not cases:
        raise SystemExit("no selected cases")
    args.out.mkdir(parents=True, exist_ok=True)
    results = []

    for index, case in enumerate(cases, 1):
        case_id = case.get("id") or case["case_id"]
        style = case["style"]
        history = case.get("history") or []
        resolved = resolve_auto(history) if style == "auto" else style
        messages = build_messages(resolved, case["diff"], case.get("guidance", ""), history)
        encoded = tokenizer.apply_chat_template(messages, tokenize=True, add_generation_prompt=True)
        prompt_tokens = len(encoded["input_ids"] if hasattr(encoded, "keys") else encoded)
        prompt_disposition = (case.get("prompt") or {}).get("disposition")
        expected_rejection = case.get("expected_rejection") or (
            "context_limit" if prompt_disposition == "reject_before_inference" else None
        )
        requirements = case.get("requirements") or case.get("expected_constraints") or {}
        record = None
        message = ""
        raw_output = ""
        parse_error = ""
        wall_seconds = 0.0
        response_body = None

        if prompt_tokens > MAX_PROMPT_TOKENS:
            status = "rejected_context_limit"
            mechanical_failures = [] if expected_rejection == "context_limit" else ["unexpected_context_rejection"]
        elif expected_rejection == "context_limit":
            status = "failed_to_reject_context_limit"
            mechanical_failures = ["expected_context_rejection"]
        else:
            payload = {
                "model": args.model,
                "messages": messages,
                "temperature": 0,
                "top_p": 1,
                "top_k": 0,
                "repeat_penalty": 1,
                "seed": 424242,
                "max_tokens": MAX_GENERATED_TOKENS,
                "response_format": {
                    "type": "json_schema",
                    "json_schema": {"name": "semantic_record", "strict": True, "schema": JSON_SCHEMA},
                },
            }
            request = urllib.request.Request(
                args.endpoint,
                data=json.dumps(payload).encode(),
                headers={"Content-Type": "application/json"},
                method="POST",
            )
            started = time.perf_counter()
            with urllib.request.urlopen(request, timeout=300) as response:
                response_body = json.loads(response.read().decode())
            wall_seconds = time.perf_counter() - started
            raw_output = response_body["choices"][0]["message"]["content"]
            mechanical_failures = []
            try:
                record = parse_record(raw_output)
                message = render(resolved, record)
                mechanical_failures.extend(check_requirements(record, message, requirements))
                normalized_history = {item.strip().splitlines()[0].lower() for item in history if item.strip()}
                if record.subject.lower() in normalized_history:
                    mechanical_failures.append("history_copy")
            except ValueError as error:
                parse_error = str(error)
                mechanical_failures.append("record_or_render_invalid")
            status = "generated"

        result = {
            "id": case_id,
            "source": case.get("source", "shadow"),
            "style": style,
            "resolved_style": resolved,
            "requirements": requirements,
            "expected_rejection": expected_rejection,
            "prompt_tokens": prompt_tokens,
            "diff_sha256": case.get("diff_sha256") or hashlib.sha256(case["diff"].encode()).hexdigest(),
            "wall_seconds": round(wall_seconds, 6),
            "status": status,
            "raw_output": raw_output,
            "record": None if record is None else {
                "type": record.type,
                "scope": record.scope,
                "subject": record.subject,
                "body": record.body,
            },
            "message": message,
            "parse_error": parse_error,
            "mechanical_failures": sorted(set(mechanical_failures)),
            "mechanical_pass": not mechanical_failures,
            "response_usage": None if response_body is None else response_body.get("usage"),
        }
        case_dir = args.out / case_id
        case_dir.mkdir(parents=True, exist_ok=True)
        (case_dir / "result.json").write_text(json.dumps(result, ensure_ascii=False, indent=2) + "\n")
        if response_body is not None:
            (case_dir / "response.json").write_text(json.dumps(response_body, ensure_ascii=False, indent=2) + "\n")
        results.append(result)
        print(json.dumps({"index": index, "id": case_id, "status": status, "mechanical_pass": not mechanical_failures, "seconds": round(wall_seconds, 3)}), flush=True)

    results_path = args.out / "results.jsonl"
    with results_path.open("w", encoding="utf-8") as stream:
        for result in results:
            stream.write(json.dumps(result, ensure_ascii=False, separators=(",", ":")) + "\n")
    summary = {
        "cases": len(results),
        "mechanical_pass": sum(item["mechanical_pass"] for item in results),
        "quality_gate_status": "awaiting_independent_semantic_scores",
        "cases_sha256": digest(args.cases),
        "results_sha256": digest(results_path),
        "decoding": {"temperature": 0, "top_p": 1, "top_k": 0, "repeat_penalty": 1, "seed": 424242, "max_tokens": MAX_GENERATED_TOKENS},
        "max_prompt_tokens": MAX_PROMPT_TOKENS,
    }
    (args.out / "summary.json").write_text(json.dumps(summary, indent=2) + "\n")
    print(json.dumps(summary, indent=2))
    return 0 if summary["mechanical_pass"] == summary["cases"] else 2


if __name__ == "__main__":
    raise SystemExit(main())
