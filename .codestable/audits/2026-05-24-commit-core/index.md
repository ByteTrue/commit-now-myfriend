---
doc_type: audit-index
audit: 2026-05-24-commit-core
scope: internal/git, internal/runtime, internal/providers commit core chain
created: 2026-05-24
status: resolved
total_findings: 3
---

# commit-core 审计报告

## 范围

本次审计范围由用户确认：`internal/git`、`internal/runtime`、`internal/providers`。扫描维度覆盖 bug、security、performance、maintainability、arch-drift，重点对照 `.codestable/architecture/ARCHITECTURE.md` 中 Git / Tool Call Runtime / Provider adapters 的边界，以及相关 decision（transactional commit、Domain Tools、Secret Blocker、无远程 telemetry）。

实际检查内容包括：静态结构和复杂度、provider HTTP 请求/响应路径、runtime tool dispatch、Git scope/commit/rollback 路径、核心测试覆盖、以及针对范围内包的 `go test ./internal/git ./internal/runtime ./internal/providers`。

## 总评

范围内核心测试全部通过，整体架构与 CodeStable architecture / decisions 大体一致：provider adapter、runtime、Git side effects 分层清楚，repair 的 read-before-write / confirmation guardrail 已有测试覆盖。本次发现的 3 条问题已全部完成修复并回归验证：provider 错误信息不再包含响应片段，rollback 恢复为 post-reset 透明校验语义，runtime tool dispatch 已拆成小 handler。

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

## 收尾结论

- finding-01 已修复：provider 错误路径不再回显 response snippet。
- finding-02 已修复：rollback 维持 `reset --mixed` 后校验 status 的事务语义，并补足测试说明。
- finding-03 已修复：`executeToolCall` 已拆分为小 handler 和统一 helper。
- 本次 audit 可以视为已闭环；后续建议转入新的审计范围或发布准备。
