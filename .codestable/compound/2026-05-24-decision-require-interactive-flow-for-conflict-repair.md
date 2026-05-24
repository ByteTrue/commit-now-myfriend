---
doc_type: decision
category: constraint
date: 2026-05-24
slug: require-interactive-flow-for-conflict-repair
status: active
area: interactive-repair
tags: [repair, conflicts, tui, autonomous-commit]
---

## 背景

Merge conflict resolution can change source semantics and should not happen in a fully non-interactive run.

## 决定

Require conflict repair to happen through the Full-screen TUI. Non-interactive Autonomous Commit reports conflicts and fails instead of editing conflicted files.

## 理由

Conflict resolution needs developer judgment. The TUI provides an explicit human-confirmed context for repair, while non-interactive flow should not silently alter source semantics.

## 后果

Autonomous Commit must fail predictably on conflicts unless it hands off to an interactive flow. Repair implementation belongs behind TUI confirmation.

## 相关文档

- `docs/adr/0011-require-interactive-flow-for-conflict-repair.md`
- `.codestable/architecture/ARCHITECTURE.md`
