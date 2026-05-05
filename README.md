# commit-now-myfriend

AI 辅助的 Git 提交工作流工具。

`commit-now-myfriend` 是一个轻量级的命令行工具，通过 AI 分析你的暂存区更改并生成符合 Conventional Commits 规范的提交信息。

## 安装

你可以全局安装此工具：

```bash
npm install -g commit-now-myfriend
```

或者通过 `npx` 直接运行：

```bash
npx commit-now-myfriend
```

如果当前环境禁用了 `npx` 或需要显式指定包名，可以使用 npm exec 回退方式：

```bash
npm exec --package commit-now-myfriend cnm
```

**注意**：由于 `cnm` 这个二进制名称在 npm 生态中可能存在冲突，如果全局安装后无法直接使用 `cnm`，请尝试使用 `npx commit-now-myfriend`、`npm exec --package commit-now-myfriend cnm` 或 `pnpm exec cnm`。

## 快速开始

1. **初始化配置**：
   运行以下命令进行基础配置（支持通过命令行参数直接设置）：
   ```bash
   cnm init --provider openai-responses --api-key <your-api-key>
   ```
   该命令会创建用户级配置文件。

2. **暂存更改**：
   像往常一样使用 `git add` 暂存你的代码更改。

3. **生成提交信息**：
   直接运行 `cnm`：
   ```bash
   cnm
   ```
   工具会分析差异，生成预览，并询问你是否确认提交。你可以选择直接提交、编辑信息、重新生成或取消。

## 核心特性

### 暂存优先工作流
`cnm` 默认只处理已暂存（staged）的更改。如果当前没有暂存任何文件，工具会询问是否暂存所有更改。它永远不会在未经确认的情况下自动暂存文件。

### 安全与隐私
- **本地扫描**：在发送给 AI 之前，工具会扫描差异内容以识别潜在的敏感信息（如 API 密钥或令牌）。如果发现可疑内容，工具会发出警告，但不会替你阻断流程；是否继续由你在预览确认阶段决定。
- **隐私说明**：暂存区的差异内容会被发送到你配置的 AI 提供商。请确保你信任该提供商处理你的代码片段。
- **无自动提交**：在 v1 版本中，所有提交操作都必须经过人工确认。

> [!WARNING]
> **安全警告**：通过 `cnm init --api-key` 或 `cnm config set apiKey` 设置的 API 密钥将以**明文**形式存储在本地配置文件中（默认为 `~/.cnm/config.json`）。在多用户系统或未加密磁盘上请谨慎使用。推荐使用环境变量作为更安全的替代方案。

### 诊断工具
如果遇到环境问题，可以使用内置的诊断命令：
```bash
cnm doctor
```
它会检查 Git 环境、配置文件完整性以及 API 密钥设置。

## 配置说明

### 环境变量
环境变量具有比配置文件更高的优先级，是配置敏感信息（如 API 密钥）的推荐方式：
- `CNM_API_KEY`: AI 提供商的 API 密钥。
- `CNM_PROVIDER`: 使用的 AI 提供商（如 `openai-responses`, `anthropic-messages` 等）。
- `CNM_MODEL`: 使用的 AI 模型。
- `CNM_BASE_URL`: OpenAI 兼容服务的自定义基础 URL。
- `CNM_CUSTOM_PROMPT`: 自定义 AI 提示词指令。
- `CNM_HOME`: 更改配置文件的存储目录（默认为 `~/.cnm`）。

### 配置文件路径
- 用户级配置：`~/.cnm/config.json`
- 项目级配置：项目根目录下的 `.cnmrc.json`（出于安全考虑，项目级配置不支持 `apiKey` 字段，该字段会被忽略并警告）。

### 交互式配置面板
在交互式终端里直接运行 `cnm config` 会打开一个轻量级配置面板，可用于：
- 配置 provider 和 model
- 选择 commit message 风格（`auto`、`conventional`、`angular`、`google`、`atom`、`plain`、`custom`）
- 设置 API key、`baseURL`、自定义 prompt
- 查看当前生效配置（API key 会脱敏显示）
- 本地测试当前配置是否缺少必要字段
- 重置单个用户配置项或清空全部用户配置

在以下场景中，`cnm config` 会保持原有的非交互式可脚本行为，而不会进入面板：
- `cnm config --json`
- `cnm --json config`
- `cnm config --dry-run`
- 非 TTY 环境中的 bare `cnm config`

