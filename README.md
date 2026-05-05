<div align="center">

# commit-now-myfriend

**AI-assisted Git commit workflow from your terminal.**

[![npm version](https://img.shields.io/npm/v/commit-now-myfriend.svg)](https://www.npmjs.com/package/commit-now-myfriend)
[![Node.js Version](https://img.shields.io/node/v/commit-now-myfriend.svg)](https://nodejs.org)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)

`cnm` inspects your staged diff, generates a commit message with your AI provider, then lets you confirm, edit, regenerate, or cancel.

[Quick Start](#quick-start) · [Configuration](#configuration) · [CLI Reference](#cli-reference) · [Troubleshooting](#troubleshooting)

</div>

---

## Why use it?

Writing good commit messages is valuable, but it can interrupt the flow. `commit-now-myfriend` keeps the workflow in Git: stage your changes, run `cnm`, review the generated message, and commit only when you approve it.

Key features:

- Generates commit messages from staged Git diffs and changed file metadata.
- Learns repository style from recent commits when `promptStyle` is `auto`.
- Supports OpenAI Responses, Anthropic Messages, Google Gemini, and OpenAI-compatible APIs.
- Warns about likely secrets, detached HEAD, unstaged files, untracked files, and interrupted Git operations.
- Offers human-friendly prompts plus JSON output for scripts.

> [!IMPORTANT]
> Your staged diff is sent to the configured AI provider. Review the provider's data handling policy before using `cnm` on sensitive code.

## Quick start

### Prerequisites

- Node.js 20 or newer
- Git available on `PATH`
- An API key for one supported AI provider

### Install

```bash
npm install -g commit-now-myfriend
```

Or run it without a global install:

```bash
npx commit-now-myfriend
```

> [!NOTE]
> The package installs the `cnm` binary. If another tool already uses that name, run `npx commit-now-myfriend` or `npm exec --package commit-now-myfriend cnm`.

### First commit

```bash
cnm init
git add <files>
cnm
```

The default flow is:

1. Inspect the repository and staged changes.
2. Generate a commit message with the configured provider.
3. Show a preview with warnings, files, and the proposed message.
4. Let you confirm, edit, regenerate, or cancel.
5. Run `git commit` only after confirmation.

If no files are staged and you are in an interactive terminal, `cnm` can ask whether to stage all current changes. In JSON or non-interactive mode, stage files manually first.

## Providers

Configure a provider interactively:

```bash
cnm init
```

Or set it directly:

```bash
cnm init \
  --provider openai-responses \
  --model gpt-5.4-mini \
  --api-key <api-key>
```

Supported provider IDs:

| Provider | Use for | Required config | Default model |
| --- | --- | --- | --- |
| `openai-responses` | OpenAI Responses API | `apiKey` | `gpt-5.4-mini` |
| `anthropic-messages` | Anthropic Messages API | `apiKey` | `claude-haiku-4-5-20251001` |
| `google-gemini` | Google Gemini API | `apiKey` | `gemini-3-flash-preview` |
| `openai-compatible` | OpenAI-compatible or local gateways | `apiKey`, `baseURL` | `gpt-5.4-mini` |

For OpenAI-compatible providers, include a base URL:

```bash
cnm init \
  --provider openai-compatible \
  --base-url <base-url> \
  --model <model> \
  --api-key <api-key>
```

## Configuration

`cnm` merges configuration in this order, with later sources taking precedence:

1. User config: `~/.cnm/config.json`
2. Project config: `.cnmrc.json`
3. Environment variables
4. CLI flags

Project config is useful for shared defaults, but `apiKey` is ignored there by design.

```json
{
  "provider": "anthropic-messages",
  "model": "claude-haiku-4-5-20251001",
  "promptStyle": "conventional"
}
```

### Environment variables

Prefer environment variables for API keys:

```bash
export CNM_PROVIDER="openai-responses"
export CNM_MODEL="gpt-5.4-mini"
export CNM_API_KEY="<api-key>"
```

Available variables:

| Variable | Maps to |
| --- | --- |
| `CNM_PROVIDER` | `provider` |
| `CNM_MODEL` | `model` |
| `CNM_BASE_URL` | `baseURL` |
| `CNM_PROMPT_STYLE` | `promptStyle` |
| `CNM_CUSTOM_PROMPT` | `customPrompt` |
| `CNM_API_KEY` | `apiKey` |
| `CNM_HOME` | user config directory, default `~/.cnm` |

> [!WARNING]
> API keys set with `cnm init --api-key` or `cnm config set apiKey` are stored in plaintext in the user config file. `cnm` attempts to set `0600` permissions on Unix-like systems, but environment variables are still the safer default.

### Manage config

```bash
# Interactive config panel in a TTY
cnm config

# Print effective config
cnm config list
cnm config list --json

# Read or update one key
cnm config get provider
cnm config set promptStyle conventional
cnm config unset baseURL
```

Config keys: `provider`, `model`, `baseURL`, `promptStyle`, `customPrompt`, `apiKey`.

### Commit message styles

| Style | Behavior |
| --- | --- |
| `auto` | Infer style from recent non-merge commits; falls back to Conventional Commits. |
| `conventional` | Use `type(scope)?: subject` with an optional body. |
| `angular` | Use Angular-style commit subjects. |
| `google` | Use a short imperative subject and an optional explanatory body. |
| `atom` | Use a concise imperative subject, with a body only when useful. |
| `plain` | Use a concise natural-language message without strict prefixes. |
| `custom` | Use your `customPrompt` instructions. |

Examples:

```bash
cnm --prompt-style conventional
cnm --prompt-style custom --custom-prompt "Write concise Chinese commit messages"
```

## CLI reference

### `cnm`

Generate a commit message for staged changes and optionally create the commit.

```bash
cnm [options]
```

| Option | Description |
| --- | --- |
| `--dry-run` | Generate a preview without creating a commit. |
| `--json` | Emit machine-readable output; the root workflow returns a preview instead of committing. |
| `--provider <provider>` | Override the provider for this run. |
| `--model <model>` | Override the model for this run. |
| `--base-url <baseUrl>` | Override `baseURL` for this run. |
| `--prompt-style <promptStyle>` | Override the commit message style. |
| `--custom-prompt <customPrompt>` | Add custom instructions for this run. |

### `cnm init`

Create or update user configuration.

```bash
cnm init [options]
```

Options: `--provider`, `--model`, `--base-url`, `--prompt-style`, `--custom-prompt`, `--api-key`, `--dry-run`, `--json`.

### `cnm config`

Inspect and edit configuration.

```bash
cnm config
cnm config get [key] [--json]
cnm config list [--json]
cnm config set <key> <value>
cnm config unset <key>
```

### `cnm doctor`

Diagnose Node.js, Git, repository state, config files, permissions, and effective provider setup.

```bash
cnm doctor
cnm doctor --json
```

## Safety and privacy

- `cnm` never runs `git push`.
- `cnm` does not amend, rebase, or modify code.
- Commits are created with `git commit -F` only after interactive confirmation.
- Large staged diffs are truncated before being sent to the provider.
- Potential secrets in staged diffs are reported as warnings before generation.
- Project-level `apiKey` values are ignored to avoid committing shared secrets.

## Troubleshooting

### `cnm is not configured yet`

Run the setup wizard or provide configuration through environment variables:

```bash
cnm init
cnm doctor
```

### `No staged changes found`

Stage files first, especially in JSON or non-interactive mode:

```bash
git add <files>
cnm
```

### OpenAI-compatible provider requires `baseURL`

Set it in config, env, or a one-off command:

```bash
cnm config set provider openai-compatible
cnm config set baseURL <base-url>
```

### Git identity is missing

Configure Git before committing:

```bash
git config user.name "Your Name"
git config user.email "you@example.com"
```

### `cnm` launches a different command

Use the package name explicitly:

```bash
npx commit-now-myfriend
npm exec --package commit-now-myfriend cnm
```

## Development

```bash
pnpm install
pnpm dev -- --help
pnpm test
pnpm typecheck
pnpm build
npm pack --dry-run
```
