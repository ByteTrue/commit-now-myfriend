---
doc_type: roadmap
slug: product-completion
status: completed
created: 2026-05-24
last_reviewed: 2026-05-24
tags: [product-completion, cnm, go, tui]
related_requirements:
  - interactive-commit
  - autonomous-commit
  - safety-and-privacy
  - setup-and-diagnostics
  - native-binary-distribution
related_architecture:
  - architecture-overview
---

# 完成 Go-native cnm 产品

## 1. 背景

这个 roadmap 归档 `commit-now-myfriend` 从旧的 staged-first / TypeScript 迁移思路，切换到 Go-native `cnm` 产品语义后的完成路线。当前源码和 `docs/implementation-todo.md` 显示各阶段已经完成，因此本 roadmap 作为历史规划层补档，承接 `PLAN.md`、requirements、architecture 和 decisions，方便后续从 CodeStable 入口追溯产品是如何拆成一组可验证能力的。

目标产品是一个本地 AI-assisted Git commit workflow CLI：默认 `cnm` 走 Interactive Commit，全屏审阅后提交；`cnm auto` 走 Autonomous Commit，本地自动创建一个或多个 commit；配置、诊断、安全、分发都围绕这两个主流程收敛。

## 2. 范围与明确不做

### 本 roadmap 覆盖

- 重新收敛 CLI command surface：`cnm`、`cnm auto`、`cnm init`、`cnm config`、`cnm doctor`。
- 建立 preferences、Secret Store、onboarding、doctor 和 provider probing 的配置/诊断能力。
- 建立 Working Tree Commit、Commit Scope、Secret Blocker、AI Exposure Summary 和安全边界。
- 建立 provider-native Tool Call Runtime、Domain Tools、provider protocol adapters。
- 建立 Autonomous Commit、File-level Commit Split、transactional commit recovery。
- 建立 Interactive Commit Full-screen TUI、TUI commit execution 和 Interactive Repair。
- 建立 Go native binary + npm wrapper + release-path verification。

### 明确不做

- 不恢复 standalone `split`、`repair`、`check`、`onboard` 命令；这些能力属于主流程或用户工作流。
- 不做 Hunk-level Commit Split；首版只支持 File-level Commit Split。
- 不把 cnm 变成项目测试 / lint runner；项目验证留给用户命令和 Git hooks。
- 不默认远程 telemetry；诊断默认本地，provider probe 必须显式触发。
- 不把 npm 作为产品 runtime；npm 只作为 native binary 的安装和启动入口。

## 3. 模块拆分（概设）

```text
product-completion
├── command-surface：收敛命令入口和运行模式
├── config-diagnostics：偏好、凭证、初始化和诊断
├── git-safety：提交范围、diff/read 预算、安全 blocker 和事务恢复
├── provider-runtime：provider-native tool-call loop 与 Domain Tools
├── interactive-tui：全屏 TUI、提交确认、交互修复
└── distribution-docs：Go binary、npm wrapper、文档和发布校验
```

### command-surface · 命令面

- **职责**：定义用户能启动的产品入口，移除旧命令，保持 help/smoke tests 与产品语义一致。
- **承载的子 feature**：`command-surface-core`。
- **触碰的现有代码 / 模块**：`cmd/cnm`、`internal/cli`、`internal/commands`。

### config-diagnostics · 配置和诊断

- **职责**：区分 private/shared preferences，默认使用 Secret Store 保存凭证，提供 onboarding/config/doctor 三个辅助入口。
- **承载的子 feature**：`preferences-onboarding`、`doctor-provider-probing`。
- **触碰的现有代码 / 模块**：`internal/config`、`internal/commands`、`internal/doctor`、`internal/tui`。

### git-safety · Git 与安全边界

- **职责**：定义 Commit Scope、Git inspection、secret detection、opaque changes、budget、hook failure 和 transaction rollback 行为。
- **承载的子 feature**：`working-tree-scope-safety`、`transactional-file-split`。
- **触碰的现有代码 / 模块**：`internal/git`、`internal/security`。

### provider-runtime · Provider 和运行时

- **职责**：把不同 provider protocol 适配成同一个本地 Tool Call Runtime；只暴露 Domain Tools，不暴露 shell/raw Git。
- **承载的子 feature**：`tool-call-runtime`、`provider-protocols`、`autonomous-commit-loop`。
- **触碰的现有代码 / 模块**：`internal/providers`、`internal/runtime`。

### interactive-tui · 交互式提交界面

- **职责**：提供 Full-screen TUI 的 scope/diff/AI activity/plan/message/repair 体验，并在副作用前要求开发者确认。
- **承载的子 feature**：`interactive-commit-tui`、`tui-commit-execution`、`interactive-repair`。
- **触碰的现有代码 / 模块**：`internal/tui`、`internal/interactive`、`internal/cli`。

