## Review

- **Correct:** 当前实现与仓库现有 `PLAN.md` 一致：它明确要求 Full-screen TUI、Autonomous Commit、Tool Call Loop、Secret Store、配置面板和 doctor（`PLAN.md:5-26`）。`go test ./...` 全部通过。
- **Blocker:** 现有计划本身与用户真实目标不一致。目标只是“快速生成 commit message”，但 README 将产品定义为工作树规划、分组并直接创建一个或多个提交的平台（`README.md:21-41`）。这不是局部过度抽象，而是产品边界整体偏离。
- **Note:** 建议**重写核心而非在现架构上逐层删减**。当前约有 10,456 行生产 Go、5,897 行 Go 测试和 527 行 Node 脚本；真正目标只需要：读取 diff → 单次模型请求 → stdout 输出消息。以下行数为物理行，测试行只应随对应能力删除，不应单独砍测试。各项按包划分，基本不重叠。

### 按最大可删收益排序

#### 1. 重写 CLI 编排层：约 5,500 行

**证据**

- `internal/cli/cli.go`：2,176 行。
- `internal/cli/cli_test.go`：3,324 行。
- `Runtime` 为测试和多产品表面注入 TUI、配置面板、Onboarding、Commit/Repair Provider 等十余项依赖（`internal/cli/cli.go:28-42`）。
- `runAuto` 不只是生成文字，而是处理 scope、冲突、Secret Blocker、JSON、dry-run 和真实提交（`internal/cli/cli.go:436-634`）。
- Interactive Repair 及 TUI handoff 位于 `internal/cli/cli.go:675-862`。
- Tool Call Loop 规划和执行位于 `internal/cli/cli.go:1005-1172`。
- 多提交计划转换、事务执行和回滚编排位于 `internal/cli/cli.go:1222-1457`。
- 默认命令进入完整交互提交流程并最终直接创建 commit（`internal/cli/cli.go:1557-1670`）。

**判断：重写。**

现文件把所有非目标能力耦合在一个入口中；原地删减会留下大量参数、结果类型和跨包依赖。目标入口应只保留：

1. 解析极少量参数；
2. 调用 Git 获取 diff；
3. 调用一次 provider；
4. 将 commit message 写到 stdout。

不得再执行 `git add`、`git commit`、`git reset` 或写工作树。

---

#### 2. 删除 Full-screen TUI：约 3,150 行及全部 Charm 依赖

**证据**

- `internal/tui/` 生产代码约 2,898 行，测试 252 行。
- 主模型定义六个页面：scope review、Agent Instruction、AI activity、message review/edit、repair review（`internal/tui/model.go:16-25`）。
- `Model` 保存 scope cursor、选择状态、viewport、spinner、theme 等完整 UI 状态（`internal/tui/model.go:78-101`）。
- 键盘状态机在 `internal/tui/model.go:159-301`。
- 富文本/纯文本双渲染、双栏和终端尺寸布局占据 `internal/tui/model.go:739-1479`。
- 配置面板又有 list/edit/choice 三态模型（`internal/tui/config_panel.go:29-51`）。
- Onboarding 是七步 wizard（`internal/tui/onboarding.go:69-142`）。
- `go.mod:6-9` 因此直接引入 Bubbles、Bubble Tea、Lip Gloss、ANSI 及大量间接依赖。

**判断：原地整包删除。**

快速生成消息不需要 alternate screen、鼠标、viewport、spinner、主题、resize 或内嵌 diff review。普通 stdout 已经是最合适的界面。

---

#### 3. 重写配置系统，删除 init/config/doctor 产品套件：约 2,988 行

**证据**

不含上述 TUI 部分：

- `internal/config/`：约 1,124 行生产代码、423 行测试。
- `internal/commands/`：约 862 行。
- `internal/doctor/`：321 行生产代码、258 行测试。
- 配置模型包含四类 provider、七种 message style、四种语言、九个 key 以及推荐 provider/model（`internal/config/schema.go:15-140`）。
- 配置解析组合 user、project、environment、flag、Secret Store，并追踪每项来源（`internal/config/service.go:218-266`、`internal/config/service.go:435-443`）。
- `cnm config` 实现 get/list/set/unset、JSON 和 dry-run（`internal/commands/config.go:22-234`）。
- `cnm init` 同时实现 TTY、非 TTY、JSON、dry-run、明文密钥和 Secret Store 流程（`internal/commands/init.go:47-240`）。
- doctor 构造 Git、repository、配置来源、provider capability、probe 和 issue dashboard（`internal/doctor/service.go:23-134`）。

**判断：配置核心重写；三个子命令原地删除。**

轻量版本只需要少量运行参数，例如 API key、model、endpoint 和可选语言。可直接使用环境变量或一个极小配置文件；不要再保留 Shared/Private Preference、Provider Recommendation、来源指示器、配置 CRUD 和诊断产品。凭据可交给环境或既有 provider 凭据机制，没必要自行建设 Onboarding/Secret Store 产品层。

---

