# Task for reviewer

第三次且最终只读复审 `/tmp/cnm-train31/data/{train,valid,test}.jsonl`。严格复用 seed `cnm-31-review-v1` 的固定 60-family 抽样方法（每 split 20），逐条核对 target 与 diff、subject 完整、style/guidance、secret/幻觉；全量检查 split/repo/family/diff 交集、manifest hash、全部 custom bullet 路径支持。确认上一轮 Hipstore family 和首轮 7 坏例均消失。不要修改文件。按严重级别给 findings 和明确 GO/STOP；小样本天然残余风险不应单独阻断本次 bounded pierce。

---
**Output:**
Write your findings to exactly this path: /tmp/cnm-train31/data-review-v3.md
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