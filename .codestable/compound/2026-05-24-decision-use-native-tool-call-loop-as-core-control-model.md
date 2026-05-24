---
doc_type: decision
category: architecture
date: 2026-05-24
slug: use-native-tool-call-loop-as-core-control-model
status: active
area: tool-call-runtime
tags: [tool-call-loop, providers, domain-tools]
---

## 背景

The product goal is for AI to drive the commit workflow through provider-native tool calls, not by emitting a declarative JSON plan that local code later interprets.

## 决定

Use a Tool Call Loop where each supported Provider Protocol maps native tool calls into the same local Tool Call Runtime.

## 理由

Provider-native tool feedback gives the model a chance to correct invalid calls without turning free-form JSON parsing into a workflow blocker.

## 后果

Providers own protocol adaptation; the local runtime validates and executes Domain Tools. This supersedes the earlier declarative Commit Plan / Execution Plan split as the core control model.

## 相关文档

- `docs/adr/0006-use-native-tool-call-loop-as-core-control-model.md`
- `2026-05-24-decision-separate-commit-plan-from-execution-plan.md`
- `.codestable/architecture/ARCHITECTURE.md`
