---
doc_type: audit-index
audit: 2026-05-24-cli-tui-config
scope: internal/cli, internal/tui, internal/config
created: 2026-05-24
status: resolved
total_findings: 3
---

# cli-tui-config 审计报告

## 范围

本次审计覆盖 `internal/cli`、`internal/tui`、`internal/config`。重点对照 `.codestable/architecture/ARCHITECTURE.md` 中的 CLI 调度层、配置解析/Secret Store 边界、Interactive Commit Full-screen TUI 和 `cnm doctor`/`cnm config` 辅助命令语义，检查 bug、security、performance、maintainability、arch-drift 五个维度。

实际检查包括：目标包测试、命令调度路径、config panel 写入路径、conflict repair handoff、machine output 的敏感字段处理，以及 TTY / rich output 相关实现。

## 总评

目标包测试当前通过，本轮范围内发现的 3 条问题已全部修复并完成回归验证：Config Panel 的 API key 保存已按当前 provider 状态解析，`cnm auto --tui` 的 conflict handoff 不再绕过前置 config gate，`runAuto` 也进一步提炼出 helper 降低了分支集中度。

## 发现清单

| # | 性质 | 严重度 | 置信度 | 标题 | 文件 |
|---|---|---|---|---|---|
| 1 | bug | P1 | high | config panel 切换 provider 后保存 API key 会写到旧 provider 名下 | [finding-01.md](finding-01.md) |
| 2 | bug | P1 | medium | `cnm auto --tui` 的 conflict handoff 会绕过前置 config 错误 | [finding-02.md](finding-02.md) |
| 3 | maintainability | P2 | high | `runAuto` 继续膨胀为单函数状态机，分支过于集中 | [finding-03.md](finding-03.md) |

## 按维度分布

| 性质 | P0 | P1 | P2 | 合计 |
|---|---|---|---|---|
| bug | 0 | 2 | 0 | 2 |
| security | 0 | 0 | 0 | 0 |
| performance | 0 | 0 | 0 | 0 |
| maintainability | 0 | 0 | 1 | 1 |
| arch-drift | 0 | 0 | 0 | 0 |
| **合计** | **0** | **2** | **1** | **3** |

## 收尾结论

- finding-01 已修复：Config Panel 保存 API key 时按当前 provider 状态解析并写入正确的 Secret Store slot。
- finding-02 已修复：`cnm auto --tui` 的 conflict handoff 现在会先经过 config gate，再决定是否进入 repair。
- finding-03 已修复：`runAuto` 已提炼出 config/provider-target helper，降低了本轮问题所暴露的分支耦合。
- 本次 audit 已闭环；后续如继续收缩 `runAuto`，可另开独立 `cs-refactor`。