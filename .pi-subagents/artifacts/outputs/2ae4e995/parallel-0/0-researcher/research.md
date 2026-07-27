# Research: Next bounded label-source pierce for cnm

## Summary

**Recommendation: GO only as a bounded filter-validation experiment, not as authorization to build a corpus or train the 0.5B student.** The proposal is materially different from #35/#36 because the 14B model no longer authors semantic targets: human source messages remain unchanged candidates and the model can only reject them. However, it partially returns to #34’s failed source-label mechanism, and two passes from the same 14B model are correlated rather than genuinely independent; the fresh pilot must therefore measure the intersection’s precision against two human reviews over a pre-frozen population, not assume that model agreement establishes correctness.

## Findings

1. **High — materially different failure mechanism, but only partly.** #35/#36 failed because the 14B teacher authored wrong-primary-intent, unsupported, and incomplete targets; 29/200 were critical after mechanical cleanup. A rejection-only teacher prevents its invented text from entering labels, so that failure channel is removed. But #34 already showed that heavily filtered human source messages can retain materially unsupported claims (1 critical in 160 reviewed after redesign). The new risk is therefore **critic false acceptance of a bad/incomplete human message**, not teacher generation error. [#34 final report](.cs/epics/001-o-offline-commit-flow/artifacts/034/final-report.md) [#36 final report](.cs/epics/001-o-offline-commit-flow/artifacts/036/final-report.md)

2. **High — “two independent critic passes” is overstated if both use the same frozen 14B.** Greedy decoding with the same model, prompt, input, and seed is replication, not independence; prompt variants or reordered contexts create some diversity but retain shared model blind spots. #36’s two genuinely separate semantic reviewers still found only 7 shared critical cases out of 29 unique, demonstrating why agreement among correlated critics cannot replace the reviewer union gate. Treat critic intersection as a precision-oriented prefilter only. [#35 issue spec](.cs/epics/001-o-offline-commit-flow/artifacts/035/issue-spec.md) [#36 semantic gate](.cs/epics/001-o-offline-commit-flow/artifacts/036/semantic-gate.json)

