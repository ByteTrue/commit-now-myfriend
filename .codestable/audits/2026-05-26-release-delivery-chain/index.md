---
doc_type: audit-index
audit: 2026-05-26-release-delivery-chain
scope: .github/workflows, scripts, docs/distribution.md, docs/release-runbook.md, package.json, .goreleaser.yml
created: 2026-05-26
status: active
total_findings: 3
---

# release-delivery-chain 审计报告

## 范围

本次审计覆盖交付链相关边界：`.github/workflows/`、`scripts/`、`docs/distribution.md`、`docs/release-runbook.md`、`package.json`、`.goreleaser.yml`。关注维度为 bug / security / performance / maintainability / arch-drift，重点检查 release workflow、npm wrapper、Windows smoke coverage、artifact contract 和文档一致性。

实际检查包括：workflow 触发与 job 顺序、installer / wrapper 脚本、Node smoke tests、package scripts、GoReleaser archive 约束，以及 distribution / runbook 的当前口径。

## 总评

最近几轮 hardening 已经把 release 链从“主要靠文档和人工步骤”推进到了“有 workflow、有 checksum verification、有 Windows contract smoke”的状态，方向正确。当前 3 条发现里，installer 临时文件隔离问题和 Windows smoke 重复调用问题都已修复；剩余主要缺口是 Windows 真机执行证据不足。

## 发现清单

| # | 性质 | 严重度 | 置信度 | 标题 | 文件 |
|---|---|---|---|---|---|
| 1 | bug | P1 | medium | Windows smoke 仍停留在 Linux 上的契约模拟，没有真实 `cnm.exe` 启动证据 | [finding-01.md](finding-01.md) |
| 2 | bug | P2 | medium | npm installer 的 archive / checksums 临时文件名按版本固定，缺少并发隔离 | [finding-02.md](finding-02.md) · 已修复 |
| 3 | maintainability | P2 | high | CI / release workflow 已重复执行 Windows smoke，验证入口开始分叉 | [finding-03.md](finding-03.md) · 已修复 |

## 按维度分布

| 性质 | P0 | P1 | P2 | 合计 |
|---|---|---|---|---|
| bug | 0 | 1 | 1 | 2 |
| security | 0 | 0 | 0 | 0 |
| performance | 0 | 0 | 0 | 0 |
| maintainability | 0 | 0 | 1 | 1 |
| arch-drift | 0 | 0 | 0 | 0 |
| **合计** | **0** | **1** | **2** | **3** |

## 下一步建议

- **P1 本迭代修**：如果你决定重新关注 Windows 平台，优先处理 finding-01，补真实 Windows 执行证据或把 contract smoke 明确降级为模拟验证。
- finding-02 和 finding-03 已完成，可从后续交付质量清单中移除。
- 本次审计当前剩余的高价值动作，已经收敛到 release 验证可信度本身。