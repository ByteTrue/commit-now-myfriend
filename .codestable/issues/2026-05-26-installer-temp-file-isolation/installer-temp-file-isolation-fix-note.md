---
doc_type: issue-fix
issue: 2026-05-26-installer-temp-file-isolation
path: fast-track
fix_date: 2026-05-26
tags: [installer, release, temp-files]
---

# installer-temp-file-isolation 修复记录

## 1. 问题描述

`scripts/npm-install.js` 会把 archive 与 `checksums.txt` 下载到按版本固定的临时文件名；同版本并发安装或重试时，不同进程可能共享相同路径并发生交叉读取或覆盖。

## 2. 根因

根因位于 `scripts/npm-install.js:101-117`：
- `archivePath` 使用 `${tmpRoot}/${archiveName}`
- `checksumsPath` 使用 `${tmpRoot}/checksums-${version}.txt`
- 只有 `extractDir` 使用了 `mkdtempSync(...)`

也就是说，一次 installer 执行并没有独享 archive / checksum 临时空间。

## 3. 修复方案

把 archive、checksum、extract 三类临时路径统一收进一次 installer 执行自己的会话目录中：
- 新增 `createInstallerSessionPaths(...)`
- 每次运行先 `mkdtempSync(...)` 出独立 sessionRoot
- 再在该目录下放置 archive、`checksums.txt` 和 extract 目录

## 4. 改动文件清单

- `scripts/npm-install-lib.js` — 新增 installer session path helper
- `scripts/npm-install.js` — 改为使用每次执行独立的 session 路径
- `scripts/npm-install-lib.test.js` — 增加并发隔离回归测试

## 5. 验证结果

- `node --test scripts/npm-install-lib.test.js scripts/windows-smoke.test.js` 通过
- `npm test` 通过
- 新增测试证明：同版本同 archive 名称下，两次 `createInstallerSessionPaths(...)` 会生成不同 sessionRoot，并把 archive / checksum / extract 隔离到各自目录

## 6. 遗留事项

- 仍未处理 release-delivery-chain finding-01（Windows 真机 smoke 证据不足）
- 仍未处理 release-delivery-chain finding-03（workflow 对 Windows smoke 的重复执行）