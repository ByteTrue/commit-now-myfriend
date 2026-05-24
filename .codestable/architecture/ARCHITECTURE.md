---
doc_type: architecture
slug: architecture-overview
scope: commit-now-myfriend Go-native CLI system overview
summary: Maps the current cnm command surface, local commit runtime, provider adapters, TUI, config, safety, and npm binary distribution boundaries.
status: current
last_reviewed: 2026-05-24
tags: [overview, cli, go, tui, git, providers]
depends_on: []
implements: []
---

# commit-now-myfriend 架构总入口

## 0. 术语

- **Interactive Commit**：默认 `cnm` 流程，开发者在 Full-screen TUI 中确认 scope、计划、消息和 repair 入口；术语来源见 `CONTEXT.md:15`。
- **Autonomous Commit**：`cnm auto` 流程，AI-led 本地提交，可创建一个或多个本地 Git commits，但不 push；术语来源见 `CONTEXT.md:7`。
- **Tool Call Runtime**：把 provider-native tool calls 转成 local Domain Tools 执行循环的运行时；当前实现入口是 `internal/runtime/runtime.go:11` 和 `internal/runtime/runtime.go:43`。
- **Domain Tools**：暴露给 AI 的受限提交领域工具，不是 shell/Git 原生命令入口；约束来源见 `PLAN.md:22` 和 `PLAN.md:23`。
- **Commit Scope**：一次提交流程选中的 working tree 变更集合；术语来源见 `CONTEXT.md:67`，代码类型见 `internal/git/types.go:134`。

## 1. 项目简介

commit-now-myfriend 是一个 Go-native、本地运行的 AI-assisted Git commit workflow CLI。对外 npm 包仍叫 `commit-now-myfriend`，命令入口是 `cnm`，但 npm 只负责分发和启动 native binary，不承载产品运行时；分发目标见 `docs/distribution.md:3`、`docs/distribution.md:7`、`docs/distribution.md:8`。

当前产品由两个主提交流程和若干辅助命令组成：`cnm` 进入 Interactive Commit Full-screen TUI，`cnm auto` 进入 Autonomous Commit；`cnm init`、`cnm config`、`cnm doctor` 分别负责 onboarding/preferences、配置读写和环境诊断。产品形态来源见 `PLAN.md:5`、`PLAN.md:7`、`PLAN.md:8`、`PLAN.md:9`；README 对外描述见 `README.md:21`、`README.md:25`、`README.md:26`。

## 2. 结构与交互

### 2.1 CLI 调度层

`cmd/cnm/main.go` 只做进程入口，调用 `internal/cli.Execute` 并用返回码退出（`cmd/cnm/main.go:9`）。`internal/cli` 负责组装默认 runtime 依赖，包括 Secret Store、TUI runners、provider factory、Git runner、HTTP client 和输出 writer（`internal/cli/cli.go:28`、`internal/cli/cli.go:44`、`internal/cli/cli.go:57`）。

命令分派集中在 `executeWithRuntime`：它先构造 output router 和配置服务，再按子命令进入 `init`、`doctor`、`config` 等辅助流程；没有显式子命令时进入默认 commit flow，`auto` 子命令进入 autonomous flow（`internal/cli/cli.go:64`、`internal/cli/cli.go:95`、`internal/cli/cli.go:108`）。这种分法让命令解析和依赖注入保持在 CLI 层，业务动作落在 commands/config/doctor/git/runtime/tui 等包里。

### 2.2 配置、偏好与 Secret Store

配置模型在 `internal/config/schema.go`，包括 provider 类型、API key source、private/shared preferences、effective config 等枚举和结构（`internal/config/schema.go:3`、`internal/config/schema.go:16`、`internal/config/schema.go:42`、`internal/config/schema.go:116`）。解析入口是 `ResolveEffectiveConfig`，它把 cwd/env/secret store/preferences 解析成运行期有效配置（`internal/config/service.go:218`）。

API keys 默认写入系统 Secret Store，而不是 plaintext config；Secret Store 适配在 `internal/config/secret_store.go`（`internal/config/secret_store.go:12`、`internal/config/secret_store.go:16`、`internal/config/secret_store.go:20`、`internal/config/secret_store.go:37`）。这个边界让 provider credential 的读取和偏好配置分离，也使 `doctor` 能报告 credential source。

