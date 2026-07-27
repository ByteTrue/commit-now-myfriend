# Task for researcher

Read-only strategy review for cnm Epic `.cs/epics/001-o-offline-commit-flow/spec.md`, artifact reports 034/035/036, and relevant dataset scripts. The maintained product constraints are fully offline shipped 0.5B <=700MB, seven styles, arbitrary guidance, multi-line. Previous source-message labels and 14B-generated labels failed zero-critical gates. Evaluate the proposed next bounded pierce: use human source commit messages as candidate targets, but a local 14B teacher acts only as a strict complete-diff support/material-completeness rejection filter; accept only intersection of two independent critic passes; select a fresh disjoint 200-family pilot and run two independent semantic reviewers. Identify whether this has a materially different failure mechanism, minimum protocol, leakage/reproducibility risks, and a better minimal alternative if any. Do not edit files or inspect source commit messages beyond what existing reports already state. Return a concise recommendation and hard STOP gates.

---
**Output:**
Write your findings to exactly this path: /Users/byte/workspace/projects/commit-now-myfriend/.pi-subagents/artifacts/outputs/2ae4e995/parallel-0/0-researcher/research.md
This path is authoritative for this run.
Ignore any other output filename or output path mentioned elsewhere, including output destinations in the base agent prompt, system prompt, or task instructions.

## Acceptance Contract
Acceptance level: attested
Completion is not accepted from prose alone. End with a structured acceptance report.

Criteria:
- criterion-1: Return concrete findings with file paths and severity when applicable

Required evidence: review-findings, residual-risks

Finish with a fenced JSON block tagged `acceptance-report` in this shape:
Use empty arrays when no items apply; array fields contain strings unless object entries are shown.
`criteriaSatisfied[].status` must be exactly one of: satisfied, not-satisfied, not-applicable.
`commandsRun[].result` must be exactly one of: passed, failed, not-run.
`manualNotes` and `notes` are optional strings; an empty string means no note and does not satisfy `manual-notes` evidence.
```acceptance-report
{
  "criteriaSatisfied": [
    {
      "id": "criterion-1",
      "status": "satisfied",
      "evidence": "specific proof"
    }
  ],
  "changedFiles": [
    "src/file.ts"
  ],
  "testsAddedOrUpdated": [
    "test/file.test.ts"
  ],
  "commandsRun": [
    {
      "command": "command",
      "result": "passed",
      "summary": "short result"
    }
  ],
  "validationOutput": [
    "validation output or concise summary"
  ],
  "residualRisks": [
    "none"
  ],
  "noStagedFiles": true,
  "diffSummary": "short description of the diff",
  "reviewFindings": [
    "blocker: file.ts:12 - issue found, or no blockers"
  ],
  "manualNotes": "anything else the parent should know"
}
```