---
doc_type: audit-finding
audit: 2026-05-24-cli-tui-config
finding_id: "bug-02"
nature: bug
severity: P1
confidence: medium
suggested_action: cs-issue
status: resolved
---

# Finding 02：`cnm auto --tui` 的 conflict handoff 会绕过前置 config 错误

## 速答

`runAuto` 在检测到 conflict 且 `--tui` 时，会直接调用 `runConflictTUIHandoff(runtime, scope, parsed, resolvedConfig.Values, message)`；但统一的 `configErr` / required-config 检查是在更后面才执行。这意味着配置解析失败或关键配置缺失时，用户可能先进入 repair handoff，再在 provider 创建或后续 repair 路径中得到次生错误，而不是稳定的 config error。

## 关键证据

- `internal/cli/cli.go:425` — `ResolveEffectiveConfig(...)` 提前执行，并把错误存到 `configErr`。
- `internal/cli/cli.go:481` — conflict 分支在 `scopeHasConflict(scope)` 后立即处理。
- `internal/cli/cli.go:483-485` — `parsed.TUI` 时直接调用 `runConflictTUIHandoff(..., resolvedConfig.Values, ...)`。
- `internal/cli/cli.go:521-526` — `configErr` 和 `autoRequiredConfigIssue(...)` 的统一 gate 在 conflict 分支之后。
- `internal/cli/cli.go:735-746` — repair handoff 最终仍会用 `resolvedConfig` 构造 `ProviderConfig` 并创建 provider。

## 影响

该问题会让“缺 API key / baseURL / 配置解析失败”的根因在 conflict+TUI 路径里被延后甚至掩盖，用户得到的不是一致的配置错误出口。置信度为 medium，因为具体报错取决于 provider 和配置状态，但控制流不一致是确定存在的。

## 修复方向

把 conflict handoff 前移到 config gate 之后，或在进入 `runConflictTUIHandoff` 前单独复用 `configErr` / `autoRequiredConfigIssue` 检查，确保 repair 路径和 normal auto path 的配置前置条件一致。

## 建议动作

`cs-issue`，因为这是命令调度顺序导致的真实控制流 bug。

## 处理结果

已修复。`runAuto` 现在只在 TUI conflict handoff 分支前做前置 config gate；缺 API key / 缺 `baseURL` 时优先返回 config error，不再进入 repair 产生次生 provider 错误。相关 CLI 回归测试已补齐。