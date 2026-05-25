---
doc_type: audit-finding
audit: 2026-05-24-cli-tui-config
finding_id: "bug-01"
nature: bug
severity: P1
confidence: high
suggested_action: cs-issue
status: resolved
---

# Finding 01：config panel 切换 provider 后保存 API key 会写到旧 provider 名下

## 速答

`runConfigPanel` 在创建 `WriteValue` 闭包时把 `resolved.Values.Provider` 固定住了；如果用户先在 panel 里把 provider 改成另一个值，再编辑 API key，Secret Store 仍会按旧 provider 写入，导致 key 被存到错误的 provider slot。

## 关键证据

- `internal/cli/cli.go:350` — `runConfigPanel` 只在进入 panel 前解析一次 `resolved`。
- `internal/cli/cli.go:374` — API key 分支判断 `key == config.ConfigKeyAPIKey && runtime.SecretStore != nil`。
- `internal/cli/cli.go:375` — provider 取自闭包里的 `resolved.Values.Provider`，不是最新 reload 后的 provider。
- `internal/cli/cli.go:390` — panel 确实提供了 `Reload`，但 reload 只在写入后更新展示，不会回填到闭包中用于下一次 `SetAPIKey`。
- `internal/tui/config_panel.go:238` / `262` — Enter 保存会直接调用 `WriteValue`，因此这是可达的真实交互路径。

## 影响

这是一个高置信度的用户可触发 bug：先改 provider，再填 API key，会把密钥写到错误 provider，后续 `ResolveEffectiveConfig` 可能表现为“新 provider 没 key、旧 provider 有 key”。它不一定泄露 secrets，但会破坏 onboarding / config flow 的正确性。

## 修复方向

API key 保存时不要依赖 panel 打开时的 provider 快照；应在写 key 前重新解析当前 effective config，或优先使用 panel 当前编辑后的 provider 值，再决定 Secret Store account。

## 建议动作

`cs-issue`，因为这是具体、可复现、影响用户配置正确性的行为缺陷。

## 处理结果

已修复。`internal/cli/runConfigPanel` 的 API key 保存路径现在会实时重解析当前 provider，再写入对应 Secret Store slot；新增 CLI 测试覆盖“先改 provider 再设 key”的场景。