### 2.3 Git / Commit Scope / Safety 层

Git 层是本地仓库事实来源：`InspectRepository` 检查仓库状态，`InspectCommitScope` 根据默认 whole-working-tree、`--staged`、pathspec、untracked 策略等构造 Commit Scope（`internal/git/service.go:52`、`internal/git/service.go:131`）。diff/read 预算有固定默认值，避免无限制把仓库内容暴露给 AI（`internal/git/service.go:15`、`internal/git/service.go:16`）。

提交执行保持在 Git 层：`CommitScopeWithMessage` 对选中 scope 进行提交，Autonomous Commit 可先 `CaptureCommitTransactionSnapshot`，失败时再 `RollbackCommitTransaction`（`internal/git/service.go:361`、`internal/git/service.go:394`、`internal/git/service.go:409`）。Secret blocker 也在 scope 检查中生成，具体文本扫描在 `internal/security/secrets.go:64`，scope 汇总在 `internal/git/service.go:218`。

### 2.4 Provider adapters 与 Tool Call Runtime

Provider 层把 OpenAI Responses、OpenAI-compatible、Anthropic Messages、Google Gemini 等 provider protocol 适配成统一的 Tool Call Provider。公开能力和配置结构在 `internal/providers/types.go`（`internal/providers/types.go:8`、`internal/providers/types.go:16`、`internal/providers/types.go:24`），HTTP provider factory 是 `CreateToolCallProvider`（`internal/providers/tool_call_provider.go:14`、`internal/providers/tool_call_provider.go:36`）。

Protocol 差异集中在 adapter：`CreateToolProtocolAdapter` 根据 provider config 选择解析和 payload builder；各 provider 的 turn parser / tool-result builder 分散在同一文件中（`internal/providers/tool_protocol.go:12`、`internal/providers/tool_protocol.go:32`、`internal/providers/tool_protocol.go:39`、`internal/providers/tool_protocol.go:44`、`internal/providers/tool_protocol.go:63`、`internal/providers/tool_protocol.go:105`、`internal/providers/tool_protocol.go:155`）。

Runtime 层只理解 provider turn、tool call request/result、repair policy 和 domain tool set。`ToolCallRuntime.Run` 驱动 turn loop，`executeToolCall` 把 provider 请求派发给本地 domain tool（`internal/runtime/runtime.go:11`、`internal/runtime/runtime.go:21`、`internal/runtime/runtime.go:43`、`internal/runtime/runtime.go:110`；类型见 `internal/runtime/types.go:30`、`internal/runtime/types.go:49`、`internal/runtime/types.go:53`、`internal/runtime/types.go:71`、`internal/runtime/types.go:76`）。这种分层让 provider 只负责“怎么表达 tool calls”，runtime 负责“什么时候执行什么本地能力”。

### 2.5 TUI / Interactive Commit

Full-screen TUI 位于 `internal/tui`，`ModelInput` 是 CLI/业务层注入 TUI 的边界，包含 scope、plan、diff provider、file reader、repair context 等（`internal/tui/model.go:27`、`internal/tui/model.go:45`、`internal/tui/model.go:51`、`internal/tui/model.go:56`）。屏幕状态由 `Screen` 枚举表达（`internal/tui/model.go:16`）。

`internal/tui/run.go:16` 是 Bubble Tea 程序入口。TUI 使用 alt-screen、mouse cell-motion、WindowSizeMsg-driven layout 的承诺来自 `PLAN.md:18`、`PLAN.md:19`，代码上通过 `tui.Run` 与 CLI runtime 注入隔离，便于测试时替换 runner（`internal/cli/cli.go:35`、`internal/cli/cli.go:52`）。

### 2.6 Doctor / Init / Config 辅助流

