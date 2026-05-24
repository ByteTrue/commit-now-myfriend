---
doc_type: requirement
slug: setup-and-diagnostics
pitch: 让开发者能配置 AI 提供商、管理偏好，并在提交前看清本地环境是否可用
status: current
last_reviewed: 2026-05-24
implemented_by:
  - architecture-overview
tags: [setup, config, doctor, credentials]
---

# 配置 AI 提供商并诊断本地环境

## 用户故事

- 作为第一次使用 cnm 的开发者，我希望有一个引导流程帮我配置 provider、model 和 API key，而不是自己猜配置文件怎么写。
- 作为一个在多个仓库间切换的人，我希望能区分个人私有偏好和仓库共享建议，而不是让项目配置覆盖我的凭证。
- 作为一个脚本或 CI 用户，我希望配置命令有稳定的非交互输出，而不是只能通过全屏界面操作。
- 作为一个提交前遇到问题的人，我希望 doctor 告诉我 Git、配置、凭证和 provider 能力哪里不对，而不是只看到提交失败。

## 为什么需要

AI 提交工具只有在 provider、凭证、仓库状态和终端能力都可用时才好用。配置散落或诊断不清楚时，用户会把时间花在猜环境问题上，而不是整理提交。Setup and Diagnostics 的价值，是让第一次配置和出错排查都变成可理解的产品流程。

## 怎么解决

`cnm init` 帮用户完成首次配置，API key 默认进入系统 Secret Store；`cnm config` 让用户查看和修改个人偏好、仓库建议和来源；`cnm doctor` 汇总本地 Git、仓库、配置、凭证和 provider 能力，并在用户明确要求时才做远程 provider probe。TTY 用户获得全屏/面板体验，非 TTY 用户获得脚本友好的输出。

## 边界

- 不允许仓库共享配置强制写入个人 provider 凭证；项目只能给出安全的推荐。
- 不默认调用远程 provider 做诊断；provider probe 必须由用户显式触发。
- 不把 API key 默认写进 plaintext config；plaintext 只作为显式 fallback。
- 不把 setup、config、doctor 混进提交流程本身；它们是提交流程的准备和诊断入口。
- 不替用户修复环境问题，只报告问题和可执行线索。
