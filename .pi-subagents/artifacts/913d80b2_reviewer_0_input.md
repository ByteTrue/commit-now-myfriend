# Task for reviewer

对 GitHub #31 的最终高风险结果做独立只读语义评审。读取：`/tmp/cnm-train31/eval/high-risk.jsonl`（含每例完整 system/user/diff）、`/tmp/cnm-train31/eval/results-bf16-adapter/results.json`、`/tmp/cnm-train31/eval/results-q4/results.json`、两份 summary。对每个 26 case 的 BF16 与 Q4 输出分别核对：是否忠实覆盖 diff 的核心变化、是否无幻觉/重复、是否满足指定 style、附加 guidance、body 期望。逐候选给出 pass/partial/fail 数量与关键失败 case，明确量化退化程度；根据 #31 的门（风格/提示词仍明显不可靠就 STOP）给唯一 GO/STOP 结论。自动结构检查不能替代语义。不要修改任何文件。

---
**Output:**
Write your findings to exactly this path: /tmp/cnm-train31/eval/semantic-review.md
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