现有脚本化子命令保持不变：
- `cnm config get [key]`
- `cnm config list [--json]`
- `cnm config set <key> <value>`
- `cnm config unset <key>`

### 提交信息风格
`promptStyle` 支持以下值：
- `auto`：默认值，尽量推断仓库风格；信息不足时回退到 Conventional Commits。
- `conventional`：Conventional Commits，例如 `feat(scope): subject`。
- `angular`：Angular 风格的 `type(scope): subject`。
- `google`：Google 风格，短祈使句标题，可选正文解释 what/why。
- `atom`：简洁祈使句标题，可选正文，不强制 type 前缀。
- `plain`：普通自然语言提交信息，不强制格式。
- `custom`：不注入预设风格说明，只使用你的 `customPrompt` 加基础安全约束。

示例：
```bash
cnm config set promptStyle google
cnm config set promptStyle custom
cnm config set customPrompt "用中文生成简短提交信息，不要 type 前缀"
```

### 提供商配置示例

#### OpenAI (Responses)
```bash
cnm init --provider openai-responses --model gpt-4o --api-key <your-api-key>
```

#### OpenAI-compatible (如 DeepSeek, Local LLMs)
```bash
cnm init --provider openai-compatible --model deepseek-chat --base-url https://api.deepseek.com --api-key <your-api-key>
```

#### Anthropic (Messages)
```bash
cnm init --provider anthropic-messages --model claude-3-5-sonnet-20240620 --api-key <your-api-key>
```

#### Google Gemini
```bash
cnm init --provider google-gemini --model gemini-1.5-pro --api-key <your-api-key>
```

## 命令行参考

```text
Usage: cnm [options] [command]

AI-assisted commit workflow CLI.

Options:
  --dry-run                        预览执行过程，不产生实际副作用。
  --json                           以 JSON 格式输出结果（适用于支持的命令）。
  --provider <provider>            覆盖本次提交工作流使用的 AI 提供商。
  --model <model>                  覆盖本次提交工作流使用的 AI 模型。
  --base-url <baseUrl>             覆盖 OpenAI-compatible 基础 URL。
  --prompt-style <promptStyle>     覆盖本次提交工作流使用的提示风格。
  --custom-prompt <customPrompt>   覆盖本次提交工作流使用的自定义提示词。
  -V, --version                    显示版本号。
  -h, --help                       显示帮助信息。

Commands:
  init [options]    初始化配置。
  config [options]  查看或修改配置。
  doctor [options]  运行环境诊断。
```

### 预览与集成
- `--dry-run`：生成提交信息但不执行 `git commit`。
- `--json`：输出结构化数据，方便与其他工具集成。在 v1 中，使用此标志不会触发交互式提交。

## 常见问题排查

- **API 密钥无效**：请检查 `cnm config list` 确认密钥是否正确设置，或检查环境变量 `CNM_API_KEY`。
- **Git 未初始化**：`cnm` 必须在 Git 仓库根目录或子目录中运行。
- **二进制冲突**：如果 `cnm` 命令指向了错误的工具，请使用 `npx commit-now-myfriend` 或 `npm exec --package commit-now-myfriend cnm`。

## 发布安全建议

发布到 npm 前，请先运行 `pnpm typecheck`、`pnpm test -- --run`、`pnpm build` 和 `npm pack --dry-run`，确认包内容只包含 `dist`、`README.md`、`LICENSE` 和 `package.json` 等必要文件。

建议使用 npm 的 Trusted Publishing 或 provenance 发布能力，让 npm 包附带可验证的构建来源。若使用传统 npm token 发布，请启用 npm 账户 2FA，并优先选择“发布和设置修改都需要 2FA”的模式。不要在仓库、CI 日志、shell 历史或 README 示例中写入真实 npm token 或 AI API key。

## v1 非目标 (Non-goals)

以下特性在 v1 版本中暂不支持：
- 自动执行 `git push`。
- 自动执行 `git commit --amend` 或 `rebase`。
- 交互式 TUI 界面。
- 插件系统。
- AI 直接修改源代码。
- 支持 Azure OpenAI 或 Vertex AI。
- 支持 `--yes` 自动确认模式。

## 许可证

MIT
