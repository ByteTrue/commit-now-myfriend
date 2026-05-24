---
doc_type: decision
category: constraint
date: 2026-05-24
slug: leave-project-checks-to-user-workflows
status: active
area: validation
tags: [checks, git-hooks, scope]
---

## 背景

The CLI should focus on AI-assisted local commit planning, messaging, splitting, and optional repair, not on becoming a project test runner.

## 决定

Do not add `--check` or a check-running Domain Tool. Developers who need validation before commits should use their own commands or Git hooks.

## 理由

Project checks are project-specific and can be expensive or side-effectful. Keeping them outside cnm keeps the product boundary focused on local commit workflow.

## 后果

cnm surfaces Git hook failures but does not own arbitrary test/lint execution. Documentation should direct users to their existing workflows for validation.

## 相关文档

- `docs/adr/0008-leave-project-checks-to-user-workflows.md`
- `.codestable/architecture/ARCHITECTURE.md`
