---
doc_type: decision
category: architecture
date: 2026-05-24
slug: replace-split-command-with-primary-commit-flows
status: active
area: cli-command-surface
tags: [cli, commit-flow, split]
---

## 背景

The Go rewrite and TUI redesign are treated as a new product shape rather than a compatibility migration. Keeping `cnm split` as a primary or compatibility command would preserve old workflow debt in the redesigned CLI.

## 决定

Remove `cnm split` as a primary/compatibility command. Commit splitting belongs inside the main `cnm` Interactive Commit and `cnm auto` Autonomous Commit flows.

## 理由

Split decisions are part of commit planning, not a standalone product workflow. Embedding split behavior in the primary flows keeps the command surface aligned with the redesigned product semantics.

## 后果

Existing scripts or habits that call `cnm split` must migrate to primary flows. Documentation and tests should describe split decisions as part of commit planning.

## 相关文档

- `docs/adr/0001-replace-split-command-with-primary-commit-flows.md`
- `.codestable/architecture/ARCHITECTURE.md`
