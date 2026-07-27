# Task for worker

只处理翻译数据，不修改仓库。读取 `/tmp/cnm-train31/data/translation-input.json`（112 条，每条有 id/subject/body），将英文 Git commit subject/body 高质量翻译成简体中文，并写入 `/tmp/cnm-train31/data/translations.json`。要求：1) 输出必须是 JSON 数组，条目数与 id 集合完全一致；2) 每条只含 id、subject、body；3) 保留代码标识符、文件名、CLI flag、issue/reference 等原样；4) subject 简洁、命令式、无句号；5) body 只翻译原文，不新增事实，输入 body 为空则输出 body 为空；6) 不要翻译 Conventional 类型前缀（输入当前没有前缀）。完成后用 Python 校验 JSON、112 条、id 唯一且一致、空 body 仍为空，并报告命令与结果。

---
**Output:**
Write your findings to exactly this path: /tmp/cnm-train31/translation-report.md
This path is authoritative for this run.
Ignore any other output filename or output path mentioned elsewhere, including output destinations in the base agent prompt, system prompt, or task instructions.

## Acceptance Contract
Acceptance level: checked
Completion is not accepted from prose alone. End with a structured acceptance report.

Criteria:
- criterion-1: Implement the requested change without widening scope

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