#!/usr/bin/env python3
"""Run one #34 MLX smoke and enforce the frozen M5 Pro hardware gate."""
from __future__ import annotations

import argparse
import hashlib
import json
import math
import os
import re
import signal
import statistics
import struct
import subprocess
import sys
import threading
import time
from pathlib import Path

import yaml

MAX_RECOMMENDED_WORKING_SET = 40_200_896_512
MAX_MLX_PEAK_BYTES = int(MAX_RECOMMENDED_WORKING_SET * 0.80)
MAX_MEDIAN_STEP_SECONDS = 2.0
MAX_PROJECTED_SECONDS = 6 * 60 * 60
MIN_FREE_PERCENT_ABORT = 8
NORMAL_PRESSURE_LEVEL = 1


def file_sha(path: Path) -> str:
    digest = hashlib.sha256()
    with path.open("rb") as stream:
        while block := stream.read(1024 * 1024):
            digest.update(block)
    return digest.hexdigest()


def vm_stats() -> dict[str, int]:
    output = subprocess.run(["vm_stat"], check=True, text=True, stdout=subprocess.PIPE).stdout
    page_size = int(re.search(r"page size of (\d+) bytes", output).group(1))
    values = {key: int(value.replace(".", "")) for key, value in re.findall(r"^([^:]+):\s+(\d+)\.$", output, re.M)}
    return {"page_size": page_size, **values}


def free_percent() -> int:
    output = subprocess.run(["memory_pressure", "-Q"], check=True, text=True, stdout=subprocess.PIPE).stdout
    return int(re.search(r"free percentage:\s*(\d+)%", output).group(1))


def pressure_level() -> int:
    output = subprocess.run(
        ["sysctl", "-n", "kern.memorystatus_vm_pressure_level"],
        check=True,
        text=True,
        stdout=subprocess.PIPE,
    ).stdout
    return int(output.strip())


def rss_kib(pid: int) -> int:
    result = subprocess.run(["ps", "-o", "rss=", "-p", str(pid)], text=True, stdout=subprocess.PIPE).stdout.strip()
    return int(result) if result else 0


def valid_safetensors(path: Path) -> bool:
    try:
        with path.open("rb") as stream:
            header_size = struct.unpack("<Q", stream.read(8))[0]
            if header_size <= 0 or header_size > min(10_000_000, path.stat().st_size - 8):
                return False
            header = json.loads(stream.read(header_size))
        tensors = [value for key, value in header.items() if key != "__metadata__"]
        return path.stat().st_size > 1_000_000 and bool(tensors) and all("dtype" in value and "shape" in value and "data_offsets" in value for value in tensors)
    except (OSError, ValueError, KeyError, MemoryError, json.JSONDecodeError, struct.error):
        return False


