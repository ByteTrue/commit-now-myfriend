---
doc_type: decision
category: architecture
date: 2026-05-24
slug: simplify-command-surface-around-interactive-and-auto
status: active
area: cli-command-surface
tags: [cli, interactive-commit, autonomous-commit]
---

## 背景

The redesigned CLI should center on two commit entry points instead of exposing each sub-capability as a command.

## 决定

Use `cnm` for Interactive Commit and `cnm auto` for Autonomous Commit. Keep `init`, `config`, and `doctor` for setup and diagnostics. Remove standalone `split`, `repair`, `check`, and `onboard` commands because those concerns belong inside primary flows or outside cnm.

## 理由

A small command surface makes the product easier to explain and prevents old workflow concepts from leaking into the redesigned architecture.

## 后果

Repair is a user-confirmed interactive action, not an autonomous flag. Command design should keep `cnm auto` focused on organizing and creating commits from non-conflicted working tree changes.

## 相关文档

- `docs/adr/0018-simplify-command-surface-around-interactive-and-auto.md`
- `.codestable/architecture/ARCHITECTURE.md`
