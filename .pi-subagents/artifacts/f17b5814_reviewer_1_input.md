# Task for reviewer

[Read from: /Users/byte/workspace/projects/commit-now-myfriend/plan.md, /Users/byte/workspace/projects/commit-now-myfriend/progress.md]

Goal: independently blind-review reviewer B's 200 frozen teacher-label cases and write exactly 10 score files.

Read ONLY:
- /Users/byte/workspace/projects/commit-now-myfriend/.cs/epics/001-o-offline-commit-flow/artifacts/035/audit-instructions.md
- /Users/byte/workspace/projects/commit-now-myfriend/.cs/epics/001-o-offline-commit-flow/artifacts/035/audit-rubric.json
- /tmp/cnm-pierce35/audit/review-input-manifest.json (only to obtain reviewer B slice hashes)
- /tmp/cnm-pierce35/audit/reviewer-b-slice-00.jsonl through reviewer-b-slice-09.jsonl
Do not read any repository/source identity, commit messages, reviewer A inputs/scores, existing /tmp/cnm-pierce35/review files, aggregate results, or web sources.

For each slice, write exactly 20 JSONL score rows to:
/tmp/cnm-pierce35/audit/scores-b/reviewer-b-slice-NN.scores.jsonl
Each object must have exactly: index, reviewer (literal B), input_slice_sha256 (the exact manifest hash for that slice), critical_error, fully_grounded, subject_quality, body_quality, evidence_quality, reason. Follow the frozen rubric strictly; inspect each complete diff semantically, not merely whether cited evidence exists. Do not repair labels. Do not modify project files. Complete all 200 independently. At the end, report totals and paths only; do not inspect reviewer A.

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