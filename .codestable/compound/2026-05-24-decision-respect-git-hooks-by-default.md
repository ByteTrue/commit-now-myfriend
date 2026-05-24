---
doc_type: decision
category: constraint
date: 2026-05-24
slug: respect-git-hooks-by-default
status: active
area: git-commit-execution
tags: [git-hooks, validation, rollback]
---

## 背景

Project validation often lives in Git hooks. Although cnm does not run project checks itself, commit creation still encounters hooks.

## 决定

Let Git hooks run during commit creation by default. Treat hook failures as commit failures that trigger Index Snapshot restoration and Commit Transaction rollback where safe.

## 理由

Respecting hooks preserves repository-local validation policy without making cnm a general-purpose test runner.

## 后果

Commit execution must surface hook failures clearly. Rollback behavior must stay conservative when safe restoration cannot be guaranteed.

## 相关文档

- `docs/adr/0009-respect-git-hooks-by-default.md`
- `.codestable/architecture/ARCHITECTURE.md`
