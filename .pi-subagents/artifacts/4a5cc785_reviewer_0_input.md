# Task for reviewer

复审修正后的 `/tmp/cnm-train31/data/{train,valid,test}.jsonl` 与 manifest。必须复用上一轮固定方法：seed `cnm-31-review-v1`，每个 split 按 `SHA-256(seed + '\0' + family)` 升序取 20 family、每 family 再按同法选 1 variant，总计 60 条；逐条检查 target 是否由 diff 支持、subject 完整、style/guidance 遵从、是否有 secret/幻觉/机械坏样本。全量复查 split/repo/family/diff 交集、manifest hash，并确认上一轮列出的 7 个具体坏例不再出现。也检查全部 custom bullet target 的两条 changed-file bullet 是否确由 diff 路径支持。不要修改文件。按 blocker/high/medium/low 输出，给明确 GO/STOP。

---
**Output:**
Write your findings to exactly this path: /tmp/cnm-train31/data-review-v2.md
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