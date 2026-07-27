## Review

- **Correct:** The proposed direction is materially better than #36: human source messages remain unchanged semantic candidates, Qwen14B only filters, the pilot is fresh/disjoint, and two blinded semantic reviewers remain the final gate. This avoids treating generated evidence as truth, which failed despite 200/200 mechanical validity in `.cs/epics/001-o-offline-commit-flow/artifacts/036/final-report.md:13-29`.
- **Correct:** Preserve the existing zero-critical union gate and joint quality thresholds from `.cs/epics/001-o-offline-commit-flow/artifacts/035/issue-spec.md:65-76`: 0 critical from either reviewer, 190 grounded, 180 subject-quality-2, all 60 required bodies useful, and 54/60 body-quality-2.

- **Blocker:** `.cs/issues/037-pierce-improve-subject-quality.md:61-79` still specifies regenerating the same 200 #35 cases with a post-score prompt and reviewers C/D. That is exactly the excluded experiment documented in `artifacts/036/final-report.md:31-35` and `spec.md:170`. It is neither fresh nor disjoint and would repeat #36. Replace and freeze this issue specification before reading any new source messages or running critics.

- **Blocker:** The current leakage machinery is insufficient once source messages become candidate targets:
  - `artifacts/035/scripts/build_pilot.py:145` deliberately does not read messages.
  - `build_pilot.py:175-184` computes target signatures from `""` and checks only diff signatures, although `artifacts/034/scripts/build_dataset.py:284-305` already exposes historical/public target exclusions.
  - `artifacts/035/pilot-manifest.json:119-122` excludes only the 360 #34 audit families; it does not exclude the 200 #35/#36 pilot families.
  - `artifacts/035/scripts/build_leakage_exclusions.py:44-90` builds diff-only exclusions and explicitly records that source-message fields were not accessed.
  
  #37 must exclude all previously exposed #34–#36 families, repository/fork components, exact and near diff signatures, and exact/near target-message signatures. The entire #37 candidate pool—not merely the final accepted 200—must remain evaluation-only and be excluded from future train/validation/test data.

- **Blocker:** “Take the intersection of two critic passes” can falsely pass through unbounded semantic post-selection. Without a frozen denominator, the operator can scan or refill until 200 easy, short, obvious source messages survive. That proves only that cherry-picked labels exist, not that the production strategy yields sufficient representative data. Freeze a candidate pool and quotas before critic output, run both critics exactly once on every row, treat malformed/timeout output as reject, prohibit refill, and require a predeclared intersection yield in every coverage stratum.

- **Blocker:** Fresh context does not prevent reviewer post-selection. `artifacts/035/scripts/merge_scores.py:67-80` verifies only that the submitted A/B run IDs are disjoint and match the chosen score hashes. It cannot detect extra reviewer attempts whose unfavorable scores were omitted. Freeze reviewer implementation/model version, prompt, slice hashes, and exactly two run slots before opening review inputs. Persist an append-only attempt ledger and make any extra content-bearing review attempt a STOP. Reviewer execution should be sandboxed to its slice and rubric rather than merely instructed not to inspect the repository.

- **Note [high]:** Two prompts against the same Qwen2.5-Coder-14B binary are two critic views, not independent critics. Their errors can be highly correlated, especially because #36 already demonstrated semantic errors from that model. Critic agreement must remain a filter only; it cannot contribute evidence toward GO. The two blinded semantic reviews must independently satisfy the complete gate.

- **Note [high]:** Deterministic evidence extraction can recreate #36’s false assurance. `artifacts/035/scripts/validate_labels.py:73-80` proves only that excerpts occur in changed lines. #36 passed that mechanical check for 200/200 labels while retaining 29 critical semantic errors. Moreover, `prepare_audit.py:53-61` shows extracted evidence to reviewers, creating confirmation anchoring. Evidence should be diagnostic only and withheld from the first semantic judgment; reviewers need only the complete diff, body policy, and exact candidate target.

- **Note [high]:** Do not blindly reuse #34 normalization. `artifacts/034/scripts/build_dataset.py:84-105` deletes selected trailer/body content and rejects all non-ASCII messages; `semantic_record()` at lines 109-123 assigns some styles using a family hash. Freeze a lossless candidate lineage: raw-message hash, normalized-message hash, and explicit deterministic transformation reasons. Normalization may standardize newlines and parse a recognized prefix, but must not invent, paraphrase, or case-specifically remove semantic claims after critic/reviewer results.

- **Note [high]:** Freeze `body_required` independently of critic outcomes. The existing pilot does this before generation at `artifacts/035/scripts/build_pilot.py:320-324`. Selecting required-body rows only after observing which source messages contain good bodies would inflate the body gate. A required-body stratum that fails to yield 60 accepted candidates must STOP rather than refill.

## Smallest frozen protocol

1. **Declare development data:** Treat every #34–#36 diff, source/generated label, score, and the invalid `/tmp/cnm-pierce37` run as development data. Persist hashes for family, repository/fork component, exact/near diff, and exact/near target message.
2. **Freeze one 400-row candidate pool:** Select by fixed seed/hash after mechanical eligibility only. Keep existing 40/100/60 file-bin proportions, at least 50 high-token rows, and predeclared required-body coverage. No semantic filtering or critic-dependent refill.
3. **Freeze the complete pipeline:** Hash raw/normalized messages, normalization code/tests, critic model/runtime/decode, two distinct prompts and schemas, input order, selection rule, thresholds, review rubric, and reviewer assignments before critic output.
4. **One-shot critics:** Two stateless calls per row; both must separately accept complete-diff support and material completeness. Parse failure, timeout, disagreement, or missing output means reject. Preserve all outputs and rejection counts.
5. **Frozen selection:** Require at least 200 intersection accepts with all coverage/body quotas. Select exactly 200 by a predeclared hash-and-quota rule. Fewer than 200 or any deficient stratum is STOP; no refill or second seed.
6. **Blinded semantic gate:** Reviewers receive only opaque index, complete diff, predeclared body policy, and the exact normalized target. Withhold source provenance, critic decisions/reasons, extracted evidence, repository identity, other scores, and aggregate progress.
7. **No reviewer shopping:** Exactly two predeclared review attempts, append-only provenance, union-of-critical decisions, and the existing #35 thresholds. Any restart after content exposure or any extra reviewer attempt is STOP.
8. **GO remains bounded:** Passing authorizes a separate full-corpus labeling issue using the unchanged pipeline. None of the 400 pilot rows may enter student training or another quality gate.

## Residual risks

- Public commits and messages may have appeared in Qwen or reviewer-model pretraining; internal disjointness cannot prove model-training disjointness. Post-cutoff commits or actual human reviewers are the strongest mitigation.
- Public diffs are untrusted prompt content and can contain instruction-like text; strict schemas and system delimiters reduce but do not eliminate prompt-injection risk.
- A 200-row zero-critical result does not establish zero population error, and critic filtering may bias labels toward easy repositories/change types.
- Even a successful label-source pilot does not prove 0.5B student quality, seven-style/Guidance behavior, quantized quality, or production Git workflow feasibility.