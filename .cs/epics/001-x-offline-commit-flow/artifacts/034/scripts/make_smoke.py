#!/usr/bin/env python3
"""Create longest-row smoke data and exact short variants of frozen training configs."""
from __future__ import annotations

import argparse
import hashlib
import json
from pathlib import Path

import yaml


def sha(path: Path) -> str:
    return hashlib.sha256(path.read_bytes()).hexdigest()


def main() -> None:
    parser = argparse.ArgumentParser()
    parser.add_argument("--data", type=Path, default=Path("/tmp/cnm-pierce34/data/augmented"))
    parser.add_argument("--out", type=Path, default=Path("/tmp/cnm-pierce34/data/smoke"))
    parser.add_argument("--artifact", type=Path, default=Path(".cs/epics/001-o-offline-commit-flow/artifacts/034"))
    args = parser.parse_args()
    rows = [json.loads(line) for line in (args.data / "train.jsonl").read_text().splitlines() if line.strip()]
    longest = max(rows, key=lambda row: int(row["meta"]["input_tokens"]))
    args.out.mkdir(parents=True, exist_ok=True)
    hashes = {}
    for split, count in (("train", 32), ("valid", 4), ("test", 4)):
        path = args.out / f"{split}.jsonl"
        with path.open("w") as stream:
            for index in range(count):
                copy = json.loads(json.dumps(longest))
                copy["meta"]["smoke_copy"] = index
                stream.write(json.dumps(copy, ensure_ascii=False, separators=(",", ":")) + "\n")
        hashes[split] = sha(path)
    configs = {}
    for name in ("a-r16", "b-r32"):
        source = args.artifact / f"config-{name}.yaml"
        config = yaml.safe_load(source.read_text())
        config.update({
            "data": str(args.out),
            "iters": 20,
            "val_batches": 4,
            "steps_per_report": 10,
            "steps_per_eval": 10,
            "adapter_path": f"/tmp/cnm-pierce34/smoke/{name}",
            "save_every": 10,
        })
        target = args.artifact / f"smoke-config-{name}.yaml"
        target.write_text(yaml.safe_dump(config, sort_keys=False))
        configs[name] = {
            "full_config_sha256": sha(source),
            "smoke_config_sha256": sha(target),
            "only_shortened_fields": ["data", "iters", "val_batches", "steps_per_report", "steps_per_eval", "adapter_path", "save_every"],
        }
    manifest = {
        "status": "frozen_pre_training",
        "longest_family": longest["meta"]["family"],
        "longest_input_tokens": longest["meta"]["input_tokens"],
        "source_train_sha256": sha(args.data / "train.jsonl"),
        "smoke_data_hashes": hashes,
        "configs": configs,
    }
    path = args.artifact / "smoke-manifest.json"
    path.write_text(json.dumps(manifest, indent=2) + "\n")
    print(json.dumps(manifest, indent=2))


if __name__ == "__main__":
    main()
