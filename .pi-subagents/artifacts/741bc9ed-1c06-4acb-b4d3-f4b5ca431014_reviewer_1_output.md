## Review

- **Correct:** 当前实现与仓库现行产品计划一致，不是无意间膨胀：`plan.md:7-25` 明确要求 Full-screen TUI、Autonomous Commit、原生 Tool Call Loop、Secret Store；`docs/implementation-todo.md:46-128` 又明确要求 runtime、多提交、repair、四种 provider 和 doctor。
- **Fixed:** 无；按要求只审不改。
- **Blocker（范围）:** 若产品目标已经收缩为“读取 diff，只生成 commit message 并输出”，则现行 `plan.md` 不再适用。继续以它作为 source of truth，会不断重新引入本次审计认为应删除的模块。
- **Note:** `progress.md` 不存在，审计基于 `plan.md`、源码、测试和依赖图完成。

### 结论

相对现行计划，这些模块不是 YAGNI；但相对“只生成 commit message”，当前架构明显过度，约 **10,454 行生产 Go + 5,897 行测试 Go**。一个保留输入校验、diff 大小限制、secret 检测和 HTTP 响应限制的简单实现，预计只需约 **800–1,800 行生产代码**。

因此可净删约：

- **8,500–9,500 行生产 Go（约 81%–91%）**
- **4,500–5,300 行测试 Go**
- 保守方案移除 5 个直接 TUI 依赖；env-only 方案可清空全部 Go 外部依赖。

### 当前 LOC 分布

| 模块 | 生产 LOC | 测试 LOC | Ponytail 判断 |
|---|---:|---:|---|
| `internal/tui` | 2,898 | 252 | 对 message-only 全删 |
| `internal/runtime` | 599 | 422 | Tool-call loop 全删 |
| `internal/providers` | 1,038 | 413 | 留一个直接 message API 客户端即可 |
| `internal/config` | 1,124 | 423 | 缩成 env/少量 flags，或最多一个小配置文件 |
| `internal/doctor` | 321 | 258 | 全删 |
| `internal/commands` | 862 | 0 | `init/config/doctor` 基本全删 |
| `internal/interactive` | 142 | 0 | Onboarding 删除后全删 |
| `internal/git` | 1,104 | 623 | 保留只读 diff/scope/safety，删除 commit/transaction |
| `internal/cli` | 2,176 | 3,324 | 重写为小型参数解析与 stdout 输出 |
| 其他 | 190 | 182 | `security`、HTTP 限制和基础输出可保留 |

## 建议删除顺序

### 1. 自动提交、多提交、repair、transaction：优先全删

它们不只是 YAGNI，而是直接超出“只生成 message”的副作用边界。

证据：

- Autonomous Commit 主流程：`internal/cli/cli.go:436-634`，符号 `runAuto`
- Interactive Repair：`internal/cli/cli.go:675-925`，符号 `runConflictTUIHandoff`、`executeInteractiveRepair`、`interactiveRepairDomainTools`
- AI 多提交规划与转换：`internal/cli/cli.go:1124-1348`
- 多提交执行与 rollback：`internal/cli/cli.go:1350-1378`，符号 `executeCommitPlan`
- Git commit/transaction：`internal/git/service.go:342-447`，符号 `CommitWithMessage`、`CommitScopeWithMessage`、`CaptureCommitTransactionSnapshot`、`RollbackCommitTransaction`
- Transaction 类型：`internal/git/types.go:203-227`
- Repair/read-before-write runtime：`internal/runtime/runtime.go:207-256`
- 多提交参数解析：`internal/runtime/runtime.go:286-387`
- 多提交/transaction/repair DTO：`internal/runtime/types.go:71-74,132-187`

这些交织代码约 **1,200–1,500 LOC**，还不含其 TUI 和测试覆盖。删除后 Git 层只需要只读地取得 diff，不能 stage、commit、reset 或写工作树。

### 2. Tool Call Runtime：全删

