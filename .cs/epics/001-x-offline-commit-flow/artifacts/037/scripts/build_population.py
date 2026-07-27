#!/usr/bin/env python3
"""Build the frozen 200-family evaluation-only population for cnm #37.

Every prior #34-#36 family/diff/signature is excluded before selection.
The source commit message is normalized (newlines, trailers, trim) but NEVER
semantically transformed, styled, or rewritten.  This is the candidate target
shown to critics and reviewers.
"""
from __future__ import annotations

import argparse
import hashlib
import json
import re
import sys
from collections import Counter
from pathlib import Path

import pyarrow.dataset as ds
from transformers import AutoTokenizer

ARTIFACT = Path(__file__).resolve().parents[1]  # artifacts/037/
ARTIFACTS_ROOT = ARTIFACT.parent  # artifacts/
CS34 = ARTIFACTS_ROOT / "034"
CS35 = ARTIFACTS_ROOT / "035"

sys.path.insert(0, str(CS34 / "scripts"))
sys.path.insert(0, str(CS35 / "scripts"))

# ── reuse the same core filters as #34 / #35 ──────────────────────────
from build_dataset import (  # noqa: E402
    SEED as _IGNORE_SEED,
    SOURCE_FILES,
    ALLOWED_LICENSES,
    MIN_DIFF_BYTES,
    MAX_DIFF_BYTES,
    MAX_FILES,
    SENSITIVE,
    GENERATED_PARTS,
    LEAK_TOKENS,
    digest,
    canonical_repo,
    repo_group,
    clean_message,
    make_diff,
    sensitive_categories,
    minhashes,
    leakage_signatures,
    signature_pairs,
    grounded_body,
    grounded,
)
from pipeline import build_messages, parse_record, render  # noqa: E402
from leakage_v2 import changed_content, signatures as v2_signatures  # noqa: E402

# ── #37-specific constants ─────────────────────────────────────────────
SEED = 37
MAX_STUDENT_TOKENS = 8192
NEAR_LIMIT_MIN = 7600
REPO_CAP = 2
FILE_BINS = {"single": (1, 1), "medium": (2, 3), "large": (4, 25)}
QUOTAS = {"single": 40, "medium": 100, "large": 60}
BODY_REQUIRED = 60
HIGH_TOKEN_MIN = 50
POPULATION = 200


def file_bin_key(count: int) -> str:
    for name, (lo, hi) in FILE_BINS.items():
        if lo <= count <= hi:
            return name
    raise ValueError(f"file count {count} out of range")


# ── target normalization (lossless, no semantic transform) ─────────────
TRAILER_RE = re.compile(r"(?im)^(?:signed-off-by|co-authored-by|acked-by|reviewed-by|tested-by|reported-by|suggested-by):")


def normalize_target(message: str) -> tuple[str, str] | None:
    """Normalize a human source message without inventing/trimming semantics."""
    value = message.replace("\r\n", "\n").replace("\r", "\n").strip()
    if not value or "\x00" in value:
        return None
    lines = value.splitlines()
    subject = lines[0].strip()
    body_lines = [
        line for line in lines[1:]
        if not re.fullmatch(r"\s*(?:fixes|closes|resolves)\s*:?(?:\s*#\d+)?\s*", line, re.I)
    ]
    body = "\n".join(body_lines).strip()
    if body.startswith("\n"):
        body = body.lstrip("\n")
    if not (10 <= len(subject) <= 72):
        return None
    lowered = subject.lower()
    if lowered.startswith(("merge ", "revert \"", "bump version", "release ")):
        return None
    if subject.endswith((":", ",", ";", "-")) or len(body) > 1200:
        return None
    # #37 allows non-ASCII messages (dropped in #34)
    return subject, body


def target_hash(subject: str, body: str) -> str:
    return digest(subject + "\n" + body)


# ── exclusion loading ──────────────────────────────────────────────────

