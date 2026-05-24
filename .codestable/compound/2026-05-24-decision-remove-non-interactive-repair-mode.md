---
doc_type: decision
category: constraint
date: 2026-05-24
slug: remove-non-interactive-repair-mode
status: active
area: autonomous-commit
tags: [repair, autonomous-commit, tui]
---

## 背景

Conflict repair belongs in the Full-screen TUI because it can change source semantics and needs developer judgment.

## 决定

Do not provide `cnm auto --repair`. Non-interactive Autonomous Commit fails on conflicts, while `cnm auto --tui` may hand off to Interactive Commit for Interactive Repair.

## 理由

A non-interactive repair mode would imply autonomous source edits during a commit workflow. That conflicts with the safety boundary for conflict resolution.

## 后果

Command design should avoid autonomous repair flags. Users who want repair must enter the interactive flow.

## 相关文档

- `docs/adr/0012-remove-non-interactive-repair-mode.md`
- `.codestable/architecture/ARCHITECTURE.md`
