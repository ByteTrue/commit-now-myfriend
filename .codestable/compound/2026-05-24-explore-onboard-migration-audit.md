# CodeStable onboard migration audit

> 日期：2026-05-24
> 模式：保守接入
> 结论：已创建 `.codestable/` 骨架；本次不移动、不删除、不重命名既有文档。

## 扫描结论

- `.codestable/`：原先不存在，本次新建。
- 旧版目录：未检测到 `easysdd/` 或 `codestable/`。
- 既有文档：存在 `README.md`、`CONTEXT.md`、`PLAN.md`、`docs/`、`.sisyphus/` 等文档资产，因此判断为“迁移路径”，不是空仓库路径。
- 本次用户选择：保守接入 —— 先建骨架和审计报告，所有既有文档保持原位。

## 迁移审计表

| 现有文件 | 推测内容类型 | 建议归入 CodeStable | 置信度 | 本次处理 |
|---|---|---|---|---|
| `CONTEXT.md` | 领域语言 / 产品语义 / 架构上下文 | `.codestable/architecture/domain-language.md` 或回填 `.codestable/architecture/ARCHITECTURE.md` | 高 | 保留原位，待后续确认 |
| `PLAN.md` | 产品完成计划 / roadmap 来源 | `.codestable/roadmap/product-completion/product-completion-roadmap.md` | 高 | 保留原位，待后续确认 |
| `docs/adr/*.md` | 架构决策记录 | `.codestable/compound/YYYY-MM-DD-decision-{slug}.md` 或汇总索引到 `architecture/ARCHITECTURE.md` | 高 | 保留原位，待后续确认 |
| `docs/implementation-todo.md` | 实现 TODO / 阶段清单 | `.codestable/roadmap/product-completion/product-completion-items.yaml` 或相关 feature checklist | 中 | 保留原位，待后续确认 |
| `docs/distribution.md` | 分发指南 / 发布说明 | 保留在 `docs/` 作为指南；必要时在 architecture 中引用分发架构 | 中 | 保留原位 |
| `docs/agents/domain.md` | agent 工作说明 / 领域文档入口 | `.codestable/attention.md` 摘要或 `.codestable/reference/` 项目补充文档 | 中 | 保留原位 |
| `docs/agents/issue-tracker.md` | GitHub Issues 工作说明 | `.codestable/attention.md` 摘要或 `.codestable/reference/` 项目补充文档 | 中 | 保留原位 |
| `docs/agents/triage-labels.md` | triage label 说明 | `.codestable/attention.md` 摘要或 `.codestable/reference/` 项目补充文档 | 中 | 保留原位 |
| `README.md` | 对外 README / 用户入口 | 继续保留在根目录；不建议迁入 CodeStable | 高 | 保留原位 |
| `AGENTS.md` | 通用 agent 指令入口 | 继续保留；CodeStable 子技能入口固定为 `.codestable/attention.md` | 高 | 保留原位 |
| `.sisyphus/plans/commit-now-myfriend-v1.md` | 历史 v1 产品计划 | `.codestable/roadmap/commit-now-myfriend-v1/commit-now-myfriend-v1-roadmap.md` 或归档为 learning/explore | 中 | 保留原位，待后续确认 |
| `.sisyphus/notepads/commit-now-myfriend-v1/decisions.md` | 历史决策笔记 | `.codestable/compound/YYYY-MM-DD-decision-{slug}.md` 拆分或归档 | 中 | 保留原位，待后续确认 |
| `.sisyphus/notepads/commit-now-myfriend-v1/issues.md` | 历史问题记录 | `.codestable/issues/` 拆分或归档为 learning | 中 | 保留原位，待后续确认 |
| `.sisyphus/notepads/commit-now-myfriend-v1/learnings.md` | 历史学习记录 | `.codestable/compound/YYYY-MM-DD-learning-{slug}.md` 拆分或归档 | 中 | 保留原位，待后续确认 |
| `.sisyphus/notepads/commit-now-myfriend-v1/problems.md` | 历史问题 / 风险记录 | `.codestable/issues/` 拆分或归档为 learning | 中 | 保留原位，待后续确认 |

## 已新建 / 刷新的 CodeStable 骨架

- `.codestable/attention.md`
- `.codestable/architecture/ARCHITECTURE.md`
- `.codestable/requirements/.gitkeep`
- `.codestable/roadmap/.gitkeep`
- `.codestable/features/.gitkeep`
- `.codestable/issues/.gitkeep`
- `.codestable/compound/.gitkeep`
- `.codestable/tools/search-yaml.py`
- `.codestable/tools/validate-yaml.py`
- `.codestable/reference/code-dimensions.md`
- `.codestable/reference/maintainer-notes.md`
- `.codestable/reference/requirement-example.md`
- `.codestable/reference/shared-conventions.md`
- `.codestable/reference/system-overview.md`
- `.codestable/reference/tools.md`

## 后续建议

1. 用 `cs-arch backfill` 从 `CONTEXT.md`、`PLAN.md`、`docs/adr/` 回填架构总览。
2. 用 `cs-roadmap new/update` 把 `PLAN.md` 中仍有效的产品完成计划整理成 roadmap。
3. 用 `cs-decide` 逐条迁移或汇总 `docs/adr/*.md` 中仍有效的长期决策。
4. 用 `cs-note` 把构建、测试、凭证、目录禁区等“每次 CodeStable 技能启动都必须知道”的短规则追加到 `.codestable/attention.md`。