def load_all_exclusions() -> dict:
    shadow = json.loads((CS34 / "shadow-manifest.json").read_text())
    public_manifest = json.loads((CS34 / "public-regression-manifest.json").read_text())
    public_regression = Path("/tmp/cnm-train31/eval/high-risk.jsonl")

    # identity
    excluded_repos = {
        canonical_repo(item["canonical"] if isinstance(item, dict) else item)
        for item in shadow["repository_exclusions"]
    }
    excluded_groups = {repo_group(repo) for repo in excluded_repos}
    excluded_commits = set(shadow.get("commit_sha256_exclusions", []))
    for item in shadow.get("hashed_commit_diff_exclusions", []):
        if item.get("source_commit_sha256"):
            excluded_commits.add(item["source_commit_sha256"])
    excluded_diffs = {item.get("complete_diff_sha256", "") for item in shadow.get("hashed_commit_diff_exclusions", [])}
    excluded_diffs.discard("")

    # near-duplicate
    entries = (shadow.get("leakage_exclusions") or {}).get("entries", [])
    excluded_patches = {item["normalized_patch_sha256"] for item in entries}
    excluded_changed_pairs = set().union(*(signature_pairs(item.get("changed_token_minhash", [])) for item in entries))
    excluded_targets = {item.get("normalized_target_sha256", "") for item in entries}
    excluded_targets.discard("")
    excluded_target_pairs = set().union(*(signature_pairs(item.get("target_token_minhash", [])) for item in entries))

    # public regression
    if public_regression.is_file():
        for line in public_regression.read_text().splitlines():
            row = json.loads(line)
            excluded_diffs.add(row["diff_sha256"])
            commit = row.get("commit_hash")
            if commit:
                excluded_commits.add(digest(commit))

    # public-regression target entries
    for item in public_manifest.get("target_entries", []):
        excluded_commits.add(item["commit_sha256"])
        excluded_targets.add(item["normalized_target_sha256"])
        excluded_target_pairs.update(signature_pairs(item.get("target_token_minhash", [])))

    # #35 prior-audit-exclusions (360 #34 audit rows)
    prior_audit = json.loads((CS35 / "prior-audit-exclusions.json").read_text())
    excluded_families = set(prior_audit.get("families", []))
    for item in prior_audit.get("diff_entries", []):
        excluded_diffs.add(item.get("diff_sha256", ""))
        excluded_patches.add(item.get("normalized_patch_sha256", ""))
        excluded_changed_pairs.update(signature_pairs(item.get("changed_token_minhash", [])))
    excluded_diffs.discard("")

    # #35 leakage-v2 exclusions
    v2_exclusions = json.loads((CS35 / "leakage-v2-exclusions.json").read_text())
    excluded_v2_hashes = set(v2_exclusions.get("content_sha256", []))
    excluded_v2_keys = set().union(*(signature_pairs(item.get("minhash", [])) for item in v2_exclusions.get("entries", [])))

    # #35 pilot 200 families (also covers #36 same families)
    pilot35_path = Path("/tmp/cnm-pierce35/data/pilot-200.jsonl")
    if pilot35_path.is_file():
        for line in pilot35_path.read_text().splitlines():
            if line:
                item = json.loads(line)
                excluded_families.add(item["family"])
                excluded_diffs.add(item.get("diff_sha256", ""))
                excluded_patches.add(item.get("normalized_patch_sha256", ""))
                excluded_changed_pairs.update(signature_pairs(item.get("changed_token_minhash", [])))
                v2_info = item.get("v2_content_sha256")
                if v2_info:
                    excluded_v2_hashes.add(v2_info)
                    excluded_v2_keys.update(signature_pairs(item.get("v2_minhash", [])))

    # #34 data audit-200
    for line in Path("/tmp/cnm-pierce34/data/audit-200.jsonl").read_text().splitlines():
        if line:
            excluded_families.add(json.loads(line)["family"])

    # #34 train/valid/test families (8000 total)
    for split in ("train", "valid", "test"):
        p = Path(f"/tmp/cnm-pierce34/data/{split}.jsonl")
        if p.is_file():
            for line in p.read_text().splitlines():
                if line:
                    meta = json.loads(line).get("meta", {})
                    excluded_families.add(meta.get("family", ""))
                    excluded_commits.add(digest(meta.get("commit", "")))
                    excluded_diffs.add(meta.get("diff_sha256", ""))
                    excluded_patches.add(meta.get("normalized_patch_sha256", ""))
                    excluded_changed_pairs.update(signature_pairs(meta.get("changed_token_minhash", ())))
                    excluded_targets.add(meta.get("normalized_target_sha256", ""))
                    excluded_target_pairs.update(signature_pairs(meta.get("target_token_minhash", ())))

    excluded_diffs.discard("")
    excluded_patches.discard("")
    excluded_targets.discard("")
    excluded_families.discard("")
    excluded_commits.discard("")

    return {
        "repos": excluded_repos,
        "repo_groups": excluded_groups,
        "commits": excluded_commits,
        "diffs": excluded_diffs,
        "patches": excluded_patches,
        "changed_pairs": excluded_changed_pairs,
        "targets": excluded_targets,
        "target_pairs": excluded_target_pairs,
        "families": excluded_families,
        "v2_hashes": excluded_v2_hashes,
        "v2_keys": excluded_v2_keys,
    }


