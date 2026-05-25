---
doc_type: compound-note
category: explore
slug: release-hardening
created: 2026-05-24
status: active
summary: Release/distribution follow-up after product-completion and post-audit fixes.
tags: [release, distribution, npm, goreleaser, docs]
---

# release hardening follow-up

## 结论

当前仓库已经具备本地 build、GoReleaser snapshot、npm wrapper 下载/启动三块基础能力，但离“可放心发布”还差几项收尾。最值得优先处理的不是新代码功能，而是**文档/产物一致性、发布自动化、以及下载校验**。

## 发现

### 1. `docs/distribution.md` 的 Windows 产物命名与 GoReleaser 配置不一致

- `docs/distribution.md:29-30` 把 Windows 产物写成 `*.tar.gz`
- `.goreleaser.yml:25-30` 明确把 Windows archive override 成 `zip`

这会让直接下载用户拿着错误的文件名去找产物，也会让发布检查表出现假阳性。

**建议动作**：先修文档；顺便把 README 的 direct binary install 文案一起核一遍，确保示例与实际文件名一致。

### 2. 仓库只有 `ci.yml`，没有明确的 release workflow

- `.github/workflows/ci.yml` 存在
- 没看到独立的 release / draft-release / npm publish workflow
- 现在 release 依赖人工执行 `make go-release-snapshot`、`npm run build:release-local` 和 GoReleaser 配置

这不代表不能发版，但意味着“谁来产出 release archives、checksums、npm package、以及它们的顺序”仍主要靠人工记忆。

**建议动作**：补一份 release runbook 或 workflow 设计，至少明确：
1. 版本号来源
2. GoReleaser 产物生成步骤
3. npm publish 与 GitHub Release 的先后顺序
4. smoke checks（`cnm --help` / `cnm doctor --json` / wrapper launch）

### 3. `scripts/npm-install.js` 下载产物但不校验 checksum

- `scripts/npm-install.js:109-122` 下载 archive、解压、复制 binary
- 同文件没有 `checksums.txt` 验证逻辑
- `.goreleaser.yml` 已生成 `checksums.txt`

这意味着 release pipeline 已具备生成校验数据的能力，但 npm installer 还没有消费它。对首版内部使用不是致命问题，但从“发布 hardening”角度看是缺口。

**建议动作**：下一轮 release hardening 可以评估：
- npm installer 是否要校验 `checksums.txt`
- 若不做，也要在 `docs/distribution.md` 里明确当前信任模型和边界

## 建议顺序

1. **docs fix**：先修 `docs/distribution.md` / README 的 Windows archive 说明。
2. **release runbook / workflow**：把人工发布顺序落成文档或 CI workflow。
3. **checksum decision**：决定 npm installer 是否接入校验；若暂不做，补一条明确 decision / 文档边界。

## 与当前工作的关系

- 这不是新的产品能力 feature，更像发布面收尾。
- 更适合走小范围 `cs-guide` / `cs-decide` / `cs-issue` 组合，而不是大 feature 流程。
- 如果你要我下一步直接动手，我建议先从**修文档不一致**开始。