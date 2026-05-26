---
doc_type: issue-report
issue: 2026-05-26-installer-temp-file-isolation
status: confirmed
severity: P2
summary: npm installer 在同版本并发/重试时复用固定临时文件名，可能导致 archive 与 checksum 交叉读取
tags: [installer, release, temp-files]
---

# installer-temp-file-isolation Issue Report

## 1. 问题现象

`scripts/npm-install.js` 在执行安装时，会把 release archive 写到 `${tmpRoot}/${archiveName}`，并把 checksum 写到 `${tmpRoot}/checksums-${version}.txt`。当同一台机器上有并发安装、重试或多个包管理进程同时处理同版本时，不同进程会共享相同的临时文件路径，存在覆盖或读到半成品文件的风险。

## 2. 复现步骤

1. 在共享 `TMPDIR` / `CNM_INSTALL_TMP_DIR` 的环境中，同时启动两个 `scripts/npm-install.js` 进程，且目标版本相同。
2. 两个进程都下载同名 archive 与同版本 `checksums.txt` 到共享临时目录。
3. 观察到：其中一个进程可能读取到另一个进程尚未写完或刚被覆盖的 archive / checksum 文件，导致安装失败或出现难复现的 checksum / extract 异常。

复现频率：暂无法稳定量化；在单进程安装下通常不触发，在并发 / 重试 / 共享环境下风险上升。

## 3. 期望 vs 实际

**期望行为**：每次 installer 执行都应拥有自己的临时文件空间，不会和同版本的并发安装互相干扰。

**实际行为**：archive 与 checksum 临时文件名按版本固定，多个进程可能共享路径并产生交叉读取或覆盖。

## 4. 环境信息

- 涉及模块 / 功能：release delivery chain / npm installer
- 相关文件 / 函数：`scripts/npm-install.js:101-117`
- 运行环境：任意共享临时目录的安装环境（dev / CI / 用户本机都可能）
- 其他上下文：extract 目录已经使用 `mkdtempSync(...)`，但 archive / checksum 文件仍是固定路径

## 5. 严重程度

**P2** — 不影响大多数单进程安装，但会给并发 / 重试 / 共享环境带来偶发且难复现的安装失败，属于值得尽快收口的交付稳定性问题。

## 备注

用户当前明确先忽略 Windows 平台相关 finding-01，优先继续处理非 Windows 方向的交付质量问题。本问题来自 `.codestable/audits/2026-05-26-release-delivery-chain/finding-02.md`。