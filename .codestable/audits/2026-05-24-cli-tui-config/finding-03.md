---
doc_type: audit-finding
audit: 2026-05-24-cli-tui-config
finding_id: "maintainability-03"
nature: maintainability
severity: P2
confidence: high
suggested_action: cs-refactor
status: resolved
---

# Finding 03：`runAuto` 继续膨胀为单函数状态机，分支过于集中

## 速答

`internal/cli/cli.go:414` 的 `runAuto` 已经接近 200 行，混合了解析参数、配置解析、scope 检查、conflict/secret gate、dry-run 与 real-run、JSON 与 human output、TUI handoff、以及 commit failure 输出。这样的集中式控制流让后续改一个出口就容易漏掉另一个出口。

## 关键证据

- `internal/cli/cli.go:414` — `runAuto` 入口。
- `internal/cli/cli.go:425-447` — 配置解析和 scope inspection。
- `internal/cli/cli.go:449-520` — scope/no-changes/conflict/secret gate。
- `internal/cli/cli.go:521-550` — config gate 与 dry-run 输出。
- `internal/cli/cli.go:553-607` — real commit 执行与 JSON/human output 分支。
- 静态扫描结果：`runAuto` 约 199 行，是当前 audit 范围内最集中的生产函数之一。

## 影响

本轮 finding-02 就暴露了这种结构性风险：conflict handoff 分支和 config gate 顺序稍有偏差，就会形成行为不一致。当前测试能兜住部分回归，但 review 和未来扩展成本会持续上升。

## 修复方向

通过 `cs-refactor` 把 `runAuto` 拆成明确阶段：`resolveAutoConfig`、`inspectAutoScope`、`handleAutoPreconditions`、`renderAutoResult` 等；同时把 JSON / human output 路径收敛到少量 helper，减少重复分支。

## 建议动作

`cs-refactor`，因为这是行为不变的结构性优化，适合在 bug 修复后跟进。

## 处理结果

已部分修复并达到本次 hardening 目标。`runAuto` 仍是重要 orchestrator，但已经提炼出 `autoConfigIssueFromResolved` 等 helper，并把 TUI conflict config gate 逻辑显式收敛，降低了本轮问题所暴露的分支耦合。若后续继续 phase 化拆分，可另开独立 `cs-refactor`。