`internal/commands/init.go:47` 运行 onboarding，`internal/commands/config.go:22` 运行 config 命令，`internal/commands/doctor.go:23` 运行 doctor 命令。Doctor 的核心服务是 `doctor.Run`，输入包含 cwd/env/secret store/provider probe 等依赖（`internal/doctor/service.go:13`、`internal/doctor/service.go:23`）。这些命令通过 CLI runtime 注入共享 config、secret store、provider factory 和 output router，避免辅助流各自重建系统边界。

### 2.7 输出与 npm/native binary 分发

输出路由在 `internal/output/router.go`，`Router` 根据 jsonMode 把 human-readable 和 JSON 输出送到 stdout/stderr（`internal/output/router.go:24`、`internal/output/router.go:30`）。这让 CLI command 层不用直接关心 JSON mode 的 writer 细节。

npm 包的 `bin.cnm` 指向 `scripts/cnm.js`，该 launcher 在已安装 native binary 和本地 build binary 之间选择，然后 `spawnSync` 传递参数执行（`scripts/cnm.js:6`、`scripts/cnm.js:9`、`scripts/cnm.js:18`）。postinstall 下载 release binary，release owner/repo/base URL 可由环境变量覆盖（`scripts/npm-install.js:12`、`scripts/npm-install.js:14`）。本地 release rehearsal 和 npm wrapper layout 见 `docs/distribution.md:34`、`docs/distribution.md:54`、`docs/distribution.md:58`、`docs/distribution.md:59`。

## 3. 数据与状态

- **仓库状态**：由 Git 层从 cwd + env + git runner 实时读取，不在 cnm 内持久化；入口见 `internal/git/service.go:52` 和 `internal/git/service.go:131`。
- **Commit Scope / Plan / Results**：scope 类型位于 `internal/git/types.go:134`，commit result 和 transaction snapshot 位于 `internal/git/types.go:212`、`internal/git/types.go:218`；TUI 展示层另有 `CommitPlanView`（`internal/tui/model.go:56`）。
- **Provider config 与 preferences**：effective config 位于 `internal/config/schema.go:116`，解析入口是 `internal/config/service.go:218`。
- **API key**：默认归系统 Secret Store 所有，读取/写入经 `SystemSecretStore`（`internal/config/secret_store.go:12`、`internal/config/secret_store.go:20`、`internal/config/secret_store.go:37`）。
- **AI turn state**：由 provider API response 和 runtime loop 暂态持有，不作为项目文件持久化；runtime 结果类型见 `internal/runtime/types.go:76`。
- **分发状态**：npm install 后 native binary 落到 `bin/`，本地打包 rehearsal 使用 `dist/release-local`；见 `docs/distribution.md:54`、`scripts/cnm.js:9`、`scripts/pack-binary.js:10`。

## 4. 关键架构决定

- 主命令面围绕 Interactive Commit 和 Autonomous Commit，而不是拆成单独 split/repair 命令；来源 `.codestable/compound/2026-05-24-decision-replace-split-command-with-primary-commit-flows.md`、`.codestable/compound/2026-05-24-decision-simplify-command-surface-around-interactive-and-auto.md`。这解释了 CLI 调度层为什么把 split/repair 嵌入主 flow。
- Interactive Commit 使用 Full-screen TUI 和 Bubble Tea 生态；来源 `.codestable/compound/2026-05-24-decision-use-full-screen-tui-for-interactive-commit.md`、`.codestable/compound/2026-05-24-decision-use-bubble-tea-ecosystem-for-tui.md`。这解释了 `internal/tui` 作为主交互层而非逐行 prompt 的原因。
- Commit planning 默认来自 working tree；来源 `.codestable/compound/2026-05-24-decision-plan-commits-from-working-tree-by-default.md`。早期“plan/execution plan split”已由 native Tool Call Loop 决策取代，见 `.codestable/compound/2026-05-24-decision-separate-commit-plan-from-execution-plan.md`。
- Provider interaction 采用 native Tool Call Loop 和 Domain Tools；来源 `.codestable/compound/2026-05-24-decision-support-four-provider-protocols-as-first-class.md`、`.codestable/compound/2026-05-24-decision-use-native-tool-call-loop-as-core-control-model.md`。这解释了 providers adapter 与 runtime/tool dispatch 的分层。
- Multi-commit creation transactional，Git hooks respected by default；来源 `.codestable/compound/2026-05-24-decision-make-multi-commit-creation-transactional.md`、`.codestable/compound/2026-05-24-decision-respect-git-hooks-by-default.md`。这解释了 snapshot/rollback 和 hook failure surfacing 的存在。
- API keys 默认存在系统 Secret Store，首版不做 remote telemetry；来源 `.codestable/compound/2026-05-24-decision-store-api-keys-in-system-secret-store-by-default.md`、`.codestable/compound/2026-05-24-decision-no-remote-telemetry-in-first-version.md`。这解释了 config/doctor/privacy 边界。
- Native Go binary 通过 npm 和 releases 分发；来源 `.codestable/compound/2026-05-24-decision-distribute-native-binary-through-npm-and-releases.md`、`docs/distribution.md:3`。这解释了 scripts 只做 wrapper/downloader。

