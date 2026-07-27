# Task for reviewer

你是 #33 的独立盲评审。不要读取本会话、GitHub issue、模型身份、自动检查结果或之前 #30/#31/#32 的结果；只读取 `/tmp/cnm-pierce33/eval/blind-review-input.jsonl`。该文件有 26 行，每行包含 frozen system/user diff、style、guidance、expect_body 与候选 output。

逐项按以下冻结尺度评审 output：
- 2：准确且足够完整地概括实际变更，没有虚构的实质行为；
- 1：与变更实质相关，但不完整、过宽或不精确；
- 0：错误、空、复述代码/diff、实质虚构或不可用。
同时分别判断：非空 guidance 是否真正满足；expect_body=true 时是否存在 `subject + blank line + useful body`；Conventional/Angular 的 subject 格式是否符合该行 system 要求；是否出现 Markdown 包裹、解释、重复或代码/diff echo。不要因为语法流畅而提高语义分；有疑问取较低分。

必须把完整报告写到 `/tmp/cnm-pierce33/eval/semantic-review.md`，每个 case 一行表格，列为 id / score / guidance(pass|fail|n/a) / body(pass|fail|n/a) / style(pass|fail) / note。末尾给出精确汇总：score 2/1/0 数量、guidance x/5、required body x/4、Conventional/Angular 格式通过数量，以及是否满足门槛（无0且至少24个2、guidance5/5、body4/4）。不要修改任何输入文件。最后回复同样的简短汇总和报告路径。

---
**Output:**
Write your findings to exactly this path: /tmp/cnm-pierce33/eval/reviewer-response.md
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