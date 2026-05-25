# config-repair-ux-hardening 验收报告

> 阶段：阶段 3（验收闭环）
> 验收日期：2026-05-25
> 关联方案 doc：`.codestable/features/2026-05-24-config-repair-ux-hardening/config-repair-ux-hardening-design.md`

## 1. 接口契约核对

对照方案第 2.1 节名词层逐一核查：

**接口示例逐项核对**：
- [x] Config Panel 写入接口（`internal/cli/cli.go:369` `runConfigPanel`）：provider 改变后再写 `apiKey` → 实际通过 `currentConfigPanelProvider(...)` 先重解析 current provider，再写入对应 Secret Store slot，一致。
- [x] Effective Config Gate（`internal/cli/cli.go:429` `autoConfigIssueFromResolved` + `internal/cli/cli.go:503` `runAuto`）：conflict + `--tui` 路径在 handoff 前先检查 config 问题，代码实际行为一致。

**名词层“现状 → 变化”逐项核对**：
- [x] Config Panel 写入目标从 panel 初始快照 → 当前 provider 状态：代码落点 `internal/cli/cli.go:374-385`，一致。
- [x] conflict handoff 前置 config gate：代码落点 `internal/cli/cli.go:503-509`，一致。

**流程图核对**（第 2.2 节开头 mermaid 图）：
- [x] 图中 `ResolveEffectiveConfig → InspectCommitScope → effective config gate → conflict handoff` 的 TUI conflict 分支在代码中有实际落点（`internal/cli/cli.go:425`、`439`、`503`、`546`）。

## 2. 行为与决策核对

对照方案第 1 节 + 第 2.2 节：

**需求摘要逐项验证**：
- [x] Config Panel provider 改完再填 key，不会写到旧 provider slot。证据：`internal/cli/cli_test.go:3279`。
- [x] `cnm auto --tui` 遇到 conflict 时，如果缺关键配置，先返回 config 问题，不进入 repair。证据：`internal/cli/cli_test.go:3158`、`3214`。

**明确不做逐项核对**：
- [x] 未重做 config schema 或 Secret Store 抽象（diff 仅在 `internal/cli/*`）。
- [x] 未新增 provider 能力或 repair domain tool（grep / diff 未见 `internal/providers` / `internal/runtime` 新能力改动）。
- [x] 未改变 commit transaction / secret blocker 语义（相关包无改动）。
- [x] 未新增命令面（`internal/cli/cli.go` 仅调整现有 flow）。

**关键决策落地**：
- [x] “provider 选择视为当前状态，不是 panel 快照” → `currentConfigPanelProvider` 在每次保存 API key 时重解析。
- [x] “conflict handoff 不是 config gate 旁路” → `runAuto` 仅在 `parsed.TUI` 的 conflict handoff 前做前置 gate。
- [x] “错误优先级：config 问题优先于 repair/provider 次生错误” → `autoConfigIssueFromResolved` 在 TUI conflict handoff 前生效。

**编排层“现状 → 变化”逐项核对**：
- [x] TUI conflict 分支先经过 config gate 再进入 `runConflictTUIHandoff`，落点 `internal/cli/cli.go:503-509`。
- [x] normal auto path 仍保留既有 `conflict` / `secret_blocked` / `config_missing` 出口顺序，落点 `internal/cli/cli.go:503-548`。

**流程级约束核对**：
- [x] 缺 API key / 缺 `baseURL` / config parse 失败时，在 TUI conflict 路径优先暴露 config 错误。证据：`internal/cli/cli_test.go:3158`、`3214`。
- [x] 配置完整时 conflict repair happy path 不回归。证据：现有 `TestExecuteWithRuntimeAutoTUIHandsOffConflictsToInteractiveRepairContext` 继续通过。

