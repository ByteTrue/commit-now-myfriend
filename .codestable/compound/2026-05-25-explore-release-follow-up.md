---
doc_type: explore
slug: release-follow-up
created: 2026-05-25
status: current
tags: [release, distribution, npm, binaries]
---

# 发布准备 follow-up 结论

## 结论

发布准备的下一阶段应优先做 **distribution hardening**，重点不是再写说明，而是把 npm wrapper + GitHub release 这条交付链补到更抗失败。

## 优先级最高的 3 个动作

### 1. 给 npm installer 增加 checksum 校验

当前 `scripts/npm-install.js` 会直接按 `package.json` 版本拼 GitHub Releases URL 下载 archive，然后解压安装；但没有校验 `checksums.txt`，也没有 signature / digest 验证。

这意味着：
- 下载损坏时只能靠解压失败或运行失败暴露。
- 用户拿不到“下载的就是 release 里声明那份二进制”的完整性保证。

证据：
- `.goreleaser.yml` 已经生成 `checksums.txt`
- `scripts/npm-install.js` 只做下载、解压、查找 binary、写入 `bin/`，未见 checksum 验证逻辑

### 2. 补一个真正的 GitHub Release 发布 workflow

当前仓库只有 `.github/workflows/ci.yml`，会做 `go test`、`go build`、`npm pack --dry-run`，但没有看到真正发布 release archive / checksum / npm 包的工作流。

这意味着：
- 文档里描述了 release artifacts，但 CI 还没有把“构建 → 产物 → GitHub Release”自动串起来。
- npm installer 依赖 GitHub Releases 存在且命名正确，但发布流程还没被仓库内工作流显式固化。

### 3. 覆盖 installer 的失败路径与跨平台验收

当前文档说明了支持 darwin/linux/windows amd64/arm64，installer 也分别走 `tar` / `PowerShell Expand-Archive`。但现有 CI 没看到：
- 按平台模拟 archive 下载 + 解压 + launcher 执行
- Windows 安装链路验证
- archive 结构变化时 `findBinary` 是否还能稳定找到目标文件

建议最少补：
- installer 单元测试 / 集成测试
- 本地或 CI 的 release smoke test
- Windows 路径的自动验证

## 为什么这比别的 release 事项更优先

因为当前产品主运行时已经在 Go 里，真正会影响用户拿到工具并跑起来的，是 binary delivery 链是否可靠。相比继续润色 README，交付链完整性和可复现性更关键。

## 推荐下一步

建议开一个 feature：`release-distribution-hardening`

建议范围：
1. checksum verification
2. GitHub release workflow
3. installer / launcher smoke tests

如果要快一点，也可以先走 `cs-feat-design release-distribution-hardening`，把这些点拆成 checklist 后再实现。