3. **High — review all 200 pre-frozen families, not only critic-accepted rows.** Freeze a fresh, disjoint 200-family population first; run both critics; then have both semantic reviewers score all 200 while blind to critic decisions. This directly measures intersection precision, false accepts, false rejects, yield, and selection skew. Reviewing only accepted rows can show precision but cannot establish whether the filter merely selects trivial diffs or destroys body/multi-file/long-context coverage. Do not top up or resample after decisions are visible. The existing deterministic selection, repository caps, token/file quotas, and output guards are reusable. [#35 pilot builder](.cs/epics/001-o-offline-commit-flow/artifacts/035/scripts/build_pilot.py)

4. **High — support and material completeness must be separate frozen decisions.** Critic A should evaluate whether every target claim is supported by the complete diff; Critic B should independently enumerate material changes and reject messages omitting or mis-centering a core change. Both may emit only `accept/reject`, reason codes, and exact evidence/coverage references; neither may rewrite, trim, supplement, or normalize semantic content. The candidate target must be the exact source subject/body after only predeclared deterministic hygiene (line-ending normalization and trailer removal). This directly targets #34’s unsupported claim and #36’s dominant-change omissions. [#34 audit rubric](.cs/epics/001-o-offline-commit-flow/artifacts/034/audit-rubric.md) [#35 audit rubric](.cs/epics/001-o-offline-commit-flow/artifacts/035/audit-rubric.json)

5. **Medium — the #34 source-label transform itself can create unsupported semantics and style noise.** `build_dataset.py` heuristically converts source messages into structured records, assigns some styles from imperative/hash rules, strips bodies using lexical overlap, and accepts “grounded” subjects via token overlap. Those operations must not be silently reused as proof of semantic validity. Freeze and audit the exact post-transform target that the student would receive, while retaining the exact source message hash/provenance. [#34 dataset builder](.cs/epics/001-o-offline-commit-flow/artifacts/034/scripts/build_dataset.py)

6. **High — this pierce cannot prove seven styles, arbitrary guidance, or multi-line behavior.** It can prove only that a sufficiently precise and representative base semantic target source exists. Source messages do not demonstrate arbitrary user guidance, and filtering may disproportionately remove required-body rows. Keep the existing product constraints unchanged, but require a separate frozen guidance/body augmentation gate before any student training authorization. The Epic explicitly requires all seven styles, arbitrary guidance, and multi-line messages in the shipped 0.5B path. [Epic](.cs/epics/001-o-offline-commit-flow/spec.md)

7. **Medium — leakage and reproducibility controls should extend, not restart.** Reuse #34/#35 exact diff, normalized patch, changed-line shingle, repository/fork component, prior-audit-family, public regression, private shadow/historical, secret/PII, binary/incomplete, and token-limit exclusions. Add source-target hashes and exclude the entire new 200-family pilot from every future train/validation/test split. Pin source revision/files, selection seed, tokenizer, model/GGUF/runtime hashes, both critic prompts/schemas/decodes, row order, raw decisions, reviewer slices, and score hashes. The #36 manual edits/re-generation and missing over-limit entrypoint evidence are explicit patterns not to repeat. [#34 dataset builder](.cs/epics/001-o-offline-commit-flow/artifacts/034/scripts/build_dataset.py) [#35 audit preparation](.cs/epics/001-o-offline-commit-flow/artifacts/035/scripts/prepare_audit.py) [#36 final report](.cs/epics/001-o-offline-commit-flow/artifacts/036/final-report.md)

## Minimum protocol

1. **Freeze before output:** source revision and file hashes; fresh 200 family IDs/diffs; all prior-gate exclusions; exact unmodified candidate targets and hashes; coverage quotas; critic prompts/schemas/model/runtime/decode; reviewer rubric/slices; and all GO/STOP thresholds.
2. **Population:** preserve #35’s 40 single-file / 100 two-to-three-file / 60 four-to-eight-file coverage, at least 50 inputs above 4,096 tokens, a frozen near-limit case, an over-limit pre-inference rejection, repository-group cap, and 60 predeclared `body_required` cases. No post-filter replacement.
3. **Critics:** two separate fresh contexts; one support pass and one material-completeness/primary-intent pass; blind to repository identity, history, other decisions, and reviewer scores. They reject only. Persist raw outputs and exact hashes.
4. **Human truth set:** two fresh-context semantic reviewers independently score **all 200** post-transform candidate targets, blind to source identity, critic decisions, other scores, and aggregate progress. Reuse the #35 critical/grounding/subject/body rubric and union-of-reviewers critical rule.
5. **Report:** confusion matrix per critic and intersection; intersection precision; accepted yield; false-negative rate; reviewer disagreement; and accepted coverage by file count, token range, body policy, language, license, and repository group.
6. **No mutation:** no repairs, deletions, retries, prompt changes, resampling, target edits, or case-specific exceptions after any critic or reviewer output exists.

## Hard STOP gates

- **Any critical error from either semantic reviewer in the critic-intersection accepted set.** Zero means zero; no confidence interval substitutes for the frozen pilot gate.
- **Any teacher-authored or post-hoc semantic text enters a target**, including trimming a supported source body based on critic output.
- **Fewer than 190/200 population rows reviewed by both, incomplete score provenance, duplicate reviewer runs, hash mismatch, or reviewer exposure to critic decisions/source identity.** Prefer all 200; missing rows are a protocol failure.
- **Intersection precision below 100% for critical correctness**, fully grounded below 95%, subject-quality-2 below 90%, any required body not useful, or required-body-quality-2 below 90%, using the existing #35 thresholds on the accepted subset where denominators apply.
- **Insufficient usable yield or collapsed coverage:** predeclare a minimum before running. Recommended minimum is at least 100 accepted rows overall, at least 30/60 accepted `body_required` rows, and nonzero accepted rows in every frozen file/token bin; otherwise the strategy is too selective to support the maintained product.
- **Any overlap** with #34/#35/#36 pilots/audits, public regression, private shadow/historical gates, or future student splits under exact, normalized-patch, changed-line-near, target-near, repository/fork-component, or family checks.
- **Any post-output rebuild/retry/prompt redesign/manual repair**, any unpinned model/runtime/source/tokenizer dependency, or inability to reproduce the same selected families and target bytes from the frozen inputs.
- **Any attempt to proceed directly to full-corpus labeling or student training.** Passing this pierce authorizes only a separately frozen full-corpus plan that includes semantic sampling and a separate guidance/body augmentation gate.

## Better minimal alternative

The best minimal version is not “find 200 accepted rows and review them”; it is **freeze 200 candidate families, score every one with both critics and both humans, then evaluate the critic intersection against human truth**. This adds no model or abstraction, avoids selection leakage, and answers the actual question—whether rejection-only filtering materially raises source-message precision—while also quantifying yield and representativeness. If even this pilot produces one critic-intersection critical error, stop automated source-message filtering and move to human-authored structured targets for a much smaller feasibility set (for example 50 representative families) before considering corpus-scale cost.

## Sources

- Kept: [Epic spec](.cs/epics/001-o-offline-commit-flow/spec.md) — maintained product constraints and accumulated pierce conclusions.
- Kept: [#34 final report](.cs/epics/001-o-offline-commit-flow/artifacts/034/final-report.md) and [audit rubric](.cs/epics/001-o-offline-commit-flow/artifacts/034/audit-rubric.md) — direct evidence that filtered source messages retained a critical semantic error.
- Kept: [#34 dataset builder](.cs/epics/001-o-offline-commit-flow/artifacts/034/scripts/build_dataset.py) and [audit slicer](.cs/epics/001-o-offline-commit-flow/artifacts/034/scripts/prepare_audit_slices.py) — actual source-message transforms, leakage checks, selection, and audit mechanics.
- Kept: [#35 final report](.cs/epics/001-o-offline-commit-flow/artifacts/035/final-report.md), [issue spec](.cs/epics/001-o-offline-commit-flow/artifacts/035/issue-spec.md), and [rubric](.cs/epics/001-o-offline-commit-flow/artifacts/035/audit-rubric.json) — frozen teacher protocol and mechanical failure evidence.
- Kept: [#35 pilot builder](.cs/epics/001-o-offline-commit-flow/artifacts/035/scripts/build_pilot.py) and [audit preparation](.cs/epics/001-o-offline-commit-flow/artifacts/035/scripts/prepare_audit.py) — reusable disjoint-selection, coverage, freeze, and reviewer-blinding controls.
- Kept: [#36 final report](.cs/epics/001-o-offline-commit-flow/artifacts/036/final-report.md) and [semantic gate](.cs/epics/001-o-offline-commit-flow/artifacts/036/semantic-gate.json) — direct semantic failure counts and reproducibility caveats.
- Dropped: `/tmp/cnm-pierce37` — explicitly invalid post-score redesign, stale provenance, incomplete output, and excluded by the maintained report.
- Dropped: source commit messages — prohibited by the task and unnecessary for strategy review.

## Gaps

No frozen spec yet defines the two rejection prompts, what qualifies them as independent, the minimum accepted yield, or full-corpus audit sampling after a pilot GO. Those must be fixed before any output. A 200-family zero-critical result is strong bounded evidence, not proof that an 8,000-family corpus contains zero critical labels.

```acceptance-report
{
  "criteriaSatisfied": [
    {
      "id": "criterion-1",
      "status": "satisfied",
      "evidence": "Concrete severity-ranked findings cite the Epic, artifact reports 034/035/036, and dataset/audit scripts; hard STOP gates and residual risks are stated."
    }
  ],
  "changedFiles": [],
  "testsAddedOrUpdated": [],
  "commandsRun": [
    {
      "command": "Read-only inspection of the Epic, reports 034/035/036, rubrics, and relevant dataset/audit scripts",
      "result": "passed",
      "summary": "Reviewed specified persisted evidence without inspecting source commit messages."
    }
  ],
  "validationOutput": [
    "Recommendation limited to a bounded filter-validation pilot; no corpus build, student training, or Git rewrite authorized.",
    "Artifact written to the authoritative runtime output path."
  ],
  "residualRisks": [
    "Two passes of one 14B model have correlated blind spots and are not genuinely independent semantic evidence.",
    "Filtering may select trivial rows and collapse body, multi-file, long-context, language, or style coverage.",
    "The pierce does not validate arbitrary guidance, seven-style conditioning, or final 0.5B student quality.",
    "A zero-critical 200-family pilot cannot guarantee zero critical labels in a later full corpus."
  ],
  "noStagedFiles": true,
  "diffSummary": "No project files edited; only the required external research artifact was written.",
  "reviewFindings": [
    "high: proposed critic intersection removes teacher-authored target errors but retains source-message false-acceptance risk proven by #34",
    "high: same-model critic passes are correlated and must remain a prefilter, never the semantic gate",
    "high: reviewing only accepted rows creates selection leakage and hides false rejects/coverage collapse; review all pre-frozen 200 families",
    "high: passing this pierce cannot authorize training because guidance, seven styles, and multi-line augmentation remain unproven",
    "medium: #34 heuristic target transforms must be frozen and audited as part of the candidate target",
    "medium: all #34/#35 leakage, freeze, provenance, token-limit, and reviewer-blinding controls must be retained"
  ],
  "manualNotes": "Read-only review; source commit messages were not inspected."
}
```
