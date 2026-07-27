## Goal

Prove or reject a **rejection-only** label-source strategy for the offline 0.5B student: human source commit messages remain unchanged semantic candidates, while the frozen local 14B model may only reject unsupported or materially incomplete candidates.

This issue changes only the label-source filter. The shipped product remains fully offline, Qwen2.5-Coder-0.5B, <=700,000,000 bytes, seven styles, arbitrary Custom Guidance, and multi-line messages. Passing this issue does not authorize corpus construction, student training, or Git workflow implementation.

## Why this is different

- #34 showed that deterministic filtering alone can retain unsupported human messages.
- #35/#36 showed that a 14B teacher generating targets produces semantic errors even after evidence becomes mechanically valid.
- Here the 14B model cannot author, trim, rewrite, or repair target text. It only produces reject decisions. Human source messages remain byte-traceable candidates, and two blinded semantic reviewers remain the truth gate.

## Frozen population

Before any critic output, select exactly 200 fresh public complete-commit families from the frozen Commit Chronicle source revision. The population is evaluation-only and excluded from every future train/validation/test split.

It must be disjoint from all #34-#36/public/private gates under:

- family and commit hash;
- canonical repository/fork component;
- exact complete diff and normalized changed-content signatures;
- changed-content near-duplicate signatures;
- exact and near normalized target-message signatures.

Coverage is frozen before source-message semantics or critic output are inspected:

- 40 single-file;
- 100 two-to-three-file;
- 60 four-to-eight-file;
- >=50 inputs above 4,096 student tokens;
- one near-limit case and one over-limit pre-inference rejection case;
- 60 `body_required` cases selected deterministically from multi-file changes;
- fixed repository cap, source/license/secret/PII/binary/incomplete filters.

No refill, second seed, top-up, deletion, replacement, or case-specific exception is allowed after critic output exists.

## Candidate target lineage

Persist the raw source-message hash and normalized-target hash without exposing repository/commit identity to critics or reviewers. Allowed normalization is fixed before output:

- CRLF/CR -> LF;
- trim outer blank lines and trailing horizontal whitespace;
- remove only recognized metadata trailers (`Signed-off-by`, `Co-authored-by`);
- enforce non-empty subject <=72 characters and bounded body.

No paraphrase, semantic trimming, hash-based style assignment, critic-driven edit, or post-score repair is allowed. The exact normalized subject/body shown to critics is the target shown to reviewers.

## Two critic views

Both use the already frozen local Qwen2.5-Coder-14B-Instruct Q6_K runtime, but they are explicitly **correlated views**, not independent truth evidence:

1. **Support critic:** reject if any target claim is unsupported by the complete diff.
2. **Completeness critic:** reject if the target omits or mis-centers a material primary change.

Each row receives one stateless call per critic under frozen prompts/schemas/decodes. Output is only `accept|reject`, fixed reason codes, and exact diff references. Critics cannot emit replacement target text. Malformed output, timeout, disagreement, missing evidence, or runtime/provenance mismatch means reject. Persist all raw decisions.

The accepted intersection must yield at least:

- 100/200 total;
- 20/40 single-file;
- 50/100 two-to-three-file;
- 30/60 four-to-eight-file;
- 25/50 high-token;
- 30/60 `body_required`.

Falling below any yield/coverage threshold is STOP.

## Blinded human truth gate

Exactly two predeclared fresh-context reviewer runs independently score **all 200 population rows**, including critic rejects. Review inputs contain only opaque index, complete diff, frozen body policy, and exact normalized candidate subject/body. They withhold source identity, critic decisions/reasons/evidence, repository/commit identity, other scores, and aggregate progress.

Freeze reviewer assignments, 10x20 slice hashes, rubric, model/runtime identity, and exactly two run slots before opening inputs. Persist an append-only attempt ledger. Any extra content-bearing reviewer attempt or omitted unfavorable score is STOP.

Report the confusion matrix for each critic and their intersection, false accepts, false rejects, yield, and accepted coverage.

## GO / STOP

The critic-intersection accepted subset passes only when:

- zero critical errors from either reviewer;
- >=95% fully grounded by both;
- >=90% subject quality 2 by both;
- every accepted `body_required` target has a useful body by both;
- >=90% accepted `body_required` targets receive body quality 2 by both;
- all yield/coverage thresholds above pass;
- all 200 rows have two complete bound reviewer scores;
- every frozen hash, leakage, source, environment, local-runtime, near-limit, and over-limit check passes.

Any post-output prompt change, retry, target repair, resampling, manual edit, reviewer shopping, or hash mismatch is STOP.

## Risk order

1. Freeze source revision, exclusions, 200-family population, target bytes/hashes, quotas, critic protocols, review slices, attempt ledger, and all gates.
2. Run reproducibility, leakage, over-limit rejection, and near-limit local-runtime smoke.
3. Execute both critic views exactly once over all 200 rows.
4. Verify yield without selecting or refilling new rows.
5. Run exactly two blinded reviews over all 200 rows and compute critic confusion/gates.
6. GO authorizes only a separate full-corpus plan plus a separate Guidance/body augmentation gate; STOP returns to maintainer strategy choice.

## Out of scope

- generated/repaired teacher targets;
- prompt search, retries, voting, preference optimization, or multiple teacher families;
- reviewing only accepted rows;
- student training, quantization, seven-style/Guidance validation, or Git workflow implementation;
- private repository labeling or hosted APIs.

## Deliverables

- frozen issue protocol and manifests;
- reproducible 200-family population and exclusion proof;
- exact target lineage and normalization tests;
- critic prompts/schemas/raw decisions/provenance;
- two complete blinded score sets and attempt ledger;
- confusion matrix, yield/coverage report, explicit GO/STOP, and Epic writeback.

