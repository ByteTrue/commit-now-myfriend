## Review

**Verdict: FIX-BEFORE-OUTPUT**

### Blockers

- **Critical — reviewer inputs can leak source commit messages.**  
  `.cs/epics/001-o-offline-commit-flow/artifacts/035/scripts/prepare_audit.py:49-53` includes `family`, which contains the public repository and commit SHA. A reviewer can therefore locate the original commit message. This contradicts the exclusive input list and withholding requirement in `issue-spec.md:62`, `audit-rubric.json:3-4`, and `audit-instructions.md:3`. Remove repository/commit identity from reviewer artifacts and use an opaque audit identifier.

- **Critical — independent double review is not enforced.**  
  `.cs/epics/001-o-offline-commit-flow/artifacts/035/scripts/merge_scores.py:40-52` accepts arbitrary score paths and permits the same files to be supplied as both reviewer A and B. It does not bind scores to reviewer-specific frozen slices or require disjoint score artifacts. One review can consequently be counted twice and produce a false GO.

- **High — the GO computation omits mandatory gates.**  
  `merge_scores.py:60-68` checks only critical, grounded, subject, body count, and 54 body-quality-2 thresholds. It does not require:
  - the frozen mechanical-validation hash to remain valid;
  - all 60 required bodies to be useful;
  - successful near-limit labeling/smoke;
  - demonstrated over-limit rejection before inference.

  These are mandatory at `issue-spec.md:69-72` and `audit-rubric.json:17-19`. Additionally, `smoke.py:41-44` accepts any index without verifying it is near-limit, while no script executes the frozen over-limit rejection case. A report can therefore say `go` without satisfying the issue gate.

- **High — the frozen teacher and local-only execution are not verified.**  
  `label_pilot.py:33` permits an arbitrary server URL, and `label_pilot.py:74-86` sends `"model": "local-teacher"` without verifying that the endpoint is the frozen Qwen model, that the frozen llama-server binary owns the process, or that the frozen GGUF is loaded. Another local model—or a remote endpoint—could generate all labels while the frozen file hashes still pass, violating `teacher-manifest.json:3-18` and `issue-spec.md:18-30`.

- **High — near-duplicate leakage signatures omit changed content.**  
  `build_pilot.py:163-168` relies on the imported `leakage_signatures`; `.cs/epics/001-o-offline-commit-flow/artifacts/034/scripts/build_dataset.py:162-174` only includes changed lines after an `@@` header. Parsing the frozen pilot found **67/200 rows containing 116 file sections without `@@` headers**. Changes in those sections do not participate in near-duplicate exclusion, so prior evaluation content can enter the pilot through an otherwise different commit. Exact diff/commit hashes do not cover this near-duplicate case.

- **High — rebuild reproducibility dependencies are not frozen.**  
  `build_pilot.py:18-33,102-109,123` imports selection/filter logic from artifact 034 and loads an external tokenizer directory. Neither the imported `034/scripts/build_dataset.py` nor tokenizer files/revision are included in `freeze.py:12-38` or hashed in `pilot-manifest.json:105-111`. `pilot-reproducibility.json:10-15` records package versions only. A later rebuild can therefore produce different eligibility, token limits, or selected rows while appearing to use the documented setup.

### Correct

- The frozen pilot contains 200 unique indices, families, and diff hashes; required coverage is present: 40/100/60 file bins, 88 high-token inputs, seven near-limit inputs, and 60 required-body cases.
- Frozen data hashes match `pilot-manifest.json:100-103`.
- Pilot records contain no source-message field, and `build_pilot.py:133` scans only `repo`, `hash`, `language`, `license`, and `mods`.
- Both audit plans cover indices 0–199 exactly once per reviewer.
- No files were edited, and no source commit messages were inspected.

### Residual risks

- The frozen Python 3.12.13 environment was unavailable; running tests under system Python 3.14 failed to import `pyarrow`. Only two audit-tool tests executed successfully.
- Freeze cannot yet run because the final GGUF is absent; only partial download files currently exist. Its final size and SHA-256 must be verified before output.