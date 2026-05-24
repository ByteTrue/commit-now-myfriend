---
doc_type: requirement
slug: safety-and-privacy
pitch: 让 AI 帮忙提交时，始终把本地仓库、凭证和用户控制权放在安全边界内
status: current
last_reviewed: 2026-05-24
implemented_by:
  - architecture-overview
tags: [safety, privacy, ai-assist]
---

# 让 AI 提交时守住安全和隐私边界

## 用户故事

- 作为一个把本地代码交给 AI 分析的开发者，我希望清楚知道哪些仓库内容会被使用，而不是无意把过多上下文发出去。
- 作为一个可能改到凭证文件的人，我希望工具发现疑似 secret 时停下来提醒我，而不是悄悄跳过或继续提交。
- 作为一个依赖 Git hooks 的项目维护者，我希望提交失败时原因被清楚展示，而不是工具自动修 hook 或吞掉失败。
- 作为一个重视隐私的用户，我希望诊断和日志默认留在本地，并且不要记录 secrets、完整 diff、prompt 或 provider 响应。

## 为什么需要

AI 提交工具会接触本地仓库内容，也可能调用用户配置的 AI provider。没有明确安全边界时，用户很难判断哪些内容会暴露、哪些操作会发生、失败时仓库会不会被半改。安全和隐私能力的价值，就是让“AI 帮忙”不变成“AI 越权”。

## 怎么解决

cnm 只把选中提交范围内、经过限制的上下文交给 AI，并让 AI 通过受控的提交工具工作。疑似凭证会阻塞当前流程；Git hook 或项目验证失败会被展示给用户；调试和诊断默认保守、显式、本地化。工具可以创建本地提交，但不会推送远端。

## 边界

- 不保证 AI provider 本身是本地服务；用户配置远程 provider 时，选中的仓库上下文可能会发送给该 provider。
- 不替用户修复 Git hook、测试、lint 或其他项目验证失败。
- 不静默跳过含疑似凭证的选中改动；用户必须缩小范围或处理 secret。
- 不给 AI shell access 或 raw Git command access。
- 不做默认远程 telemetry；未来如果要加，需要单独重新拍板。
