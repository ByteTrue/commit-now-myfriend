#!/usr/bin/env python3
"""Build and measure the conservative macOS size skeleton for cnm #34."""
from __future__ import annotations

import hashlib
import json
import shutil
from pathlib import Path

ROOT = Path("/tmp/cnm-pierce34")
MODEL = ROOT / "skeleton/model.gguf"
INSTALL = ROOT / "skeleton/install"
EVIDENCE = Path(".cs/epics/001-o-offline-commit-flow/artifacts/034/size-skeleton.json")
EXPECTED_MODEL_BYTES = 522_186_624
EXPECTED_MODEL_SHA256 = "7f6a0b1670c9fff2cccbf8746dff19bb9c023708c8ca7702e4d49805985ecbac"
CAP = 700_000_000
GROWTH_RESERVE = 32_000_000

FILES = {
    "bin/llama-server": "/opt/homebrew/Cellar/llama.cpp/9430/bin/llama-server",
    "lib/libllama-server-impl.dylib": "/opt/homebrew/Cellar/llama.cpp/9430/lib/libllama-server-impl.dylib",
    "lib/libllama-common.0.0.9430.dylib": "/opt/homebrew/Cellar/llama.cpp/9430/lib/libllama-common.0.0.9430.dylib",
    "lib/libmtmd.0.0.9430.dylib": "/opt/homebrew/Cellar/llama.cpp/9430/lib/libmtmd.0.0.9430.dylib",
    "lib/libllama.0.0.9430.dylib": "/opt/homebrew/Cellar/llama.cpp/9430/lib/libllama.0.0.9430.dylib",
    "lib/libggml.0.13.1.dylib": "/opt/homebrew/Cellar/ggml/0.13.1/lib/libggml.0.13.1.dylib",
    "lib/libggml-base.0.13.1.dylib": "/opt/homebrew/Cellar/ggml/0.13.1/lib/libggml-base.0.13.1.dylib",
    "lib/libssl.3.dylib": "/opt/homebrew/Cellar/openssl@3/3.6.2/lib/libssl.3.dylib",
    "lib/libcrypto.3.dylib": "/opt/homebrew/Cellar/openssl@3/3.6.2/lib/libcrypto.3.dylib",
    "lib/libomp.dylib": "/opt/homebrew/Cellar/libomp/22.1.7/lib/libomp.dylib",
    "licenses/QWEN-APACHE-2.0.txt": "/tmp/cnm-pierce34/licenses/QWEN-APACHE-2.0.txt",
    "licenses/LLAMA-MIT.txt": "/opt/homebrew/Cellar/llama.cpp/9430/LICENSE",
    "licenses/GGML-MIT.txt": "/opt/homebrew/Cellar/ggml/0.13.1/LICENSE",
    "licenses/OPENSSL-APACHE-2.0.txt": "/opt/homebrew/Cellar/openssl@3/3.6.2/LICENSE.txt",
    "licenses/LIBOMP-APACHE-2.0-LLVM.txt": "/opt/homebrew/Cellar/libomp/22.1.7/LICENSE.TXT",
}

ALIASES = {
    "lib/libllama-common.0.dylib": "libllama-common.0.0.9430.dylib",
    "lib/libmtmd.0.dylib": "libmtmd.0.0.9430.dylib",
    "lib/libllama.0.dylib": "libllama.0.0.9430.dylib",
    "lib/libggml.0.dylib": "libggml.0.13.1.dylib",
    "lib/libggml-base.0.dylib": "libggml-base.0.13.1.dylib",
}


def sha256(path: Path) -> str:
    digest = hashlib.sha256()
    with path.open("rb") as stream:
        while block := stream.read(1024 * 1024):
            digest.update(block)
    return digest.hexdigest()


def main() -> None:
    if MODEL.stat().st_size != EXPECTED_MODEL_BYTES or sha256(MODEL) != EXPECTED_MODEL_SHA256:
        raise SystemExit("pinned Q5 model size/hash mismatch")
    if INSTALL.exists():
        shutil.rmtree(INSTALL)
    for relative, source_name in FILES.items():
        source = Path(source_name)
        if not source.is_file():
            raise SystemExit(f"missing skeleton input: {source}")
        target = INSTALL / relative
        target.parent.mkdir(parents=True, exist_ok=True)
        shutil.copy2(source, target)
    shutil.copy2(MODEL, INSTALL / "model.gguf")
    launcher = INSTALL / "bin/cnm"
    launcher.write_text("#!/bin/sh\nexec \"$(dirname \"$0\")/llama-server\" \"$@\"\n", encoding="utf-8")
    launcher.chmod(0o755)
    for relative, target_name in ALIASES.items():
        link = INSTALL / relative
        link.symlink_to(target_name)

    rows = []
    logical_bytes = 0
    for path in sorted(INSTALL.rglob("*")):
        relative = str(path.relative_to(INSTALL))
        if path.is_symlink():
            rows.append({"path": relative, "kind": "symlink", "target": str(path.readlink()), "bytes": 0})
        elif path.is_file():
            size = path.stat().st_size
            logical_bytes += size
            rows.append({"path": relative, "kind": "file", "bytes": size,
                         "sha256": EXPECTED_MODEL_SHA256 if relative == "model.gguf" else sha256(path)})
    projected = logical_bytes + GROWTH_RESERVE
    result = {
        "status": "pass" if projected <= CAP else "fail",
        "layout": str(INSTALL),
        "cap_bytes": CAP,
        "logical_file_bytes": logical_bytes,
        "growth_reserve_bytes": GROWTH_RESERVE,
        "projected_bytes": projected,
        "headroom_bytes": CAP - projected,
        "runtime_source": {"llama_cpp": "Homebrew 9430", "ggml": "0.13.1", "openssl": "3.6.2", "libomp": "22.1.7"},
        "caveat": "Pre-training macOS size skeleton only; final fused runtime must be made standalone, stripped, tested, and remeasured.",
        "files": rows,
    }
    EVIDENCE.parent.mkdir(parents=True, exist_ok=True)
    EVIDENCE.write_text(json.dumps(result, indent=2) + "\n", encoding="utf-8")
    print(json.dumps({key: result[key] for key in ("status", "logical_file_bytes", "growth_reserve_bytes", "projected_bytes", "headroom_bytes")}, indent=2))
    if result["status"] != "pass":
        raise SystemExit(2)


if __name__ == "__main__":
    main()
