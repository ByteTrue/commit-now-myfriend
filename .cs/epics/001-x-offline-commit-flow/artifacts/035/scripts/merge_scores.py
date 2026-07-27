#!/usr/bin/env python3
"""Merge two bound, independent #35 score sets without adjudication or label repair."""
from __future__ import annotations

import hashlib
import json
from collections import Counter
from pathlib import Path

ARTIFACT = Path(__file__).resolve().parents[1]
ROOT = Path("/tmp/cnm-pierce35")
REQUIRED = {
    "index", "reviewer", "input_slice_sha256", "critical_error", "fully_grounded",
    "subject_quality", "body_quality", "evidence_quality", "reason",
}


def digest(path: Path) -> str:
    return hashlib.sha256(path.read_bytes()).hexdigest()


def load_jsonl(path: Path) -> list[dict]:
    return [json.loads(line) for line in path.read_text().splitlines() if line]


def expected_slices(reviewer: str, manifest: dict) -> list[tuple[Path, Path, str]]:
    result = []
    for number in range(10):
        input_name = f"reviewer-{reviewer.lower()}-slice-{number:02d}.jsonl"
        input_path = ROOT / "audit" / input_name
        score_path = ROOT / "audit" / f"scores-{reviewer.lower()}" / input_name.replace(".jsonl", ".scores.jsonl")
        expected_hash = manifest["slices"].get(input_name)
        if expected_hash is None or digest(input_path) != expected_hash:
            raise SystemExit(f"missing or changed reviewer {reviewer} input: {input_path}")
        result.append((input_path, score_path, expected_hash))
    return result


def read_scores(reviewer: str, slices: list[tuple[Path, Path, str]]) -> tuple[dict[int, dict], dict[str, str]]:
    result: dict[int, dict] = {}
    hashes: dict[str, str] = {}
    for input_path, score_path, input_hash in slices:
        expected_indices = {row["index"] for row in load_jsonl(input_path)}
        if len(expected_indices) != 20 or not score_path.is_file():
            raise SystemExit(f"missing/bad reviewer {reviewer} slice: {score_path}")
        rows = load_jsonl(score_path)
        if len(rows) != 20 or {row.get("index") for row in rows} != expected_indices:
            raise SystemExit(f"reviewer {reviewer} scores do not bind to {input_path.name}")
        for row in rows:
            if set(row) != REQUIRED or row["index"] in result:
                raise SystemExit(f"invalid/duplicate reviewer {reviewer} score in {score_path}")
            if row["reviewer"] != reviewer or row["input_slice_sha256"] != input_hash:
                raise SystemExit(f"reviewer/slice identity mismatch in {score_path}")
            if not isinstance(row["critical_error"], bool) or not isinstance(row["fully_grounded"], bool):
                raise SystemExit(f"invalid boolean score in {score_path}")
            if row["subject_quality"] not in {0, 1, 2} or row["evidence_quality"] not in {0, 1, 2} or row["body_quality"] not in {None, 0, 1, 2}:
                raise SystemExit(f"invalid quality score in {score_path}")
            if not isinstance(row["reason"], str) or not row["reason"].strip():
                raise SystemExit(f"missing reviewer reason in {score_path}")
            result[row["index"]] = row
        hashes[score_path.name] = digest(score_path)
    if set(result) != set(range(200)):
        raise SystemExit(f"incomplete reviewer {reviewer} scores: {len(result)}/200")
    return result, hashes


def verify_provenance(score_hashes: dict[str, dict[str, str]]) -> dict:
    path = ROOT / "audit/reviewer-provenance.json"
    provenance = json.loads(path.read_text())
    if provenance.get("status") != "complete" or set(provenance.get("reviewers", {})) != {"A", "B"}:
        raise SystemExit("reviewer provenance is incomplete")
    run_sets = {}
    for reviewer in ("A", "B"):
        entry = provenance["reviewers"][reviewer]
        runs = set(entry.get("run_ids", []))
        if entry.get("context") != "fresh" or not runs or entry.get("score_hashes") != score_hashes[reviewer]:
            raise SystemExit(f"reviewer {reviewer} provenance mismatch")
        run_sets[reviewer] = runs
    if run_sets["A"] & run_sets["B"]:
        raise SystemExit("reviewer runs are not independent")
    return provenance


