---
doc_type: explore
slug: next-feature-candidates
created: 2026-05-25
status: current
tags: [feature, planning, follow-up]
---

# 下一步 feature 候选结论

## 结论

下一条最值得进入 `cs-feat` 的能力增量是：**hunk-level commit split**。

## 为什么是它

1. 这是当前能力边界里最明确、最有用户价值、且被显式延期的缺口。
2. 现有 requirement / decision / implementation todo 都已经把它定义成“已知限制”，语义清楚，不需要重新发明问题定义。
3. 它直接补强 Interactive Commit 和 Autonomous Commit 两条核心链路，而不是只优化外围配套。

## 证据

- `.codestable/compound/2026-05-24-decision-defer-hunk-level-commit-split.md`
  - 明确写了 first version 只支持 file-level split，并把 same-file split 视为 limitation。
- `docs/implementation-todo.md`
  - Phase 7 明确记录：`Defer Hunk-level Commit Split`，说明这是刻意留到后续做的能力，而不是偶然缺失。
- `.codestable/requirements/interactive-commit.md`
  - 用户故事强调“工作区里同时有多件事时，希望先收窄提交范围，不把不相关改动一起带进去”。
- `.codestable/requirements/autonomous-commit.md`
  - 用户故事强调“希望 AI 能把一次工作区里的多类改动整理成合理提交”，hunk-level split 会提升 same-file mixed changes 的处理质量。

## 建议的 feature 定义

建议直接开：`cs-feat-design hunk-level-commit-split`

目标不是“一次做到最强”，而是先把下面问题定义清楚：

1. same-file mixed intent 怎么表达
2. AI plan 与最终 patch / stage 结果怎样对齐
3. rollback / repair / secret blocker 在 hunk 级别怎么保持安全边界
4. Interactive 与 Autonomous 是否共享同一套 split primitive

## 不建议优先做的方向

- 再做外围文档类 feature：价值不如补核心 commit 语义缺口高。
- 做全新 provider / UI 花样能力：当前主链路已经齐，same-file split 的产品缺口更真实。
- 直接跳到大范围 release polish：可以做，但它更像发布准备，不是最强 feature 增量。

## 推荐下一步

1. 先用 `cs-feat-design` 起草 `hunk-level-commit-split` 方案。
2. 设计时必须把现有 file-level split 的 truthful fallback 语义保住。
3. 设计完成后，再决定是否拆成多个实现 item。