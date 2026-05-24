---
doc_type: decision
category: constraint
date: 2026-05-24
slug: store-api-keys-in-system-secret-store-by-default
status: active
area: configuration
tags: [api-keys, secret-store, security]
---

## 背景

The redesigned CLI should not normalize plaintext API keys in `~/.cnm/config.json`.

## 决定

Store API keys in the platform Secret Store by default. Keep environment variables as explicit overrides and require a clear opt-in for plaintext storage.

## 理由

API keys are sensitive, and a local commit tool should minimize accidental credential exposure in filesystem config.

## 后果

Config, onboarding, and doctor flows must distinguish Secret Store, environment, and plaintext sources. Plaintext API key storage should be diagnosed as an explicit choice.

## 相关文档

- `docs/adr/0013-store-api-keys-in-system-secret-store-by-default.md`
- `.codestable/architecture/ARCHITECTURE.md`
