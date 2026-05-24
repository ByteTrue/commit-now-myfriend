---
doc_type: decision
category: constraint
date: 2026-05-24
slug: defer-hunk-level-commit-split
status: active
area: commit-split
tags: [commit-split, hunk-level, scope]
---

## 背景

Hunk-level Commit Split would allow different hunks in one file to belong to different commits, but it requires reliable patch application, index recovery, and TUI hunk selection.

## 决定

Defer hunk-level split from the first version. Support File-level Commit Split only and report same-file split needs as a limitation rather than attempting unsafe partial staging.

## 理由

File-level split is safer and matches the first-version reliability target. Hunk-level split needs additional UI and Git index recovery guarantees.

## 后果

The product should explicitly describe same-file split limitations. Implementation should avoid partial staging flows that could corrupt user intent.

## 相关文档

- `docs/adr/0014-defer-hunk-level-commit-split.md`
- `.codestable/architecture/ARCHITECTURE.md`
