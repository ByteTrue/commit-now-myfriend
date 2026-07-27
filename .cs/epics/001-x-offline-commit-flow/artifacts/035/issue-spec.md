## Goal

Prove or reject one reproducible local-teacher labeling path for the Qwen2.5-Coder-0.5B student after #34 showed that filtered source commit messages still contain critical semantic errors.

This issue changes only the label source. The final product remains a <=700,000,000-byte, fully offline 0.5B runtime with seven styles, Custom Guidance, and multi-line messages. No Git workflow implementation or student training starts until this data gate passes.

## Maintainer decision

On 2026-07-25 the maintainer explicitly approved the recommended next step:

- stop trusting source commit messages as labels;
- use a stronger local code model as teacher over public complete commit diffs;
- perform independent double-blind review;
- keep all #34 product, size, privacy, and student-model constraints unchanged.

Public dataset diffs may be processed locally. Private project diffs are not teacher-training input and are not sent to any labeling service.

## Frozen teacher

- Model: `Qwen/Qwen2.5-Coder-14B-Instruct-GGUF`
- Revision: `d0a692ef765eefbf2fabb130b3cb2e8917e3d225`
- Quant: Q6_K
- File: `qwen2.5-coder-14b-instruct-q6_k.gguf`
- Bytes: `12,124,683,712`
- SHA-256: `302a079369ad8b66c8e8ec1bfa62d109c64d8015e9bfd52d9c6cf4c6c9f36b5f`
- License: Apache-2.0
- Runtime: llama.cpp `9430` (`d48a56eff`), server SHA-256 `7b1151b319e977564a5422aa00902f64edb87b350c02b65f8dc44c5e183deeea`
- Tooling: Python `3.12.13`, uv `0.11.14`, and a generated-hash 31-package transitive `requirements.lock` (including the isolated environment's pip bootstrap); runtime verification binds the base Python/uv binaries, requires uv isolation, and rejects missing, changed, or extra distributions
- Decode: greedy, temperature 0, seed 424242, max 384 generated tokens, JSON grammar, no content retries

The 14B Q6 teacher is intentionally not part of the shipped product. It fits the M5 Pro's 48 GB unified memory and can later fit the RTX 5080's 16 GB VRAM with a bounded context if throughput becomes the bottleneck. No second teacher family or quant is allowed in this issue.

## Teacher record

For each complete public diff, the teacher returns exactly one JSON record:

- `type`, `scope`, `subject`, `body`: candidate student target;
- `subject_evidence`: 1–3 exact excerpts from changed diff lines supporting the subject;
- `body_evidence`: exact excerpts supporting each non-empty body claim.

Evidence fields are audit scaffolding and are removed from student targets. Deterministic validation requires exact excerpts to exist in the complete diff, valid target schema, subject <=72 characters, no wrappers/explanations, no secrets/PII, and body evidence whenever body is non-empty. Evidence presence does not substitute for semantic review.

The original source commit message is withheld from the teacher and reviewers. It remains provenance metadata only and cannot select, repair, or score the teacher label.

## Frozen pilot

Build one 200-family pilot from `JetBrains-Research/commit-chronicle` revision `5fd076e67b812a9f3d1999e5e40f71715f84bb51`, using complete textual `mods[]` and the existing #34 license, secret/PII, binary/incomplete, hidden/public overlap, repository-component, exact/near-duplicate, and token-limit exclusions. The #35 near-duplicate signature covers changed lines plus every metadata-only file section (rename/mode/binary sections without `@@` hunks); its hashed exclusions are rebuilt source-message-blind from all prior public/private audit diffs.

The pilot is disjoint from #34's public regression, private shadow/historical gate, all 160 rows exposed in the first blind audit, and all 200 rows in the redesigned frozen audit (including its 160 reviewed rows). It is also excluded from every future student train/validation/test split. Freeze all IDs, diffs, hashes, prompt, grammar, decoder, exact teacher/runtime binaries, tokenizer files, imported selection code, and audit rubric before the first teacher output.

Coverage:

- 40 single-file, 100 two-to-three-file, and 60 four-to-eight-file commits;
- at least 50 inputs above 4,096 student tokens;
- at least one valid near-limit input and explicit over-limit rejection check;
- 60 predeclared `body_required` cases selected from multi-file changes; remaining cases allow an empty body when the subject is sufficient;
- report language, license, repository, file-count, byte, and tokenizer-length distributions.

The pilot is evaluation-only and never enters student training.

## Independent double-blind gate

Each of the 200 teacher records is reviewed independently by two fresh reviewers. Each reviewer receives only an opaque case index, the frozen complete diff, body policy, teacher target, teacher evidence, and rubric—not repository identity, commit SHA, source messages, other scores, or model identity. Score files are bound to reviewer-specific frozen slice hashes and disjoint fresh-context run provenance; the merge tool cannot count one score set twice.

A record is accepted only when both reviewers agree it has no critical error and is fully grounded. The whole pilot passes only when:

- zero critical errors from either reviewer across all 400 review decisions;
- at least 190/200 records are fully grounded by both reviewers;
- at least 180/200 subjects receive quality 2 from both reviewers;
- all 60 `body_required` records contain a useful body, and at least 54/60 receive body quality 2 from both reviewers;
- all records pass exact schema/evidence/secret/PII checks and that validation hash is the frozen review input;
- the frozen near-limit case passes local hardware and mechanical smoke, and the frozen over-limit case is excluded before inference;
- the exact frozen runtime owns the localhost listener and serves the exact frozen model for both smoke and full labeling;
- reviewer disagreement, bound slice hashes, disjoint fresh-context run provenance, and score hashes are persisted without post-hoc repair.

There is no prompt redesign inside this issue. Any critical error or threshold failure yields STOP. Fixing, deleting, relabeling, retrying, or resampling after scores are visible requires a new maintainer decision and a new disjoint pilot.

## Risk order

1. Freeze teacher, pilot, exclusions, prompt, grammar, decoder, audit slices, and hashes.
2. Execute the frozen over-limit input through the same label entry point with no server listener and require its dedicated pre-inference rejection code; then start the exact frozen llama-server binary through the frozen local-only owner script, verify its localhost PID/model identity, and run one frozen near-limit hardware/latency/mechanical smoke. Stop if it causes OOM, sustained non-normal pressure, or run-attributable swap/pageout; do not silently switch model/quant.
3. Generate exactly 200 records locally, one deterministic output per case.
4. Run deterministic validation. Any malformed or unsupported-evidence record is a pilot failure, not a retry opportunity.
5. Run two independent blind audits and apply the frozen gate.
6. If and only if the pilot passes, close this data pierce as GO and create a separate bounded issue for full teacher labeling, full-corpus audit, M5 student smoke, and the two frozen 0.5B configurations.

## Quality commitments

- **Functional suitability:** teacher targets must accurately and sufficiently describe each complete diff; evidence and double review produce the acceptance proof.
- **Reliability:** immutable pilot, one output per case, no source-message leakage, two independent reviewers, and no post-score repair prevent optimistic relabeling.
- **Information security:** only public allowlisted data is labeled; secrets/PII are rejected before model/reviewer processing; inference is local.
- **Performance efficiency:** teacher latency and memory are measured for feasibility of full 8,000-family labeling, but teacher size does not count against the 700 MB shipped-product cap.
- **Maintainability:** reuse #34's complete-diff, leakage, schema, renderer, and manifest logic; add only the teacher evidence record and pilot gate, not a general labeling platform.

## Out of scope

- student LoRA training or hyperparameter changes;
- changing Qwen2.5-Coder-0.5B, the 700 MB cap, seven styles, Guidance, body, or offline-product constraints;
- using original commit messages as labels;
- private repository labeling or hosted labeling APIs;
- multiple teacher models, prompt search, self-consistency voting, retries, preference optimization, or a labeling service abstraction;
- Git staging/commit implementation or old-product rewrite.

## Deliverables

- teacher acquisition/runtime manifest and hashes;
- frozen 200-family pilot manifest, coverage and exclusion proof;
- exact prompt, grammar, decoder, evidence validator and tests;
- sanitized teacher hardware/latency smoke;
- 200 raw teacher records and deterministic validation result;
- two complete independent audit score sets and merged gate result;
- explicit GO/STOP conclusion and Epic writeback.
