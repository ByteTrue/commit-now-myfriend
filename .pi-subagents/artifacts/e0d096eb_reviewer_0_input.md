# Task for reviewer

只读审查 `/tmp/cnm-train31/data/{train,valid,test}.jsonl` 和 `/tmp/cnm-train31/data/families.jsonl`，重点判断这个 #31 微调数据是否足以作为可审计穿刺：按固定哈希方式跨 train/valid/test 各抽样至少 20 条（总计至少 60），检查 assistant target 是否被 diff 支持、style/guidance 是否遵从、是否有明显错误/幻觉/机械重复、仓库 split 泄漏或训练/评测污染风险。也核对 `/tmp/cnm-train31/data/manifest.json` 与 `/tmp/cnm-train31/eval/manifest.json`。不要修改任何文件。按 blocker/high/medium/low 输出 findings，给出明确 go/stop 建议；如果没有阻断项也要说清残余风险。

---
**Output:**
Write your findings to exactly this path: /tmp/cnm-train31/data-review.md
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