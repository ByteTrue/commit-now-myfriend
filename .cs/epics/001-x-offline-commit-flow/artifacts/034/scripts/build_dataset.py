#!/usr/bin/env python3
"""Build the frozen complete-commit family splits for cnm #34."""
from __future__ import annotations

import argparse
import hashlib
import json
import re
from collections import Counter, defaultdict
from dataclasses import asdict, dataclass
from itertools import combinations
from pathlib import Path

import pyarrow.dataset as ds
from transformers import AutoTokenizer

from pipeline import build_messages, parse_record, render

SEED = 340_526  # one permitted redesign after the first blind audit stopped on critical errors
SOURCE_REVISION = "5fd076e67b812a9f3d1999e5e40f71715f84bb51"
SOURCE_FILES = {
    "commit-chronicle-00.parquet": "fadd63f1b896e992d75bb37b9fb5045db1dc2e3b53c749d1ef7e30996ccf117f",
    "commit-chronicle-01.parquet": "405a203bc196f99d25cb31e25cb99f8dc3435dca8407ea843cb731441c12cbdd",
}
ALLOWED_LICENSES = {
    "Apache License 2.0",
    "MIT License",
    "BSD 3-Clause New or Revised License",
}
TARGETS = {"train": 6000, "valid": 1000, "test": 1000}
REPO_CAPS = {"train": 60, "valid": 60, "test": 60}
MAX_TRAIN_TOKENS = 8192
MIN_DIFF_BYTES = 300
MAX_DIFF_BYTES = 60_000
MAX_FILES = 25
CONVENTIONAL = re.compile(
    r"^(?P<type>build|chore|ci|docs|feat|fix|perf|refactor|revert|style|test)"
    r"(?:\((?P<scope>[^)]+)\))?!?:\s+(?P<subject>.+)$",
    re.I,
)
WORDS = re.compile(r"[A-Za-z][A-Za-z0-9_-]+")
LEAK_TOKENS = re.compile(r"[A-Za-z0-9_]+|[^\s]")
STOP = {
    "about", "after", "again", "allow", "also", "before", "change", "changes", "from", "have",
    "into", "more", "only", "other", "than", "that", "their", "then", "there", "these", "this",
    "those", "through", "update", "when", "where", "which", "with", "without",
}
IMPERATIVE = {
    "add", "allow", "avoid", "build", "cache", "check", "clean", "disable", "document", "drop", "enable",
    "ensure", "fix", "handle", "improve", "include", "keep", "make", "move", "prevent", "reduce", "refactor",
    "reject", "remove", "rename", "restore", "return", "simplify", "skip", "support", "test", "update", "use",
    "validate", "verify", "write",
}
BAD_SUBJECTS = {"update", "updates", "fix", "fix bug", "minor changes", "cleanup", "wip", "work in progress"}
GENERATED_PARTS = {"node_modules", "vendor", "third_party", "dist", "build", "coverage"}
SENSITIVE = (
    ("private_key", re.compile(r"-----BEGIN (?:RSA |EC |OPENSSH )?PRIVATE KEY-----")),
    ("aws_key", re.compile(r"\bAKIA[0-9A-Z]{16}\b")),
    ("github_token", re.compile(r"\bgh[pousr]_[A-Za-z0-9_]{30,}\b")),
    ("openai_key", re.compile(r"\bsk-[A-Za-z0-9_-]{20,}\b")),
    ("basic_auth_url", re.compile(r"(?i)://[^\s/:@]+:[^\s/@]+@[^\s/]+")),
    ("assigned_secret", re.compile(r'''(?i)\b(?:api[_-]?key|access[_-]?token|secret[_-]?key|password)\s*[:=]\s*["'][^"']{12,}["']''')),
    ("home_path_pii", re.compile(r"(?i)(?:/Users|/home)/[A-Za-z0-9._-]+/")),
    ("email", re.compile(r"(?i)\b[A-Z0-9._%+-]+@[A-Z0-9.-]+\.[A-Z]{2,}\b")),
    ("ipv4", re.compile(r"(?<![\d.])(?:25[0-5]|2[0-4]\d|1?\d?\d)(?:\.(?:25[0-5]|2[0-4]\d|1?\d?\d)){3}(?![\d.])")),
)


def digest(value: str | bytes) -> str:
    if isinstance(value, str):
        value = value.encode()
    return hashlib.sha256(value).hexdigest()