**挂载点反向核对（可卸载性）**：
- [x] `internal/cli.runConfigPanel`：实际落点 `internal/cli/cli.go:350-414`，一致。
- [x] `internal/config.ResolveEffectiveConfig` / source summary：实际由 `currentConfigPanelProvider` 与 `runAuto` 调用，清单一致。
- [x] `internal/cli.runAuto`：实际落点 `internal/cli/cli.go:414-612`，一致。
- [x] `internal/cli.runConflictTUIHandoff`：由 `runAuto` TUI conflict 分支调用，清单一致。
- [x] 相关 tests：新增 3 条 CLI 回归测试，清单一致。
- [x] 反向核查：本 feature 的新增 helper / 测试引用均落在上述清单内，无清单外挂载点。
- [x] 拔除沙盘推演：逆向移除上述挂载点后，provider-aware key 保存与 TUI conflict config gate 能力都会消失，没有额外挂入残留。

## 3. 验收场景核对

- [x] **S1**：先改 provider 再设 key，新 provider 被识别为有 key，旧 provider 不会被误写
  - 证据来源：CLI 测试 `internal/cli/cli_test.go:3279`
  - 结果：通过

- [x] **S2**：`cnm auto --tui` 在 conflict + 缺 API key 时先报 api-key-missing，不进入 repair
  - 证据来源：CLI 测试 `internal/cli/cli_test.go:3158`
  - 结果：通过

- [x] **S3**：`cnm auto --tui` 在 conflict + openai-compatible 缺 baseURL 时先报 base_url_missing，行为与 normal auto path 一致
  - 证据来源：CLI 测试 `internal/cli/cli_test.go:3214`
  - 结果：通过

- [x] **S4**：配置完整时，conflict repair happy path 仍可进入 Interactive Repair 并继续 commit handoff
  - 证据来源：现有 CLI 测试 `TestExecuteWithRuntimeAutoTUIHandsOffConflictsToInteractiveRepairContext`
  - 结果：通过

## 4. 术语一致性

- Config Panel：代码命中与方案一致（`runConfigPanel` / `ConfigPanelInput` / `ConfigPanelRunner`）✓
- Conflict Handoff：代码命中与方案一致（`runConflictTUIHandoff`）✓
- Effective Config Gate：通过 `autoConfigIssueFromResolved` + `writeAutoConfigMissing` 落地，命名一致 ✓
- Provider Slot：代码未新建冲突术语，继续复用 Secret Store provider account 语义 ✓
- 防冲突：未引入设计外的新产品概念名 ✓

## 5. 架构归并

- [x] 架构 doc：`.codestable/architecture/ARCHITECTURE.md`
  - 在“2.2 配置、偏好与 Secret Store”补入：Config Panel 保存 API key 时按**当前 provider 状态**决定 Secret Store slot，而不是使用 panel 初始快照。
  - 在“2.6 Doctor / Init / Config 辅助流”补入：`cnm auto --tui` 的 conflict handoff 不是 config gate 的旁路，repair 依赖 provider 时必须先过 effective config / required-config 检查。

本 feature 引入的是稳定、系统级可见的配置/repair 边界约束，因此已实际回写 architecture，而不是只保留 feature 归档。

## 6. requirement 回写

- [x] `requirement` 为空，且本次是现有 CLI/TUI/config flow 的 hardening，不新增独立用户能力愿景 → 写“无 requirement 回写”。

## 7. roadmap 回写

- [x] 方案 frontmatter 无 `roadmap` / `roadmap_item` → 写“非 roadmap 起头”，跳过。

## 8. attention.md 候选盘点

- [x] 本 feature 未暴露需要补入 `attention.md` 的新环境 / 工具 / 工作流信息。

## 9. 遗留

- 后续优化点：`runAuto` 仍偏胖，已在 audit 里记录为 `cs-refactor` 候选。
- 已知限制：本次只 harden config/repair UX，不扩展 provider 能力或 Secret Store 抽象。
- 实现阶段顺手发现：无新的范围外代码问题需要单独记 issue。