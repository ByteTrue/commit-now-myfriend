#!/usr/bin/env python3
"""Build hashed v2 leakage exclusions without reading source commit messages."""
from __future__ import annotations

import argparse
import importlib.util
import json
import sys
from pathlib import Path

import pyarrow.dataset as ds

HERE = Path(__file__).resolve().parent
ARTIFACT = HERE.parent
ROOT = Path("/tmp/cnm-pierce35")
CS34_SCRIPTS = ARTIFACT.parent / "034/scripts"
sys.path.insert(0, str(CS34_SCRIPTS))


def load_module(name: str, path: Path):
    spec = importlib.util.spec_from_file_location(name, path)
    module = importlib.util.module_from_spec(spec)
    sys.modules[name] = module
    spec.loader.exec_module(module)
    return module


base = load_module("build_dataset_034_v2_exclusions", CS34_SCRIPTS / "build_dataset.py")
v2 = load_module("leakage_v2_exclusions", HERE / "leakage_v2.py")


def jsonl(path: Path):
    with path.open() as stream:
        for line in stream:
            if line.strip():
                yield json.loads(line)


def main() -> None:
    parser = argparse.ArgumentParser()
    parser.add_argument("--output", type=Path, default=ARTIFACT / "leakage-v2-exclusions.json")
    args = parser.parse_args()

    prior = json.loads((ARTIFACT / "prior-audit-exclusions.json").read_text())
    wanted = set(prior["families"])
    signatures: list[dict] = []
    source_counts = {"prior_audits": 0, "public_26": 0, "private_shadow": 0, "private_historical": 0}

    seen_prior: set[str] = set()
    source_paths = [Path("/tmp/cnm-pierce34/source") / name for name in sorted(base.SOURCE_FILES)]
    dataset = ds.dataset([str(path) for path in source_paths], format="parquet")
    scanner = dataset.scanner(columns=["repo", "hash", "language", "license", "mods"], batch_size=512)
    for batch in scanner.to_batches():
        for row in batch.to_pylist():
            repo = base.canonical_repo(row.get("repo") or "")
            commit = row.get("hash") or ""
            family = f"commitchronicle:{repo}:{commit}"
            if family not in wanted or family in seen_prior:
                continue
            result = base.make_diff(row.get("mods") or [])
            if not result:
                raise SystemExit(f"prior audit family no longer reconstructs: {family}")
            diff, _paths = result
            signatures.append(v2.signatures(diff))
            seen_prior.add(family)
            source_counts["prior_audits"] += 1
    if seen_prior != wanted:
        missing = sorted(wanted - seen_prior)
        raise SystemExit(f"missing prior audit families: {missing[:5]} ({len(missing)} total)")

    raw_sources = [
        ("public_26", Path("/tmp/cnm-pierce34/eval/public-26.jsonl")),
        ("private_shadow", Path("/tmp/cnm-pierce34/private/shadow.jsonl")),
        ("private_historical", Path("/tmp/cnm-pierce34/private/historical.jsonl")),
    ]
    for name, path in raw_sources:
        for row in jsonl(path):
            signatures.append(v2.signatures(row["diff"]))
            source_counts[name] += 1

    unique = {(item["content_sha256"], tuple(item["minhash"])): item for item in signatures}
    output = {
        "schema": "cnm-leakage-v2-exclusions-v1",
        "sources": source_counts,
        "raw_signature_count": len(signatures),
        "unique_signature_count": len(unique),
        "content_sha256": sorted({item["content_sha256"] for item in unique.values()}),
        "minhash": sorted({tuple(item["minhash"]) for item in unique.values() if item["minhash"]}),
        "raw_diffs_persisted": False,
        "source_message_fields_accessed": False,
        "inputs": {
            "commit_chronicle_revision": base.SOURCE_REVISION,
            "commit_chronicle_files": base.SOURCE_FILES,
            "prior_audit_exclusions_sha256": base.digest((ARTIFACT / "prior-audit-exclusions.json").read_bytes()),
            "public_26_sha256": base.digest(Path("/tmp/cnm-pierce34/eval/public-26.jsonl").read_bytes()),
            "private_shadow_sha256": base.digest(Path("/tmp/cnm-pierce34/private/shadow.jsonl").read_bytes()),
            "private_historical_sha256": base.digest(Path("/tmp/cnm-pierce34/private/historical.jsonl").read_bytes()),
        },
    }
    output["minhash"] = [list(item) for item in output["minhash"]]
    args.output.write_text(json.dumps(output, indent=2, sort_keys=True) + "\n")
    print(json.dumps({"output": str(args.output), **source_counts, "unique": len(unique)}))


if __name__ == "__main__":
    main()
