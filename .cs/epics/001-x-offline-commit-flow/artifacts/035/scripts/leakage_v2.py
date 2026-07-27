#!/usr/bin/env python3
"""Leakage signatures that cover both hunked changes and metadata-only file sections."""
from __future__ import annotations

import hashlib
import itertools
import re

TOKEN_RE = re.compile(r"[A-Za-z0-9_./:-]+")
STRUCTURAL = (
    "diff --git ", "new file mode ", "deleted file mode ", "old mode ", "new mode ",
    "similarity index ", "dissimilarity index ", "rename from ", "rename to ",
    "copy from ", "copy to ", "Binary files ", "GIT binary patch",
)
EVIDENCE_STRUCTURAL = STRUCTURAL[1:]


def evidence_lines(diff: str) -> list[str]:
    values: list[str] = []
    for line in diff.splitlines():
        if line.startswith(EVIDENCE_STRUCTURAL):
            values.append(line.strip())
        elif line.startswith(("+", "-")) and not line.startswith(("+++ ", "--- ")):
            value = line[1:].strip()
            if value:
                values.append(value)
    return values


def changed_content(diff: str) -> str:
    lines: list[str] = []
    for line in diff.splitlines():
        if line.startswith(STRUCTURAL):
            lines.append(line.strip())
        elif line.startswith("@@"):
            lines.append(re.sub(r"^@@[^@]*@@", "@@", line).strip())
        elif line.startswith(("+++ ", "--- ")):
            lines.append(line.strip())
        elif line.startswith(("+", "-")):
            lines.append(line.rstrip())
    return "\n".join(lines)


def minhashes(text: str, count: int = 8) -> list[int]:
    tokens = TOKEN_RE.findall(text.lower())
    if not tokens:
        return []
    shingles = [" ".join(tokens[index:index + 5]) for index in range(max(1, len(tokens) - 4))]
    return [min(int.from_bytes(hashlib.blake2b(f"{seed}:{shingle}".encode(), digest_size=8).digest(), "big") for shingle in shingles) for seed in range(count)]


def signatures(diff: str) -> dict:
    content = changed_content(diff)
    return {
        "content_sha256": hashlib.sha256(content.encode()).hexdigest(),
        "minhash": minhashes(content),
        "content_lines": len(content.splitlines()),
    }


def near_keys(values: list[int] | tuple[int, ...], required_matches: int = 7) -> set[tuple[tuple[int, int], ...]]:
    if len(values) < required_matches:
        return set()
    return {
        tuple((index, values[index]) for index in indices)
        for indices in itertools.combinations(range(len(values)), required_matches)
    }


def near_duplicate(left: list[int], right: list[int], threshold: float = 0.8) -> bool:
    if not left or not right or len(left) != len(right):
        return False
    return sum(a == b for a, b in zip(left, right)) / len(left) >= threshold
