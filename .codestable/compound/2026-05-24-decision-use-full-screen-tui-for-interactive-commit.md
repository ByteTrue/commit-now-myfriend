---
doc_type: decision
category: architecture
date: 2026-05-24
slug: use-full-screen-tui-for-interactive-commit
status: active
area: interactive-commit
tags: [tui, interactive-commit, ux]
---

## 背景

Interactive Commit must support commit planning, diff review, message editing, replanning, and natural-language developer feedback in one cohesive flow.

## 决定

Build `cnm` Interactive Commit around a Full-screen TUI rather than a sequence of prompts.

## 理由

The interaction model is closer to a terminal application than to a confirmation wizard. A full-screen interface can keep scope, diff, AI activity, plan review, and message editing visible in one workflow.

## 后果

Interactive Commit implementation centers on `internal/tui`; non-interactive output paths still need graceful plain-text/JSON behavior where applicable.

## 相关文档

- `docs/adr/0002-use-full-screen-tui-for-interactive-commit.md`
- `.codestable/architecture/ARCHITECTURE.md`
