---
doc_type: decision
category: architecture
date: 2026-05-24
slug: support-four-provider-protocols-as-first-class
status: active
area: providers
tags: [providers, openai, anthropic, gemini, tool-calls]
---

## 背景

The redesigned product should not narrow the CLI to a single AI provider, even though native tool calling differs across APIs.

## 决定

Support OpenAI Responses, OpenAI-compatible chat completions, Anthropic Messages, and Google Gemini as first-class Provider Protocols by adapting each into the same local Tool Call Runtime contract.

## 理由

Provider protocols differ at the HTTP/payload/tool-call layer, but the local product needs a stable runtime contract for Domain Tools and commit workflow control.

## 后果

Every first-class provider must implement native tool-call adaptation for the flows it supports. Low-level Git sequencing remains owned by local code behind Domain Tools.

## 相关文档

- `docs/adr/0005-support-four-provider-protocols-as-first-class.md`
- `.codestable/architecture/ARCHITECTURE.md`
