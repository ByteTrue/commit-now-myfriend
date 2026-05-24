---
doc_type: decision
category: architecture
date: 2026-05-24
slug: plan-commits-from-working-tree-by-default
status: active
area: commit-scope
tags: [git, working-tree, commit-scope]
---

## 背景

The previous CLI centered on staged diffs, but the redesigned product aims to let a developer run one command and have AI plan commit work from the repository state.

## 决定

Make `cnm` and `cnm auto` plan from the working tree by default, considering staged, unstaged tracked, and untracked files while preserving safety blockers and explicit staged-only/path-limited modes.

## 理由

Working Tree Commit matches the product goal of AI-assisted local commit planning from the developer's current changes, while flags still allow narrowing the Commit Scope.

## 后果

Git inspection must consider more than staged files, and UI/diagnostics must make selected scope and untracked exposure visible to the developer.

## 相关文档

- `docs/adr/0003-plan-commits-from-working-tree-by-default.md`
- `.codestable/architecture/ARCHITECTURE.md`
