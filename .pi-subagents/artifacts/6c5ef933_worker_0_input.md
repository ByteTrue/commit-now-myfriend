# Task for worker

翻译 `/tmp/cnm-train31/data/translation-missing.json` 中全部 92 条英文 commit message。读取该文件；把每条 `subject` 和非空 `body` 翻译成自然、准确、简洁的简体中文，保留代码标识符、文件名、反引号内容、`#123` 等原样；空 body 必须仍为空。只写 `/tmp/cnm-train31/data/translations-missing.json`，内容必须是 JSON 数组，每项严格只有 `id`、`subject`、`body`，id 与输入一一对应且顺序一致。不要改其他文件。完成后用 Python 验证 92 条、ID 集相等、无空 subject、空 body 保持为空，并报告结果。

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