#### 4. 删除 Tool Call Runtime，重写 provider 为单次文本请求：约 2,472 行

**证据**

- `internal/providers/`：约 1,038 行生产代码、413 行测试。
- `internal/runtime/`：599 行生产代码、422 行测试。
- `ToolCallRuntime` 实现 provider retry、no-tool reminder、循环上限、时限和工具调度（`internal/runtime/runtime.go:11-129`）。
- 它还实现 inspect-before-create、read-before-write、repair confirmation 等策略（`internal/runtime/runtime.go:189-255`）。
- `DomainToolSet` 为六个函数建立一层包装（`internal/runtime/types.go:83-109`）。
- `ToolProtocolAdapter` 抽象 build/parse/continuation 三套操作（`internal/providers/types.go:16-22`）。
- 四种协议分别拥有 request builder、turn parser 和 tool-result payload（`internal/providers/tool_protocol.go:44-251`）。
- HTTP provider 为四种协议维护四套会话状态（`internal/providers/tool_call_provider.go:21-33`、`147-233`）。
- 八种 Domain Tool 还有专门 JSON Schema（`internal/providers/tool_schema.go:7-115`）。

**判断：重写。**

生成一条 commit message 不需要 agent loop、工具调用、continuation、finish/abort 或 provider capability metadata。保留一个实际使用的文本生成协议即可；只有确认存在真实多-provider 用户需求后，再增加第二个薄适配器，不要预先维护四种 tool-call 方言。

---

#### 5. 将 Git 层缩成“取得 diff”：约 1,829 行可大幅删除

**证据**

- `internal/git/`：约 1,104 行生产代码、623 行测试。
- `internal/security/`：102 行。
- `git` 类型层建模 Context Policy、Budget、Opaque Change、AI Exposure、repository state、Index Snapshot 和 transaction（`internal/git/types.go:40-171`、`218-227`）。
- `InspectCommitScope` 获取 staged/unstaged/untracked、binary metadata、secret scan、read budget 和 exposure summary（`internal/git/service.go:131-215`）。
- `CommitScopeWithMessage` 主动暂存并提交（`internal/git/service.go:361-392`）。
- transaction snapshot/rollback 位于 `internal/git/service.go:394-447`。

**判断：重写。**

保留一个执行 `git diff --cached` 的薄函数即可；若产品明确要支持未暂存变更，再加一个直接选项。可以保留简单字节上限，但无需把预算、可见文件、配置来源和 opaque metadata 建成完整领域模型。

---

#### 6. 删除现有产品语义与治理文档：约 1,113 行

**证据**

- `CONTEXT.md`：331 行，定义 Autonomous Commit、Interactive Repair、Tool Call Runtime、Commit Transaction 等大量领域术语。
- `README.md`：418 行。
- `PLAN.md`：47 行。
- 21 个 ADR：约 151 行。
- `docs/implementation-todo.md`：166 行、15 个阶段；其中 Tool Runtime、split/recovery、四 provider、TUI、repair、doctor 都已被产品化（`docs/implementation-todo.md:46-165`）。

**判断：完成核心重写后原地删除并重写 README。**

保留一页文档即可：安装、API key、运行 `cnm`、输出示例。旧 ADR 可以放历史分支，不应继续充当新产品约束。

---

#### 7. npm 原生二进制分发层：最多约 922 行，条件性删除

**证据**

- `scripts/`：527 行，包括安装、下载、校验、解压、launcher 和 Windows smoke tests。
- `.goreleaser.yml`、Makefile、CI/release workflow：约 227 行。
- distribution/release-runbook：168 行。
- npm 安装器自行处理平台/架构、release URL、checksum 和系统解压命令（`scripts/npm-install.js:21-128`）。

**判断：若 npm 不是已验证的主要获客渠道，则原地删除，改用 GitHub Release 或 `go install`。**

这是分发复杂性而非产品核心；若 npm 用户量已经成立，可以保留，但应与核心重写解耦。

---

### 建议的最终边界

```text
cnm
 ├─ 读取少量环境变量/flags
 ├─ 获取 Git diff
 ├─ 单次 HTTP 文本生成请求
 └─ stdout 仅输出 commit message
```

明确不包含：

- 自动暂存或创建 commit；
- 多 commit 规划和拆分；
- 冲突修复或文件写入；
- TUI、onboarding、config、doctor；
- Tool Call Loop 和 Domain Tools；
- Machine Output Contract、AI Exposure dashboard；
- provider probing、capability metadata；
- 四协议同步演进。

## Review conclusion

- **Blocker:** 继续沿当前 `PLAN.md` 原地完善会进一步偏离真实目标。
- **Note:** 建议新建极小核心实现，验证输出后一次性删除旧产品包；不要尝试在 `internal/cli/cli.go` 内逐功能打补丁。
- **Note:** 本审计按要求未评价正确性、安全漏洞或性能。仓库没有请求中的小写 `plan.md`/`progress.md`；实际读取了现有 `PLAN.md`，未发现 `progress.md`。