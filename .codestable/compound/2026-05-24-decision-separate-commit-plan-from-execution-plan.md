---
doc_type: decision
category: architecture
date: 2026-05-24
slug: separate-commit-plan-from-execution-plan
status: superseded
superseded-by: use-native-tool-call-loop-as-core-control-model
area: tool-call-runtime
tags: [commit-plan, execution-plan, runtime]
---

**[已取代]** 见 `2026-05-24-decision-use-native-tool-call-loop-as-core-control-model.md`。

## 背景

The AI should decide commit grouping, messages, and developer-facing intent, but should not directly control low-level Git command order.

## 决定

Have the AI produce a declarative Commit Plan and let the Tool Call Runtime derive a validated Execution Plan.

## 理由

This kept local side effects predictable and testable while retaining AI control over user-facing commit intent.

## 后果

This decision has been superseded by the native Tool Call Loop control model, which keeps provider-native tool feedback in the loop instead of relying on a declarative JSON plan as the core workflow contract.

## 相关文档

- `docs/adr/0004-separate-commit-plan-from-execution-plan.md`
- `docs/adr/0006-use-native-tool-call-loop-as-core-control-model.md`
