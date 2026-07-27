#!/usr/bin/env python3
"""Freeze all #35 pilot inputs before the first teacher output."""
from __future__ import annotations

import hashlib
import json
import subprocess
import sys
from pathlib import Path

ARTIFACT = Path(__file__).resolve().parents[1]
ROOT = Path("/tmp/cnm-pierce35")
OUT = ARTIFACT / "freeze-manifest.json"
ENVIRONMENT = json.loads((ARTIFACT / "environment-manifest.json").read_text())
FILES = [
    ARTIFACT / "issue-spec.md",
    ARTIFACT / "teacher-manifest.json",
    ARTIFACT / "teacher-prompt.txt",
    ARTIFACT / "teacher-schema.json",
    ARTIFACT / "audit-rubric.json",
    ARTIFACT / "audit-instructions.md",
    ARTIFACT / "prior-audit-exclusions.json",
    ARTIFACT / "leakage-v2-exclusions.json",
    ARTIFACT / "tokenizer-manifest.json",
    ARTIFACT.parent / "034/scripts/build_dataset.py",
    ARTIFACT.parent / "034/scripts/pipeline.py",
    Path("/tmp/cnm-pierce34/base/config.json"),
    Path("/tmp/cnm-pierce34/base/tokenizer.json"),
    Path("/tmp/cnm-pierce34/base/tokenizer_config.json"),
    Path("/tmp/cnm-pierce34/base/special_tokens_map.json"),
    Path("/tmp/cnm-pierce34/base/added_tokens.json"),
    Path("/tmp/cnm-pierce34/base/merges.txt"),
    Path("/tmp/cnm-pierce34/base/vocab.json"),
    ARTIFACT / "teacher-environment.txt",
    ARTIFACT / "requirements.in",
    ARTIFACT / "requirements.lock",
    ARTIFACT / "environment-manifest.json",
    Path(ENVIRONMENT["base_python_executable"]),
    Path(ENVIRONMENT["uv_executable"]),
    ARTIFACT / "pilot-manifest.json",
    ARTIFACT / "pilot-reproducibility.json",
    ARTIFACT / "limit-check.json",
    ARTIFACT / "scripts/build_pilot.py",
    ARTIFACT / "scripts/build_leakage_exclusions.py",
    ARTIFACT / "scripts/leakage_v2.py",
    ARTIFACT / "scripts/label_pilot.py",
    ARTIFACT / "scripts/run_teacher.py",
    ARTIFACT / "scripts/verify_environment.py",
    ARTIFACT / "scripts/validate_labels.py",
    ARTIFACT / "scripts/test_pilot.py",
    ARTIFACT / "scripts/test_leakage_v2.py",
    ARTIFACT / "scripts/test_teacher_tools.py",
    ARTIFACT / "scripts/test_audit_tools.py",
    ARTIFACT / "scripts/smoke.py",
    ARTIFACT / "scripts/check_limits.py",
    ARTIFACT / "scripts/prepare_audit.py",
    ARTIFACT / "scripts/merge_scores.py",
    ARTIFACT / "scripts/freeze.py",
    ROOT / "data/pilot-200.jsonl",
    ROOT / "data/over-limit.jsonl",
    ROOT / "data/audit-plan.json",
    ROOT / "model/qwen2.5-coder-14b-instruct-q6_k.gguf",
    Path("/opt/homebrew/bin/llama-server"),
]


def digest(path: Path) -> str:
    value = hashlib.sha256()
    with path.open("rb") as stream:
        for chunk in iter(lambda: stream.read(8 * 1024 * 1024), b""):
            value.update(chunk)
    return value.hexdigest()


if any((ROOT / "labels").iterdir()):
    raise SystemExit("teacher output already exists; refusing late freeze")
subprocess.check_call([sys.executable, str(ARTIFACT / "scripts/verify_environment.py")])
missing = [str(path) for path in FILES if not path.is_file()]
if missing:
    raise SystemExit(f"missing freeze inputs: {missing}")
manifest = {
    "status": "frozen_pre_output",
    "files": {str(path): {"bytes": path.stat().st_size, "sha256": digest(path)} for path in FILES},
    "labels_directory_empty": True,
}
OUT.write_text(json.dumps(manifest, ensure_ascii=False, indent=2) + "\n")
print(json.dumps({"status": manifest["status"], "files": len(FILES), "sha256": digest(OUT)}))
