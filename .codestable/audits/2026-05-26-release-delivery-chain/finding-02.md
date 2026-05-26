---
doc_type: audit-finding
audit: 2026-05-26-release-delivery-chain
finding_id: "bug-02"
nature: bug
severity: P2
confidence: medium
suggested_action: cs-issue
status: resolved
---

# Finding 02：npm installer 的 archive / checksums 临时文件名按版本固定，缺少并发隔离

## 速答

`scripts/npm-install.js` 会把下载的 archive 写到 `${tmpRoot}/${archiveName}`，并把 checksum 写到 `${tmpRoot}/checksums-${version}.txt`。同一台机器上如果有并发安装、重试或多个包管理进程同时处理同版本，临时文件会共享同一路径，存在覆盖或交叉读取风险。

## 关键证据

- `scripts/npm-install.js:77-80` — `archivePath` 和 `checksumsPath` 都直接基于 `tmpRoot + version/name` 生成，不是 `mkdtemp` 下的唯一文件。
- `scripts/npm-install.js:84-89` — 下载 archive / checksums 后立刻消费固定路径上的内容。
- `scripts/npm-install.js:80` — 只有 extract 目录用了 `mkdtempSync(...)`，说明当前唯一性仅覆盖了解包阶段。

## 影响

大多数单进程安装不会触发，但在 CI 并发、重试、或同一共享环境上并发安装同版本时，路径复用可能导致一个进程读到另一个进程的半成品文件。影响通常表现为偶发安装失败或难复现的 checksum / extract 异常，因此定级 P2、置信度 medium。

## 修复方向

把 archive 和 checksum 下载都放进同一个 `mkdtemp` 会话目录中，或为这两个文件名加入进程级随机后缀，确保一次 installer 执行拥有独立临时空间。

## 建议动作

`cs-issue`，因为这是具体、可定点修复的 installer 稳定性问题。

## 处理结果

已修复。installer 的 archive / checksum / extract 路径现在统一收进每次执行独享的 session 目录；新增 Node 测试覆盖同版本多次执行时的路径隔离。