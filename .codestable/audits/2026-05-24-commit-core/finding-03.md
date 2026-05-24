---
doc_type: audit-finding
audit: 2026-05-24-commit-core
finding_id: "maintainability-03"
nature: maintainability
severity: P2
confidence: high
suggested_action: cs-refactor
status: open
---

# Finding 03：runtime tool dispatch 函数过长且分支集中

## 速答

`ToolCallRuntime.executeToolCall` 一个函数负责所有 tool 的参数校验、策略检查、执行和错误包装，当前超过 100 行、分支约 30+，后续加新 tool 时容易漏掉 inspect-before-create、read-before-write、confirmation 等横切 guardrail。

## 关键证据

- `internal/runtime/runtime.go:110` — `executeToolCall` 入口。
- `internal/runtime/runtime.go:115` — `inspect_commit_scope` 分支开始。
- `internal/runtime/runtime.go:134` — `read_file` 分支包含 context policy、tool availability、参数校验、执行和 state update。
- `internal/runtime/runtime.go:164` — `create_commits` 分支包含 inspect-before-create guardrail 和参数解析。
- `internal/runtime/runtime.go:180` — `repair_file` 分支包含 read-before-write、allowed path、confirmation、执行等多个 guardrail。
- 静态扫描结果：`executeToolCall` 为 104 行，分支计数约 32。

## 影响

当前测试覆盖了主要 guardrail，所以这不是立即 bug；但 tool dispatch 是 commit core 的高风险扩展点，未来新增 tool 或调整参数时，单函数集中分支会增加回归概率和 review 成本。

## 修复方向

把每个 tool 的处理拆成小 handler（如 `executeInspectScope`、`executeReadFile`、`executeCreateCommits`、`executeRepairFile`），保留统一的 result/error helper；横切 guardrail 用小函数显式命名。

## 建议动作

`cs-refactor`，因为这是行为不变的结构优化，适合先设计低风险拆分再改。
