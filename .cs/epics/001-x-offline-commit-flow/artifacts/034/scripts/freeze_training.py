#!/usr/bin/env python3
"""Freeze every #34 training/evaluation input before any candidate output is generated."""
from __future__ import annotations

import hashlib
import json
from pathlib import Path

ROOT = Path(".cs/epics/001-o-offline-commit-flow/artifacts/034")
DATA = Path("/tmp/cnm-pierce34/data")


def sha(path: Path) -> str:
    return hashlib.sha256(path.read_bytes()).hexdigest()


def main() -> None:
    accepted = json.loads((DATA / "accepted.json").read_text())
    if accepted.get("status") != "accepted":
        raise SystemExit("data audit has not accepted the frozen corpus")
    smoke = json.loads((ROOT / "smoke-manifest.json").read_text())
    if smoke.get("status") != "frozen_pre_training":
        raise SystemExit("smoke inputs are not frozen")
    scripts = sorted((ROOT / "scripts").glob("*.py"))
    inputs = [
        ROOT / "base-model-manifest.json",
        ROOT / "data-manifest.json",
        ROOT / "data-redesign-1.json",
        ROOT / "audit-rubric.md",
        ROOT / "guidance-manifest.json",
        ROOT / "public-regression-manifest.json",
        ROOT / "shadow-manifest.json",
        ROOT / "smoke-manifest.json",
        ROOT / "training-environment.txt",
        ROOT / "tokenizer-compatibility.json",
        ROOT / "config-a-r16.yaml",
        ROOT / "config-b-r32.yaml",
        ROOT / "smoke-config-a-r16.yaml",
        ROOT / "smoke-config-b-r32.yaml",
        DATA / "accepted.json",
        *scripts,
    ]
    manifest = {
        "status": "frozen_pre_output",
        "candidate_budget": 2,
        "selection_rule": "lowest frozen validation loss; tie -> smaller rank",
        "clean_retrain_required": True,
        "shadow_open_count": 0,
        "files": {str(path): sha(path) for path in inputs},
    }
    path = ROOT / "training-freeze.json"
    path.write_text(json.dumps(manifest, indent=2) + "\n")
    print(json.dumps({"files": len(inputs), "sha256": sha(path)}, indent=2))


if __name__ == "__main__":
    main()
