---
doc_type: decision
category: tech-stack
date: 2026-05-24
slug: use-bubble-tea-ecosystem-for-tui
status: active
area: tui
tags: [bubble-tea, bubbles, lip-gloss, tui]
---

## 背景

Interactive Commit needs a polished Full-screen TUI while keeping the CLI Go-native and lightweight.

## 决定

Build the TUI with Bubble Tea, Lip Gloss, and Bubbles. Keep workflow and Tool Call Runtime code independent of the TUI framework.

## 理由

Bubble Tea's model/update/view architecture supports testable terminal state. Lip Gloss and Bubbles provide the styling and components needed for the Focused TUI standard.

## 后果

Non-interactive `cnm auto` must not depend on full-screen rendering. TUI code should remain an adapter around workflow state, not the owner of core runtime behavior.

## 相关文档

- `docs/adr/0017-use-bubble-tea-ecosystem-for-tui.md`
- `.codestable/architecture/ARCHITECTURE.md`
