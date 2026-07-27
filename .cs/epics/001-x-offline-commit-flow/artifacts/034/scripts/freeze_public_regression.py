#!/usr/bin/env python3
"""Convert the immutable 26-case public corpus to #34's semantic-record schema."""
from __future__ import annotations

import argparse
import hashlib
import json
import re
import subprocess
from pathlib import Path

EXPECTED_SOURCE_SHA256 = "fdd9d4e587a913187ee849b2ba62bafbb3c5ea53058269b9278e6f6fa3a4175f"


def digest(value: str | bytes) -> str:
    return hashlib.sha256(value.encode() if isinstance(value, str) else value).hexdigest()


def target_signatures(message: str) -> tuple[str, list[str]]:
    normalized = re.sub(r"\s+", " ", message.lower()).strip()
    tokens = re.findall(r"[a-z0-9_]+|[^\s\w]", normalized)
    shingles = [" ".join(tokens[index:index + 3]) for index in range(max(0, len(tokens) - 2))]
    return digest(normalized), sorted({digest(value) for value in shingles})[:8]


def section(value: str, marker: str) -> str:
    if marker not in value:
        return ""
    return value.split(marker, 1)[1].strip()


def main() -> None:
    parser = argparse.ArgumentParser()
    parser.add_argument("--source", type=Path, default=Path("/tmp/cnm-train31/eval/high-risk.jsonl"))
    parser.add_argument("--out", type=Path, default=Path("/tmp/cnm-pierce34/eval/public-26.jsonl"))
    parser.add_argument("--repo", type=Path, default=Path.cwd())
    parser.add_argument("--manifest", type=Path, default=Path(".cs/epics/001-o-offline-commit-flow/artifacts/034/public-regression-manifest.json"))
    args = parser.parse_args()
    source_hash = hashlib.sha256(args.source.read_bytes()).hexdigest()
    if source_hash != EXPECTED_SOURCE_SHA256:
        raise SystemExit("public regression source hash changed")
    rows = []
    target_entries: dict[str, dict] = {}
    for line in args.source.read_text().splitlines():
        source = json.loads(line)
        user = source["user"]
        match = re.search(r"```diff\n(.*)\n```\s*$", user, re.S)
        if not match:
            raise SystemExit(f"cannot recover frozen diff: {source['id']}")
        guidance = section(source["system"], "Additional user guidance:\n")
        if "\nRecent commit messages (style reference only):\n" in guidance:
            guidance = guidance.split("\nRecent commit messages (style reference only):\n", 1)[0].strip()
        if source["style"] == "auto":
            history = subprocess.run(
                ["git", "log", "--no-merges", "--max-count=10", "--pretty=format:%s", f"{source['commit_hash']}^"],
                cwd=args.repo,
                check=True,
                text=True,
                stdout=subprocess.PIPE,
            ).stdout.splitlines()
        else:
            history = []
        requirements = {}
        case_id = source["id"]
        if case_id == "custom-chinese":
            requirements = {"simplified_chinese": True, "body_forbidden": True}
        elif case_id == "custom-bullets":
            requirements = {"body_required": True, "exact_body_bullets": 2}
        elif case_id == "custom-security-prefix":
            requirements = {"subject_prefix": "SECURITY:", "body_required": True}
        elif case_id == "guided-conventional-issue-body":
            requirements = {"body_required": True, "reference": "#123"}
        elif case_id == "guided-google-why-body":
            requirements = {"body_required": True}
        commit = source["commit_hash"]
        if commit not in target_entries:
            message = subprocess.run(
                ["git", "show", "-s", "--format=%B", commit],
                cwd=args.repo,
                check=True,
                text=True,
                stdout=subprocess.PIPE,
            ).stdout.strip()
            target_hash, target_minhash = target_signatures(message)
            target_entries[commit] = {
                "commit_sha256": digest(commit),
                "normalized_target_sha256": target_hash,
                "target_token_minhash": target_minhash,
            }
        rows.append({
            "id": case_id,
            "commit": source["commit_hash"],
            "diff": match.group(1),
            "diff_sha256": source["diff_sha256"],
            "style": source["style"],
            "guidance": guidance,
            "history": history,
            "requirements": requirements,
            "source": "cnm-frozen-26",
        })
    args.out.parent.mkdir(parents=True, exist_ok=True)
    with args.out.open("w", encoding="utf-8") as stream:
        for row in rows:
            stream.write(json.dumps(row, ensure_ascii=False, separators=(",", ":")) + "\n")
    manifest = {
        "status": "frozen_pre_output",
        "cases": len(rows),
        "source_sha256": source_hash,
        "output_sha256": hashlib.sha256(args.out.read_bytes()).hexdigest(),
        "target_signature_algorithm": "cnm.leakage-exclusion.v1",
        "target_entries": sorted(target_entries.values(), key=lambda item: item["commit_sha256"]),
    }
    args.manifest.parent.mkdir(parents=True, exist_ok=True)
    args.manifest.write_text(json.dumps(manifest, indent=2) + "\n")
    print(json.dumps(manifest, indent=2))


if __name__ == "__main__":
    main()
