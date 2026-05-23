# Product Completion Plan

This repository is now being completed as the redesigned Go-native `cnm` product described by `CONTEXT.md`, `docs/adr/`, PRD #15, and issues #16-#29. The old TypeScript migration plan is no longer the source of truth.

## Current Product Shape

- `cnm`: Interactive Commit in the Full-screen TUI. Two-pane Charm layout with Commit Scope, live diff, AI activity spinner, Commit Plan review, message edit, and Interactive Repair entry.
- `cnm auto`: Autonomous Commit without step-by-step confirmation. Drives a provider-native Tool Call Loop that creates one or more local commits.
- `cnm init`: Full-screen Onboarding wizard (TTY) or flag-driven non-interactive init. API keys land in the system Secret Store by default.
- `cnm config`: Full-screen preferences panel in TTY (read, edit, unset, source indicators); legacy line-based subcommands (`get`, `list`, `set`, `unset`) still work for non-TTY scripts.
- `cnm doctor`: Bordered Charm dashboard summarising Git, repository, configuration sources, provider capability, optional probe, and issues. `--json` and non-TTY environments still get the existing machine-readable output.
- Removed product surfaces: `cnm split`, `cnm repair`, `cnm check`, and `cnm onboard`.

Commit splitting and conflict repair are not standalone commands. File-level Commit Split belongs inside Interactive Commit and Autonomous Commit. Interactive Repair belongs inside the Full-screen TUI and requires developer confirmation.

## Architectural Commitments

- Go 1.24+ is the runtime. The TUI is built on Bubble Tea, Bubbles (`viewport`, `spinner`), and Lip Gloss with a Charm-style purple/pink theme that degrades to bold ASCII when `NoColor` is set or terminal width drops below 80 columns.
- The TUI uses the alternate screen, mouse cell-motion, and `WindowSizeMsg`-driven layout so it never pollutes scrollback and resizes cleanly.
- npm is a distribution wrapper for the native binary, not the product runtime. `cnm --version` is injected at build time from `package.json`.
- Commit planning uses the Working Tree Commit model by default: staged, unstaged tracked, and untracked non-ignored files are considered unless flags narrow the Commit Scope.
- Provider interaction uses native Tool Call Loops adapted into the local Tool Call Runtime. The AI does not emit JSON plans for local code to interpret. Tool definitions and JSON schemas (with required arguments) are sent on every request, including continuations and reminder turns, so stateless provider proxies still know the tool surface.
- AI receives Domain Tools only. It does not receive shell access or raw Git command access.
- API keys are stored in the system Secret Store by default. Plaintext config requires explicit opt-in.
- `standingInstruction` is the only persistent prompt-like preference. Legacy `customPrompt` is intentionally rejected.
- Project checks stay outside cnm. Git hook failures are surfaced and may roll back commit transactions where safe, but cnm does not repair hook or project validation failures.

## Completion Source Of Truth

Use `docs/implementation-todo.md` for the live implementation checklist. Phase 14 records the completion audit items that must remain true before calling the product complete.

Before declaring completion, verify at minimum:

- `go test ./...`
- `go vet ./...`
- `make go-build`
- `npm pack --dry-run`
- `./dist/go/cnm --help`
- `./dist/go/cnm auto --help`
- `node scripts/cnm.js --version`
- Removed command smoke tests for `split`, `repair`, `check`, and `onboard`.
- TTY smoke test that `cnm`, `cnm config`, `cnm init`, and `cnm doctor` enter their respective Charm panels and exit cleanly without polluting the user's shell scrollback.
- Real provider smoke test: `cnm auto --dry-run` with a configured provider produces a plausible `commitPlan` via `create_commits`, and `cnm auto` creates the planned commits in a clean test repo.

## Historical Note

The previous TypeScript implementation is historical reference only and may live outside this main product path. Do not use the old staged-first or standalone split workflow as compatibility pressure for the redesigned Go product.
