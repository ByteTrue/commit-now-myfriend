# Task for worker

独立增强 cnm #34 hidden gate 的公开泄漏签名；可以读取 `/tmp/cnm-pierce34/private/shadow.jsonl` 和 historical.jsonl，但禁止在回复/公开文件泄露 raw diff、commit、message、guidance。

在 `.cs/epics/001-o-offline-commit-flow/artifacts/034/shadow-manifest.json` 新增 `leakage_exclusions`，对 hidden 两集合每个 underlying commit（去重）输出仅哈希：
1. `normalized_patch_sha256`：移除 diff/path/index/hunk-header 元数据，只保留 +/- changed line，去前缀、strip、collapse whitespace、lower 后连接并 SHA256；
2. `changed_token_minhash`：对上述 changed content tokenize 为 `[A-Za-z0-9_]+|非空白符号`，5-token shingles SHA256，保存字典序最小 8 个（不足则全量）；
3. historical `normalized_target_sha256` 与 target 3-token shingle 最小8哈希；shadow 无 gold 不写 target；
4. 每项只保留 `set`（shadow/historical）、这些 signatures 与原有 source_commit_sha256/complete_diff_sha256 哈希，不保留 raw ID。
5. 新增 algorithm version/exact normalization说明、counts、更新 manifest 自身之外的 bundle hash。验证 private file原 hash/0600不变。

还要新增 `repository_groups`：将公开 repository_exclusions 的 canonical repo 映射为自身 canonical group（若 clone remote 能证明 fork parent 则映射 parent，否则自身），不猜测。

回复只给 PASS/FAIL、条目数量和公开 manifest 路径。

---
**Output:**
Write your findings to exactly this path: /tmp/cnm-shadow-signature-summary.md
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