#!/usr/bin/env python3
"""Run both frozen critics over the #37 population. One-shot, stateless, no retries."""
from __future__ import annotations

import argparse
import hashlib
import json
import sys
import time
import urllib.request
from pathlib import Path

ARTIFACT = Path(__file__).resolve().parents[1]
POPULATION = Path("/tmp/cnm-pierce37/data/population-200.jsonl")
OUT = Path("/tmp/cnm-pierce37/critics")

CRITIC_VIEWS = [
    ("support", "support-critic-prompt.txt", "support-critic-schema.json"),
    ("completeness", "completeness-critic-prompt.txt", "completeness-critic-schema.json"),
]


def digest(data: bytes) -> str:
    return hashlib.sha256(data).hexdigest()


def load_resource(name: str) -> str:
    return (ARTIFACT / name).read_text()


def api_request(url: str, body: dict, timeout: int = 120) -> dict:
    data = json.dumps(body, ensure_ascii=False).encode()
    req = urllib.request.Request(url, data=data, headers={"Content-Type": "application/json"})
    with urllib.request.urlopen(req, timeout=timeout) as resp:
        return json.loads(resp.read())


def run_critic(view_name: str, prompt: str, schema: str, manifest: dict, population: list[dict]) -> list[dict]:
    url = f"http://{manifest['runtime']['host']}:{manifest['runtime']['port']}/v1/chat/completions"
    decode = manifest["decode"]
    results: list[dict] = []
    out_path = OUT / f"{view_name}-decisions.jsonl"
    out_path.parent.mkdir(parents=True, exist_ok=True)

    with out_path.open("w") as f:
        for i, item in enumerate(population):
            diff = item["diff"]
            target = json.dumps({
                "type": None,
                "scope": None,
                "subject": item["subject"],
                "body": item["body"],
            }, ensure_ascii=False, separators=(",", ":"))

            body_policy = item["body_policy"]
            user_content = f"COMPLETE_DIFF:\n{diff}\n\nCANDIDATE_TARGET:\n{target}\n\nBODY_POLICY: {body_policy}"

            request_body = {
                "model": "local-teacher",
                "messages": [
                    {"role": "system", "content": prompt},
                    {"role": "user", "content": user_content},
                ],
                "temperature": decode["temperature"],
                "top_p": decode["top_p"],
                "seed": decode["seed"],
                "max_tokens": decode["max_tokens"],
                "response_format": {
                    "type": "json_schema",
                    "json_schema": {
                        "name": f"{view_name}_critic",
                        "schema": json.loads(schema),
                        "strict": True,
                    },
                },
            }

            start = time.monotonic()
            try:
                resp = api_request(url, request_body, timeout=120)
                content = resp["choices"][0]["message"]["content"]
                parsed = json.loads(content)
                status = "ok"
            except Exception as exc:
                parsed = {"decision": "reject", "reason_codes": ["outside_diff_context"], "evidence": []}
                status = f"error: {exc}"

            elapsed = time.monotonic() - start

            result = {
                "index": i,
                "view": view_name,
                "decision": parsed.get("decision", "reject"),
                "reason_codes": parsed.get("reason_codes", []),
                "evidence": parsed.get("evidence", []),
                "status": status,
                "elapsed": round(elapsed, 3),
            }
            results.append(result)
            f.write(json.dumps(result, ensure_ascii=False, separators=(",", ":")) + "\n")

            if (i + 1) % 20 == 0:
                accepted = sum(1 for r in results if r["decision"] == "accept")
                print(f"  {view_name}: {i + 1}/200 ({accepted} accepted)", flush=True)

    accepted = sum(1 for r in results if r["decision"] == "accept")
    print(f"  {view_name}: DONE {accepted}/200 accepted", flush=True)
    return results


def main() -> None:
    parser = argparse.ArgumentParser()
    parser.add_argument("--views", nargs="*", default=["support", "completeness"])
    parser.add_argument("--manifest", type=Path, default=ARTIFACT / "critic-manifest.json")
    args = parser.parse_args()

    manifest = json.loads(args.manifest.read_text())
    population = [json.loads(line) for line in POPULATION.read_text().splitlines() if line]
    assert len(population) == 200, f"Expected 200, got {len(population)}"

    print(f"Running critics on {len(population)} rows", flush=True)
    print(f"Server: {manifest['runtime']['host']}:{manifest['runtime']['port']}", flush=True)

    all_results: dict[str, list[dict]] = {}
    for view_name, prompt_file, schema_file in CRITIC_VIEWS:
        if view_name not in args.views:
            continue
        prompt = load_resource(prompt_file)
        schema = load_resource(schema_file)
        all_results[view_name] = run_critic(view_name, prompt, schema, manifest, population)

    # Compute intersection
    if len(all_results) == 2:
        support = {r["index"]: r for r in all_results["support"]}
        comp = {r["index"]: r for r in all_results["completeness"]}
        accepted = []
        rejected_support = []
        rejected_comp = []
        rejected_both = []
        for i in range(200):
            s = support[i]["decision"] == "accept"
            c = comp[i]["decision"] == "accept"
            if s and c:
                accepted.append(i)
            elif s and not c:
                rejected_comp.append(i)
            elif not s and c:
                rejected_support.append(i)
            else:
                rejected_both.append(i)

        summary = {
            "accepted_both": len(accepted),
            "rejected_support_only": len(rejected_support),
            "rejected_comp_only": len(rejected_comp),
            "rejected_both": len(rejected_both),
            "accepted_indices": accepted,
        }
        (OUT / "intersection.json").write_text(json.dumps(summary, ensure_ascii=False, indent=2) + "\n")
        print(f"\nIntersection: {len(accepted)} accepted by both critics", flush=True)

        # Check yield thresholds
        gates = json.loads((ARTIFACT / "gates.json").read_text())
        yield_req = gates["yield"]
        accepted_set = set(accepted)
        accepted_pop = [population[i] for i in accepted]

        bins = {}
        for item in accepted_pop:
            bins[item["file_bin"]] = bins.get(item["file_bin"], 0) + 1
        high_token = sum(1 for item in accepted_pop if item["input_tokens"] >= 4096)
        body_req = sum(1 for item in accepted_pop if item["body_policy"] == "required")

        yield_report = {
            "total": len(accepted),
            "total_min": yield_req["total"],
            "single": bins.get("single", 0),
            "single_min": yield_req["single"],
            "medium": bins.get("medium", 0),
            "medium_min": yield_req["medium"],
            "large": bins.get("large", 0),
            "large_min": yield_req["large"],
            "high_token": high_token,
            "high_token_min": yield_req["high_token"],
            "body_required": body_req,
            "body_required_min": yield_req["body_required"],
        }

        yield_ok = all([
            len(accepted) >= yield_req["total"],
            bins.get("single", 0) >= yield_req["single"],
            bins.get("medium", 0) >= yield_req["medium"],
            bins.get("large", 0) >= yield_req["large"],
            high_token >= yield_req["high_token"],
            body_req >= yield_req["body_required"],
        ])

        yield_report["yield_ok"] = yield_ok
        (OUT / "yield-report.json").write_text(json.dumps(yield_report, ensure_ascii=False, indent=2) + "\n")
        print(f"Yield OK: {yield_ok}", flush=True)
        if not yield_ok:
            print("STOP: insufficient yield", flush=True)


if __name__ == "__main__":
    main()