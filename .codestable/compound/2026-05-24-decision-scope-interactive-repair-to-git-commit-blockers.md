---
doc_type: decision
category: constraint
date: 2026-05-24
slug: scope-interactive-repair-to-git-commit-blockers
status: active
area: interactive-repair
tags: [repair, conflicts, hooks]
---

## 背景

Interactive Repair could expand into a broad project test, lint, or hook-failure fixer if left undefined.

## 决定

Scope Interactive Repair to Git commit blockers such as merge conflicts. Git hook failures stop the flow and are surfaced to the developer instead of being repaired by cnm.

## 理由

Conflict repair is directly tied to making a Git commit possible, while arbitrary test/lint/hook repair would turn cnm into a general coding agent.

## 后果

Repair UI and Domain Tools should focus on commit blockers. Hook failures remain visible terminal outcomes for the developer to address.

## 相关文档

- `docs/adr/0010-scope-interactive-repair-to-git-commit-blockers.md`
- `.codestable/architecture/ARCHITECTURE.md`
