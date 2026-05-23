<div align="center">

# commit-now-myfriend

**AI-assisted local commit planning and execution from your terminal.**

[![npm version](https://img.shields.io/npm/v/commit-now-myfriend.svg)](https://www.npmjs.com/package/commit-now-myfriend)
[![Go Version](https://img.shields.io/badge/Go-1.24+-00ADD8?logo=go)](https://go.dev)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)

`cnm` helps turn your current working tree into one or more local Git commits. It can run as a Full-screen TUI for step-by-step review, or as `cnm auto` for fast Autonomous Commit.

[Quick Start](#quick-start) | [Configuration](#configuration) | [CLI Reference](#cli-reference) | [Safety And Privacy](#safety-and-privacy) | [Migration](#migration)

</div>

---

## What it does

`commit-now-myfriend` is a lightweight Go-native CLI for developers who want AI help with commit planning, grouping, and messages without leaving Git.

The redesigned product has two primary commit flows:

- `cnm`: Interactive Commit in a Full-screen TUI. A two-pane Charm-style layout shows the working-tree Commit Scope on the left and the live diff / AI activity / Commit Plan preview on the right. You review the scope, give optional Agent Instructions, watch the AI plan with a spinner, then accept or edit before any side effect.
- `cnm auto`: Autonomous Commit. The tool drives a provider-native Tool Call Loop that ends in one or more local commits. It never pushes.

The TUI is full-screen (alt-screen, mouse, resize-aware) and degrades gracefully to a plain text preview in non-TTY environments (CI, pipes, `--json`).

Key behavior:

- Plans from the working tree by default: staged files, unstaged tracked files, and untracked non-ignored files.
- Supports `--staged`, `--no-untracked`, `--diff-only`, and Git pathspecs after `--` to narrow the Commit Scope.
- Uses provider-native tool calls through a local Tool Call Runtime rather than trusting arbitrary JSON plans.
- Sends every tool definition on every request so stateless provider proxies still know the tool surface.
- Splits clearly independent file groups into multiple local commits when safe.
- Blocks suspected secrets in the selected Commit Scope instead of silently skipping them.
- Runs Git hooks by default and supports explicit `--no-verify`.
- Keeps project checks outside cnm; use your own Git hooks or workflow runner.

In TTY mode `cnm config`, `cnm init`, and `cnm doctor` all render as bordered Charm panels with keyboard navigation; in non-TTY they fall back to the original line-based output so existing scripts keep working.

## Quick start

### Prerequisites

- Git available on `PATH`
- An API key for one supported provider
- Node.js 20 or newer only when installing through npm

### Install

#### npm wrapper install

```bash
npm install -g commit-now-myfriend
```

The npm package is a thin installer and launcher for the native Go binary.

#### Direct binary install

Download a release archive for your platform and place `cnm` on your `PATH`.

Published targets:

- macOS amd64 / arm64
- Linux amd64 / arm64
- Windows amd64 / arm64

#### One-off execution

```bash
npx commit-now-myfriend
```

The package installs the `cnm` command. If another tool already uses that name, run `npx commit-now-myfriend` or `npm exec --package commit-now-myfriend cnm`.

### First run

```bash
cnm init
cnm
```

The default `cnm` flow:

1. Inspects the Commit Scope from the current working tree.
2. Shows the selected files and AI Exposure Summary in the Full-screen TUI.
3. Lets you provide an Agent Instruction for the active workflow.
4. Produces a commit plan and messages.
5. Requires confirmation before creating commits.

For a fast non-interactive local commit run:

```bash
cnm auto
```

Preview the plan without side effects:

```bash
cnm auto --dry-run
cnm auto --dry-run --json
```

Limit the Commit Scope:

```bash
cnm --staged
cnm auto --no-untracked
cnm auto -- src docs/guide.md
```

## Providers

Configure a provider interactively:

```bash
cnm init
```

Or set it directly:

```bash
cnm init \
  --provider openai-responses \
  --model gpt-5-mini \
  --api-key <api-key>
```

Supported provider IDs:

| Provider | Protocol | Required config | Default model |
| --- | --- | --- | --- |
| `openai-responses` | OpenAI Responses native tool calls | `apiKey` | `gpt-5-mini` |
| `openai-compatible` | OpenAI-compatible chat completions tool calls | `apiKey`, `baseURL` | `gpt-5-mini` |
| `anthropic-messages` | Anthropic Messages native tool calls | `apiKey` | `claude-sonnet-4-20250514` |
| `google-gemini` | Gemini function calls | `apiKey` | `gemini-2.5-flash` |

For OpenAI-compatible providers, include a base URL:

```bash
cnm init \
  --provider openai-compatible \
  --base-url <base-url> \
  --model <model> \
  --api-key <api-key>
```

Use `cnm doctor --json` to inspect provider capability metadata. Use `cnm doctor --probe-provider` only when you explicitly want to send a fixed non-repository probe to the configured provider.

## Configuration

cnm separates shared repository preferences from private developer preferences.

- Shared Preference: repository-level config such as prompt style, message language, standing instructions, and provider recommendations.
- Private Preference: developer-local config such as provider, model, base URL, API key, theme, and personal standing instructions.

Configuration sources:

1. User config: `~/.cnm/config.json`
2. Project config: `.cnmrc.json`
3. Environment variables
4. CLI flags

Project config can recommend a provider or model, but it must not force private provider credentials. Project-level `apiKey`, `provider`, `model`, and `baseURL` values are ignored or converted to recommendations where safe.

Example project config:

```json
{
  "promptStyle": "conventional",
  "messageLanguage": "en",
  "standingInstruction": "Keep commit subjects concise.",
  "recommendedProvider": "openai-responses",
  "recommendedModel": "gpt-5-mini"
}
```

Environment variables:

| Variable | Maps to |
| --- | --- |
| `CNM_PROVIDER` | private provider |
| `CNM_MODEL` | private model |
| `CNM_BASE_URL` | private base URL |
| `CNM_PROMPT_STYLE` | prompt style |
| `CNM_MESSAGE_LANGUAGE` | message language |
| `CNM_STANDING_INSTRUCTION` | standing instruction |
| `CNM_API_KEY` | API key for this process |
| `CNM_HOME` | user config directory, default `~/.cnm` |

API keys are resolved from environment variables, Secret Store, or explicit plaintext opt-in. Environment variables are useful for temporary sessions. Secret Store is the default persistent credential target. Plaintext user config is supported only as an explicit fallback and is redacted in normal output.

Manage config:

```bash
cnm config
cnm config list
cnm config list --json
cnm config get provider
cnm config set promptStyle conventional
cnm config set standingInstruction "Prefer concise subjects."
cnm config unset baseURL
```

Config keys include `provider`, `model`, `baseURL`, `promptStyle`, `messageLanguage`, `standingInstruction`, `recommendedProvider`, `recommendedModel`, and `apiKey`.

## CLI Reference

### `cnm`

Start Interactive Commit in the Full-screen TUI.

```bash
cnm [flags] [-- <pathspec...>]
```

Flags:

| Flag | Description |
| --- | --- |
| `--json` | Emit a TUI preview Machine Output Contract instead of full-screen rendering. |
| `--staged` | Restrict Commit Scope to staged changes. |
| `--no-untracked` | Exclude untracked files from the Commit Scope. |
| `--diff-only` | Disable bounded working-tree file reads. |
| `--no-verify` | Pass `--no-verify` to `git commit` after confirmation. |
| `--provider <provider>` | Override provider for this run. |
| `--model <model>` | Override model for this run. |
| `--base-url <url>` | Override provider base URL for this run. |
| `--prompt-style <style>` | Override commit message style for this run. |
| `--message-language <language>` | Override generated message language. |
| `--standing-instruction <text>` | Add standing instruction for this run. |

`cnm` requires confirmation before creating commits. In a non-TTY environment it prints a no-side-effect preview.

### `cnm auto`

Run Autonomous Commit without step-by-step confirmation.

```bash
cnm auto [flags] [-- <pathspec...>]
```

Flags:

| Flag | Description |
| --- | --- |
| `--dry-run` | Show a Commit Plan Preview without creating commits. |
| `--json` | Emit the Machine Output Contract. |
| `--tui`, `-i` | Hand off conflicts to the Full-screen TUI. |
| `--staged` | Restrict Commit Scope to staged changes. |
| `--no-untracked` | Exclude untracked files. |
| `--diff-only` | Use diff-only Context Policy. |
| `--no-verify` | Pass `--no-verify` to `git commit`. |
| `--verbose`, `-v` | Show detailed run output. |

`cnm auto` can create multiple local commits when the selected files clearly belong to independent groups. It fails non-interactively on conflicts. Conflict repair is only available through the Full-screen TUI.

### `cnm init`

Run Onboarding and configure preferences.

```bash
cnm init [flags]
```

Common flags: `--provider`, `--model`, `--base-url`, `--prompt-style`, `--message-language`, `--standing-instruction`, `--api-key`, `--dry-run`, `--json`.

### `cnm config`

Inspect and edit preferences.

```bash
cnm config
cnm config get <key> [--json]
cnm config list [--json]
cnm config set <key> <value>
cnm config unset <key>
```

### `cnm doctor`

Diagnose local setup.

```bash
cnm doctor
cnm doctor --json
cnm doctor --probe-provider
```

Default doctor output is local-only and read-only. It reports Git availability, repository status, config paths, credential source, effective provider/model, and Provider Capability metadata. `--probe-provider` sends only fixed non-repository content to the configured provider.

Removed commands:

- `cnm split`: split planning now belongs inside `cnm` and `cnm auto`.
- `cnm repair`: conflict repair now belongs inside the Full-screen TUI.
- `cnm check`: project checks belong in user workflows or Git hooks.
- `cnm onboard`: use `cnm` or `cnm init`.

## Safety And Privacy

cnm is local-first, but selected repository context can be sent to your configured AI provider during AI-assisted flows.

- No remote telemetry is collected by cnm in the first version.
- cnm never runs `git push`.
- cnm does not run arbitrary shell commands as AI tools.
- AI receives validated Domain Tools exposed by the Tool Call Runtime.
- Git hooks run by default during commit creation.
- Hook failures are surfaced; cnm does not repair hook or project validation failures.
- `--no-verify` is an explicit developer override.
- Secret Blocker stops the selected flow when suspected credentials are detected.
- Context Policy controls whether bounded file reads are allowed or diff-only mode is used.
- Diff Budget and Read Budget bound how much repository content can be exposed.
- AI Exposure Summary reports selected file count, visible files, budget use, opaque changes, and preference sources.
- Opaque Changes such as binary files are described by metadata only.

Debug logging should remain local, explicit, and conservative. Avoid logging secrets, full diffs, prompts, provider responses, or API keys.

## Migration

Previous versions centered on staged diffs and standalone split/repair commands. The redesigned CLI treats the working tree as the default Commit Scope and makes split/repair part of the primary flows.

| Old habit | New flow |
| --- | --- |
| `git add <files>; cnm` | `cnm` or `cnm auto` from the working tree |
| staged-only commit | `cnm --staged` or `cnm auto --staged` |
| `cnm split` | `cnm` or `cnm auto`; split planning is automatic when safe |
| `cnm repair` | `cnm auto --tui` or `cnm` conflict context in the Full-screen TUI |
| `cnm check` | Git hooks or your own project workflow |
| `cnm onboard` | `cnm init` or first-run Onboarding through `cnm` |

If you need exact control over included files, pass pathspecs after `--`:

```bash
cnm auto -- src docs/guide.md
```

## Troubleshooting

### Configuration is missing

Run setup or inspect the current state:

```bash
cnm init
cnm doctor
cnm doctor --json
```

### No selected changes

By default cnm includes staged, unstaged tracked, and untracked non-ignored files. If no changes are selected, check pathspecs and flags:

```bash
cnm auto --dry-run --json
cnm auto --staged
cnm auto -- src
```

### Conflicts are present

`cnm auto` fails on conflicts by default:

```bash
cnm auto --tui
```

Conflict repair requires the Full-screen TUI and developer confirmation.

### OpenAI-compatible provider requires `baseURL`

```bash
cnm config set provider openai-compatible
cnm config set baseURL <base-url>
```

### Git identity is missing

```bash
git config user.name "Your Name"
git config user.email "you@example.com"
```

### `cnm` launches a different command

```bash
npx commit-now-myfriend
npm exec --package commit-now-myfriend cnm
```

## Development

```bash
npm run dev
npm test
npm run fmt
npm run build
npm run build:release-local
npm pack --dry-run
```

For a release snapshot:

```bash
make go-release-snapshot
```

See `docs/distribution.md` for release packaging details.

## Archive branch

The legacy TypeScript implementation is preserved outside the new Go product path. Use the archive branch only for historical reference.

```bash
git switch archive/typescript-runtime
```