def main() -> int:
    parser = argparse.ArgumentParser()
    parser.add_argument("--out", type=Path, required=True)
    parser.add_argument("--log", type=Path, required=True)
    parser.add_argument("--config", type=Path, required=True)
    parser.add_argument("--runner", type=Path, required=True)
    parser.add_argument("--adapter-dir", type=Path, required=True)
    parser.add_argument("--smoke-iters", type=int, default=20)
    parser.add_argument("--projected-iters", type=int, required=True)
    parser.add_argument("command", nargs=argparse.REMAINDER)
    args = parser.parse_args()
    command = args.command[1:] if args.command[:1] == ["--"] else args.command
    if not command:
        raise SystemExit("missing command")
    expected_command = [str(Path(sys.executable).resolve()), str(args.runner.resolve()), "--config", str(args.config.resolve())]
    actual_command = (
        [str(Path(command[0]).resolve()), str(Path(command[1]).resolve()), command[2], str(Path(command[3]).resolve())]
        if len(command) == 4 else command
    )
    if actual_command != expected_command:
        raise SystemExit(f"smoke command must be the frozen invocation; expected={expected_command!r} actual={actual_command!r}")
    runner_sha_before = file_sha(args.runner)
    config_data = yaml.safe_load(args.config.read_text())
    if config_data.get("iters") != args.smoke_iters or Path(config_data.get("adapter_path", "")).resolve() != args.adapter_dir.resolve():
        raise SystemExit("smoke config iteration/adapter path mismatch")
    config_sha_before = file_sha(args.config)
    args.out.parent.mkdir(parents=True, exist_ok=True)
    args.log.parent.mkdir(parents=True, exist_ok=True)
    before = vm_stats()
    samples: list[dict] = []
    abort_reason = ""
    monitor_errors: list[str] = []
    stop = threading.Event()

    with args.log.open("w", encoding="utf-8") as log:
        started = time.perf_counter()
        timed_command = ["/usr/bin/time", "-l", *command]
        process = subprocess.Popen(timed_command, stdout=log, stderr=subprocess.STDOUT, text=True, start_new_session=True)

        def monitor() -> None:
            nonlocal abort_reason
            while not stop.wait(1):
                try:
                    sample = {
                        "elapsed": round(time.perf_counter() - started, 3),
                        "rss_kib": rss_kib(process.pid),
                        "free_percent": free_percent(),
                        "pressure_level": pressure_level(),
                    }
                    samples.append(sample)
                    current = vm_stats()
                    pageouts = current.get("Pageouts", 0) - before.get("Pageouts", 0)
                    swapouts = current.get("Swapouts", 0) - before.get("Swapouts", 0)
                    if sample["pressure_level"] != NORMAL_PRESSURE_LEVEL:
                        abort_reason = "memory pressure no longer normal"
                    elif sample["free_percent"] < MIN_FREE_PERCENT_ABORT:
                        abort_reason = f"memory free percentage below {MIN_FREE_PERCENT_ABORT}"
                    elif pageouts > 0 or swapouts > 0:
                        abort_reason = "pageout/swapout activity detected"
                    if abort_reason and process.poll() is None:
                        os.killpg(process.pid, signal.SIGINT)
                        return
                except (OSError, subprocess.SubprocessError, ValueError) as error:
                    monitor_errors.append(str(error))
                    abort_reason = "memory monitor failed closed"
                    if process.poll() is None:
                        os.killpg(process.pid, signal.SIGINT)
                    return

        thread = threading.Thread(target=monitor, daemon=True)
        thread.start()
        exit_code = process.wait()
        stop.set()
        thread.join(timeout=5)
        wall_seconds = time.perf_counter() - started
    after = vm_stats()
    text = args.log.read_text(errors="replace")
    telemetry_matches = re.findall(r"^CNM_MLX_TELEMETRY=(\{.*\})$", text, re.M)
    telemetry = json.loads(telemetry_matches[-1]) if len(telemetry_matches) == 1 else {}
    peak_bytes = int(telemetry.get("peak_memory", 0))
    rates = [float(item) for item in re.findall(r"It/sec ([0-9.]+)", text)]
    train_matches = re.findall(r"Iter (\d+): Train loss ([^,\s]+).*?Trained Tokens (\d+)", text)
    train_rows = [(int(step), float(loss), int(tokens)) for step, loss, tokens in train_matches]
    val_rows = [(int(step), float(loss)) for step, loss in re.findall(r"Iter (\d+): Val loss ([^,\s]+)", text)]
    actual_steps = max((step for step, _loss, _tokens in train_rows), default=0)
    finite_losses = bool(train_rows and val_rows) and all(math.isfinite(loss) for _step, loss, _tokens in train_rows) and all(math.isfinite(loss) for _step, loss in val_rows)
    median_step = 1 / statistics.median(rates) if rates else float("inf")
    projected = wall_seconds / actual_steps * args.projected_iters if actual_steps else float("inf")
    pageouts = after.get("Pageouts", 0) - before.get("Pageouts", 0)
    swapouts = after.get("Swapouts", 0) - before.get("Swapouts", 0)
    compressor = after.get("Pages occupied by compressor", 0) - before.get("Pages occupied by compressor", 0)
    rss_matches = [int(item) for item in re.findall(r"^\s*(\d+)\s+maximum resident set size$", text, re.M)]
    maximum_rss = rss_matches[-1] if len(rss_matches) == 1 else 0
    checkpoints = sorted(args.adapter_dir.glob("*.safetensors"))
    checkpoint_hashes = {path.name: file_sha(path) for path in checkpoints}
    exact_checkpoint = args.adapter_dir / f"{args.smoke_iters:07d}_adapters.safetensors"
    final_checkpoint = args.adapter_dir / "adapters.safetensors"
    checkpoint_format_valid = all(valid_safetensors(path) for path in (exact_checkpoint, final_checkpoint))
    adapter_config_path = args.adapter_dir / "adapter_config.json"
    adapter_config = json.loads(adapter_config_path.read_text()) if adapter_config_path.is_file() else {}
    expected_report_steps = list(range(int(config_data["steps_per_report"]), args.smoke_iters + 1, int(config_data["steps_per_report"])))
    observed_report_steps = [step for step, _loss, _tokens in train_rows]
    tokens_strictly_increase = all(right > left for left, right in zip([0, *[row[2] for row in train_rows[:-1]]], [row[2] for row in train_rows]))
    adapter_config_valid = (
        adapter_config.get("iters") == args.smoke_iters
        and Path(adapter_config.get("adapter_path", "")).resolve() == args.adapter_dir.resolve()
        and Path(adapter_config.get("model", "")).resolve() == Path(config_data["model"]).resolve()
        and adapter_config.get("lora_parameters") == config_data.get("lora_parameters")
    )
    config_stable = args.config.is_file() and file_sha(args.config) == config_sha_before
    runner_stable = args.runner.is_file() and file_sha(args.runner) == runner_sha_before
    observed_pressure = [item["pressure_level"] for item in samples]
    gate = {
        "exit_code": exit_code,
        "abort_reason": abort_reason,
        "monitor_errors": monitor_errors,
        "config": str(args.config),
        "config_sha256": config_sha_before,
        "config_stable": config_stable,
        "runner": str(args.runner),
        "runner_sha256": runner_sha_before,
        "runner_stable": runner_stable,
        "wall_seconds": round(wall_seconds, 3),
        "actual_train_step": actual_steps,
        "expected_report_steps": expected_report_steps,
        "observed_report_steps": observed_report_steps,
        "tokens_strictly_increase": tokens_strictly_increase,
        "validation_rows": len(val_rows),
        "finite_losses": finite_losses,
        "checkpoint_hashes": checkpoint_hashes,
        "checkpoint_format_valid": checkpoint_format_valid,
        "adapter_config_valid": adapter_config_valid,
        "maximum_rss_bytes": maximum_rss,
        "sampled_parent_rss_peak_bytes": max((item["rss_kib"] for item in samples), default=0) * 1024,
        "mlx_telemetry": telemetry,
        "mlx_peak_bytes": peak_bytes,
        "max_allowed_mlx_peak_bytes": MAX_MLX_PEAK_BYTES,
        "free_percent_min": min((item["free_percent"] for item in samples), default=free_percent()),
        "pressure_levels": sorted(set(observed_pressure)),
        "pageouts_delta": pageouts,
        "swapouts_delta": swapouts,
        "compressor_pages_delta": compressor,
        "median_step_seconds": round(median_step, 6),
        "projected_full_seconds": round(projected, 3),
        "samples": samples,
    }
    gate["pass"] = (
        exit_code == 0
        and not abort_reason
        and not monitor_errors
        and config_stable
        and runner_stable
        and actual_steps == args.smoke_iters
        and observed_report_steps == expected_report_steps
        and tokens_strictly_increase
        and finite_losses
        and len(val_rows) >= 1
        and bool(checkpoint_hashes)
        and checkpoint_format_valid
        and adapter_config_valid
        and len(telemetry_matches) == 1
        and 0 < peak_bytes < MAX_MLX_PEAK_BYTES
        and maximum_rss > 0
        and pageouts == 0
        and swapouts == 0
        and bool(observed_pressure)
        and all(level == NORMAL_PRESSURE_LEVEL for level in observed_pressure)
        and median_step <= MAX_MEDIAN_STEP_SECONDS
        and projected <= MAX_PROJECTED_SECONDS
    )
    args.out.write_text(json.dumps(gate, indent=2) + "\n")
    print(json.dumps({key: gate[key] for key in gate if key not in {"samples", "mlx_telemetry"}}, indent=2))
    return 0 if gate["pass"] else 2


if __name__ == "__main__":
    raise SystemExit(main())
