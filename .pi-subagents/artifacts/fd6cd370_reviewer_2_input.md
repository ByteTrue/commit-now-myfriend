# Task for reviewer

[Read from: /Users/byte/workspace/projects/commit-now-myfriend/plan.md, /Users/byte/workspace/projects/commit-now-myfriend/progress.md]

独立盲审 /tmp/cnm-pierce34/audit/slices/base-02.jsonl 的全部20行。只读该 slice 与 .cs/epics/001-o-offline-commit-flow/artifacts/034/audit-rubric.md，不读其他 corpus/score/训练脚本。逐行检查完整 diff 与 record，绝不可跳过。写恰好20行 JSONL 到 /tmp/cnm-pierce34/audit/scores/base-02.jsonl：family,critical_error(bool),fully_grounded(bool),subject_quality(0/1/2),body_quality(0/1/2),reason。写后验证20个唯一 family 和类型。不要改 repo/corpus；最后只报计数。

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