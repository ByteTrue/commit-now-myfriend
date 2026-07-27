# Task for reviewer

[Read from: /Users/byte/workspace/projects/commit-now-myfriend/plan.md, /Users/byte/workspace/projects/commit-now-myfriend/progress.md]

你是一个独立的盲审 reviewer B。你的任务是审查 /tmp/cnm-pierce35/review/blind-review-B.jsonl 中的 200 个案例。

对于每个案例，你需要：
1. 阅读完整的 diff
2. 查看 body_policy（optional 或 required）
3. 评估 teacher_target（subject 和 body）
4. 检查 teacher_evidence（subject_evidence 和 body_evidence）

然后根据以下标准评分：

**critical_error** (boolean): true 表示存在材料性错误，如：
- subject/body 描述与 diff 实际变更不符
- 捏造了 diff 中不存在的功能
- 遗漏了核心变更
- 对 required body 做出了 unsupported 的声明

**fully_grounded** (boolean): true 表示 subject 和 body 中的所有材料性声明都能从 diff 中得到支持

**subject_quality** (integer 0-2):
- 2: 准确、具体、简洁，聚焦于主要变更
- 1: 基于 diff 但模糊、不完整或聚焦于次要变更
- 0: 误导性或不可用

**body_quality** (integer 0-2 或 null):
- 如果 body_policy 是 optional 且 body 为空，则为 null
- 2: 提供了有用的理由/细节，且基于 diff
- 1: 基于 diff 但薄弱、冗余或作用最小
- 0: unsupported、矛盾或在 required 时无用

**evidence_quality** (integer 0-2):
- 2: 引用的 changed-line 摘录直接支持所有材料性声明
- 1: 部分支持
- 0: 不支持

**reason** (string): 一句话解释你的评分理由

输出要求：
1. 将结果写入 /tmp/cnm-pierce35/review/scores-B.jsonl
2. 每行一个 JSON 对象，包含字段：index, reviewer (固定为 "B"), input_slice_sha256 (从 review-manifest.json 读取), critical_error, fully_grounded, subject_quality, body_quality, evidence_quality, reason
3. 按 index 顺序（0-199）输出

重要约束：
- 不要修复标签
- 不要使用外部资源
- 不要搜索原始 commit
- 不要推断仓库身份
- 独立完成所有 200 个案例

完成后输出 'Reviewer B complete' 和你的评分文件路径。

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