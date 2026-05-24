---
doc_type: audit-index
audit: 2026-05-24-commit-core
scope: internal/git, internal/runtime, internal/providers commit core chain
created: 2026-05-24
status: active
total_findings: 3
---

# commit-core 审计报告

## 范围

本次审计范围由用户确认：`internal/git`、`internal/runtime`、`internal/providers`。扫描维度覆盖 bug、security、performance、maintainability、arch-drift，重点对照 `.codestable/architecture/ARCHITECTURE.md` 中 Git / Tool Call Runtime / Provider adapters 的边界，以及相关 decision（transactional commit、Domain Tools、Secret Blocker、无远程 telemetry）。

实际检查内容包括：静态结构和复杂度、provider HTTP 请求/响应路径、runtime tool dispatch、Git scope/commit/rollback 路径、核心测试覆盖、以及针对范围内包的 `go test ./internal/git ./internal/runtime ./internal/providers`。

## 总评

范围内核心测试全部通过，整体架构与 CodeStable architecture / decisions 大体一致：provider adapter、runtime、Git side effects 分层清楚，repair 的 read-before-write / confirmation guardrail 已有测试覆盖。本次发现 3 条问题：1 条 security P1，1 条 bug P1，1 条 maintainability P2。最值得优先处理的是 provider 错误响应直接进入错误信息，以及 rollback 在 reset 前不先检查工作区是否相对 snapshot 发生并发变化。

## 发现清单

| # | 性质 | 严重度 | 置信度 | 标题 | 文件 |
|---|---|---|---|---|---|
| 1 | security | P1 | medium | Provider 错误/解析失败会把响应片段原样放进错误信息 | [finding-01.md](finding-01.md) |
| 2 | bug | P1 | medium | rollback 在执行 reset 前没有先做并发工作区变化保护 | [finding-02.md](finding-02.md) |
| 3 | maintainability | P2 | high | runtime 的 tool dispatch 函数过长且分支集中 | [finding-03.md](finding-03.md) |

## 按维度分布

| 性质 | P0 | P1 | P2 | 合计 |
|---|---|---|---|---|
| bug | 0 | 1 | 0 | 1 |
| security | 0 | 1 | 0 | 1 |
| performance | 0 | 0 | 0 | 0 |
| maintainability | 0 | 0 | 1 | 1 |
| arch-drift | 0 | 0 | 0 | 0 |
| **合计** | **0** | **2** | **1** | **3** |

## 下一步建议

- **P1 本迭代修**：finding-01 建议走 `cs-issue`，先定义 provider error redaction / snippet policy；finding-02 建议走 `cs-issue`，补 rollback 并发保护测试后修。
- **P2 有空再看**：finding-03 建议走 `cs-refactor`，把每个 tool 的 validation/execution 拆成小 handler。
- 本次未发现 P0；不建议在 `cs-audit` 内直接修改代码。
