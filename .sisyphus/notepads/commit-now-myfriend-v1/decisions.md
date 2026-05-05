## 2026-05-04 Task: startup-baseline
- Task 1 should create a fresh Node/TypeScript CLI scaffold while leaving `.sisyphus/` untouched and excluded from publish artifacts.

## 2026-05-04 Task: provider-adapters
- V1 provider families are exactly OpenAI-compatible Chat Completions, OpenAI Responses, Anthropic Messages, and Google Gemini API; Google provider intentionally uses Gemini API key auth only and does not pass Vertex options.
- The sanitizer enforces Conventional Commit subjects and the configured maximum subject length, even when a custom prompt is supplied, to preserve downstream safety gates.

## 2026-05-04 Task: config-onboarding-redaction
- `cnm config get` without a key returns the full effective config so the v1 QA flow can use `cnm config get --json` directly.
- Project config secrets are not fatal: the CLI warns on stderr, ignores only the secret field, and continues using remaining non-secret project values.
- Mutating config commands honor existing global `--dry-run/--json` shell behavior instead of introducing a separate config-only contract.

## 2026-05-04 Task: main-commit-workflow
- Task 6 keeps the root `--json` path explicitly safe and non-committing by returning a structured preview-only result; the full machine-readable contract is deferred to Task 8 instead of partially committing in non-interactive mode.
- Edit/regenerate actions are implemented through prompt and workflow seams now, but validation/polish remains intentionally lightweight so Task 7 can refine the UX without rewriting the state machine.

## 2026-05-04 Task: doctor-diagnostics
- `cnm doctor` exits with the existing error code contract: it prints the full diagnostic report or JSON payload first, then returns exit code `1` only when error-severity issues exist; warnings alone do not fail the command.
- Doctor treats `not_git_repository` as a warning in its own report even though git inspection marks it blocking for commit workflows, preserving useful diagnostics outside repos without crashing.
- `provider_config_missing` is emitted when doctor only has built-in provider/model defaults and no API key, so empty first-run environments surface setup guidance even though effective defaults still exist.

## 2026-05-04 Task: dry-run-json-contracts
- Keep the machine surface preview-only in v1: --json may inspect staged changes and generate a message, but committed must remain false and prompt text must stay out of stdout.
- Strip internal workflow-only fields before JSON emission so stdout stays a stable external contract while human mode keeps richer internal control flow.
