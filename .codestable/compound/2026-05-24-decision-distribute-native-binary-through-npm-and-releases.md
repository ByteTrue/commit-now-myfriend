---
doc_type: decision
category: tech-stack
date: 2026-05-24
slug: distribute-native-binary-through-npm-and-releases
status: active
area: distribution
tags: [go, npm, releases, binary]
---

## 背景

The CLI should run as a native Go binary while remaining easy for JavaScript ecosystem users to install.

## 决定

Publish native release binaries and keep npm as a distribution wrapper for those binaries rather than a TypeScript or Node.js runtime path.

## 理由

Go provides the product runtime and cross-platform binaries. npm remains a convenient installation surface for existing users.

## 后果

The npm package should launch/download native binaries. Release automation must build platform artifacts for macOS, Linux, and Windows.

## 相关文档

- `docs/adr/0020-distribute-native-binary-through-npm-and-releases.md`
- `docs/distribution.md`
- `.codestable/architecture/ARCHITECTURE.md`
