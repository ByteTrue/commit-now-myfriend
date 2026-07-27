# Task for reviewer

Second pre-output audit for #35 after fixes. Do not edit and never inspect source commit messages. Read GitHub #35, current issue-spec, manifests/rubric/instructions, and all artifact 035 scripts/tests. Specifically verify closure of prior blockers: reviewer slices expose no repo/commit identity; one score set cannot count twice and fresh-run provenance is required; merge enforces mechanical/body/smoke/over-limit gates; exact frozen localhost teacher process/model ownership is enforced; metadata-only file sections enter near-duplicate leakage; all imported/tokenizer dependencies are frozen. Look for new false-GO paths or code/runtime errors in run_teacher.py, leakage_v2.py, build_leakage_exclusions.py, build_pilot.py, prepare_audit.py, merge_scores.py. The v2 pilot is rebuilding, so transient old pilot hashes are not a blocker. Return concise severity-ranked findings and GO/FIX-BEFORE-OUTPUT.

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