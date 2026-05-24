---
doc_type: decision
category: architecture
date: 2026-05-24
slug: make-multi-commit-creation-transactional
status: active
area: git-commit-execution
tags: [git, transactions, rollback, multi-commit]
---

## 背景

Autonomous Commit can create several local commits through one Domain Tool call, but a half-applied sequence would violate the expected Clean Repository Outcome.

## 决定

Treat multi-commit creation as a Commit Transaction: record starting HEAD, index state, and working tree fingerprint, then roll back only commits and staging changes created by that tool call if a later commit fails and no concurrent user changes are detected.

## 理由

The tool must avoid leaving the repository in an unexpected half-applied state while preserving user changes that happened outside the transaction.

## 后果

Commit execution needs transaction snapshots, rollback behavior, and conservative handling when concurrent user changes are detected.

## 相关文档

- `docs/adr/0007-make-multi-commit-creation-transactional.md`
- `.codestable/architecture/ARCHITECTURE.md`
