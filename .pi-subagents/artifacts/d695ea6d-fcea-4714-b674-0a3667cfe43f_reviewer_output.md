## Review

- **Blocker — dependency freeze is incomplete:** `.cs/epics/001-o-offline-commit-flow/artifacts/035/teacher-environment.txt:3-7` records only direct packages and a few tokenizer dependencies. `transformers` runtime dependencies such as `huggingface-hub`, `safetensors`, `packaging`, `regex`, `requests`, `filelock`, `fsspec`, and `PyYAML` are neither version-pinned nor hashed. Moreover, `freeze.py:31` freezes only the environment text, without verifying the executing environment. This does not close the requirement that all imported/tokenizer dependencies be frozen.

- **High — ownership gate accepts mismatched provenance artifacts:** `scripts/merge_scores.py:132-136` only checks each server provenance record for `status`, `local_only`, and frozen model metadata. It does not require the expected `mode`, `listener_owner_verified`, frozen runtime metadata, command, model response/path, or bind the smoke provenance to the smoke report’s PID/index. For example, a copied full-run provenance file placed at the smoke provenance path can satisfy this gate. `run_teacher.py:89-101` performs the correct live checks, but the final gate does not verify the evidence distinguishing those runs.

- **Medium — over-limit “no output slot” check is vacuous:** `scripts/build_pilot.py:352-356` writes the rejected candidate without an `index`; consequently, `scripts/check_limits.py:31` searches label filenames using the fallback string `"never"`. That condition will normally pass regardless of whether inference occurred. The remaining checks prove the case is outside the pilot, but there is no executable rejection attempt demonstrating that the inference entry point rejects this frozen over-limit input.

- **Correct — reviewer slices omit explicit repository/commit identity:** `scripts/prepare_audit.py:49-58` emits only opaque index, body policy, complete diff, target, and evidence. `audit-rubric.json:3-5` and `audit-instructions.md:3-4` consistently prohibit identity and cross-review information.

- **Correct — score artifacts are structurally separated and provenance is required:** `scripts/merge_scores.py:25-63` binds each score file to its reviewer-specific slice hash and prevents shared paths/duplicates; `merge_scores.py:66-80` requires fresh-context provenance with disjoint run IDs and matching score hashes.

- **Correct — mechanical/body/smoke gates are present:** `scripts/merge_scores.py:95-137` enforces reviewer thresholds, required-body scores, frozen mechanical validation, near-limit smoke, and frozen limit results. Required non-empty bodies are also enforced by `scripts/validate_labels.py:61-70`.

- **Correct — metadata-only changes enter leakage signatures:** `scripts/leakage_v2.py:10-41` includes rename, copy, mode, and binary metadata. `scripts/test_leakage_v2.py:15-19` covers a metadata-only rename.

- **Validation:** All scripts compile. Unit discovery ran seven tests: five passed and two modules could not load because the current Python environment lacks `pyarrow`. This is an environment limitation, but it also reinforces the missing executable dependency lock.

**Verdict: FIX-BEFORE-OUTPUT.**

No files were edited, and no source commit messages were inspected. Transient v2 pilot/freeze hash state was not treated as a blocker.