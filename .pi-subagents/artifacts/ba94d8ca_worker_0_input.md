# Task for worker

接手并在 10 分钟内完成 cnm #34 unseen gate。前一 worker 已把 10 个 public repos 的 bare clone 和 tokenizer 放在 `/tmp/cnm-pierce34/`，并有 `/tmp/cnm-pierce34/private/build_gate.py`，但全量扫描太慢。不要读取或修改训练数据。不要在回复中泄露 raw diff、commit SHA、guidance wording 或 gold。

必须采用有界重写/修正：
1. `TOKENIZERS_PARALLELISM=false`，每 repo 按固定 seed 最多检查 80 个 post-2024-11-18 non-merge commit；累计 100 个符合 clean/complete/multi-file candidate 就停止。
2. 至少 8 个 repo exclusion；只允许 MIT/Apache-2.0/BSD-3-Clause。
3. 冻结 `/tmp/cnm-pierce34/private/shadow.jsonl` 至少 30 case（覆盖7 styles、>=10组合guidance、>=5 body、history-conflict、near-limit/over-limit，token 必须 `len(encoded['input_ids'])`）；冻结 `/tmp/cnm-pierce34/private/historical.jsonl` 至少60。underlying commit 可在两个集合重叠，但每个文件内部唯一。
4. 写 public manifest 到 `.cs/epics/001-o-offline-commit-flow/artifacts/034/shadow-manifest.json`，只包含 hashes/counts/coverage/rubric/repository_exclusions、hashed commit/diff exclusions，不含 raw/gold/guidance wording。
5. 对 private files 做 secret/PII scan、权限0600、SHA256；验证 manifest 与文件一致。若 near/over 大 commit 找不到，最多额外定向扫描各一个，不要重新全量扫。
6. 回复只给 PASS/FAIL、公开 manifest 路径、private 路径、计数和覆盖，不泄露 raw。

可直接替换旧 builder；不要无限扫描。

---
**Output:**
Write your findings to exactly this path: /tmp/cnm-shadow-worker-summary-v2.md
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