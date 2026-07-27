#!/usr/bin/env python3
"""Small resumable range downloader used only by cnm #34."""
from __future__ import annotations

import argparse
import subprocess
from concurrent.futures import ThreadPoolExecutor, as_completed
from pathlib import Path


def main() -> None:
    parser = argparse.ArgumentParser()
    parser.add_argument("url")
    parser.add_argument("output", type=Path)
    parser.add_argument("size", type=int)
    parser.add_argument("--workers", type=int, default=8)
    args = parser.parse_args()

    args.output.parent.mkdir(parents=True, exist_ok=True)
    part = args.output.with_suffix(args.output.suffix + ".part")
    prefix = part.stat().st_size if part.exists() else 0
    if prefix > args.size:
        raise SystemExit(f"partial too large: {prefix}")
    if prefix == args.size:
        part.replace(args.output)
        return

    chunk = (args.size - prefix + args.workers - 1) // args.workers
    ranges = []
    for index in range(args.workers):
        start = prefix + index * chunk
        end = min(args.size - 1, start + chunk - 1)
        if start <= end:
            ranges.append((index, start, end))

    def download(item: tuple[int, int, int]) -> tuple[int, int]:
        index, start, end = item
        destination = Path(f"{part}.{index}")
        expected = end - start + 1
        while (destination.stat().st_size if destination.exists() else 0) < expected:
            actual = destination.stat().st_size if destination.exists() else 0
            temporary = Path(f"{destination}.next")
            subprocess.run(
                ["curl", "--http1.1", "-L", "--fail", "--silent", "--show-error",
                 "--retry", "5", "--retry-all-errors", "--range", f"{start + actual}-{end}",
                 args.url, "-o", str(temporary)],
                check=True,
                timeout=900,
            )
            with destination.open("ab") as target, temporary.open("rb") as source:
                while block := source.read(1024 * 1024):
                    target.write(block)
            temporary.unlink()
            if destination.stat().st_size > expected:
                raise RuntimeError(f"range {index} exceeded expected bytes")
        return index, destination.stat().st_size

    print(f"prefix={prefix} ranges={len(ranges)}", flush=True)
    with ThreadPoolExecutor(max_workers=args.workers) as executor:
        futures = [executor.submit(download, item) for item in ranges]
        for future in as_completed(futures):
            index, size = future.result()
            print(f"range {index}: {size}", flush=True)

    with part.open("ab") as target:
        for index, _, _ in ranges:
            segment = Path(f"{part}.{index}")
            with segment.open("rb") as source:
                while block := source.read(1024 * 1024):
                    target.write(block)
            segment.unlink()
    if part.stat().st_size != args.size:
        raise SystemExit(f"assembled size mismatch: {part.stat().st_size}")
    part.replace(args.output)
    print(f"assembled {args.output}: {args.size}", flush=True)


if __name__ == "__main__":
    main()
