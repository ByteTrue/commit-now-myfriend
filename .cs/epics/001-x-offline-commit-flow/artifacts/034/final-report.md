# cnm #34 final report — STOP before training

## Decision

`STOP` at risk-order step 3 (data validity and blind audit).

No model output was generated. No hardware smoke, base baseline, LoRA training, checkpoint selection, adapter fusion, quantization, shadow opening, historical scoring, final packaging, or Git workflow implementation was run. The two-configuration training budget remains unused (`0/2`).

## Frozen candidate

- Base: `mlx-community/Qwen2.5-Coder-0.5B-Instruct-bf16`
- Revision: `89cbb0caaa52fc80ce8a8a3a015abdc5b35d9ecc`
- Base files: 999,610,552 bytes, 11 content-hashed files
- Distribution target: Q5_K_M with deterministic style rendering
- Complete installed limit: 700,000,000 bytes
- Pre-training skeleton: 570,514,209 bytes; 129,485,791 bytes margin

## Gates completed before data audit

- Independent private shadow/historical gate frozen before data selection.
- Public regression target-message signatures frozen and excluded.
- Normalized-patch, changed-line shingle, target-message, repository/fork-component, secret, PII, schema, and cross-split checks persisted.
- Parser, resolver, renderer, evaluator, semantic-score gate, mutation checks, smoke monitor, and two LoRA configurations persisted before output.
- No private gate diff or gold output was inspected by the training operator.

## First blind data audit

Seed `340525` produced 6,000/1,000/1,000 complete-commit train/validation/test families. Independent reviewers processed 160 of the frozen 200 base-audit rows before early stop:

- critical errors: 4 (required: 0)
- fully grounded: 146/160 (91.25%; required: >=95%)

The remaining 40 rows and guidance audit were not consumed because the zero-critical threshold was already impossible. The one issue-authorized redesign changed the seed to `340526`, strengthened subject evidence to at least 80% meaningful-token overlap, removed unsupported source bodies, removed dangling issue trailers, and added absolute-home-path PII rejection.

## Redesigned blind data audit

The rebuilt corpus again produced 6,000/1,000/1,000 complete-commit families, plus 500 train and 80 validation deterministic guidance variants. Source processing retained 39,817 schema-valid candidates after rejecting 127,829 weakly grounded labels, 25,467 unsupported bodies, 11,652 sensitive rows, 1,616 private/public near overlaps, and 15,358 remaining near duplicates.

Independent reviewers processed 160 of the redesigned frozen 200 base-audit rows before early stop:

- critical errors: 1 (required: 0)
- fully grounded: 157/160 (98.125%; required: >=95%)

The critical row was:

- family: `commitchronicle:leits/meetingbar:b2eea34a65bee3340b6eac304bf51336ff01203f`
- finding: the diff adds a test target and `Equatable` conformance but contains no `getMeetingLink` test; the source label therefore makes a materially unsupported claim.

Although the groundedness percentage passed, the frozen zero-critical requirement failed. Issue #34 permits one source/filter redesign; that allowance was already consumed. Deleting or relabeling this row after seeing the blind result would be a second post-audit redesign and would invalidate the gate, so the run stopped.

## Hardware answer

The M5 Pro did not reach the smoke stage because the earlier data gate failed. Therefore #34 has no measured 0.5B training peak-memory, pageout, swapout, or step-time result and makes no claim that full training would pass on this Mac. No migration to the RTX 5080 is needed for this stopped run. If the maintainer explicitly authorizes another data-design attempt, the next executable gate remains the frozen longest-row smoke on the M5 Pro; migration happens only if that measured gate fails.

## Evidence

- Epic: `.cs/epics/001-o-offline-commit-flow/spec.md`
- Issue: `https://github.com/ByteTrue/commit-now-myfriend/issues/34`
- Persisted pierce assets: `.cs/epics/001-o-offline-commit-flow/artifacts/034/`
- Runtime evidence root: `/tmp/cnm-pierce34/`
- First audit outcome: `.cs/epics/001-o-offline-commit-flow/artifacts/034/data-redesign-1.json`
- Redesigned audit score hashes:
  - `base-00`: `8613547b512068dad25b46d954363c6b07cf380ef25cbf9e9d1278da35710ebf`
  - `base-01`: `42c03a6abe65f2e7ff82e2f57c0225011163ce987d988708fac636b8862cc393`
  - `base-02`: `88cc575490357d62de4642ee8c5c4be85172071cbabeda4e31c402561b056eb4`
  - `base-03`: `268717ee97436608f669b6f686ec37d18168422ecb1e393397e74d999adf18b8`
  - `base-04`: `441f2aa60b2adba844b22d95edab5e79c91a210f87d4ee5dbf98aba59dadff7d`
  - `base-05`: `6f1d87c5918d83306616d0b005a39c675b5b73bc7628578d504ce0bdaf5e49fa`
  - `base-06`: `d685e7301705c1fae47c46aec4719b1582baabdb18d442145da7bebca1a94d65`
  - `base-07`: `c703a372815f758b08e06115b72c895dd1922c99106abfc1f2ca4044c80797ba`

## Constraint needed to continue

A further run requires an explicit maintainer change to #34's data-search budget, for example allowing one additional independently reseeded redesign with a higher-quality labeling strategy. The current run cannot continue under its frozen rules.
