#!/usr/bin/env python3
"""Add deterministic, provenance-preserving guidance variants to #34 family splits."""
from __future__ import annotations

import argparse
import hashlib
import json
import re
from collections import Counter
from pathlib import Path

from transformers import AutoTokenizer

from pipeline import build_messages, check_requirements, parse_record, render
from build_dataset import sensitive_categories

TRAIN_TARGETS = {"issue": 120, "no_body": 120, "bullets": 100, "body": 100, "prefix": 60}
VALID_TARGETS = {"issue": 20, "no_body": 20, "bullets": 15, "body": 15, "prefix": 10}
MAX_TOKENS = 8192


def digest(value: str | bytes) -> str:
    return hashlib.sha256(value.encode() if isinstance(value, str) else value).hexdigest()


def read_jsonl(path: Path) -> list[dict]:
    return [json.loads(line) for line in path.read_text().splitlines() if line.strip()]


def write_jsonl(path: Path, rows: list[dict]) -> str:
    with path.open("w", encoding="utf-8") as stream:
        for row in rows:
            stream.write(json.dumps(row, ensure_ascii=False, separators=(",", ":")) + "\n")
    return digest(path.read_bytes())


def parts(row: dict) -> tuple[str, dict]:
    marker = "Generate the semantic commit record for this complete diff:\n\n"
    user = row["messages"][1]["content"]
    if not user.startswith(marker):
        raise ValueError("unexpected frozen user message")
    return user[len(marker):], json.loads(row["messages"][-1]["content"])


def segments(body: str) -> list[str]:
    values = []
    for part in re.split(r"\n+|(?<=[.!?])\s+", body):
        part = part.strip().lstrip("-* ").strip()
        if 8 <= len(part) <= 180:
            values.append(part)
    return values


def guidance(kind: str, split: str, number: int, prefix: str = "") -> str:
    train = {
        "issue": ["Include tracking reference {ref} in a short body.", "Add a concise body ending with issue {ref}.", "Mention {ref} in the body."],
        "no_body": ["Return a subject only with no body.", "Do not include a body.", "Use exactly one subject line."],
        "bullets": ["Use exactly two concise bullet points in the body.", "Add a body containing exactly two '-' bullets.", "Summarize supporting details as two bullet lines."],
        "body": ["Include the existing useful supporting body.", "Keep a concise body with supporting details.", "Use a subject and the supplied grounded body."],
        "prefix": ["Start the subject with `{prefix}`.", "Prefix the subject exactly with `{prefix}`.", "The subject must begin `{prefix}`."],
    }
    valid = {
        "issue": ["Put {ref} in the explanatory body."],
        "no_body": ["Omit all body text; keep only the subject."],
        "bullets": ["The body must consist of two and only two dash bullets."],
        "body": ["Include the grounded supporting body rather than a subject alone."],
        "prefix": ["Begin the subject verbatim with `{prefix}`."],
    }
    template = (train if split == "train" else valid)[kind][number % len((train if split == "train" else valid)[kind])]
    return template.format(ref=f"#{123 + number % 7}", prefix=prefix)


def make_variant(row: dict, split: str, kind: str, number: int, tokenizer) -> dict | None:
    diff, record = parts(row)
    record = dict(record)
    style = row["meta"]["style"]
    prefix = ""
    if kind == "issue":
        reference = f"#{123 + number % 7}"
        record["body"] = (record["body"] + "\n\n" if record["body"] else "") + f"Tracking: {reference}"
    elif kind == "no_body":
        record["body"] = ""
    elif kind == "bullets":
        values = segments(record["body"])
        if len(values) < 2:
            return None
        record["body"] = f"- {values[0]}\n- {values[1]}"
        style = "custom"
        record["type"] = record["scope"] = None
    elif kind == "body":
        if not record["body"]:
            return None
    elif kind == "prefix":
        haystack = (record["subject"] + "\n" + diff).lower()
        if any(word in haystack for word in ("security", "credential", "secret", "auth", "token")):
            prefix = "SECURITY:"
        elif any(word in haystack for word in ("performance", "faster", "latency", "cache")):
            prefix = "PERF:"
        elif any(word in haystack for word in ("readme", "document", "docs")):
            prefix = "DOCS:"
        else:
            prefix = "IMPORTANT:"
        if len(prefix) + 1 + len(record["subject"]) > 72:
            return None
        record["subject"] = prefix + " " + record["subject"]
        record["type"] = record["scope"] = None
        style = "custom"
    else:
        raise ValueError(kind)

    instruction = guidance(kind, split, number, prefix)
    messages = build_messages(style, diff, instruction)
    messages.append({"role": "assistant", "content": json.dumps(record, ensure_ascii=False, separators=(",", ":"))})
    encoded = tokenizer.apply_chat_template(messages, tokenize=True, add_generation_prompt=False)
    tokens = len(encoded["input_ids"] if hasattr(encoded, "keys") else encoded)
    if tokens > MAX_TOKENS:
        return None
    meta = dict(row["meta"])
    meta.update({
        "id": meta["family"] + ":guidance:" + kind,
        "variant": kind,
        "guidance": instruction,
        "style": style,
        "derived_from": meta["family"],
        "input_tokens": tokens,
    })
    return {"messages": messages, "meta": meta}


