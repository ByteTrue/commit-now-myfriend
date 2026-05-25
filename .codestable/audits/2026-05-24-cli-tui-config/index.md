---
doc_type: audit-index
audit: 2026-05-24-cli-tui-config
scope: internal/cli, internal/tui, internal/config
created: 2026-05-24
status: active
total_findings: 3
---

# cli-tui-config 审计报告

## 范围

本次审计覆盖 `internal/cli`、`internal/tui`、`internal/config`。重点对照 `.codestable/architecture/ARCHITECTURE.md` 中的 CLI 调度层、配置解析/Secret Store 边界、Interactive Commit Full-screen TUI 和 `cnm doctor`/`cnm config` 辅助命令语义，检查 bug、security、performance、maintainability、arch-drift 五个维度。

实际检查包括：目标包测试、命令调度路径、config panel 写入路径、conflict repair handoff、machine output 的敏感字段处理，以及 TTY / rich output 相关实现。

## 总评

目标包测试当前通过，但本轮范围内仍有 3 条值得处理的发现：2 条 bug、1 条 maintainability。最优先的是 config panel 对 API key 的写入逻辑会把 key 绑定到“打开 panel 时的 provider”，而不是用户刚在 panel 里切换后的 provider；其次是 `cnm auto --tui` 的 conflict handoff 在配置解析失败时仍会继续走 repair 分支，导致用户拿到次生错误而不是明确的配置错误。

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

## 下一步建议

- **P1 本迭代修**：先走 finding-01，避免用户在 config panel 中误把密钥写入错误 provider 的 Secret Store slot；再走 finding-02，保证 conflict handoff 对配置缺失/解析失败给出确定性错误。
- **P2 下个迭代修**：finding-03 建议通过 `cs-refactor` 拆解 `runAuto`，把 scope 检查、config gate、TUI handoff、JSON/non-JSON 输出分层。
- 本次未发现 P0；修复应分别进入 `cs-issue` / `cs-refactor`，不在 `cs-audit` 中顺手改。