# Task for reviewer

[Read from: /Users/byte/workspace/projects/commit-now-myfriend/plan.md, /Users/byte/workspace/projects/commit-now-myfriend/progress.md]

Adversarial read-only review of the proposed #37 strategy in cnm: fresh disjoint public complete-diff pilot; candidate labels come from human source commit messages; Qwen2.5-Coder-14B acts only as accept/reject critic for complete-diff support and material completeness; accept intersection of two independently prompted critic passes; deterministic normalization/evidence extraction; then two fresh-context blinded semantic reviewers. Read `.cs/epics/001-o-offline-commit-flow/spec.md`, artifacts 034-036 reports/gates, and relevant scripts. Find ways this could falsely pass, leak prior/evaluation data, post-select on reviewers, or repeat #34/#36. Recommend the smallest frozen protocol that closes those holes. Do not edit files.

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