#!/usr/bin/env python3
"""Freeze the source-message-blind 200-family teacher pilot for cnm #35."""
from __future__ import annotations

import argparse
import hashlib
import heapq
import json
import sys
from collections import Counter
from dataclasses import asdict, dataclass
from pathlib import Path

import pyarrow.dataset as ds
from transformers import AutoTokenizer

ARTIFACT = Path(__file__).resolve().parents[1]
CS34 = ARTIFACT.parent / "034"
sys.path.insert(0, str(CS34 / "scripts"))
sys.path.insert(0, str(ARTIFACT / "scripts"))
from leakage_v2 import near_keys as v2_near_keys, signatures as v2_signatures  # noqa: E402
from build_dataset import (  # noqa: E402
    ALLOWED_LICENSES,
    SOURCE_FILES,
    SOURCE_REVISION,
    canonical_repo,
    digest,
    leakage_signatures,
    load_exclusions,
    make_diff,
    repo_group,
    sensitive_categories,
    signature_pairs,
    source_hashes,
)

SEED = 350_725
MAX_INPUT_TOKENS = 8192
POOL_LIMIT = 3000
FILE_TARGETS = {"single": 40, "two_three": 100, "four_eight": 60}
HIGH_TARGET = 50
REPO_CAP = 3


@dataclass
class Candidate:
    family: str
    repo: str
    repo_group: str
    commit: str
    language: str
    license: str
    paths: list[str]
    file_count: int
    diff: str
    diff_sha256: str
    normalized_patch_sha256: str
    changed_token_minhash: tuple[str, ...]
    v2_content_sha256: str
    v2_minhash: tuple[int, ...]
    input_tokens: int


def file_bin(count: int) -> str | None:
    if count == 1:
        return "single"
    if 2 <= count <= 3:
        return "two_three"
    if 4 <= count <= 8:
        return "four_eight"
    return None


def push_smallest(heap: list, score: int, candidate: Candidate, limit: int = POOL_LIMIT) -> None:
    entry = (-score, candidate.family, candidate)
    if len(heap) < limit:
        heapq.heappush(heap, entry)
    elif score < -heap[0][0]:
        heapq.heapreplace(heap, entry)


def changed_line_texts(diff: str) -> list[str]:
    values: list[str] = []
    inside = False
    for line in diff.splitlines():
        if line.startswith("diff --git "):
            inside = False
        elif line.startswith("@@"):
            inside = True
        elif inside and line[:1] in {"+", "-"} and not line.startswith(("+++", "---")):
            value = line[1:].strip()
            if value:
                values.append(value)
    return values


def write_jsonl(path: Path, rows: list[dict]) -> str:
    with path.open("w", encoding="utf-8") as stream:
        for row in rows:
            stream.write(json.dumps(row, ensure_ascii=False, separators=(",", ":")) + "\n")
    return digest(path.read_bytes())


