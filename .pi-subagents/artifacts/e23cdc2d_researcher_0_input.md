# Task for researcher

[Read from: /Users/byte/workspace/projects/commit-now-myfriend/.cs/epics/001-o-offline-commit-flow/spec.md]

Read-only research for GitHub issue #30 in ByteTrue/commit-now-myfriend. Find the smallest credible locally runnable model/runtime candidates as of 2026-07 for git diff -> commit message with instruction following, seven configurable styles, arbitrary custom guidance, and optional multiline body. Hard ceiling complete installed bundle 300 MB. Prefer GGUF + llama.cpp or similarly redistributable static runtime. For each candidate report exact parameter/quantized file sizes, license/redistribution status from primary sources, architecture/runtime support, and likely quality risk. Pay special attention to hks350d/git-diff-to-commit-gemma-3-270m, Gemma license obligations, and whether Q8 can be requantized to Q4. No edits or issue comments. Return source URLs.

---
**Output:**
Write your findings to exactly this path: /Users/byte/workspace/projects/commit-now-myfriend/.pi-subagents/artifacts/outputs/e23cdc2d/research.md
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