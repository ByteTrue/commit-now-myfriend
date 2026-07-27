# Task for scout

在 /Users/byte/workspace/projects/commit-now-myfriend 建立最小核心路径地图，只读：如果产品唯一职责是读取 git diff 并生成一条 commit message，当前哪些文件/函数可复用，哪些整个目录可删？估计最小保留方案与当前代码的 LOC/依赖差异，指出是否有比推倒重来更低风险的路径。不要修改任何文件，不要运行子代理。

---
Update progress at: /Users/byte/workspace/projects/commit-now-myfriend/.pi-subagents/artifacts/progress/741bc9ed-1c06-4acb-b4d3-f4b5ca431014/progress.md

---
**Output:**
Write your findings to exactly this path: /Users/byte/workspace/projects/commit-now-myfriend/.pi-subagents/artifacts/outputs/741bc9ed-1c06-4acb-b4d3-f4b5ca431014/parallel-0/2-scout/context.md
This path is authoritative for this run.
Ignore any other output filename or output path mentioned elsewhere, including output destinations in the base agent prompt, system prompt, or task instructions.

## Acceptance Contract
Acceptance level: reviewed
Completion is not accepted from prose alone. End with a structured acceptance report.

Criteria:
- criterion-1: Implement the requested change without widening scope
- criterion-2: Return evidence sufficient for an independent acceptance review

Required evidence: changed-files, tests-added, commands-run, validation-output, residual-risks, no-staged-files

Review gate: required by reviewer.

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
    },
    {
      "id": "criterion-2",
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