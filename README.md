# cnm — Git Commit Message Generator

一个轻量级 CLI 工具，调用 AI API 从 git diff 生成 commit message。

## 安装

```bash
# 下载二进制（macOS arm64）
curl -L https://github.com/ByteTrue/commit-now-myfriend/releases/latest/download/cnm-darwin-arm64 -o /usr/local/bin/cnm
chmod +x /usr/local/bin/cnm
```

## 使用

```bash
cnm setup      # 一次性配置 API key 和偏好
cnm            # 生成 commit message，确认后提交
cnm --yes      # 跳过确认直接提交
```

### 风格

```bash
cnm --style conventional     # feat(auth): add JWT validation
cnm --style angular          # feat(auth): add JWT validation
cnm --style google           # feat: add JWT validation
cnm --style atom             # :sparkles: feat: add JWT validation
cnm --style plain            # add JWT validation
cnm --style custom           # 配合 --prompt 使用
```

### 自定义 prompt

```bash
cnm --prompt "用中文写commit message"
cnm --prompt "Include issue number in the subject"
```

### API Provider

```bash
cnm --provider anthropic-message --model claude-haiku-3-5
cnm --provider openai-response --model gpt-4o
```

## 配置

`cnm setup` 交互式配置，保存到 `~/.config/cnm/config.json`。

也可通过环境变量：
```bash
export CNM_API_KEY=sk-xxx
export CNM_STYLE=conventional
export CNM_MODEL=gpt-4o-mini
```

CLI flags 优先级最高，可临时覆盖配置。

## License

MIT
