---
doc_type: decision
category: constraint
date: 2026-05-24
slug: no-remote-telemetry-in-first-version
status: active
area: privacy
tags: [telemetry, privacy, diagnostics]
---

## 背景

The CLI handles local repository content and sends selected context to the user's configured AI provider. Adding a separate telemetry channel would weaken trust.

## 决定

Do not collect remote telemetry in the first version. Limit diagnostics to explicit local debug output.

## 理由

A local commit tool should minimize hidden data flows, especially when repository content and provider prompts are involved.

## 后果

Debug logging must be opt-in and avoid secrets, full diffs, prompts, and provider responses by default. Future telemetry proposals need a fresh decision.

## 相关文档

- `docs/adr/0016-no-remote-telemetry-in-first-version.md`
- `.codestable/architecture/ARCHITECTURE.md`
