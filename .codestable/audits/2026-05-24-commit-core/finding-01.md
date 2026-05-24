---
doc_type: audit-finding
audit: 2026-05-24-commit-core
finding_id: "security-01"
nature: security
severity: P1
confidence: medium
suggested_action: cs-issue
status: open
---

# Finding 01：Provider 错误响应片段可能泄露敏感内容

## 速答

Provider HTTP 错误和响应解析失败会把响应 body 的前 500 个字符直接拼进错误信息；如果 provider/proxy 把 prompt、tool result 或其他敏感上下文回显到错误 body，cnm 可能把这些内容暴露到日志、TUI 或 JSON 输出。

## 关键证据

- `internal/providers/tool_call_provider.go:71` — 非 2xx 响应进入错误路径。
- `internal/providers/tool_call_provider.go:72` — `fmt.Errorf("http %d: %s", response.StatusCode, responseSnippet(body))` 直接使用响应片段。
- `internal/providers/tool_call_provider.go:78` — parse error 同样把 `responseSnippet(body)` 放入错误信息。
- `internal/providers/helpers.go:66` — `responseSnippet` 只压缩 whitespace 并截断，没有 redaction。
- `internal/providers/helpers.go:69` — 截断长度为 500 rune，仍足够包含 provider 回显的 prompt、file content 或 token-like text。

## 影响

触发条件取决于 provider 或代理是否在错误 body 中回显敏感内容，所以置信度为 medium。但项目的 safety/privacy requirement 明确要求调试和诊断避免 secrets、full diffs、prompts、provider responses；当前路径至少会暴露 provider response 片段，和该边界有冲突风险。

## 修复方向

统一 provider error snippet policy：默认只返回 status/provider/code/request id 等低敏信息；如保留 snippet，先做 secret/prompt-like redaction，并只在 explicit verbose/debug 下展示。

## 建议动作

`cs-issue`，因为这是安全边界上的具体行为缺陷，需要先写复现和期望输出，再补测试修复。
