#!/usr/bin/env python3
"""Verify the exact locked Python environment used by #35."""
from __future__ import annotations

import argparse
import hashlib
import importlib.metadata
import json
import re
import shutil
import subprocess
import sys
from pathlib import Path

ARTIFACT = Path(__file__).resolve().parents[1]
LOCK = ARTIFACT / "requirements.lock"
MANIFEST = ARTIFACT / "environment-manifest.json"
REQUIREMENT = re.compile(r"^([A-Za-z0-9_.-]+)==([^\\ ]+)")


def digest(path: Path) -> str:
    value = hashlib.sha256()
    with path.open("rb") as stream:
        for chunk in iter(lambda: stream.read(8 * 1024 * 1024), b""):
            value.update(chunk)
    return value.hexdigest()


def canonical(name: str) -> str:
    return re.sub(r"[-_.]+", "-", name).lower()


def locked() -> dict[str, str]:
    result = {}
    for line in LOCK.read_text().splitlines():
        match = REQUIREMENT.match(line)
        if match:
            result[canonical(match.group(1))] = match.group(2)
    if len(result) != 31:
        raise SystemExit(f"unexpected lock size: {len(result)}")
    return result


def current() -> dict:
    expected_versions = locked()
    actual_versions = {
        canonical(distribution.metadata["Name"]): distribution.version
        for distribution in importlib.metadata.distributions()
    }
    if actual_versions != expected_versions:
        missing = sorted(set(expected_versions) - set(actual_versions))
        extra = sorted(set(actual_versions) - set(expected_versions))
        wrong = sorted(name for name in set(actual_versions) & set(expected_versions) if actual_versions[name] != expected_versions[name])
        raise SystemExit(f"environment distribution mismatch: missing={missing}, extra={extra}, wrong={wrong}")
    if sys.version_info[:3] != (3, 12, 13):
        raise SystemExit(f"unexpected Python: {sys.version}")
    if Path(sys.prefix).resolve() == Path(sys.base_prefix).resolve():
        raise SystemExit("locked packages must execute in uv's isolated environment, not the bare base interpreter")
    base_executable = (Path(sys.base_prefix) / "bin/python3.12").resolve()
    uv = Path(shutil.which("uv") or "").resolve()
    uv_version = subprocess.check_output([str(uv), "--version"], text=True).strip()
    if uv_version != "uv 0.11.14 (3fdfdc7d4 2026-05-12 aarch64-apple-darwin)":
        raise SystemExit(f"unexpected uv: {uv_version}")
    return {
        "python": ".".join(map(str, sys.version_info[:3])),
        "base_python_executable": str(base_executable),
        "base_python_executable_sha256": digest(base_executable),
        "isolated_environment_required": True,
        "runner_command": "uv run --python 3.12 --with-requirements .cs/epics/001-o-offline-commit-flow/artifacts/035/requirements.lock python <script>",
        "uv": uv_version,
        "uv_executable": str(uv),
        "uv_executable_sha256": digest(uv),
        "requirements_lock_sha256": digest(LOCK),
        "packages": expected_versions,
    }


def main() -> None:
    parser = argparse.ArgumentParser()
    parser.add_argument("--write-manifest", action="store_true")
    args = parser.parse_args()
    value = current()
    if args.write_manifest:
        MANIFEST.write_text(json.dumps({"status": "frozen_pre_output", **value}, indent=2) + "\n")
    else:
        expected = json.loads(MANIFEST.read_text())
        if expected != {"status": "frozen_pre_output", **value}:
            raise SystemExit("executing environment differs from frozen manifest")
    print(json.dumps({"status": "pass", "python": value["python"], "packages": len(value["packages"])}))


if __name__ == "__main__":
    main()
