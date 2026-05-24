---
doc_type: decision
category: constraint
date: 2026-05-24
slug: treat-secret-detection-as-selected-scope-blocker
status: active
area: safety
tags: [secrets, commit-scope, blockers]
---

## 背景

Autonomous commit flows should not silently work around suspected credentials.

## 决定

If secret detection finds a suspected credential inside the Commit Scope, cnm blocks the selected flow and creates no commits rather than excluding that file and continuing.

## 理由

Silently excluding a file would hide a security-relevant condition and could produce incomplete commits. Blocking keeps the developer in control.

## 后果

Secret detection is a selected-scope blocker. The product must report findings safely without exposing full secrets.

## 相关文档

- `docs/adr/0015-treat-secret-detection-as-selected-scope-blocker.md`
- `.codestable/architecture/ARCHITECTURE.md`
