---
doc_type: decision
category: constraint
date: 2026-05-24
slug: make-provider-probing-explicit-in-doctor
status: active
area: doctor
tags: [doctor, providers, diagnostics, privacy]
---

## 背景

Doctor should diagnose local setup by default without sending data to an AI provider or consuming tokens.

## 决定

Keep provider API and native tool-call capability probing behind an explicit `cnm doctor --probe-provider` option using fixed non-repository test content.

## 理由

Users running diagnostics should not accidentally spend tokens or transmit repository content. Capability probing is useful, but it must be an explicit action.

## 后果

Default doctor output is local-only. Provider probe code must use fixed non-repository content and clearly report when a remote provider call is made.

## 相关文档

- `docs/adr/0019-make-provider-probing-explicit-in-doctor.md`
- `.codestable/architecture/ARCHITECTURE.md`