def main() -> None:
    parser = argparse.ArgumentParser()
    parser.add_argument("--source", type=Path, default=Path("/tmp/cnm-pierce34/source"))
    parser.add_argument("--tokenizer", type=Path, default=Path("/tmp/cnm-pierce34/base"))
    parser.add_argument("--shadow-manifest", type=Path, default=CS34 / "shadow-manifest.json")
    parser.add_argument("--public-regression", type=Path, default=Path("/tmp/cnm-train31/eval/high-risk.jsonl"))
    parser.add_argument("--public-manifest", type=Path, default=CS34 / "public-regression-manifest.json")
    parser.add_argument("--prior-audit-exclusions", type=Path, default=ARTIFACT / "prior-audit-exclusions.json")
    parser.add_argument("--v2-exclusions", type=Path, default=ARTIFACT / "leakage-v2-exclusions.json")
    parser.add_argument("--prompt", type=Path, default=ARTIFACT / "teacher-prompt.txt")
    parser.add_argument("--schema", type=Path, default=ARTIFACT / "teacher-schema.json")
    parser.add_argument("--out", type=Path, default=Path("/tmp/cnm-pierce35/data"))
    parser.add_argument("--labels", type=Path, default=Path("/tmp/cnm-pierce35/labels"))
    parser.add_argument("--manifest", type=Path, default=ARTIFACT / "pilot-manifest.json")
    args = parser.parse_args()

    if any(args.labels.iterdir()):
        raise SystemExit("teacher output already exists; refusing post-output pilot rebuild")
    source_paths = [args.source / name for name in sorted(SOURCE_FILES)]
    source_hashes(source_paths)
    exclusions = load_exclusions(args.shadow_manifest, args.public_regression, args.public_manifest)
    prior_audit = json.loads(args.prior_audit_exclusions.read_text())
    prior_families = set(prior_audit["families"])
    v2_exclusions = json.loads(args.v2_exclusions.read_text())
    excluded_v2_hashes = set(v2_exclusions["content_sha256"])
    excluded_v2_keys = {
        key
        for values in v2_exclusions["minhash"]
        for key in v2_near_keys(values)
    }

    tokenizer = AutoTokenizer.from_pretrained(args.tokenizer, local_files_only=True)
    prompt = args.prompt.read_text().strip()
    schema = json.loads(args.schema.read_text())
    dataset = ds.dataset([str(path) for path in source_paths], format="parquet")
    counters: Counter[str] = Counter()
    pools: dict[tuple[str, str], list] = {
        (kind, size): [] for kind in FILE_TARGETS for size in ("normal", "high")
    }
    overlimit: list = []

    scanner = dataset.scanner(columns=["repo", "hash", "language", "license", "mods"], batch_size=2048)
    for batch in scanner.to_batches():
        for row in batch.to_pylist():
            counters["source_rows"] += 1
            repo = canonical_repo(row.get("repo") or "")
            commit = row.get("hash") or ""
            group = repo_group(repo) if repo else ""
            family = f"commitchronicle:{repo}:{commit}"
            if (
                not repo
                or repo in exclusions["repos"]
                or group in exclusions["repo_groups"]
                or family in prior_families
                or digest(commit) in exclusions["commits"]
            ):
                counters["excluded_identity"] += 1
                continue
            if row.get("license") not in ALLOWED_LICENSES:
                counters["license"] += 1
                continue
            result = make_diff(row.get("mods") or [])
            if not result:
                counters["incomplete_or_size"] += 1
                continue
            diff, paths = result
            kind = file_bin(len(paths))
            if not kind:
                counters["file_count"] += 1
                continue
            diff_sha = digest(diff)
            patch_sha, changed_minhash, _target, _target_minhash = leakage_signatures(diff, "")
            content_v2 = v2_signatures(diff)
            content_v2_keys = v2_near_keys(content_v2["minhash"])
            if (
                diff_sha in exclusions["diffs"]
                or patch_sha in exclusions["patches"]
                or signature_pairs(changed_minhash) & exclusions["changed_pairs"]
                or content_v2["content_sha256"] in excluded_v2_hashes
                or bool(content_v2_keys & excluded_v2_keys)
            ):
                counters["excluded_overlap"] += 1
                continue
            found = sensitive_categories(diff)
            if found:
                counters["sensitive"] += 1
                for category in found:
                    counters[f"sensitive_{category}"] += 1
                continue
            body_policy = "required" if kind != "single" else "optional"
            user = f"BODY_POLICY: {body_policy}\n\nCOMPLETE_DIFF:\n{diff}"
            messages = [{"role": "system", "content": prompt}, {"role": "user", "content": user}]
            encoded = tokenizer.apply_chat_template(messages, tokenize=True, add_generation_prompt=True)
            tokens = len(encoded["input_ids"] if hasattr(encoded, "keys") else encoded)
            candidate = Candidate(
                family=family,
                repo=repo,
                repo_group=group,
                commit=commit,
                language=row.get("language") or "unknown",
                license=row["license"],
                paths=paths,
                file_count=len(paths),
                diff=diff,
                diff_sha256=diff_sha,
                normalized_patch_sha256=patch_sha,
                changed_token_minhash=changed_minhash,
                v2_content_sha256=content_v2["content_sha256"],
                v2_minhash=tuple(content_v2["minhash"]),
                input_tokens=tokens,
            )
            score = int(digest(f"pilot:{SEED}:{family}"), 16)
            if tokens > MAX_INPUT_TOKENS:
                counters["over_limit"] += 1
                if tokens <= 16_384:
                    push_smallest(overlimit, score, candidate, 20)
                continue
            size = "high" if tokens >= 4096 else "normal"
            push_smallest(pools[(kind, size)], score, candidate)
            counters["eligible"] += 1

    candidates = [entry[2] for heap in pools.values() for entry in heap]
    candidates.sort(key=lambda item: digest(f"dedupe:{SEED}:{item.family}"))
    deduped: list[Candidate] = []
    seen_diffs: set[str] = set()
    seen_patches: set[str] = set()
    seen_pairs: set[tuple[str, str]] = set()
    seen_v2_hashes: set[str] = set()
    seen_v2_keys: set[tuple[tuple[int, int], ...]] = set()
    for item in candidates:
        pairs = signature_pairs(item.changed_token_minhash)
        content_keys = v2_near_keys(item.v2_minhash)
        if (
            item.diff_sha256 in seen_diffs
            or item.normalized_patch_sha256 in seen_patches
            or pairs & seen_pairs
            or item.v2_content_sha256 in seen_v2_hashes
            or content_keys & seen_v2_keys
        ):
            counters["near_duplicate"] += 1
            continue
        seen_diffs.add(item.diff_sha256)
        seen_patches.add(item.normalized_patch_sha256)
        seen_pairs.update(pairs)
        seen_v2_hashes.add(item.v2_content_sha256)
        seen_v2_keys.update(content_keys)
        deduped.append(item)

    selected: list[Candidate] = []
    selected_ids: set[str] = set()
    repo_counts: Counter[str] = Counter()

    def pick(kind: str, high_only: bool, wanted: int) -> None:
        pool = sorted(
            (
                item for item in deduped
                if file_bin(item.file_count) == kind
                and (not high_only or item.input_tokens >= 4096)
            ),
            key=lambda item: digest(f"select:{SEED}:{item.family}"),
        )
        have = sum(file_bin(item.file_count) == kind and (not high_only or item.input_tokens >= 4096) for item in selected)
        for item in pool:
            if have >= wanted:
                return
            if item.family in selected_ids or repo_counts[item.repo_group] >= REPO_CAP:
                continue
            selected.append(item)
            selected_ids.add(item.family)
            repo_counts[item.repo_group] += 1
            have += 1
        raise SystemExit(f"insufficient pilot coverage for {kind} high={high_only}: {have}/{wanted}")

    def pick_high(wanted: int) -> None:
        pool = sorted(
            (item for item in deduped if item.input_tokens >= 4096),
            key=lambda item: digest(f"select-high:{SEED}:{item.family}"),
        )
        have = sum(item.input_tokens >= 4096 for item in selected)
        for item in pool:
            if have >= wanted:
                return
            kind = file_bin(item.file_count)
            if (
                item.family in selected_ids
                or repo_counts[item.repo_group] >= REPO_CAP
                or sum(file_bin(current.file_count) == kind for current in selected) >= FILE_TARGETS[kind]
            ):
                continue
            selected.append(item)
            selected_ids.add(item.family)
            repo_counts[item.repo_group] += 1
            have += 1
        raise SystemExit(f"insufficient high-token pilot coverage: {have}/{wanted}")

    pick_high(HIGH_TARGET)
    for kind, wanted in FILE_TARGETS.items():
        pick(kind, False, wanted)

    near = [item for item in deduped if 7169 <= item.input_tokens <= MAX_INPUT_TOKENS]
    if not any(item.input_tokens >= 7169 for item in selected):
        replacement = next((item for item in sorted(near, key=lambda row: digest(f"near:{SEED}:{row.family}")) if repo_counts[item.repo_group] < REPO_CAP), None)
        if replacement is None:
            raise SystemExit("no valid near-limit pilot input")
        kind = file_bin(replacement.file_count)
        victim = next(
            item for item in reversed(selected)
            if file_bin(item.file_count) == kind and item.input_tokens >= 4096 and item.input_tokens < 7169
        )
        selected.remove(victim)
        selected_ids.remove(victim.family)
        repo_counts[victim.repo_group] -= 1
        selected.append(replacement)
        selected_ids.add(replacement.family)
        repo_counts[replacement.repo_group] += 1

    selected.sort(key=lambda item: digest(f"order:{SEED}:{item.family}"))
    multi = sorted((item for item in selected if item.file_count >= 2), key=lambda item: digest(f"body:{SEED}:{item.family}"))
    required = {item.family for item in multi[:60]}
    if len(required) != 60:
        raise SystemExit("insufficient body-required cases")
    over = sorted((entry[2] for entry in overlimit), key=lambda item: digest(f"over:{SEED}:{item.family}"))
    if not over:
        raise SystemExit("no sanitized over-limit rejection input")
    rejected = over[0]

    rows = []
    for index, item in enumerate(selected):
        rows.append({
            "index": index,
            "family": item.family,
            "repo": item.repo,
            "commit": item.commit,
            "language": item.language,
            "license": item.license,
            "paths": item.paths,
            "file_count": item.file_count,
            "diff": item.diff,
            "diff_sha256": item.diff_sha256,
            "normalized_patch_sha256": item.normalized_patch_sha256,
            "changed_token_minhash": item.changed_token_minhash,
            "v2_content_sha256": item.v2_content_sha256,
            "v2_minhash": item.v2_minhash,
            "input_tokens": item.input_tokens,
            "body_policy": "required" if item.family in required else "optional",
            "changed_line_count": len(changed_line_texts(item.diff)),
        })
    output_hash = write_jsonl(args.out / "pilot-200.jsonl", rows)
    over_hash = write_jsonl(args.out / "over-limit.jsonl", [{
        **{key: value for key, value in asdict(rejected).items() if key != "diff"},
        "diff": rejected.diff,
        "expected": "reject_before_inference",
    }])
    slices = []
    for reviewer in ("A", "B"):
        order = sorted(rows, key=lambda row: digest(f"audit:{reviewer}:{SEED}:{row['family']}"))
        slices.append({
            "reviewer": reviewer,
            "slices": [[row["index"] for row in order[start:start + 20]] for start in range(0, 200, 20)],
        })
    audit_plan = {
        "status": "frozen_pre_output",
        "reviewers": slices,
        "rubric": {
            "critical_errors_required": 0,
            "fully_grounded_by_both": 190,
            "subject_quality_2_by_both": 180,
            "body_required": 60,
            "body_useful_by_both": 60,
            "body_quality_2_by_both": 54,
            "mechanical_smoke_limit_and_local_teacher_gates": "all_pass",
        },
    }
    audit_path = args.out / "audit-plan.json"
    audit_path.write_text(json.dumps(audit_plan, ensure_ascii=False, indent=2) + "\n")

    stats = {
        "families": len(rows),
        "repositories": len({row["repo"] for row in rows}),
        "repository_groups": len({repo_group(row["repo"]) for row in rows}),
        "file_bins": Counter(file_bin(row["file_count"]) for row in rows),
        "high_token_inputs": sum(row["input_tokens"] >= 4096 for row in rows),
        "near_limit_inputs": sum(row["input_tokens"] >= 7169 for row in rows),
        "body_required": sum(row["body_policy"] == "required" for row in rows),
        "languages": Counter(row["language"] for row in rows),
        "licenses": Counter(row["license"] for row in rows),
        "token_min": min(row["input_tokens"] for row in rows),
        "token_max": max(row["input_tokens"] for row in rows),
        "token_mean": round(sum(row["input_tokens"] for row in rows) / len(rows), 2),
    }
    manifest = {
        "status": "frozen_pre_output",
        "seed": SEED,
        "source": {
            "dataset": "JetBrains-Research/commit-chronicle",
            "revision": SOURCE_REVISION,
            "files": SOURCE_FILES,
            "license_policy": sorted(ALLOWED_LICENSES),
            "columns_read": ["repo", "hash", "language", "license", "mods"],
            "source_message_read": False,
        },
        "limits": {"max_input_tokens": MAX_INPUT_TOKENS, "max_files": 8, "repo_group_cap": REPO_CAP},
        "targets": {"file_bins": FILE_TARGETS, "high_token": 50, "body_required": 60, "families": 200},
        "filters": dict(counters),
        "stats": stats,
        "outputs": {
            "pilot_200_sha256": output_hash,
            "over_limit_sha256": over_hash,
            "audit_plan_sha256": digest(audit_path.read_bytes()),
        },
        "inputs": {
            "prompt_sha256": digest(args.prompt.read_bytes()),
            "schema_sha256": digest(args.schema.read_bytes()),
            "shadow_manifest_sha256": digest(args.shadow_manifest.read_bytes()),
            "public_regression_sha256": digest(args.public_regression.read_bytes()),
            "public_manifest_sha256": digest(args.public_manifest.read_bytes()),
            "prior_audit_exclusions_sha256": digest(args.prior_audit_exclusions.read_bytes()),
            "leakage_v2_exclusions_sha256": digest(args.v2_exclusions.read_bytes()),
            "tokenizer_manifest_sha256": digest((ARTIFACT / "tokenizer-manifest.json").read_bytes()),
            "build_dataset_034_sha256": digest((CS34 / "scripts/build_dataset.py").read_bytes()),
            "pipeline_034_sha256": digest((CS34 / "scripts/pipeline.py").read_bytes()),
            "build_pilot_sha256": digest(Path(__file__).read_bytes()),
            "leakage_v2_code_sha256": digest((ARTIFACT / "scripts/leakage_v2.py").read_bytes()),
        },
        "prior_audit_exclusions": {
            "families": len(prior_families),
            "reason": "excludes all 160 first-audit reviewed rows and all 200 rows in the redesigned frozen audit",
        },
        "teacher_output_guard": "builder refuses to run when labels directory is non-empty",
        "pilot_role": "evaluation_only_never_student_training",
    }
    args.manifest.write_text(json.dumps(manifest, ensure_ascii=False, indent=2, default=dict) + "\n")
    print(json.dumps({"status": manifest["status"], "filters": manifest["filters"], "stats": stats}, ensure_ascii=False, indent=2, default=dict))


if __name__ == "__main__":
    main()