def augment(rows: list[dict], split: str, targets: dict[str, int], tokenizer) -> tuple[list[dict], Counter]:
    ordered = sorted(rows, key=lambda row: digest(f"guidance:{split}:{row['meta']['family']}"))
    variants: list[dict] = []
    counts: Counter[str] = Counter()
    used: set[tuple[str, str]] = set()
    for kind, target in targets.items():
        for number, row in enumerate(ordered):
            if counts[kind] >= target:
                break
            key = (row["meta"]["family"], kind)
            if key in used:
                continue
            variant = make_variant(row, split, kind, number, tokenizer)
            if variant is None:
                continue
            variants.append(variant)
            used.add(key)
            counts[kind] += 1
        if counts[kind] != target:
            raise SystemExit(f"insufficient {split} {kind} variants: {counts[kind]}/{target}")
    return rows + variants, counts


def validate_variant(row: dict) -> None:
    diff, _expected = parts(row)
    raw = row["messages"][-1]["content"]
    record = parse_record(raw)
    message = render(row["meta"]["style"], record)
    kind = row["meta"]["variant"]
    requirements = {}
    if kind == "issue":
        reference = re.search(r"#\d+", row["meta"]["guidance"])
        requirements = {"body_required": True, "reference": reference.group(0) if reference else ""}
    elif kind == "no_body":
        requirements = {"body_forbidden": True}
    elif kind == "bullets":
        requirements = {"body_required": True, "exact_body_bullets": 2}
    elif kind == "body":
        requirements = {"body_required": True}
    elif kind == "prefix":
        requirements = {"subject_prefix": record.subject.split(maxsplit=1)[0]}
    failures = check_requirements(record, message, requirements)
    if failures:
        raise ValueError(f"invalid deterministic guidance label: {kind}: {failures}")
    if sensitive_categories(diff + "\n" + raw):
        raise ValueError(f"sensitive deterministic guidance label: {kind}")


def main() -> None:
    parser = argparse.ArgumentParser()
    parser.add_argument("--base", type=Path, default=Path("/tmp/cnm-pierce34/data"))
    parser.add_argument("--out", type=Path, default=Path("/tmp/cnm-pierce34/data/augmented"))
    parser.add_argument("--tokenizer", type=Path, default=Path("/tmp/cnm-pierce34/base"))
    parser.add_argument("--manifest", type=Path, default=Path(".cs/epics/001-o-offline-commit-flow/artifacts/034/guidance-manifest.json"))
    args = parser.parse_args()
    tokenizer = AutoTokenizer.from_pretrained(args.tokenizer, local_files_only=True)
    train = read_jsonl(args.base / "train.jsonl")
    valid = read_jsonl(args.base / "valid.jsonl")
    test = read_jsonl(args.base / "test.jsonl")
    train_out, train_counts = augment(train, "train", TRAIN_TARGETS, tokenizer)
    valid_out, valid_counts = augment(valid, "valid", VALID_TARGETS, tokenizer)
    for row in [*train_out[len(train):], *valid_out[len(valid):]]:
        validate_variant(row)
    args.out.mkdir(parents=True, exist_ok=True)
    guidance_audit = sorted(
        [*train_out[len(train):], *valid_out[len(valid):]],
        key=lambda row: digest("audit-guidance:" + row["meta"]["id"]),
    )[:100]
    hashes = {
        "train": write_jsonl(args.out / "train.jsonl", train_out),
        "valid": write_jsonl(args.out / "valid.jsonl", valid_out),
        "test": write_jsonl(args.out / "test.jsonl", test),
        "audit_guidance_100": write_jsonl(args.out / "audit-guidance-100.jsonl", guidance_audit),
    }
    manifest = {
        "status": "awaiting_blind_audit",
        "method": "deterministic transformations of source-grounded records; no teacher model",
        "automated_variant_checks": ["schema", "renderer", "guidance_constraint", "secret", "pii"],
        "train_base": len(train),
        "train_variants": dict(train_counts),
        "valid_base": len(valid),
        "valid_variants": dict(valid_counts),
        "test": len(test),
        "output_hashes": hashes,
        "max_tokens": MAX_TOKENS,
    }
    args.manifest.write_text(json.dumps(manifest, indent=2) + "\n")
    print(json.dumps(manifest, indent=2))


if __name__ == "__main__":
    main()