def canonical_repo(value: str) -> str:
    return value.strip().lower().removesuffix(".git")


def repo_group(repo: str) -> str:
    # Forks normally preserve the repository basename; over-grouping is safer than leakage.
    return canonical_repo(repo).rsplit("/", 1)[-1]


def clean_message(value: str) -> tuple[str, str] | None:
    value = value.replace("\r\n", "\n").replace("\r", "\n").strip()
    if not value or "\x00" in value or re.search(r"(?im)^(?:signed-off-by|co-authored-by):", value):
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
    lowered = subject.lower()
    if not (10 <= len(subject) <= 72) or lowered in BAD_SUBJECTS:
        return None
    if lowered.startswith(("merge ", "revert \"", "bump version", "release ")):
        return None
    if subject.endswith((":", ",", ";", "-")) or len(body) > 1200:
        return None
    if any(ord(char) > 127 for char in value):
        return None
    return subject, body


def semantic_record(subject: str, body: str, family_hash: str) -> tuple[str, dict]:
    match = CONVENTIONAL.match(subject)
    if match:
        content = match.group("subject").strip()
        scope = (match.group("scope") or "").lower() or None
        if scope and not re.fullmatch(r"[a-z0-9][a-z0-9._/-]{0,31}", scope):
            scope = None
        style = "angular" if content[:1].islower() and not content.endswith(".") and int(family_hash[:2], 16) % 2 else "conventional"
        return style, {"type": match.group("type").lower(), "scope": scope, "subject": content, "body": body}
    first = subject.split(maxsplit=1)[0].lower().rstrip(":")
    if first in IMPERATIVE and not subject.endswith("."):
        style = "google" if int(family_hash[:2], 16) % 2 else "atom"
    else:
        style = "plain"
    return style, {"type": None, "scope": None, "subject": subject, "body": body}


def make_diff(mods: list[dict]) -> tuple[str, list[str]] | None:
    if not mods or len(mods) > MAX_FILES:
        return None
    pieces: list[str] = []
    paths: list[str] = []
    for mod in mods:
        patch = (mod.get("diff") or "").strip("\n")
        old = (mod.get("old_path") or mod.get("new_path") or "").strip()
        new = (mod.get("new_path") or mod.get("old_path") or "").strip()
        if not patch or not old or not new or "Binary files " in patch or "GIT binary patch" in patch:
            return None
        old_header = "/dev/null" if mod.get("change_type") == "ADD" else f"a/{old}"
        new_header = "/dev/null" if mod.get("change_type") == "DELETE" else f"b/{new}"
        pieces.append(f"diff --git a/{old} b/{new}\n--- {old_header}\n+++ {new_header}\n{patch}\n")
        paths.append(new if new_header != "/dev/null" else old)
    value = "\n".join(pieces)
    if not (MIN_DIFF_BYTES <= len(value.encode()) <= MAX_DIFF_BYTES):
        return None
    if all(any(part.lower() in GENERATED_PARTS for part in Path(path).parts) for path in paths):
        return None
    return value, paths


def sensitive_categories(value: str) -> list[str]:
    return [name for name, pattern in SENSITIVE if pattern.search(value)]


def minhashes(normalized: str, width: int) -> tuple[str, ...]:
    tokens = LEAK_TOKENS.findall(normalized)
    values = {
        digest(json.dumps(tokens[index:index + width], ensure_ascii=False, separators=(",", ":")))
        for index in range(max(0, len(tokens) - width + 1))
    }
    return tuple(sorted(values)[:8])


def leakage_signatures(diff: str, target: str) -> tuple[str, tuple[str, ...], str, tuple[str, ...]]:
    changed: list[str] = []
    inside_hunk = False
    for line in diff.splitlines():
        if line.startswith("@@"):
            inside_hunk = True
            continue
        if line.startswith("diff --git "):
            inside_hunk = False
            continue
        if inside_hunk and line[:1] in {"+", "-"}:
            changed.append(re.sub(r"\s+", " ", line[1:].strip()).lower())
    normalized_patch = "\n".join(changed)
    normalized_target = re.sub(r"\s+", " ", target.strip()).lower()
    return (
        digest(normalized_patch),
        minhashes(normalized_patch, 5),
        digest(normalized_target),
        minhashes(normalized_target, 3),
    )


def signature_pairs(values: tuple[str, ...] | list[str]) -> set[tuple[str, str]]:
    return set(combinations(values, 2))