`internal/runtime` 的核心是一个代理执行器，而生成 message 只需要一次受限请求：

- 八个 Domain Tools：`internal/runtime/types.go:9-20`
- Provider interface：`internal/runtime/types.go:53-55`
- 六个函数注入组成的 `DomainToolSet`：`internal/runtime/types.go:83-109`
- 循环、重试、reminder、tool dispatch：`internal/runtime/runtime.go:11-129`
- Repair guardrail：`internal/runtime/runtime.go:207-256`

删除范围：

- `internal/runtime/**`：约 **599 生产 LOC + 422 测试 LOC**
- `internal/providers/tool_schema.go`：约 **115 LOC**
- `internal/providers/tool_protocol.go` 中 continuation/tool-result 编解码
- `internal/providers/tool_call_provider.go` 中多轮历史和 reminder 分支

Ponytail 替代是：`git diff` → 单个 HTTP 请求 → 解析文本 → stdout。无需 `finish`、`abort`、`preview_commit`、`create_commits` 或循环限制。

预计 runtime/provider 合计可从 **1,637 LOC** 缩至 **200–350 LOC**，净删约 **1,300 LOC**。

### 3. Full-screen TUI：全删

TUI 是最大单块复杂度：

- 六屏状态机：`internal/tui/model.go:16-25`
- 注入式 diff/file/planner API：`internal/tui/model.go:27-76`
- 主 Model 状态：`internal/tui/model.go:78-101`
- Bubble Tea 状态转移：`internal/tui/model.go:103-280`
- Alternate screen/mouse runtime：`internal/tui/run.go:16-41`
- Config panel：`internal/tui/config_panel.go:11-100`
- Onboarding wizard：`internal/tui/onboarding.go:11-142`
- Doctor dashboard：`internal/tui/doctor_panel.go:10-203`

可直接删除 `internal/tui/**`，约 **2,898 生产 LOC + 252 测试 LOC**。同时删除 `internal/cli/cli.go` 中 TUI wiring，如 `runConfigPanel`、`runConflictTUIHandoff`、`runRoot` 的 TUI 分支。

### 4. 四 Provider Adapter：只留一个

四协议分别在：

- OpenAI Responses：`internal/providers/tool_protocol.go:63-103`
- OpenAI-compatible：`internal/providers/tool_protocol.go:105-153`
- Anthropic：`internal/providers/tool_protocol.go:155-199`
- Gemini：`internal/providers/tool_protocol.go:201-252`

此外 `httpToolCallProvider` 为四协议维护独立历史和分支：

- 并行状态字段：`internal/providers/tool_call_provider.go:21-34`
- reminder/initial/continuation 分支：`internal/providers/tool_call_provider.go:102-233`
- endpoint/header/state capture：`internal/providers/tool_call_provider.go:235-340`

对于 message-only，建议只保留当前默认协议 OpenAI Responses；默认值证据在 `internal/config/schema.go:129-140`。如果产品实际核心是第三方网关，则改留 OpenAI-compatible，但仍只选一个。

单 provider + 单轮文本响应预计可再删 **700–850 生产 LOC**。四 provider 的“兼容性矩阵”没有 message-only 需求支撑。

### 5. Config / Doctor / Onboarding：大幅删除

当前配置系统支持：

- user/project/env/flag 四层合并：`internal/config/service.go:98-260`
- Shared/Private preference 和 source tracking：`internal/config/schema.go:46-126`
- Secret Store：`internal/config/service.go:30-37`、`internal/config/secret_store.go:12-56`
- 完整 init：`internal/commands/init.go:47-554`
- config CRUD：`internal/commands/config.go:22-234`
- doctor：`internal/doctor/service.go:13-206`、`internal/commands/doctor.go:13-72`
- TUI onboarding/config/doctor panels：合计约 **1,168 LOC**

不重复计算 TUI 时，`config + doctor + commands + interactive` 已有约 **2,449 生产 LOC**。Message-only 最小配置只需 provider/model/key，优先从环境变量读取，约 **100–200 LOC** 足够。