# ── main ───────────────────────────────────────────────────────────────

def main() -> None:
    parser = argparse.ArgumentParser()
    parser.add_argument("--source", type=Path, default=Path("/tmp/cnm-pierce34/source"))
    parser.add_argument("--tokenizer", type=Path, default=Path("/tmp/cnm-pierce34/base"))
    parser.add_argument("--out", type=Path, default=Path("/tmp/cnm-pierce37/data"))
    parser.add_argument("--manifest", type=Path, default=ARTIFACT / "population-manifest.json")
    args = parser.parse_args()
    args.out.mkdir(parents=True, exist_ok=True)

    print("Loading exclusions...", flush=True)
    exclusions = load_all_exclusions()
    print(f"  excluded repos: {len(exclusions['repos'])}")
    print(f"  excluded families: {len(exclusions['families'])}")
    print(f"  excluded diffs: {len(exclusions['diffs'])}")
    print(f"  excluded patches: {len(exclusions['patches'])}")
    print(f"  excluded targets: {len(exclusions['targets'])}")
    print(f"  excluded v2 hashes: {len(exclusions['v2_hashes'])}")

    print("Loading tokenizer...", flush=True)
    tokenizer = AutoTokenizer.from_pretrained(str(args.tokenizer), local_files_only=True)

    # Verify source files
    paths = [args.source / name for name in sorted(SOURCE_FILES)]
    for p in paths:
        assert p.is_file(), f"Missing source: {p}"
    # source_hashes would go here but they're already verified in #34

    print("Scanning Commit Chronicle...", flush=True)
    dataset = ds.dataset([str(p) for p in paths], format="parquet")
    counters: Counter[str] = Counter()
    pool: list[dict] = []
    seen_target_hashes: set[str] = set()

    for batch in dataset.scanner(batch_size=2048).to_batches():
        for row in batch.to_pylist():
            counters["source_rows"] += 1
            repo = canonical_repo(row.get("repo") or "")
            commit = row.get("hash") or ""
            if not repo or repo in exclusions["repos"] or repo_group(repo) in exclusions["repo_groups"] or digest(commit) in exclusions["commits"]:
                counters["excluded_identity"] += 1
                continue
            family = f"commitchronicle:{repo}:{commit}"
            if family in exclusions["families"]:
                counters["excluded_family"] += 1
                continue
            if row.get("license") not in ALLOWED_LICENSES:
                counters["license"] += 1
                continue
            normalized = normalize_target(row.get("message") or "")
            if not normalized:
                counters["message"] += 1
                continue
            diff_result = make_diff(row.get("mods") or [])
            if not diff_result:
                counters["incomplete_or_size"] += 1
                continue
            diff, file_paths = diff_result
            diff_hash = digest(diff)
            if diff_hash in exclusions["diffs"]:
                counters["excluded_diff"] += 1
                continue
            subject, body = normalized
            # grounded body filter (reuse #34 logic, but don't trim body)
            body_cleaned = grounded_body(body, diff, file_paths) if body else ""
            # #37 body_policy: multi-file commits with non-empty source body are required
            body_policy = "required" if body and len(file_paths) >= 2 else "optional"
            body_kept = body_cleaned if body_policy == "required" else (body_cleaned if body_cleaned else "")
            target_text = subject + ("\n" + body_kept if body_kept else "")
            patch_hash, changed_minhash, target_hash_val, target_minhash = leakage_signatures(diff, target_text)
            if (
                patch_hash in exclusions["patches"]
                or signature_pairs(changed_minhash) & exclusions["changed_pairs"]
                or target_hash_val in exclusions["targets"]
                or signature_pairs(target_minhash) & exclusions["target_pairs"]
            ):
                counters["excluded_near_overlap"] += 1
                continue
            # v2 signatures
            v2 = v2_signatures(diff)
            v2_content_hash = v2["content_sha256"]
            v2_minhash = v2["minhash"]
            if v2_content_hash in exclusions["v2_hashes"] or bool(signature_pairs(v2_minhash) & exclusions["v2_keys"]):
                counters["excluded_v2_overlap"] += 1
                continue
            combined = diff + "\n" + subject + "\n" + (body_kept or "")
            found = sensitive_categories(combined)
            if found:
                counters["sensitive"] += 1
                for cat in found:
                    counters[f"sensitive_{cat}"] += 1
                continue
            if not grounded(subject, diff, file_paths):
                counters["ungrounded_heuristic"] += 1
                continue
            if target_hash_val in seen_target_hashes:
                counters["duplicate_target"] += 1
                continue
            # token count using the same system prompt as #34
            style = "plain"  # #37 does not assign styles
            msgs = build_messages(style, diff)
            record_json = json.dumps({"type": None, "scope": None, "subject": subject, "body": body_kept}, ensure_ascii=False, separators=(",", ":"))
            msgs.append({"role": "assistant", "content": record_json})
            encoded = tokenizer.apply_chat_template(msgs, tokenize=True, add_generation_prompt=False)
            tokens = len(encoded["input_ids"] if hasattr(encoded, "keys") else encoded)
            if tokens > MAX_STUDENT_TOKENS:
                counters["too_many_tokens"] += 1
                continue
            seen_target_hashes.add(target_hash_val)
            file_count = len(file_paths)
            bin_name = file_bin_key(file_count)
            pool.append({
                "family": family,
                "repo": repo,
                "repo_group": repo_group(repo),
                "commit": commit,
                "language": row.get("language") or "unknown",
                "license": row["license"],
                "paths": file_paths,
                "file_count": file_count,
                "file_bin": bin_name,
                "diff": diff,
                "diff_sha256": diff_hash,
                "normalized_patch_sha256": patch_hash,
                "changed_token_minhash": list(changed_minhash),
                "v2_content_sha256": v2_content_hash,
                "v2_minhash": v2_minhash,
                "subject": subject,
                "body": body_kept,
                "body_policy": body_policy,
                "target_sha256": target_hash_val,
                "target_token_minhash": list(target_minhash),
                "input_tokens": tokens,
                "source_message_sha256": digest(row.get("message") or ""),
                "normalized_message_sha256": target_hash_val,
            })
            counters["accepted"] += 1

    print(f"\nScan complete: {counters['source_rows']} source rows, {counters['accepted']} accepted", flush=True)

    # ── hash-based deterministic selection with quotas ──────────────────
    def selection_key(item: dict) -> str:
        return digest(f"{SEED}:{item['family']}")

    pool.sort(key=selection_key)
    selected: list[dict] = []
    selected_families: set[str] = set()
    bin_counts: Counter[str] = Counter()
    repo_counts: Counter[str] = Counter()
    body_required_count = 0
    high_token_count = 0

    def select_pass(predicate, max_items: int, label: str) -> int:
        nonlocal body_required_count, high_token_count
        added = 0
        for item in pool:
            if added >= max_items:
                break
            if item["family"] in selected_families:
                continue
            if repo_counts[item["repo"]] >= REPO_CAP:
                continue
            bin_name = item["file_bin"]
            if bin_counts[bin_name] >= QUOTAS[bin_name]:
                continue
            if not predicate(item):
                continue
            selected.append(item)
            selected_families.add(item["family"])
            bin_counts[bin_name] += 1
            repo_counts[item["repo"]] += 1
            if item["body_policy"] == "required":
                body_required_count += 1
            if item["input_tokens"] >= 4096:
                high_token_count += 1
            added += 1
        return added

    # Pass 1: high_token from any bin (these are naturally more common in large bins)
    select_pass(lambda item: item["input_tokens"] >= 4096, HIGH_TOKEN_MIN, "high_token")
    # Pass 2: body_required from medium+large bins
    select_pass(lambda item: item["body_policy"] == "required", BODY_REQUIRED, "body_required")
    # Pass 3: fill remaining quotas
    remaining = POPULATION - len(selected)
    select_pass(lambda item: True, remaining, "fill")

    # Check near-limit and over-limit
    near_limit_found = any(item["input_tokens"] >= NEAR_LIMIT_MIN for item in selected)
    over_limit_found = any(item["input_tokens"] > MAX_STUDENT_TOKENS for item in selected)

    # The over-limit rejection case must be separately sourced from excluded rows.
    # If none survived the selection pass, record that the pool had them.
    if not over_limit_found:
        print(f"  NOTE: no over-limit case in population; {counters.get('too_many_tokens', 0)} were excluded during scan")

    print(f"\nSelected {len(selected)} candidates", flush=True)
    print(f"  bins: {dict(bin_counts)}", flush=True)
    print(f"  body_required: {body_required_count}", flush=True)
    print(f"  high_token: {high_token_count}", flush=True)
    print(f"  near_limit: {near_limit_found}", flush=True)

    # Verify quotas
    for bin_name, quota in QUOTAS.items():
        if bin_counts.get(bin_name, 0) != quota:
            print(f"  WARNING: {bin_name} bin has {bin_counts.get(bin_name, 0)}/{quota}", flush=True)

    # Assign indices
    for i, item in enumerate(selected):
        item["index"] = i

    # Write population
    population_path = args.out / "population-200.jsonl"
    with population_path.open("w") as f:
        for item in selected:
            f.write(json.dumps(item, ensure_ascii=False, separators=(",", ":")) + "\n")

    # Write manifest
    manifest = {
        "status": "frozen_pre_output",
        "seed": SEED,
        "population": POPULATION,
        "counts": dict(counters),
        "bin_counts": dict(bin_counts),
        "body_required": body_required_count,
        "high_token": high_token_count,
        "near_limit_found": near_limit_found,
        "over_limit_found": over_limit_found,
        "population_sha256": digest(population_path.read_bytes()),
        "exclusion_counts": {
            "repos": len(exclusions["repos"]),
            "repo_groups": len(exclusions["repo_groups"]),
            "commits": len(exclusions["commits"]),
            "diffs": len(exclusions["diffs"]),
            "patches": len(exclusions["patches"]),
            "changed_pairs": len(exclusions["changed_pairs"]),
            "targets": len(exclusions["targets"]),
            "target_pairs": len(exclusions["target_pairs"]),
            "families": len(exclusions["families"]),
            "v2_hashes": len(exclusions["v2_hashes"]),
            "v2_keys": len(exclusions["v2_keys"]),
        },
    }
    manifest_path = args.manifest
    manifest_path.write_text(json.dumps(manifest, ensure_ascii=False, indent=2) + "\n")
    print(f"\nManifest: {manifest_path}", flush=True)
    print(json.dumps({"status": "frozen_pre_output", "population": POPULATION, "bins": dict(bin_counts), "body_required": body_required_count}, ensure_ascii=False, indent=2))


if __name__ == "__main__":
    main()