def normalized_words(value: str) -> set[str]:
    value = re.sub(r"([a-z0-9])([A-Z])", r"\1 \2", value)
    result = set()
    for raw in WORDS.findall(value.lower()):
        word = raw
        for suffix in ("ingly", "edly", "ing", "ed", "es", "s"):
            if len(word) > len(suffix) + 3 and word.endswith(suffix):
                word = word[:-len(suffix)]
                break
        result.add(word)
    return result


def semantic_terms(value: str) -> list[str]:
    generic = STOP | IMPERATIVE | {"code", "file", "files", "test", "tests", "thing", "stuff", "version"}
    return [word for word in normalized_words(value) if len(word) >= 4 and word not in generic]


def grounded(subject: str, diff: str, paths: list[str]) -> bool:
    haystack = normalized_words(" ".join(paths) + "\n" + diff)
    terms = semantic_terms(subject)
    matches = sum(term in haystack for term in terms)
    required = len(terms) if len(terms) <= 2 else max(2, (len(terms) * 4 + 4) // 5)
    return bool(terms) and matches >= required


def grounded_body(body: str, diff: str, paths: list[str]) -> str:
    if not body:
        return ""
    if re.search(r"(?i)\b(?:because|faster|speedup|cannot|can't|due to|pull request|previous pr|unverifiable)\b", body):
        return ""
    haystack = normalized_words(" ".join(paths) + "\n" + diff)
    sentences = [part.strip() for part in re.split(r"(?:\n+|(?<=[.!?])\s+)", body) if part.strip()]
    for sentence in sentences:
        terms = semantic_terms(sentence)
        if not terms:
            continue
        matches = sum(term in haystack for term in terms)
        required = len(terms) if len(terms) <= 2 else max(2, (len(terms) * 7 + 9) // 10)
        if matches < required:
            return ""
    return body


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
    message_sha256: str
    normalized_patch_sha256: str
    changed_token_minhash: tuple[str, ...]
    normalized_target_sha256: str
    target_token_minhash: tuple[str, ...]
    style: str
    record: dict
    input_tokens: int
    component: str = ""
    split: str = ""


def source_hashes(paths: list[Path]) -> None:
    for path in paths:
        expected = SOURCE_FILES[path.name]
        actual = digest(path.read_bytes())
        if actual != expected:
            raise SystemExit(f"source hash mismatch: {path}: {actual}")


def load_exclusions(shadow_manifest: Path, public_regression: Path, public_manifest: Path) -> dict:
    shadow = json.loads(shadow_manifest.read_text())
    repository_rows = shadow["repository_exclusions"]
    excluded_repos = {
        canonical_repo(item["canonical"] if isinstance(item, dict) else item)
        for item in repository_rows
    }
    excluded_groups = {repo_group(repo) for repo in excluded_repos}
    exclusion_rows = shadow.get("hashed_commit_diff_exclusions", [])
    excluded_commits = {
        item["source_commit_sha256"] for item in exclusion_rows if item.get("source_commit_sha256")
    }
    excluded_diffs = {
        item["complete_diff_sha256"] for item in exclusion_rows if item.get("complete_diff_sha256")
    }
    excluded_commits.update(shadow.get("commit_sha256_exclusions", []))
    excluded_diffs.update(shadow.get("diff_sha256_exclusions", []))
    entries = (shadow.get("leakage_exclusions") or {}).get("entries", [])
    excluded_patches = {item["normalized_patch_sha256"] for item in entries}
    excluded_changed_pairs = set().union(*(signature_pairs(item.get("changed_token_minhash", [])) for item in entries))
    excluded_targets = {item["normalized_target_sha256"] for item in entries if item.get("normalized_target_sha256")}
    excluded_target_pairs = set().union(*(signature_pairs(item.get("target_token_minhash", [])) for item in entries))
    if public_regression.is_file():
        for line in public_regression.read_text().splitlines():
            row = json.loads(line)
            excluded_diffs.add(row["diff_sha256"])
            commit = row.get("commit_hash")
            if commit:
                excluded_commits.add(digest(commit))
            match = re.search(r"```diff\n(.*)\n```\s*$", row.get("user", ""), re.S)
            if match:
                patch, changed, _target, _target_minhash = leakage_signatures(match.group(1), "")
                excluded_patches.add(patch)
                excluded_changed_pairs.update(signature_pairs(changed))
    if public_manifest.is_file():
        public = json.loads(public_manifest.read_text())
        if public.get("source_sha256") != digest(public_regression.read_bytes()):
            raise SystemExit("public regression manifest/source hash mismatch")
        for item in public.get("target_entries", []):
            excluded_commits.add(item["commit_sha256"])
            excluded_targets.add(item["normalized_target_sha256"])
            excluded_target_pairs.update(signature_pairs(item.get("target_token_minhash", [])))
    else:
        raise SystemExit("public regression target signatures must be frozen before data selection")
    return {
        "repos": excluded_repos,
        "repo_groups": excluded_groups,
        "commits": excluded_commits,
        "diffs": excluded_diffs,
        "patches": excluded_patches,
        "changed_pairs": excluded_changed_pairs,
        "targets": excluded_targets,
        "target_pairs": excluded_target_pairs,
    }


def dedupe_near(candidates: list[Candidate]) -> tuple[list[Candidate], int]:
    kept: list[Candidate] = []
    changed_pairs: set[tuple[str, str]] = set()
    target_pairs: set[tuple[str, str]] = set()
    patches: set[str] = set()
    targets: set[str] = set()
    diffs: set[str] = set()
    for candidate in sorted(candidates, key=lambda item: digest(f"near:{SEED}:{item.family}")):
        item_changed = signature_pairs(candidate.changed_token_minhash)
        item_target = signature_pairs(candidate.target_token_minhash)
        if (
            candidate.diff_sha256 in diffs
            or candidate.normalized_patch_sha256 in patches
            or candidate.normalized_target_sha256 in targets
            or item_changed & changed_pairs
            or item_target & target_pairs
        ):
            continue
        diffs.add(candidate.diff_sha256)
        patches.add(candidate.normalized_patch_sha256)
        targets.add(candidate.normalized_target_sha256)
        changed_pairs.update(item_changed)
        target_pairs.update(item_target)
        kept.append(candidate)
    return kept, len(candidates) - len(kept)


def canonical_repo_components(candidates: list[Candidate]) -> tuple[dict[str, str], dict[str, int]]:
    """Freeze conservative fork components from names plus shared Git/DAG evidence."""
    repos = sorted({item.repo for item in candidates})
    parent = {repo: repo for repo in repos}

    def find(repo: str) -> str:
        while parent[repo] != repo:
            parent[repo] = parent[parent[repo]]
            repo = parent[repo]
        return repo

    def union(left: str, right: str) -> None:
        left, right = find(left), find(right)
        if left != right:
            parent[max(left, right)] = min(left, right)

    names: dict[str, list[str]] = defaultdict(list)
    commits: dict[str, set[str]] = defaultdict(set)
    patches: dict[str, set[str]] = defaultdict(set)
    for item in candidates:
        names[item.repo_group].append(item.repo)
        commits[item.commit].add(item.repo)
        patches[item.normalized_patch_sha256].add(item.repo)
    basename_links = 0
    for values in names.values():
        unique = sorted(set(values))
        for repo in unique[1:]:
            union(unique[0], repo)
            basename_links += 1
    shared_commit_links = 0
    for values in commits.values():
        unique = sorted(values)
        for repo in unique[1:]:
            union(unique[0], repo)
            shared_commit_links += 1
    patch_pair_counts: Counter[tuple[str, str]] = Counter()
    for values in patches.values():
        unique = sorted(values)
        if 1 < len(unique) <= 20:
            patch_pair_counts.update(combinations(unique, 2))
    repeated_patch_links = 0
    for pair, count in patch_pair_counts.items():
        if count >= 2:
            union(*pair)
            repeated_patch_links += 1
    mapping = {repo: find(repo) for repo in repos}
    return mapping, {
        "repositories": len(repos),
        "components": len(set(mapping.values())),
        "basename_links": basename_links,
        "shared_commit_links": shared_commit_links,
        "repeated_patch_links": repeated_patch_links,
    }


def assign_components(candidates: list[Candidate], repo_components: dict[str, str]) -> None:
    for candidate in candidates:
        candidate.component = repo_components[candidate.repo]
        value = int(digest(f"{SEED}:{candidate.component}")[:8], 16) % 1000
        candidate.split = "train" if value < 750 else "valid" if value < 875 else "test"


def select(candidates: list[Candidate], split: str) -> list[Candidate]:
    target = TARGETS[split]
    repo_cap = REPO_CAPS[split]
    pool = sorted((item for item in candidates if item.split == split), key=lambda item: digest(f"{SEED}:{item.family}"))
    chosen: list[Candidate] = []
    chosen_ids: set[str] = set()
    repo_counts: Counter[str] = Counter()

    def pick(predicate, wanted: int) -> None:
        for item in pool:
            if len(chosen) >= target or sum(1 for current in chosen if predicate(current)) >= wanted:
                break
            if item.family in chosen_ids or repo_counts[item.repo] >= repo_cap or not predicate(item):
                continue
            chosen.append(item)
            chosen_ids.add(item.family)
            repo_counts[item.repo] += 1

    pick(lambda item: bool(item.record["body"]), round(target * 0.15))
    pick(lambda item: item.input_tokens >= 4000, round(target * 0.10))
    pick(lambda item: item.file_count >= 4, round(target * 0.20))
    pick(lambda item: item.file_count >= 2, round(target * 0.60))
    pick(lambda item: True, target)
    if len(chosen) != target:
        raise SystemExit(f"insufficient {split} rows after quotas: {len(chosen)}/{target}")
    if sum(item.file_count >= 2 for item in chosen) < round(target * 0.60):
        raise SystemExit(f"{split} multi-file coverage below 60%")
    return chosen


def serialize(candidate: Candidate) -> dict:
    assistant = json.dumps(candidate.record, ensure_ascii=False, separators=(",", ":"))
    messages = build_messages(candidate.style, candidate.diff)
    messages.append({"role": "assistant", "content": assistant})
    meta = asdict(candidate)
    meta.pop("diff")
    meta.pop("record")
    return {"messages": messages, "meta": meta}


def write_jsonl(path: Path, rows: list[dict]) -> str:
    with path.open("w", encoding="utf-8") as stream:
        for row in rows:
            stream.write(json.dumps(row, ensure_ascii=False, separators=(",", ":")) + "\n")
    return digest(path.read_bytes())


def main() -> None:
    parser = argparse.ArgumentParser()
    parser.add_argument("--source", type=Path, default=Path("/tmp/cnm-pierce34/source"))
    parser.add_argument("--tokenizer", type=Path, default=Path("/tmp/cnm-pierce34/base"))
    parser.add_argument("--shadow-manifest", type=Path, default=Path(".cs/epics/001-o-offline-commit-flow/artifacts/034/shadow-manifest.json"))
    parser.add_argument("--public-regression", type=Path, default=Path("/tmp/cnm-train31/eval/high-risk.jsonl"))
    parser.add_argument("--public-manifest", type=Path, default=Path(".cs/epics/001-o-offline-commit-flow/artifacts/034/public-regression-manifest.json"))
    parser.add_argument("--out", type=Path, default=Path("/tmp/cnm-pierce34/data"))
    parser.add_argument("--manifest", type=Path, default=Path(".cs/epics/001-o-offline-commit-flow/artifacts/034/data-manifest.json"))
    args = parser.parse_args()
    paths = [args.source / name for name in sorted(SOURCE_FILES)]
    source_hashes(paths)
    if not args.shadow_manifest.is_file():
        raise SystemExit("independent shadow manifest must be frozen before data selection")
    exclusions = load_exclusions(args.shadow_manifest, args.public_regression, args.public_manifest)
    tokenizer = AutoTokenizer.from_pretrained(args.tokenizer, local_files_only=True)
    dataset = ds.dataset([str(path) for path in paths], format="parquet")
    counters: Counter[str] = Counter()
    candidates: list[Candidate] = []
    seen_message: set[str] = set()

    for batch in dataset.scanner(batch_size=2048).to_batches():
        for row in batch.to_pylist():
            counters["source_rows"] += 1
            repo = canonical_repo(row.get("repo") or "")
            commit = row.get("hash") or ""
            if (
                not repo
                or repo in exclusions["repos"]
                or repo_group(repo) in exclusions["repo_groups"]
                or digest(commit) in exclusions["commits"]
            ):
                counters["excluded_identity"] += 1
                continue
            if row.get("license") not in ALLOWED_LICENSES:
                counters["license"] += 1
                continue
            cleaned = clean_message(row.get("message") or "")
            if not cleaned:
                counters["message"] += 1
                continue
            diff_result = make_diff(row.get("mods") or [])
            if not diff_result:
                counters["incomplete_or_size"] += 1
                continue
            diff, paths_for_row = diff_result
            diff_hash = digest(diff)
            if diff_hash in exclusions["diffs"]:
                counters["excluded_diff"] += 1
                continue
            subject, body = cleaned
            grounded_source_body = grounded_body(body, diff, paths_for_row)
            if body and not grounded_source_body:
                counters["body_not_grounded"] += 1
            body = grounded_source_body
            patch_hash, changed_minhash, target_hash, target_minhash = leakage_signatures(
                diff, subject + "\n" + body
            )
            if (
                patch_hash in exclusions["patches"]
                or signature_pairs(changed_minhash) & exclusions["changed_pairs"]
                or target_hash in exclusions["targets"]
                or signature_pairs(target_minhash) & exclusions["target_pairs"]
            ):
                counters["excluded_near_overlap"] += 1
                continue
            combined = diff + "\n" + subject + "\n" + body
            found = sensitive_categories(combined)
            if found:
                counters["sensitive"] += 1
                for category in found:
                    counters[f"sensitive_{category}"] += 1
                continue
            if not grounded(subject, diff, paths_for_row):
                counters["ungrounded_heuristic"] += 1
                continue
            message_hash = target_hash
            if message_hash in seen_message:
                counters["duplicate_message"] += 1
                continue
            family = f"commitchronicle:{repo}:{commit}"
            family_hash = digest(family)
            style, record = semantic_record(subject, body, family_hash)
            raw_record = json.dumps(record, ensure_ascii=False, separators=(",", ":"))
            try:
                parsed_record = parse_record(raw_record)
                render(style, parsed_record)
            except ValueError:
                counters["record_schema"] += 1
                continue
            messages = build_messages(style, diff)
            messages.append({"role": "assistant", "content": raw_record})
            encoded = tokenizer.apply_chat_template(messages, tokenize=True, add_generation_prompt=False)
            tokens = len(encoded["input_ids"] if hasattr(encoded, "keys") else encoded)
            if tokens > MAX_TRAIN_TOKENS:
                counters["too_many_tokens"] += 1
                continue
            seen_message.add(message_hash)
            candidates.append(Candidate(
                family=family,
                repo=repo,
                repo_group=repo_group(repo),
                commit=commit,
                language=row.get("language") or "unknown",
                license=row["license"],
                paths=paths_for_row,
                file_count=len(paths_for_row),
                diff=diff,
                diff_sha256=diff_hash,
                message_sha256=message_hash,
                normalized_patch_sha256=patch_hash,
                changed_token_minhash=changed_minhash,
                normalized_target_sha256=target_hash,
                target_token_minhash=target_minhash,
                style=style,
                record=record,
                input_tokens=tokens,
            ))
            counters["accepted_candidates"] += 1

    repo_components, component_evidence = canonical_repo_components(candidates)
    candidates, near_removed = dedupe_near(candidates)
    counters["near_duplicate_candidates"] = near_removed
    assign_components(candidates, repo_components)
    selected = {name: select(candidates, name) for name in TARGETS}
    all_selected = [item for rows in selected.values() for item in rows]
    for key in (
        "diff_sha256",
        "message_sha256",
        "normalized_patch_sha256",
        "normalized_target_sha256",
        "family",
        "component",
    ):
        owners: dict[str, set[str]] = defaultdict(set)
        for split, rows in selected.items():
            for item in rows:
                owners[str(getattr(item, key))].add(split)
        leaks = [value for value, splits in owners.items() if len(splits) > 1]
        if leaks:
            raise SystemExit(f"cross-split {key} leakage: {len(leaks)}")
    pair_owners: dict[tuple[str, tuple[str, str]], set[str]] = defaultdict(set)
    for split, rows in selected.items():
        for item in rows:
            for pair in signature_pairs(item.changed_token_minhash):
                pair_owners[("changed", pair)].add(split)
            for pair in signature_pairs(item.target_token_minhash):
                pair_owners[("target", pair)].add(split)
    if any(len(splits) > 1 for splits in pair_owners.values()):
        raise SystemExit("cross-split shingle leakage")
    if any(
        item.repo in exclusions["repos"]
        or item.repo_group in exclusions["repo_groups"]
        or item.diff_sha256 in exclusions["diffs"]
        or item.normalized_patch_sha256 in exclusions["patches"]
        or signature_pairs(item.changed_token_minhash) & exclusions["changed_pairs"]
        or item.normalized_target_sha256 in exclusions["targets"]
        or signature_pairs(item.target_token_minhash) & exclusions["target_pairs"]
        for item in all_selected
    ):
        raise SystemExit("shadow/public exclusion leakage")
    if any(sensitive_categories(item.diff + "\n" + item.record["subject"] + "\n" + item.record["body"]) for item in all_selected):
        raise SystemExit("selected secret/PII scan failed")

    args.out.mkdir(parents=True, exist_ok=True)
    output_hashes = {}
    for split, rows in selected.items():
        output_hashes[split] = write_jsonl(args.out / f"{split}.jsonl", [serialize(item) for item in rows])
    audit = sorted(all_selected, key=lambda item: digest(f"audit:{SEED}:{item.family}"))[:200]
    audit_hash = write_jsonl(args.out / "audit-200.jsonl", [
        {
            "family": item.family,
            "repo": item.repo,
            "commit": item.commit,
            "paths": item.paths,
            "diff": item.diff,
            "style": item.style,
            "record": item.record,
        }
        for item in audit
    ])

    stats = {}
    for split, rows in selected.items():
        stats[split] = {
            "families": len(rows),
            "repositories": len({item.repo for item in rows}),
            "repository_groups": len({item.component for item in rows}),
            "multi_file": sum(item.file_count >= 2 for item in rows),
            "four_plus_files": sum(item.file_count >= 4 for item in rows),
            "with_body": sum(bool(item.record["body"]) for item in rows),
            "styles": Counter(item.style for item in rows),
            "languages": Counter(item.language for item in rows),
            "token_min": min(item.input_tokens for item in rows),
            "token_max": max(item.input_tokens for item in rows),
            "token_mean": round(sum(item.input_tokens for item in rows) / len(rows), 2),
        }
    manifest = {
        "status": "awaiting_manual_audit",
        "seed": SEED,
        "source": {
            "dataset": "JetBrains-Research/commit-chronicle",
            "revision": SOURCE_REVISION,
            "configuration": "subset_cmg/test",
            "files": SOURCE_FILES,
            "license_policy": sorted(ALLOWED_LICENSES),
            "dataset_card_warning": "Provenance-preserving research data; original per-repository licenses and privacy warning apply.",
        },
        "tokenizer": str(args.tokenizer),
        "max_train_tokens": MAX_TRAIN_TOKENS,
        "targets": TARGETS,
        "filters": dict(counters),
        "stats": stats,
        "outputs": {**output_hashes, "audit_200": audit_hash},
        "shadow_manifest_sha256": digest(args.shadow_manifest.read_bytes()),
        "public_regression_sha256": digest(args.public_regression.read_bytes()) if args.public_regression.is_file() else None,
        "public_regression_manifest_sha256": digest(args.public_manifest.read_bytes()),
        "leakage": {
            "algorithm": "cnm.leakage-exclusion.v1",
            "normalized_patch": "path/hunk metadata removed; changed lines normalized",
            "changed_token_minhash": "smallest 8 SHA-256 hashes of 5-token shingles; sharing >=2 is overlap",
            "target_token_minhash": "smallest 8 SHA-256 hashes of 3-token shingles; sharing >=2 is overlap",
            "hidden_patch_exclusions": len(exclusions["patches"]),
            "hidden_changed_pair_exclusions": len(exclusions["changed_pairs"]),
            "hidden_target_exclusions": len(exclusions["targets"]),
            "hidden_target_pair_exclusions": len(exclusions["target_pairs"]),
            "repository_group_map": {item.repo: item.component for item in all_selected},
            "repository_group_algorithm": "same basename OR shared Git commit OR at least two shared normalized patches",
            "repository_group_evidence": component_evidence,
        },
        "audit_policy": {"rows": 200, "required_critical_errors": 0, "required_fully_grounded_fraction": 0.95},
    }
    args.manifest.parent.mkdir(parents=True, exist_ok=True)
    args.manifest.write_text(json.dumps(manifest, ensure_ascii=False, indent=2, default=dict) + "\n", encoding="utf-8")
    print(json.dumps({"status": manifest["status"], "filters": manifest["filters"], "stats": stats}, ensure_ascii=False, indent=2, default=dict))


if __name__ == "__main__":
    main()