### distribution-docs · 分发和完成校验

- **职责**：通过 npm wrapper 交付 Go native binary，补齐 README/CLI docs/release rehearsal，并跑最终产品语义清理和 smoke tests。
- **承载的子 feature**：`distribution-docs-release`、`completion-audit`。
- **触碰的现有代码 / 模块**：`scripts/`、`.goreleaser.yml`、`package.json`、`docs/`、`README.md`。

## 4. 模块间接口契约 / 共享协议（架构层详设）

### 4.1 CLI command surface

**方向**：用户 / shell → `internal/cli`
**形式**：CLI 命令协议

**契约**：

```text
cnm [--staged] [--no-untracked] [--diff-only] [--json] [-- pathspec...]
cnm auto [--tui|-i] [--staged] [--no-untracked] [--diff-only] [--json] [-- pathspec...]
cnm init [flags...]
cnm config [list|get|set|unset ...]
cnm doctor [--json] [--probe-provider]
cnm --version
```

**约束**：

- `split`、`repair`、`check`、`onboard` 不再是产品命令。
- Commit Scope flags 在 Interactive Commit 和 Autonomous Commit 中语义一致。
- `doctor --probe-provider` 是显式远程 provider 探测入口；默认 doctor 不调用 provider。

### 4.2 Effective config and credential resolution

**方向**：CLI / commands → config service / Secret Store
**形式**：配置来源合并协议

**契约**：

```text
EffectiveConfig:
  provider: ProviderType | null
  model: string | null
  baseURL: string | null
  promptStyle: string
  messageLanguage: string
  standingInstruction: string | null
  apiKeySource: env | secret_store | plaintext | missing
  sources: per-field source metadata
```

**约束**：

- Project config may recommend provider/model, but must not force private credentials.
- API keys resolve from env, Secret Store, or explicit plaintext opt-in.
- Secret Store is the default persistent credential target.

### 4.3 Commit Scope and AI exposure

**方向**：CLI / TUI / runtime → Git layer
**形式**：本地 Git inspection 数据结构

**契约**：

```text
CommitScopeOptions:
  cwd: string
  stagedOnly: bool
  includeUntracked: bool
  pathspecs: string[]
  diffOnly: bool

CommitScope:
  files: FileStatus[]
  diffMetadata: DiffMetadata
  aiExposureSummary: AIExposureSummary
  secretBlockers: SecretFinding[]
  opaqueChanges: OpaqueChange[]
```

**约束**：

- 默认从 working tree 规划，包含 staged、unstaged tracked、untracked non-ignored。
- Secret Blocker blocks the selected flow; it is not silently skipped.
- Diff/read budgets bound AI exposure.

### 4.4 Provider-native Tool Call Runtime

**方向**：Provider adapters → Tool Call Runtime → Domain Tools
**形式**：provider turn / tool call / tool result 协议

**契约**：

```text
ProviderTurn:
  message: string | null
  toolCalls: ToolCallRequest[]
  final: bool

ToolCallRequest:
  name: inspect_scope | create_commits | repair_file | ...
  arguments: object

ToolCallResult:
  call_id: string
  ok: bool
  result: object | null
  error: structured_error | null
```

**约束**：

- Provider adapters own wire-format differences; runtime owns local validation and side effects.
- Tool definitions and required JSON schemas are sent on every provider request.
- AI never receives shell access or raw Git command access.

### 4.5 TUI confirmation and repair handoff

**方向**：Interactive TUI ↔ Git/runtime/repair tools
**形式**：screen-state workflow contract

**契约**：

```text
Interactive Commit screens:
  scope_review -> ai_activity -> plan_review -> message_edit -> confirm_commit
  optional: repair_review -> repair_apply_confirmation
```

**约束**：

- TUI commit execution requires developer confirmation before side effects.
- Interactive Repair can write only eligible conflicted files after read-before-write and developer confirmation.
- Non-interactive Autonomous Commit fails on conflicts unless explicitly handed off to TUI.

### 4.6 Distribution wrapper

**方向**：npm package → native binary
**形式**：launcher / postinstall file protocol

**契约**：

```text
scripts/cnm.js:
  choose bin/cnm(.exe) if installed, else dist/go/cnm(.exe)
  spawn native binary with original argv

scripts/npm-install.js:
  resolve platform artifact for package version
  download release binary into bin/
```

**约束**：

- npm is not the product runtime.
- `cnm --version` is injected at build time from package metadata.
- Local release rehearsal must catch wrapper/binary path mismatches before publish.

## 5. 子 feature 清单

1. **command-surface-core** — 收敛 `cnm` / `cnm auto` / setup commands，并移除旧命令入口。
   - 所属模块：command-surface
   - 依赖：无
   - 状态：done
   - 对应 feature：未启动（历史补档）

