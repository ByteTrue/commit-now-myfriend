#!/usr/bin/env python3
"""Split frozen blind audits into bounded independent-review slices."""
from __future__ import annotations

import hashlib
import json
from pathlib import Path

ROOT = Path("/tmp/cnm-pierce34")


def sha(path: Path) -> str:
    return hashlib.sha256(path.read_bytes()).hexdigest()


def split(source: Path, prefix: str, expected: int) -> list[dict]:
    rows = [line for line in source.read_text().splitlines() if line.strip()]
    if len(rows) != expected:
        raise SystemExit(f"{prefix} audit expected {expected}, got {len(rows)}")
    out = ROOT / "audit" / "slices"
    out.mkdir(parents=True, exist_ok=True)
    manifest = []
    for index in range(0, len(rows), 20):
        path = out / f"{prefix}-{index // 20:02d}.jsonl"
        path.write_text("\n".join(rows[index:index + 20]) + "\n")
        manifest.append({"path": str(path), "rows": min(20, len(rows) - index), "sha256": sha(path)})
    return manifest


def main() -> None:
    base = ROOT / "data" / "audit-200.jsonl"
    guidance = ROOT / "data" / "augmented" / "audit-guidance-100.jsonl"
    manifest = {
        "status": "frozen_pre_audit",
        "base_source_sha256": sha(base),
        "guidance_source_sha256": sha(guidance),
        "base": split(base, "base", 200),
        "guidance": split(guidance, "guidance", 100),
    }
    path = ROOT / "audit" / "slices-manifest.json"
    path.write_text(json.dumps(manifest, indent=2) + "\n")
    print(json.dumps({"base_slices": len(manifest["base"]), "guidance_slices": len(manifest["guidance"]), "sha256": sha(path)}, indent=2))


if __name__ == "__main__":
    main()
