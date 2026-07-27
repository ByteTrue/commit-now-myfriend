# Task for reviewer

Independently evaluate the frozen cnm #32 Gemma 3 1B Q5_K_M outputs. Read `/tmp/cnm-pierce32/eval/high-risk.jsonl` (the exact prompt/diff per case) and `/tmp/cnm-pierce32/eval/results-q5/results.json` (outputs/checks). For every one of 26 IDs assign semantic score 0/1/2 using the frozen rubric: 2 = accurate and complete enough for a useful commit message with no invented material behavior; 1 = materially relevant but incomplete/imprecise; 0 = wrong, empty, code/diff echo, materially invented, or unusable. Separately mark explicit guidance compliance and required-body usefulness where applicable. Be strict and evidence-based. Summarize counts, list each case with score and one-line rationale, state whether acceptance (no 0; >=24/26 score 2; guidance 5/5; body 4/4) passes. Also judge whether observed failures look like a quantization/runtime defect that could justify the one allowed second 1B candidate, or a base model capability failure. Do not modify project/source files. Return only the review report through the configured output artifact.

---
**Output:**
Write your findings to exactly this path: /tmp/cnm-pierce32/eval/semantic-review.md
This path is authoritative for this run.
Ignore any other output filename or output path mentioned elsewhere, including output destinations in the base agent prompt, system prompt, or task instructions.

## Acceptance Contract
Acceptance level: checked
Completion is not accepted from prose alone. End with a structured acceptance report.

Criteria:
- criterion-1: Return concrete findings with file paths and severity when applicable

Required evidence: changed-files, tests-added, commands-run, residual-risks, no-staged-files

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