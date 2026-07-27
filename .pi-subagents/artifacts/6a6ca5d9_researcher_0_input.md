# Task for researcher

为 cnm GitHub #34 做只读研究：寻找可合法用于训练 Qwen2.5-Coder-0.5B 的“完整多文件 commit diff + commit message”公开数据源。优先官方/原始数据源，确认每个候选的数据形状（是否完整 commit 而非单文件 slice）、规模、语言、license 字段、下载方式、能否 repository-isolated split、已知泄漏/质量风险。阅读 GitHub #34（gh issue view 34）和 /tmp/cnm-train31/build_dataset.py 理解旧数据问题。给出一个最小、可执行、无需建设训练平台的数据方案；不修改仓库。把完整报告写到 /tmp/cnm-34-dataset-research.md，并在最终回复给出结论。

---
Update progress at: /Users/byte/workspace/projects/commit-now-myfriend/.pi-subagents/artifacts/progress/6a6ca5d9/progress.md

---
**Output:**
Write your findings to exactly this path: /tmp/cnm-34-dataset-research.md
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