def expected_label_pid_counts(smoke_pid: int, full_pid: int) -> dict[str, int]:
    values = Counter({smoke_pid: 1})
    values[full_pid] += 199
    return {str(key): value for key, value in sorted(values.items())}


def main() -> None:
    review_manifest = json.loads((ROOT / "audit/review-input-manifest.json").read_text())
    if review_manifest["status"] != "frozen_pre_review":
        raise SystemExit("review inputs are not frozen")
    slices = {reviewer: expected_slices(reviewer, review_manifest) for reviewer in ("A", "B")}
    all_score_paths = [item[1].resolve() for values in slices.values() for item in values]
    if len(all_score_paths) != len(set(all_score_paths)):
        raise SystemExit("reviewers share score artifacts")
    a, hashes_a = read_scores("A", slices["A"])
    b, hashes_b = read_scores("B", slices["B"])
    provenance = verify_provenance({"A": hashes_a, "B": hashes_b})

    pilot = load_jsonl(ROOT / "data/pilot-200.jsonl")
    required_body = {row["index"] for row in pilot if row["body_policy"] == "required"}
    for index in required_body:
        if a[index]["body_quality"] is None or b[index]["body_quality"] is None:
            raise SystemExit(f"required body lacks a score: {index}")
    critical = [index for index in range(200) if a[index]["critical_error"] or b[index]["critical_error"]]
    grounded = [index for index in range(200) if a[index]["fully_grounded"] and b[index]["fully_grounded"]]
    subject_2 = [index for index in range(200) if a[index]["subject_quality"] == 2 and b[index]["subject_quality"] == 2]
    body_useful = [index for index in required_body if a[index]["body_quality"] >= 1 and b[index]["body_quality"] >= 1]
    body_2 = [index for index in required_body if a[index]["body_quality"] == 2 and b[index]["body_quality"] == 2]
    evidence_2 = [index for index in range(200) if a[index]["evidence_quality"] == 2 and b[index]["evidence_quality"] == 2]

    validation_path = ROOT / "labels/validation.json"
    validation = json.loads(validation_path.read_text())
    smoke_path = ROOT / "logs/teacher-smoke.json"
    smoke = json.loads(smoke_path.read_text())
    smoke_validation = json.loads((ROOT / "logs/smoke-validation.json").read_text())
    limit_check_path = ARTIFACT / "limit-check.json"
    limit_check = json.loads(limit_check_path.read_text())
    executable_limit = json.loads((ROOT / "logs/over-limit-entrypoint.json").read_text())
    freeze = json.loads((ARTIFACT / "freeze-manifest.json").read_text())
    smoke_case = pilot[smoke["case_index"]] if smoke.get("case_index") in range(200) else {}
    server_smoke = json.loads((ROOT / "logs/teacher-server-smoke-provenance.json").read_text())
    server_full = json.loads((ROOT / "logs/teacher-server-full-provenance.json").read_text())
    frozen_model = freeze["files"][str(ROOT / "model/qwen2.5-coder-14b-instruct-q6_k.gguf")]
    frozen_runtime = freeze["files"]["/opt/homebrew/bin/llama-server"]
    expected_command = [
        "/opt/homebrew/bin/llama-server", "--model", str(ROOT / "model/qwen2.5-coder-14b-instruct-q6_k.gguf"),
        "--alias", "local-teacher", "--host", "127.0.0.1", "--port", "63286", "--ctx-size", "9216",
        "--parallel", "1", "--n-gpu-layers", "99", "--threads", "12", "--flash-attn", "on", "--no-ui",
    ]
    def owned_teacher(item: dict, mode: str) -> bool:
        models = item.get("models_response", {}).get("data", []) if isinstance(item.get("models_response"), dict) else []
        served_path = item.get("props_model_path")
        return (
            item.get("status") == "pass"
            and item.get("mode") == mode
            and item.get("local_only") is True
            and item.get("server_url") == "http://127.0.0.1:63286"
            and item.get("listener_owner_verified") is True
            and isinstance(item.get("pid"), int)
            and item["pid"] > 0
            and item.get("model") == frozen_model
            and item.get("runtime") == frozen_runtime
            and item.get("command") == expected_command
            and [model.get("id") for model in models] == ["local-teacher"]
            and (served_path is None or Path(served_path).resolve() == (ROOT / "model/qwen2.5-coder-14b-instruct-q6_k.gguf").resolve())
        )
    expected_pids = (
        expected_label_pid_counts(server_smoke.get("pid"), server_full.get("pid"))
        if isinstance(server_smoke.get("pid"), int) and isinstance(server_full.get("pid"), int)
        else {}
    )
    checks = {
        "critical_errors_zero": len(critical) == 0,
        "fully_grounded_by_both_at_least_190": len(grounded) >= 190,
        "subject_quality_2_by_both_at_least_180": len(subject_2) >= 180,
        "body_required_count_60": len(required_body) == 60,
        "all_60_required_bodies_useful_by_both": len(body_useful) == 60,
        "body_quality_2_by_both_at_least_54": len(body_2) >= 54,
        "mechanical_validation_passed_200": validation.get("status") == "pass" and validation.get("cases") == 200,
        "mechanical_validation_is_frozen_review_input": digest(validation_path) == review_manifest.get("validation_sha256"),
        "near_limit_smoke_passed": smoke.get("status") == "pass" and 7169 <= smoke_case.get("input_tokens", 0) <= 8192,
        "near_limit_smoke_output_unchanged": smoke.get("label_sha256") == digest(ROOT / "labels" / f"{smoke['case_index']:03d}.json"),
        "near_limit_smoke_mechanically_valid": smoke_validation.get("status") == "pass" and smoke_validation.get("cases") == 1,
        "over_limit_rejected_before_inference": (
            limit_check.get("status") == "pass"
            and all(limit_check.get("checks", {}).values())
            and executable_limit.get("status") == "pass"
            and all(executable_limit.get("checks", {}).values())
            and executable_limit.get("returncode") == 42
        ),
        "over_limit_check_was_frozen": freeze["files"][str(limit_check_path)]["sha256"] == digest(limit_check_path),
        "frozen_local_teacher_owned_smoke_and_full": (
            owned_teacher(server_smoke, "smoke")
            and owned_teacher(server_full, "full")
            and server_smoke.get("pid") == smoke.get("server_pid")
            and server_smoke.get("smoke_index") == smoke.get("case_index")
            and server_full.get("smoke_index") is None
            and server_smoke.get("label_server_pid_counts") == {str(server_smoke.get("pid")): 1}
            and smoke_validation.get("server_pid_counts") == {str(server_smoke.get("pid")): 1}
            and server_full.get("label_server_pid_counts") == expected_pids
            and validation.get("server_pid_counts") == expected_pids
        ),
        "independent_reviewer_provenance_passed": provenance.get("status") == "complete",
    }
    report = {
        "status": "go" if all(checks.values()) else "stop",
        "checks": checks,
        "counts": {
            "critical_errors_from_either": len(critical),
            "fully_grounded_by_both": len(grounded),
            "subject_quality_2_by_both": len(subject_2),
            "body_useful_by_both": len(body_useful),
            "body_quality_2_by_both": len(body_2),
            "evidence_quality_2_by_both": len(evidence_2),
        },
        "critical_indices": critical,
        "score_hashes": {"reviewer_a": hashes_a, "reviewer_b": hashes_b},
        "reviewer_provenance_sha256": digest(ROOT / "audit/reviewer-provenance.json"),
        "per_case": [{"index": index, "reviewer_a": a[index], "reviewer_b": b[index]} for index in range(200)],
        "adjudication": None,
        "post_score_repairs": 0,
    }
    out = ROOT / "audit/gate-result.json"
    out.write_text(json.dumps(report, ensure_ascii=False, indent=2) + "\n")
    print(json.dumps({"status": report["status"], "counts": report["counts"], "checks": checks}))
    if report["status"] != "go":
        raise SystemExit(1)


if __name__ == "__main__":
    main()
