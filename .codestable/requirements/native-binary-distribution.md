---
doc_type: requirement
slug: native-binary-distribution
pitch: 用 npm 安装入口交付 Go 原生 cnm，让不同平台的用户都能直接运行
status: current
last_reviewed: 2026-05-24
implemented_by:
  - architecture-overview
tags: [distribution, npm, go, binary]
---

# 通过 npm 入口交付 Go 原生命令

## 用户故事

- 作为一个 JavaScript 生态用户，我希望继续用 `npm install -g commit-now-myfriend` 安装 cnm，而不是额外学习新的安装渠道。
- 作为一个 macOS、Linux 或 Windows 用户，我希望安装后拿到适合当前平台的原生命令，而不是依赖本机 Go 或 TypeScript runtime。
- 作为一个本地开发者，我希望 release 前能演练打包结果，而不是等发布后才发现 npm wrapper 找不到二进制。
- 作为一个命令名冲突的用户，我希望还有 `npx` / `npm exec` 这类替代启动方式，而不是安装后无法使用。

## 为什么需要

产品运行时已经是 Go 原生 CLI，但很多潜在用户习惯通过 npm 安装开发工具。如果完全放弃 npm，会丢掉熟悉的安装入口；如果继续把 npm 当 runtime，又会把旧的 Node/TypeScript 包袱带回来。分发能力要同时满足“安装顺手”和“运行时原生”。

## 怎么解决

项目发布 Go 原生二进制，并保留 npm 包作为下载和启动 wrapper。安装时 npm wrapper 获取当前平台对应的 release binary；运行时 `cnm` 只是把参数转交给 native binary。本地开发和发布前提供打包演练，确保 wrapper、binary 路径和 release artifact 对得上。

## 边界

- npm 不是产品运行时，只是安装和启动入口。
- 不要求用户本机安装 Go 才能运行已发布的 cnm。
- 不把 TypeScript/Node 实现作为长期兼容路径保留。
- 不保证 `cnm` 命令名一定没有冲突；冲突时用户可以走 package-qualified 启动方式。
- 不把 release 失败隐藏成运行时问题；打包和下载失败需要明确报错。
