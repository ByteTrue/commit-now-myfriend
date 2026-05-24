---
doc_type: decision
category: architecture
date: 2026-05-24
slug: rebuild-core-architecture-around-new-product-semantics
status: active
area: core-architecture
tags: [go-rewrite, working-tree-commit, tool-call-loop, command-surface]
---

## 背景

The current Go rewrite preserved much of the staged-first message-generation workflow, which conflicted with the redesigned product semantics.

## 决定

Reuse reliable low-level pieces where useful, but rebuild the core workflow, provider abstraction, TUI, configuration schema, and command routing around Working Tree Commit, Tool Call Loop, and the simplified command surface.

## 理由

The redesigned product is not just a TypeScript-to-Go compatibility migration. Core architecture must match the new product language and commit workflow semantics.

## 后果

Older staged-first assumptions are no longer the source of truth. Architecture, tests, docs, and implementation should align with `CONTEXT.md`, the ADR set, and the Go-native `cnm` product shape.

## 相关文档

- `docs/adr/0021-rebuild-core-architecture-around-new-product-semantics.md`
- `CONTEXT.md`
- `PLAN.md`
- `.codestable/architecture/ARCHITECTURE.md`