可删除：

- `internal/doctor/**`
- `internal/commands/doctor.go`
- `internal/commands/init.go`
- `internal/commands/config.go`
- `internal/interactive/**`
- `internal/config` 中 project recommendation、source summary、写配置、unset、onboarding 支持

如果必须持久化 API key，可保留 `SystemSecretStore`；否则使用 `CNM_API_KEY`，不落盘也不需要 onboarding。

## 单实现接口审计

### 明显可删：`ToolProtocolAdapter`

- 接口：`internal/providers/types.go:16-22`
- 唯一生产 concrete wrapper：`toolProtocolAdapter`，`internal/providers/tool_protocol.go:12-30`
- 工厂再向该 wrapper 填三个函数指针：`internal/providers/tool_protocol.go:44-60`

这是“接口 + 一个函数指针实现 + switch 工厂”三层包装。即便暂时保留四协议，直接按 provider 调用函数也更短；收缩到单 provider 后应整体消失。

### 仅是测试 seam，不是主要问题

- `HTTPDoer`：`internal/providers/http.go:11-17`  
  生产路径只是 `*http.Client`，但一方法接口可注入 fake HTTP；若保留 provider 单元测试，它成本很低。
- `SecretStore` / `WritableSecretStore`：`internal/config/service.go:30-37`  
  唯一生产实现是 `SystemSecretStore`（`internal/config/secret_store.go:12-42`），其他实现来自测试。若保留 credential persistence，这个边界合理；env-only 后整体删除。
- `ToolCallProvider`：`internal/runtime/types.go:53-55`  
  有真实 HTTP 实现和 `FakeProviderTracer`（`internal/runtime/fake_provider.go:8-25`），不算严格单实现；但 message-only 不需要 runtime，因此无需继续讨论接口设计。

另一个比单实现接口更大的复杂度来源是 `internal/cli/cli.go:28-42` 的 `Runtime` service-locator：它注入三种 TUI runner、两个 provider、secret store 和 HTTP client，主要用于支撑庞大测试矩阵。功能删除后该结构也应退化成普通参数或消失。

## 依赖删除

`go.mod:5-12` 有 6 个直接依赖。

### 删除 TUI 后可移除

- `github.com/charmbracelet/bubbles`
- `github.com/charmbracelet/bubbletea`
- `github.com/charmbracelet/lipgloss`
- `github.com/charmbracelet/x/ansi`
- `golang.org/x/term`

其中 `x/term` 只服务于 rich doctor 的 terminal width：`internal/cli/cli.go:263-275`。

运行 `go mod tidy` 后，Charm 带来的 display-width、terminal、cell buffer、color 等间接依赖也会消失。

### env-only 配置后还可移除

- `github.com/zalando/go-keyring`

其唯一生产使用在 `internal/config/secret_store.go:7-56`。随后 `wincred`、`dbus` 等平台间接依赖也会消失。最终 Go 代码可以只用标准库，删除 `go.mod` 中全部 **6 个直接 + 21 个间接 require 条目**。

## 应保留的非 YAGNI 边界

即使只生成 message，也不建议为了 LOC 删除：

- diff/body 大小限制：`internal/providers/http.go:19-31`
- secret 检测，因为 diff 会离开本机：`internal/security/secrets.go`
- provider 响应和配置输入校验
- 无副作用保证：只读 Git，绝不 stage、commit、reset 或写文件

这些属于信任边界和安全措施，不是产品功能膨胀。

## 最小目标架构

只保留一条路径：

1. 解析少量 flags/env。
2. 调用原生 `git diff --cached`，或明确选择一种 scope。
3. 做 diff budget 与 secret 检查。
4. 向一个 provider 发一次请求。
5. 将 commit message 写到 stdout。

无需接口工厂、tool loop、TUI、doctor、onboarding、provider capability、commit plan、repair、transaction 或 Git 写操作。