## 5. 代码锚点

- `cmd/cnm/main.go:main` — 进程入口。
- `internal/cli/cli.go:ExecuteWithRuntime` — 命令分派和依赖注入入口。
- `internal/config/service.go:ResolveEffectiveConfig` — effective config 解析。
- `internal/config/secret_store.go:SystemSecretStore` — 系统 Secret Store adapter。
- `internal/git/service.go:InspectCommitScope` — Commit Scope 构造、diff/secret exposure 汇总入口。
- `internal/git/service.go:CommitScopeWithMessage` — scoped commit 执行入口。
- `internal/providers/tool_call_provider.go:CreateToolCallProvider` — provider HTTP tool-call client factory。
- `internal/providers/tool_protocol.go:CreateToolProtocolAdapter` — provider protocol adapter 选择。
- `internal/runtime/runtime.go:ToolCallRuntime.Run` — provider-native tool-call loop。
- `internal/tui/run.go:Run` — Full-screen TUI Bubble Tea 入口。
- `internal/doctor/service.go:Run` — doctor 诊断服务。
- `scripts/cnm.js` — npm bin launcher 到 native binary。
- `scripts/npm-install.js` — postinstall release binary downloader。

## 6. 已知约束 / 硬边界

- cnm 不 push remote；Autonomous Commit 也只创建 local commits。术语约束见 `CONTEXT.md:7`。
- AI 不获得 shell access 或 raw Git command access，只能通过 Domain Tools 工作；架构承诺见 `PLAN.md:22`、`PLAN.md:23`。
- Project checks 不属于 cnm；Git hook failures 会被 surfaced，能安全回滚时才回滚。架构承诺见 `PLAN.md:26`。
- API keys 默认必须走系统 Secret Store；plaintext config 需要显式 opt-in。架构承诺见 `PLAN.md:24` 和实现 `internal/config/secret_store.go:12`。
- 首版不做 remote telemetry；相关长期决策见 `.codestable/compound/2026-05-24-decision-no-remote-telemetry-in-first-version.md`。
- Hunk-level split 暂不作为当前能力边界；术语和限制见 `CONTEXT.md:167` 以及 `.codestable/compound/2026-05-24-decision-defer-hunk-level-commit-split.md`。
- Interactive Repair 必须在 Full-screen TUI 中显式启用，不在 non-interactive Autonomous Commit 中静默运行；术语约束见 `CONTEXT.md:11`、`CONTEXT.md:12`，决策见 `.codestable/compound/2026-05-24-decision-require-interactive-flow-for-conflict-repair.md`、`.codestable/compound/2026-05-24-decision-remove-non-interactive-repair-mode.md`。

## 7. 相关文档

- `CONTEXT.md` — 产品领域语言与术语边界。
- `PLAN.md` — 当前产品完成计划和架构承诺。
- `README.md` — 对外用户入口与 CLI 行为说明。
- `.codestable/compound/` 下的 `2026-05-24-decision-*` 文档 — CodeStable 决策归档；原始 ADR 仍保留在 `docs/adr/`。
- `docs/distribution.md` — Go-first distribution / npm wrapper 细节。
- `.codestable/compound/2026-05-24-explore-onboard-migration-audit.md` — onboard 迁移审计。
