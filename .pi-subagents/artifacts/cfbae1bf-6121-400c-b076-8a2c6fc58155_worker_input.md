# Task for worker

为 GitHub #34 独立冻结训练操作方不可见的 shadow/historical gate。必须先读取 `gh issue view 34 --repo ByteTrue/commit-now-myfriend` 和 `.cs/epics/001-o-offline-commit-flow/artifacts/034/decision.json`。你是唯一可查看 raw gate 的构建者；最终回复和 public manifest 严禁泄露 raw diff、commit SHA、gold wording 或 Guidance 原句。

输出：
1) 私有 raw：`/tmp/cnm-pierce34/private/shadow.jsonl`，至少 30 case；`/tmp/cnm-pierce34/private/historical.jsonl`，60 个 commit。创建目录并 chmod 700/600。
2) 公开 manifest：`.cs/epics/001-o-offline-commit-flow/artifacts/034/shadow-manifest.json`，只含 source policy、bundle SHA256、case counts、coverage counts、canonical repository exclusion groups（repo 名允许公开，用于训练排除）、date range、token bins、构建脚本/命令说明、secret scan counts，不含 commit SHA/diff/guidance/gold。
3) 只含方法和统计、不含 raw 的 `/tmp/cnm-shadow-worker-summary.md`。

选样要求：
- 来自至少 8 个公开 canonical repositories，commit 时间严格晚于 2024-11-18；这些 repo 将全部排除于训练。
- non-merge，完整多文件 textual diff；通过 GitHub API/必要时浅 clone 验证所有 changed files 都进入 diff，binary/missing/truncated patch 整个 commit 丢弃。
- shadow 30+：覆盖七 style；至少 10 个未公开措辞的 Guidance，至少 6 required-body；包括 Chinese+Conventional、prefix+body、exact bullets+issue reference、subject-only 与原 body 冲突等组合，但不要在 public manifest 写具体措辞；至少一个完整 prompt 接近 16,000-token 产品上限，一个应因超过 16,000 prompt tokens 而在推理前拒绝。用 Qwen/Qwen2.5-Coder-0.5B-Instruct@ea3f2471... 官方 tokenizer 计算。每 case 保存 diff/style/guidance/history/expected constraint metadata；不要求固定 gold message，后续盲评可直接对 diff 评分。
- historical 60：commit IDs/原始消息/diff/parent/repo，按固定 seed 从上述或额外 excluded repos 选择，覆盖小中长、多语言；public 只保存 commit-ID 列表的整体 hash。
- secret/credential/high-entropy/PII scanner 在本地先跑；任何疑似命中整 case 删除。不要把本项目私有/未发布 diff 发送到外部服务。
- 使用固定 seed `cnm-34-shadow-v1`，raw JSONL 每行规范 JSON；记录自身 schema version。计算文件和 case canonical hashes。
- raw 文件生成后不要打印内容；只打印路径、hash、计数和公开 repo exclusions。

如无法得到完整多文件 diff、接近 token 上限样本或可靠 tokenizer，不得伪造成功；报告 blocker。不要修改产品源码、Epic spec 或 GitHub issue。

---
**Output:**
Write your findings to exactly this path: /tmp/cnm-shadow-worker-summary.md
This path is authoritative for this run.
Ignore any other output filename or output path mentioned elsewhere, including output destinations in the base agent prompt, system prompt, or task instructions.

## Acceptance Contract
Acceptance level: reviewed
Completion is not accepted from prose alone. End with a structured acceptance report.

Criteria:
- criterion-1: Implement the requested change without widening scope
- criterion-2: Return evidence sufficient for an independent acceptance review

Required evidence: changed-files, tests-added, commands-run, validation-output, residual-risks, no-staged-files

Review gate: optional by reviewer.

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