2. **preferences-onboarding** — 建立 private/shared preferences、Secret Store 默认凭证保存和 first-run onboarding。
   - 所属模块：config-diagnostics
   - 依赖：command-surface-core
   - 状态：done
   - 对应 feature：未启动（历史补档）

3. **working-tree-scope-safety** — 建立 working tree Commit Scope、pathspec/staged/untracked 策略、预算和安全 blocker。
   - 所属模块：git-safety
   - 依赖：command-surface-core
   - 状态：done
   - 对应 feature：未启动（历史补档）

4. **tool-call-runtime** — 建立 provider-native Tool Call Loop、Domain Tools、tool validation 和 loop limits。
   - 所属模块：provider-runtime
   - 依赖：working-tree-scope-safety
   - 状态：done
   - 对应 feature：未启动（历史补档）

5. **autonomous-commit-loop** — 用 Tool Call Runtime 驱动 Autonomous Commit 的 plan 和 local commit creation。
   - 所属模块：provider-runtime / git-safety
   - 依赖：tool-call-runtime
   - 状态：done
   - 对应 feature：未启动（历史补档）

6. **transactional-file-split** — 支持 File-level Commit Split、split limitation、index snapshot 和 commit transaction rollback。
   - 所属模块：git-safety
   - 依赖：autonomous-commit-loop
   - 状态：done
   - 对应 feature：未启动（历史补档）

7. **provider-protocols** — 适配 OpenAI Responses、OpenAI-compatible、Anthropic Messages、Google Gemini 的 native tool calls。
   - 所属模块：provider-runtime
   - 依赖：tool-call-runtime
   - 状态：done
   - 对应 feature：未启动（历史补档）

8. **interactive-commit-tui** — 建立 Bubble Tea Full-screen TUI shell、scope/diff/AI activity/plan review/message review。
   - 所属模块：interactive-tui
   - 依赖：working-tree-scope-safety, tool-call-runtime
   - 状态：done
   - 对应 feature：未启动（历史补档）

9. **tui-commit-execution** — 在 TUI 中执行 single/multi commit，复用 hook respect、message retry、snapshot 和 transaction 行为。
   - 所属模块：interactive-tui / git-safety
   - 依赖：interactive-commit-tui, transactional-file-split
   - 状态：done
   - 对应 feature：未启动（历史补档）

10. **interactive-repair** — 支持 conflict repair 的 TUI handoff、eligible file writes、read-before-write 和确认应用。
    - 所属模块：interactive-tui
    - 依赖：tui-commit-execution
    - 状态：done
    - 对应 feature：未启动（历史补档）

11. **doctor-provider-probing** — 让 doctor 默认本地诊断，并提供显式 provider probe 和 machine output 状态。
    - 所属模块：config-diagnostics
    - 依赖：preferences-onboarding, provider-protocols
    - 状态：done
    - 对应 feature：未启动（历史补档）

12. **distribution-docs-release** — 更新 README/CLI docs/privacy/distribution，配置 native binary release 与 npm wrapper。
    - 所属模块：distribution-docs
    - 依赖：command-surface-core
    - 状态：done
    - 对应 feature：未启动（历史补档）

13. **completion-audit** — 对产品语义、help、JSON output、smoke tests、release path 做最终清理和验证。
    - 所属模块：distribution-docs / 全模块
    - 依赖：全部前置条目
    - 状态：done
    - 对应 feature：未启动（历史补档）

**最小闭环**：第 5 条 `autonomous-commit-loop` 做完后，系统已经能通过 provider-native Tool Call Loop 从选中本地改动生成本地 commit，是端到端最窄产品路径。

## 6. 排期思路

历史推进顺序先收敛命令面和配置，再建立 Git scope / safety，再建立 Tool Call Runtime，随后打通 Autonomous Commit 最小闭环。之后补齐 split/transaction/provider adapters，再进入 Interactive TUI、TUI commit execution 和 repair。最后完成 doctor/provider probing、docs/distribution、completion audit。这个顺序按技术依赖推进：没有 scope 和 runtime 就无法可靠做 autonomous flow；没有 commit execution 和 TUI confirmation 就不能做 interactive repair。

## 7. 观察项

- 这是对已完成 `docs/implementation-todo.md` 的 CodeStable roadmap 补档；items 全部标为 `done`，`feature` 字段为空，表示这些工作发生在 CodeStable feature 流程接入之前。
- `PLAN.md` 和 `docs/implementation-todo.md` 仍可作为历史来源保留；后续新工作应优先从 `.codestable/roadmap/` 或 `.codestable/features/` 发起。
- 当前没有新的 planned 子 feature；如果要继续产品演进，应另开 active roadmap 或直接走 `cs-feat`。
