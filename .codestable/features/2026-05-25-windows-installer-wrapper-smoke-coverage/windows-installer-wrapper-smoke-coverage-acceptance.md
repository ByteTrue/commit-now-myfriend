# windows-installer-wrapper-smoke-coverage 验收报告

> 阶段：阶段 3（验收闭环）
> 验收日期：2026-05-26
> 关联方案 doc：`.codestable/features/2026-05-25-windows-installer-wrapper-smoke-coverage/windows-installer-wrapper-smoke-coverage-design.md`

## 1. 接口契约核对

对照方案第 2.1 节名词层逐一核查：

**接口示例逐项核对**：
- [x] Windows artifact contract（`scripts/npm-install-lib.js:5` `archiveExt` / `scripts/npm-install-lib.js:9` `archiveNameFor`）：Windows archive 仍定义为 `.zip`，代码实际行为一致。
- [x] Wrapper binary 选择接口（`scripts/npm-install-lib.js:87` `resolveWrapperBinaryPath`）：wrapper 现在通过显式 helper 判断 installed/built `cnm.exe` 路径，代码实际行为与方案一致。
- [x] Installer binary 定位接口（`scripts/npm-install-lib.js:70` `findBinary`）：Windows smoke 仍能从解包树中定位 `cnm.exe`，代码实际行为一致。

**名词层“现状 → 变化”逐项核对**：
- [x] 引入 Windows smoke fixture / helper → 代码中新增 `platformName`、`archName`、`targetExecutableName`、`resolveWrapperBinaryPath` 和 `scripts/windows-smoke.test.js`，一致。
- [x] 引入 Windows smoke step 到 workflow → `.github/workflows/ci.yml` 与 `.github/workflows/release.yml` 都有明确步骤，一致。

**流程图核对**（第 2.2 节开头 mermaid 图）：
- [x] `build + test → artifact smoke helper → windows smoke evidence → release` 的关键节点在代码和 workflow 中都有实际落点（`package.json:24-27`、`.github/workflows/ci.yml:24-34`、`.github/workflows/release.yml:52-69`）。

## 2. 行为与决策核对

对照方案第 1 节 + 第 2.2 节：

**需求摘要逐项验证**：
- [x] Windows archive 命名、zip 路径、checksum 条目关系已有自动验证。证据：`scripts/windows-smoke.test.js:16-23`。
- [x] Windows installer 分支对 zip 校验、解压和 `cnm.exe` 定位有自动 smoke 证据。证据：`scripts/windows-smoke.test.js:25-33`、`scripts/windows-smoke.test.js:60-74`。
- [x] Wrapper 对 Windows `cnm.exe` 选择路径有自动 smoke 证据。证据：`scripts/windows-smoke.test.js:35-58`。

**明确不做逐项核对**：
- [x] 未新增 MSI / winget / Scoop 分发面。
- [x] 未把 release workflow 扩成真实跨平台发布实验室；仍然是 contract-level smoke。
- [x] 未引入必须真实 `npm publish` 才能跑的验证。

**关键决策落地**：
- [x] “优先做可复用 smoke primitive” → helper 逻辑集中在 `scripts/npm-install-lib.js`，workflow 只消费 `npm run test:windows-smoke`。
- [x] “验证重点放在 artifact contract 和启动路径” → smoke test 覆盖 archive naming、checksum、`cnm.exe` 定位和 wrapper binary 选择。

**编排层“现状 → 变化”逐项核对**：
- [x] Windows smoke 已从 runbook 手工步骤升级为自动化步骤。落点：`.github/workflows/ci.yml:27-28`、`.github/workflows/release.yml:55-56`、`.github/workflows/release.yml:68-69`。
- [x] `npm test` 继续跑 Go tests，同时新增 Node smoke，不破坏现有验证入口。落点：`package.json:24-27`。

**流程级约束核对**：
- [x] Windows smoke 不依赖真实 `npm publish`。实现方式是 Node test + workflow step，满足约束。
- [x] Wrapper 失败仍有明确错误信息；现在 `resolveWrapperBinaryPath` 为 null 时也会走显式失败提示。落点：`scripts/cnm.js:20-22`。

**挂载点反向核对（可卸载性）**：
- [x] `.github/workflows/ci.yml`：新增 Windows smoke step，清单一致。
- [x] `.github/workflows/release.yml`：新增 Windows smoke / explicit evidence step，清单一致。
- [x] `scripts/npm-install.js` / `scripts/npm-install-lib.js`：新增 helper seam 与 installer 消费路径，清单一致。
- [x] `scripts/cnm.js`：改为消费 wrapper helper，清单一致。
- [x] `docs/distribution.md` / `docs/release-runbook.md`：文档已同步 Windows smoke 自动化现状，清单一致。
- [x] 反向核查：本 feature 的新增引用均落在 design 第 2.3 节清单内，无清单外挂载点。
- [x] 拔除沙盘推演：删除 workflow steps、helper、Windows smoke test、文档说明后，该能力即完全消失，无额外残留。

## 3. 验收场景核对

- [x] **S1**：运行 Windows smoke helper / 测试时，`windows_amd64.zip` / `windows_arm64.zip` 与 checksum 条目关系可被验证
  - 证据来源：`scripts/windows-smoke.test.js`
  - 结果：通过

- [x] **S2**：Windows 平台分支下，installer 能给出 zip 校验、解压和 `cnm.exe` 定位的通过证据
  - 证据来源：`scripts/windows-smoke.test.js`
  - 结果：通过

- [x] **S3**：wrapper 对 Windows binary 路径能给出成功执行或明确失败信息的 smoke 证据
  - 证据来源：`scripts/windows-smoke.test.js` + `scripts/cnm.js`
  - 结果：通过

- [x] **S4**：CI / release workflow 中的 Windows smoke 已自动化，不再只是 runbook 里的人工步骤
  - 证据来源：`.github/workflows/ci.yml`、`.github/workflows/release.yml`
  - 结果：通过

## 4. 术语一致性

- Windows Installer：代码与方案命名一致 ✓
- Wrapper：`scripts/cnm.js` 与设计术语一致 ✓
- Release Artifact Contract：helper、workflow、docs 中的叫法一致 ✓
- Smoke Coverage：workflow / tests / docs 中无冲突命名 ✓
- 防冲突：未引入新的分发面术语（如 MSI / winget / Scoop）✓

## 5. 架构归并

- [x] 不需要更新 `.codestable/architecture/ARCHITECTURE.md`
  - 理由：本 feature 的系统级变化主要是 workflow、script seam 和 distribution docs 收敛，并未改变主运行时模块结构；release/distribution 约束已在 `docs/distribution.md` 与 `docs/release-runbook.md` 中得到准确归档。

## 6. requirement 回写

- [x] 已更新 `.codestable/requirements/native-binary-distribution.md`
  - `last_reviewed` 更新为 `2026-05-26`
  - `implemented_by` 增加 `2026-05-25-windows-installer-wrapper-smoke-coverage`
  - “怎么解决”补入 checksum verification 和 Windows smoke automation 的当前实现事实

## 7. roadmap 回写

- [x] 非 roadmap 起头。

## 8. attention.md 候选盘点

- [x] 本 feature 未暴露需要补入 `attention.md` 的新环境 / 工具 / 工作流信息。

## 9. 遗留

- 后续优化点：若未来需要更强发布证据，可再做真实 Windows runner 的 end-to-end release smoke。
- 已知限制：当前 Windows smoke 仍是 contract-level / helper-level 证据，不是完整平台原生安装实验室。
- 实现阶段顺手发现：无新的范围外问题需要另开 issue。