# commit-now-myfriend v1 Product CLI

## TL;DR
> **Summary**: Build `commit-now-myfriend` as an npm-published, product-grade TypeScript CLI exposing the `cnm` command for AI-assisted git commits with safe preview-and-confirm workflow. v1 includes onboarding, user/project config, doctor diagnostics, four provider adapters, dry-run/JSON surfaces, tests, docs, and publish readiness.
> **Deliverables**:
> - Node.js 20+ TypeScript CLI package with `bin: { "cnm": "dist/index.js" }`
> - `cnm` workflow: git inspection → staged diff → AI Conventional Commit message → preview → confirm/edit/regenerate/cancel → `git commit`
> - `cnm init`, `cnm config`, `cnm doctor`, `--dry-run`, `--json`
> - Provider adapters for OpenAI-compatible Chat Completions, OpenAI Responses, Anthropic Messages, Google Gemini API
> - Vitest unit/integration tests using temp git repos and mocked providers
> - README, publishing checks, `npm pack --dry-run`, npm security/provenance guidance
> **Effort**: Large
> **Parallel**: YES - 4 waves
> **Critical Path**: Task 1 → Tasks 2/3/4/5 → Tasks 6/7/8/9 → Tasks 10/11 → Final Verification

## Context
### Original Request
- User wants a simple developer CLI named `commit-now-myfriend`, short command `cnm`, where running `cnm` in a git repository uses AI to automatically complete a commit without opening heavy AI tools.

### Interview Summary
- Product is for other developers, not only personal/internal use.
- v1 should be high-completion, including onboarding, config, doctor-style diagnostics, and publish readiness.
- CLI classification: Human-primary Workflow/Orchestration CLI; brief interactive confirmation; Config-Stateful; High Side-Effect; secondary machine surfaces only.
- Default commit behavior: show diff summary and AI-generated message, then commit only after explicit confirmation.
- Submit scope: use staged changes first; if no staged changes, show changed files and ask whether to stage current changes.
- No real-commit `--yes` in v1.
- Default commit message style: Conventional Commits; configurable custom prompt.
- Provider support: OpenAI-compatible Chat Completions, OpenAI Responses, Anthropic Messages, Google Gemini API only; user selects one default provider.
- Config: onboarding/config commands store configuration in user directory `~/.cnm`; project-level config is supported but not auto-created.
- Tech stack: Node.js 20+, TypeScript, pnpm, Vitest, tsup, commander, @clack/prompts, conf, execa, official provider SDKs.
- Publish scope: include npm pack checks, README, 2FA/provenance recommendations, and bin conflict troubleshooting.

### Metis Review (gaps addressed)
- Fixed `--json` contract: stdout is pure JSON and v1 never commits in `--json` mode.
- Fixed exit-code contract: success/dry-run/no-change = 0; user cancel = 130; errors = 1.
- Fixed secret/config rules: user config may contain API keys but must warn, redact, prefer env overrides, chmod `0600` where possible; project config must reject/ignore secrets.
- Fixed Git safety rules: no push/amend/rebase/no-verify; no auto-stage unless TTY user explicitly confirms; block merge/rebase/cherry-pick in progress.
- Fixed testing rules: all config tests use temporary `CNM_HOME`; provider tests must mock network; git tests use temporary repositories.

## Work Objectives
### Core Objective
Create a safe, polished npm CLI that lets developers run `cnm` inside a git repo to generate and commit an AI-produced Conventional Commit message while preserving explicit human control over git side effects.

### Deliverables
- Package scaffold with TypeScript, pnpm, Vitest, tsup, lint/typecheck/build/test scripts, GitHub Actions CI, and npm publish readiness.
- CLI command shell with help output and stable command/flag semantics.
- Config and onboarding system using `~/.cnm` plus optional project config.
- Provider adapter layer with four concrete API-family adapters.
- Git service and safety checks around staged/unstaged changes, repo state, diff size, binary files, secrets, hooks, and commit execution.
- Interactive commit workflow with preview, confirm, edit, regenerate, cancel.
- Dry-run and JSON machine surfaces with stable output shape.
- Doctor diagnostics command.
- README and publish QA.

### Definition of Done (verifiable conditions with commands)
- `pnpm install` exits `0`.
- `pnpm typecheck` exits `0`.
- `pnpm test -- --run` exits `0`.
- `pnpm build` exits `0`.
- `node dist/index.js --help` exits `0` and includes `Usage: cnm`, `init`, `config`, `doctor`, `--dry-run`, `--json`.
- `npm pack --dry-run` exits `0` and excludes `.sisyphus/`, tests, real config, and secrets.
- Integration tests prove real commits only occur after explicit confirmation.

### Must Have
- `cnm`, `cnm init`, `cnm config`, `cnm doctor`.
- `--dry-run`, `--json`; no `--yes`.
- Internal provider contract: `generateCommitMessage(input): Promise<CommitMessageResult>`.
- Config precedence: CLI flags > env vars > project config > user config > defaults.
- User config home override for tests: `CNM_HOME`.
- Redacted config output.
- Project config rejects or ignores secrets with warning/error.
- All git subprocess calls use `execa` argument arrays, never shell string interpolation.
- Commit messages are passed safely, preferably via temp file or stdin; never shell-escaped string concatenation.

### Must NOT Have (guardrails, AI slop patterns, scope boundaries)
- No push, amend, squash, rebase, reset, stash, branch creation, PR creation, issue linkage, changelog generation, TUI, daemon, session system, plugin system, local model manager, Vertex AI, Azure-specific API, AI code edits, or AI file selection.
- No `--no-verify`; never bypass user git hooks.
- No true-commit `--yes`.
- No secret storage in project config.
- No real provider network calls in tests.
- No writing to real `~/.cnm` in tests.
- No prompts or spinners in stdout when `--json` is active.

## Verification Strategy
> ZERO HUMAN INTERVENTION - all verification is agent-executed.
- Test decision: tests-after per task, using Vitest with mocked provider adapters and temp git repo integration tests.
- QA policy: Every task has agent-executed scenarios.
- Evidence: `.sisyphus/evidence/task-{N}-{slug}.{ext}`

## Execution Strategy
### Parallel Execution Waves
> Target: 5-8 tasks per wave. <3 per wave (except final) = under-splitting.
> Extract shared dependencies as Wave-1 tasks for max parallelism.

Wave 1: Task 1 foundation only; blocks all implementation.
Wave 2: Tasks 2, 3, 4, 5 can run after scaffold; command shell, config, providers, git service are independent modules.
Wave 3: Tasks 6, 7, 8, 9 can run after their dependencies; workflow, JSON/dry-run, doctor, and security polish.
Wave 4: Tasks 10, 11 finalize docs, publish QA, and end-to-end hardening.

### Dependency Matrix (full, all tasks)
- Task 1 blocks Tasks 2-11.
- Task 2 blocks Tasks 6, 8, 9, 10, 11.
- Task 3 blocks Tasks 6, 8, 9, 10, 11.
- Task 4 blocks Tasks 6, 8, 9, 10.
- Task 5 blocks Tasks 6, 7, 8, 9, 10.
- Task 6 blocks Tasks 7, 8, 10, 11.
- Task 7 blocks Tasks 10, 11.
- Task 8 blocks Tasks 10, 11.
- Task 9 blocks Tasks 10, 11.
- Task 10 blocks Task 11.
- Task 11 blocks Final Verification.

### Agent Dispatch Summary (wave → task count → categories)
- Wave 1 → 1 task → unspecified-high
- Wave 2 → 4 tasks → quick, unspecified-high, deep
- Wave 3 → 4 tasks → unspecified-high, deep
- Wave 4 → 2 tasks → writing, unspecified-high
- Final → 4 review agents → oracle, unspecified-high, unspecified-high, deep

## TODOs
> Implementation + Test = ONE task. Never separate.
> EVERY task MUST have: Agent Profile + Parallelization + QA Scenarios.

- [x] 1. Initialize product-grade TypeScript CLI package

  **What to do**: Create the npm package foundation from the empty repo: `package.json`, `pnpm-lock.yaml`, `tsconfig.json`, `vitest.config.ts`, `tsup.config.ts`, `.gitignore`, `.npmignore` or `files` whitelist, `src/index.ts`, `src/cli.ts`, `tests/`, `.github/workflows/ci.yml`, and minimal README placeholder. Configure Node.js `>=20`, ESM output, `bin: { "cnm": "dist/index.js" }`, shebang in built CLI, and scripts: `build`, `test`, `typecheck`, `lint`, `dev`, `pack:dry-run`. Install dependencies: `commander`, `@clack/prompts`, `conf`, `execa`, `openai`, `@anthropic-ai/sdk`, `@google/genai`; dev dependencies: `typescript`, `vitest`, `tsup`, `tsx`, `@types/node`, lint tooling chosen consistently with the repo.
  **Must NOT do**: Do not implement provider calls or git commit workflow in this task. Do not add publishing tokens or secrets. Do not create real user config.

  **Recommended Agent Profile**:
  - Category: `unspecified-high` - Reason: bootstraps the whole repo and CI/publish foundation.
  - Skills: [] - No specialized skill required beyond Node/TS packaging.
  - Omitted: [`frontend-design`, `playwright-cli`] - No browser UI.

  **Parallelization**: Can Parallel: NO | Wave 1 | Blocks: Tasks 2-11 | Blocked By: none

  **References** (executor has NO interview context - be exhaustive):
  - Requirement: `.sisyphus/drafts/commit-now-myfriend-cli.md:19-29` - product distribution, command surface, tech stack, publish closure.
  - Research: `.sisyphus/drafts/commit-now-myfriend-cli.md:31-39` - repo is empty; recommended dependencies; npm package/bin finding.
  - External: `https://github.com/tj/commander.js/blob/master/Readme.md` - commander command/help patterns.
  - External: `https://github.com/egoist/tsup/blob/main/docs/README.md` - TypeScript package bundling.
  - External: `https://vitest.dev/guide/` - Vitest test runner.

  **Acceptance Criteria** (agent-executable only):
  - [ ] `pnpm install` exits `0`.
  - [ ] `pnpm typecheck` exits `0`.
  - [ ] `pnpm test -- --run` exits `0` with at least one smoke test.
  - [ ] `pnpm build` exits `0` and creates `dist/index.js` containing a valid Node shebang or preserving executable entry behavior.
  - [ ] `node dist/index.js --help` exits `0` and prints `Usage: cnm`.
  - [ ] `npm pack --dry-run` exits `0` and output does not include `.sisyphus/` or test fixtures.

  **QA Scenarios** (MANDATORY - task incomplete without these):
  ```
  Scenario: Package builds runnable CLI
    Tool: Bash
    Steps: Run `pnpm install && pnpm build && node dist/index.js --help` from repo root.
    Expected: Exit code 0; stdout contains `Usage: cnm`.
    Evidence: .sisyphus/evidence/task-1-package-help.txt

  Scenario: Publish contents exclude planning/secrets
    Tool: Bash
    Steps: Run `npm pack --dry-run`.
    Expected: Exit code 0; output includes `dist/index.js`, `package.json`, `README.md`; output excludes `.sisyphus/`, `.env`, `.cnm`, and test fixture secrets.
    Evidence: .sisyphus/evidence/task-1-pack-dry-run.txt
  ```

  **Commit**: YES | Message: `chore(package): initialize cnm typescript cli` | Files: [`package.json`, `pnpm-lock.yaml`, `tsconfig.json`, `vitest.config.ts`, `tsup.config.ts`, `src/**`, `tests/**`, `.github/workflows/ci.yml`, `.gitignore`, `README.md`]

- [x] 2. Implement CLI command shell and output routing

  **What to do**: Build the commander-based CLI shell for `cnm`, `cnm init`, `cnm config`, `cnm doctor`, global `--dry-run`, `--json`, `--version`, `--help`, and shared exit handling. Implement stdout/stderr routing rules: human output may use prompts/spinners on TTY; `--json` writes one JSON object to stdout and no prompt/spinner text to stdout. Define exit codes: success/dry-run/no-change = `0`; user cancel/Ctrl-C = `130`; operational/config/provider/git errors = `1`.
  **Must NOT do**: Do not implement real provider calls or real git commit yet. Do not add `--yes`.

  **Recommended Agent Profile**:
  - Category: `unspecified-high` - Reason: establishes public CLI contract and error behavior.
  - Skills: [`cli-design-framework`] - Needed to preserve human-primary CLI surface and secondary machine contract.
  - Omitted: [`frontend-design`] - No UI styling beyond terminal prompts.

  **Parallelization**: Can Parallel: YES | Wave 2 | Blocks: Tasks 6, 8, 9, 10, 11 | Blocked By: Task 1

  **References**:
  - Decision: `.sisyphus/drafts/commit-now-myfriend-cli.md:14-20` - CLI classification and command surface.
  - Decision: `.sisyphus/drafts/commit-now-myfriend-cli.md:25` - no true-commit `--yes`.
  - Metis: Context section in this plan - `--json` never commits and stdout must be pure JSON.
  - External: `https://github.com/tj/commander.js/blob/master/Readme.md` - commands/options/help.

  **Acceptance Criteria**:
  - [ ] `node dist/index.js --help` contains `init`, `config`, `doctor`, `--dry-run`, `--json`; does not mention `--yes`.
  - [ ] `node dist/index.js --version` exits `0` and matches `package.json` version.
  - [ ] `node dist/index.js --json --help` does not corrupt help output contract; if unsupported, exits `1` with clear error.
  - [ ] Unit tests cover exit code mapping and JSON stdout/human stderr separation.

  **QA Scenarios**:
  ```
  Scenario: Help is discoverable for humans
    Tool: Bash
    Steps: Run `pnpm build && node dist/index.js --help`.
    Expected: Exit code 0; stdout contains command examples for `cnm`, `cnm init`, `cnm config`, `cnm doctor`; stdout does not contain `--yes`.
    Evidence: .sisyphus/evidence/task-2-help.txt

  Scenario: Unknown command fails safely
    Tool: Bash
    Steps: Run `node dist/index.js nope`.
    Expected: Exit code 1; stderr contains a clear unknown-command error and suggests `--help`; no files are modified.
    Evidence: .sisyphus/evidence/task-2-unknown-command.txt
  ```

  **Commit**: YES | Message: `feat(cli): add cnm command shell` | Files: [`src/cli.ts`, `src/index.ts`, `src/output/**`, `tests/cli/**`]

- [x] 3. Implement user/project config, onboarding, and redaction

  **What to do**: Implement config service and commands. User config lives under `~/.cnm` by default and must be overrideable with `CNM_HOME` for tests. Store provider type, default model, baseURL where applicable, custom prompt/style, and user-supplied API key fields. Show an explicit plaintext-key warning during `cnm init`/`cnm config set`. Attempt `0600` permissions on config files. Support project-level config (e.g. `.cnmrc.json`) for non-secret settings only; do not auto-create it during `cnm init`. Implement precedence: CLI flags > env vars > project config > user config > defaults. Redact secrets in all `config get/list/doctor` output.
  **Must NOT do**: Do not write to real `~/.cnm` in tests. Do not allow project config to contain API keys. Do not print full API keys.

  **Recommended Agent Profile**:
  - Category: `unspecified-high` - Reason: config/security is a core product and safety surface.
  - Skills: [] - Standard Node/TS implementation.
  - Omitted: [`cli-design-framework`] - CLI contract already established by Task 2.

  **Parallelization**: Can Parallel: YES | Wave 2 | Blocks: Tasks 4, 6, 8, 9, 10, 11 | Blocked By: Task 1

  **References**:
  - Requirement: `.sisyphus/drafts/commit-now-myfriend-cli.md:18,24,26` - config model, storage, project config.
  - Research: `.sisyphus/drafts/commit-now-myfriend-cli.md:37` - key storage warning and redaction guardrail.
  - External: `https://www.npmjs.com/package/conf` - user config storage library.

  **Acceptance Criteria**:
  - [ ] `CNM_HOME=$(mktemp -d) node dist/index.js config set provider openai-responses` writes under that temp directory only.
  - [ ] Config read precedence is covered by unit tests for flags, env, project config, user config, defaults.
  - [ ] Full API key never appears in stdout/stderr snapshots.
  - [ ] Project config containing `apiKey` causes explicit error or warning and is not used.
  - [ ] Config file permission test asserts `0600` on supported platforms or documents skip reason.

  **QA Scenarios**:
  ```
  Scenario: User config writes to CNM_HOME and redacts key
    Tool: Bash
    Steps: Set `CNM_HOME` to a temp directory; run config commands to set provider/model/apiKey; run `cnm config get --json`.
    Expected: Config file exists only under temp CNM_HOME; stdout JSON contains provider/model; stdout does not contain full API key; key appears masked.
    Evidence: .sisyphus/evidence/task-3-config-redaction.json

  Scenario: Project config cannot store secrets
    Tool: Bash
    Steps: Create temp repo with `.cnmrc.json` containing `apiKey`; run `cnm config get` with temp CNM_HOME.
    Expected: Exit code 1 or warning per implementation; full key not printed; provider resolution does not use project key.
    Evidence: .sisyphus/evidence/task-3-project-secret-block.txt
  ```

  **Commit**: YES | Message: `feat(config): add cnm onboarding and config storage` | Files: [`src/config/**`, `src/commands/init.ts`, `src/commands/config.ts`, `tests/config/**`]

- [x] 4. Implement normalized AI provider adapters

  **What to do**: Create provider domain types and adapter registry around a single internal contract: `generateCommitMessage(input): Promise<CommitMessageResult>`. Implement adapters for `openai-compatible` using Chat Completions and configurable `baseURL`, `openai-responses` using OpenAI Responses API, `anthropic-messages` using Anthropic Messages, and `google-gemini` using `@google/genai`. Normalize inputs: staged file list, diff text, repo metadata, message style, custom prompt, max subject length. Sanitize outputs: trim, remove markdown fences, remove explanatory paragraphs, reject empty messages. Default prompt must produce Conventional Commit subject, recommended ≤72 chars, optional body.
  **Must NOT do**: Do not expose provider-specific request/response shapes outside adapters. Do not call real networks in tests. Do not implement Vertex AI, Azure, plugin providers, or streaming UI.

  **Recommended Agent Profile**:
  - Category: `deep` - Reason: multiple provider APIs need clean abstraction and robust error normalization.
  - Skills: [] - Use official SDK docs and existing config contract.
  - Omitted: [`cli-design-framework`] - Product-level CLI shape already decided.

  **Parallelization**: Can Parallel: YES | Wave 2 | Blocks: Tasks 6, 8, 9, 10 | Blocked By: Task 1

  **References**:
  - Requirement: `.sisyphus/drafts/commit-now-myfriend-cli.md:17,23,27` - provider families, Conventional default, Gemini-only Google scope.
  - Research: `.sisyphus/drafts/commit-now-myfriend-cli.md:36,38` - `generateCommitMessage` abstraction and SDK recommendations.
  - External: `https://developers.openai.com/api/docs/guides/migrate-to-responses` - OpenAI Responses guidance.
  - External: `https://github.com/anthropics/anthropic-sdk-typescript` - Anthropic TypeScript SDK.
  - External: `https://ai.google.dev/gemini-api/docs/libraries` - Google Gemini SDK libraries.

  **Acceptance Criteria**:
  - [ ] Unit tests cover all four adapters with mocked SDK/client responses.
  - [ ] Adapter tests prove no real network is required.
  - [ ] Markdown fenced output like ```` ```text\nfeat: add x\n``` ```` sanitizes to `feat: add x`.
  - [ ] Empty or explanatory-only provider output returns a typed provider error.
  - [ ] OpenAI-compatible adapter supports custom `baseURL` and standard bearer API key config.

  **QA Scenarios**:
  ```
  Scenario: All provider adapters return normalized commit messages
    Tool: Bash
    Steps: Run `pnpm test -- --run tests/providers`.
    Expected: Exit code 0; tests cover openai-compatible, openai-responses, anthropic-messages, google-gemini; no live API calls occur.
    Evidence: .sisyphus/evidence/task-4-provider-tests.txt

  Scenario: Malformed AI output fails safely
    Tool: Bash
    Steps: Run provider unit test fixture where adapter returns empty/explanatory text.
    Expected: Test asserts typed error; workflow would not call git commit with invalid message.
    Evidence: .sisyphus/evidence/task-4-provider-malformed.txt
  ```

  **Commit**: YES | Message: `feat(providers): add ai commit message adapters` | Files: [`src/providers/**`, `src/prompt/**`, `tests/providers/**`]

- [x] 5. Implement Git service, diff safety, and repo-state guardrails

  **What to do**: Implement Git service using `execa` argument arrays. Detect git repo from subdirectories, bare repo, initial commit, staged changes, unstaged changes, untracked files, binary files, deleted/renamed files, merge/rebase/cherry-pick in progress, detached HEAD, and missing `user.name`/`user.email`. Generate staged file list and staged diff. If staged changes exist, never include unstaged changes automatically; show warning metadata. If no staged changes, expose a TTY-only operation to explicitly run `git add -A` after showing changed file list. Implement diff size limit/truncation metadata and basic secret pattern scanning before provider call.
  **Must NOT do**: Do not execute `git commit` in this task except in isolated tests if necessary. Do not use shell strings. Do not auto-stage without explicit interactive confirmation.

  **Recommended Agent Profile**:
  - Category: `deep` - Reason: git edge cases and safety behavior require careful integration testing.
  - Skills: [] - Standard process/git testing.
  - Omitted: [`git-master`] - This is code implementation planning, not repository history operations.

  **Parallelization**: Can Parallel: YES | Wave 2 | Blocks: Tasks 6, 7, 8, 9, 10 | Blocked By: Task 1

  **References**:
  - Decision: `.sisyphus/drafts/commit-now-myfriend-cli.md:15,22,35` - git safety and staged-first semantics.
  - Metis: Context section in this plan - edge cases for repo state, diff truncation, secret scanning.
  - External: `https://github.com/sindresorhus/execa/blob/HEAD/readme.md` - process execution patterns.

  **Acceptance Criteria**:
  - [ ] Temp repo tests cover non-git repo, subdirectory repo, initial commit, staged-only, unstaged-only, untracked-only, staged+unstaged, binary file, rename/delete, merge/rebase/cherry-pick state.
  - [ ] Missing git identity produces a clear preflight issue before commit attempt.
  - [ ] Secret scan warning prevents provider call if user cancels.
  - [ ] Diff truncation metadata is included when configured size limit is exceeded.

  **QA Scenarios**:
  ```
  Scenario: Staged-only semantics are preserved
    Tool: Bash
    Steps: Run Vitest integration creating staged.txt staged and unstaged.txt unstaged; inspect Git service output.
    Expected: Staged diff includes staged.txt only; metadata warns unstaged changes exist; unstaged.txt remains unstaged.
    Evidence: .sisyphus/evidence/task-5-staged-only.txt

  Scenario: Merge/rebase state blocks workflow
    Tool: Bash
    Steps: Run Git service test fixture with simulated merge/rebase/cherry-pick marker files.
    Expected: Service returns blocking issue; no provider or commit operation is allowed.
    Evidence: .sisyphus/evidence/task-5-repo-state-block.txt
  ```

  **Commit**: YES | Message: `feat(git): add repository safety service` | Files: [`src/git/**`, `src/security/**`, `tests/git/**`]

- [x] 6. Implement main `cnm` commit workflow state machine

  **What to do**: Compose config, Git service, provider adapter, and command shell into the main workflow. State machine: validate git repo → load config → inspect staged changes → if none staged, list current changes and ask whether to stage all current changes → collect staged diff → run secret/size checks → call provider → preview files/status/message/action → offer confirm/edit/regenerate/cancel → on confirm execute `git commit` safely using temp message file or stdin → report result. Support pre-commit hook failures by showing git stderr/stdout summary and exiting `1`; do not retry or amend.
  **Must NOT do**: Do not push, amend, bypass hooks, modify files, choose files with AI, or commit in non-TTY when confirmation is required.

  **Recommended Agent Profile**:
  - Category: `unspecified-high` - Reason: central workflow wiring and user-visible behavior.
  - Skills: [`cli-design-framework`] - Ensure workflow remains human-primary and safety-gated.
  - Omitted: [`frontend-design`] - Terminal interaction only.

  **Parallelization**: Can Parallel: NO | Wave 3 | Blocks: Tasks 7, 8, 10, 11 | Blocked By: Tasks 2, 3, 4, 5

  **References**:
  - Requirement: `.sisyphus/drafts/commit-now-myfriend-cli.md:7,16,20,22,25` - core command, preview/confirm, command surface, staged handling, no yes.
  - Metis: Context section in this plan - required state machine and exit code contract.
  - Pattern: Task 5 Git service and Task 4 provider adapter contracts.

  **Acceptance Criteria**:
  - [ ] Integration test proves only confirmed workflow creates a commit.
  - [ ] Integration test proves cancel exits `130` and creates no commit.
  - [ ] Integration test proves existing unstaged changes are not included when staged changes exist.
  - [ ] Integration test proves no staged changes triggers stage-all prompt and default No creates no commit/provider call.
  - [ ] Failing pre-commit hook exits `1`, creates no commit, and surfaces hook stderr summary.

  **QA Scenarios**:
  ```
  Scenario: Confirmed staged commit succeeds
    Tool: Bash
    Steps: Run integration test in temp repo: stage staged.txt; leave unstaged.txt unstaged; mock provider returns `feat: add staged file`; mock prompt confirm.
    Expected: Latest commit message is `feat: add staged file`; commit contains staged.txt; unstaged.txt is not committed and remains in working tree.
    Evidence: .sisyphus/evidence/task-6-confirmed-commit.txt

  Scenario: User cancellation prevents commit
    Tool: Bash
    Steps: Run integration test in temp repo with staged file; mock provider returns message; mock prompt cancel.
    Expected: Exit code 130; commit count unchanged; no git commit command is executed after cancel.
    Evidence: .sisyphus/evidence/task-6-cancel-no-commit.txt
  ```

  **Commit**: YES | Message: `feat(workflow): implement safe ai commit flow` | Files: [`src/workflow/**`, `src/commands/commit.ts`, `tests/workflow/**`]

- [x] 7. Implement terminal interaction details: preview, edit, regenerate, cancel

  **What to do**: Implement @clack/prompts presentation for the human path. Preview must show file list, staged/unstaged status, secret/size warnings, generated commit message, and exact git operation (`git commit`). User actions: confirm, edit, regenerate, cancel. Fix the edit mechanism as follows: first support inline text edit via Clack text input; do not launch external editors in v1. Validate edited message: non-empty; if style is Conventional, subject must match basic Conventional Commit pattern or prompt user to edit again/cancel. Regenerate must call provider again with same normalized input and updated attempt metadata. Ctrl-C maps to exit `130`.
  **Must NOT do**: Do not build a TUI, REPL, or multi-turn chat. Do not let custom prompt bypass preview/confirmation. Do not commit from edit/regenerate path without final confirm.

  **Recommended Agent Profile**:
  - Category: `unspecified-high` - Reason: user interaction safety and polish.
  - Skills: [`cli-design-framework`] - Preserve human-primary discoverability and confirmation semantics.
  - Omitted: [`playwright-cli`] - No browser.

  **Parallelization**: Can Parallel: YES | Wave 3 | Blocks: Tasks 10, 11 | Blocked By: Tasks 5, 6

  **References**:
  - Decision: `.sisyphus/drafts/commit-now-myfriend-cli.md:14-16,23,25` - human-primary CLI, confirm-before-commit, Conventional default, no yes.
  - External: `https://github.com/bombshell-dev/clack/blob/main/packages/prompts/README.md` - prompt primitives.

  **Acceptance Criteria**:
  - [ ] Tests cover confirm, edit, regenerate, cancel, Ctrl-C/cancel signal mapping.
  - [ ] Edited empty message cannot be committed.
  - [ ] Edited invalid Conventional subject is rejected or requires another edit/cancel.
  - [ ] Regenerate calls provider exactly once per regenerate action and never commits before final confirm.

  **QA Scenarios**:
  ```
  Scenario: Edit message before commit
    Tool: Bash
    Steps: Run workflow test with staged file; mock provider returns `chore: initial`; mock user selects edit and enters `fix: corrected message`, then confirm.
    Expected: Latest commit message is `fix: corrected message`; no intermediate commit with `chore: initial` exists.
    Evidence: .sisyphus/evidence/task-7-edit-message.txt

  Scenario: Regenerate then cancel
    Tool: Bash
    Steps: Run workflow test with staged file; provider returns two messages; mock user selects regenerate then cancel.
    Expected: Provider called twice; exit code 130; no commit created.
    Evidence: .sisyphus/evidence/task-7-regenerate-cancel.txt
  ```

  **Commit**: YES | Message: `feat(interaction): add commit preview actions` | Files: [`src/prompts/**`, `src/workflow/**`, `tests/prompts/**`, `tests/workflow/**`]

- [x] 8. Implement `--dry-run` and `--json` contracts

  **What to do**: Implement dry-run and JSON surfaces as secondary machine/inspection modes. `--dry-run` may call provider and display the message/preview, but must never call `git commit`. `--json` must output one parseable JSON object to stdout with stable fields: `ok`, `committed`, `dryRun`, `message`, `files`, `warnings`, `error`. In v1, `--json` never commits and must imply safe preview behavior; if a path would require interactive confirmation, return JSON with `committed: false`. Keep human prompts/spinners out of stdout when `--json` is active.
  **Must NOT do**: Do not implement true commit automation via `--json`. Do not mix human prompt text with JSON stdout. Do not add `--yes`.

  **Recommended Agent Profile**:
  - Category: `unspecified-high` - Reason: output contract and non-interactive safety must be precise.
  - Skills: [`cli-design-framework`] - Secondary machine surface contract.
  - Omitted: [`frontend-design`] - No UI.

  **Parallelization**: Can Parallel: YES | Wave 3 | Blocks: Tasks 10, 11 | Blocked By: Tasks 2, 3, 4, 5, 6

  **References**:
  - Decision: `.sisyphus/drafts/commit-now-myfriend-cli.md:20,25` - dry-run/json surface, no real `--yes`.
  - Metis: Context section in this plan - JSON fields and never-commit rule.

  **Acceptance Criteria**:
  - [ ] `cnm --dry-run` in temp repo with staged file produces message preview and leaves commit count unchanged.
  - [ ] `cnm --json` stdout is valid JSON parseable by `JSON.parse`.
  - [ ] `cnm --json` output includes stable fields: `ok`, `committed`, `dryRun`, `message`, `files`, `warnings`, `error`.
  - [ ] `cnm --json` always has `committed: false` in v1.
  - [ ] Tests assert no Clack prompt text is emitted to stdout in JSON mode.

  **QA Scenarios**:
  ```
  Scenario: Dry-run previews without commit
    Tool: Bash
    Steps: In temp git repo with staged file and mock provider `fix: dry run message`, run `cnm --dry-run`.
    Expected: Output contains `fix: dry run message`; git commit count unchanged; exit code 0.
    Evidence: .sisyphus/evidence/task-8-dry-run.txt

  Scenario: JSON output is machine-parseable and safe
    Tool: Bash
    Steps: In temp git repo with staged file, run `cnm --json`; pipe stdout into `node -e 'JSON.parse(require("fs").readFileSync(0,"utf8"))'`.
    Expected: Parser exits 0; JSON has `committed:false`; no git commit occurred; stdout contains no prompt glyphs/text.
    Evidence: .sisyphus/evidence/task-8-json-contract.json
  ```

  **Commit**: YES | Message: `feat(output): add dry-run and json contracts` | Files: [`src/output/**`, `src/commands/commit.ts`, `tests/output/**`, `tests/workflow/**`]

- [x] 9. Implement `cnm doctor` diagnostics

  **What to do**: Implement `cnm doctor` and `cnm doctor --json`. Checks: Node version, git executable, whether current path is a git repo, repo state warnings, git identity (`user.name`/`user.email`), config directory accessibility, user config validity, project config validity, default provider presence, model presence, API key presence, file permission warning, and bin conflict guidance. By default, doctor must not send a live provider request; add an explicit `--check-provider` flag only if implemented safely with timeout and clear warning. JSON doctor output must use stable issue codes such as `provider_config_missing`, `api_key_missing`, `git_identity_missing`.
  **Must NOT do**: Do not call real providers by default. Do not print full API keys. Do not mutate config or repo.

  **Recommended Agent Profile**:
  - Category: `unspecified-high` - Reason: diagnostics cross config, git, and output contracts.
  - Skills: [`cli-design-framework`] - Help/discoverability and diagnostics design.
  - Omitted: [`playwright-cli`] - No browser.

  **Parallelization**: Can Parallel: YES | Wave 3 | Blocks: Tasks 10, 11 | Blocked By: Tasks 2, 3, 5

  **References**:
  - Decision: `.sisyphus/drafts/commit-now-myfriend-cli.md:20,24,29,39` - doctor command, config storage, publishing/bin conflict.
  - Metis: Context section in this plan - doctor default no live provider call.

  **Acceptance Criteria**:
  - [ ] `cnm doctor` outside a git repo exits `0` or `1` per implemented severity, but does not crash; output clearly reports git repo status.
  - [ ] `cnm doctor --json` stdout is parseable JSON with issue codes.
  - [ ] Missing provider config reports `provider_config_missing`.
  - [ ] Full API key is never printed in human or JSON doctor output.
  - [ ] Default doctor path does not call provider network client.

  **QA Scenarios**:
  ```
  Scenario: Doctor reports missing config without crashing
    Tool: Bash
    Steps: Set `CNM_HOME` to empty temp directory; run `cnm doctor --json` outside git repo.
    Expected: stdout JSON parses; includes `provider_config_missing`; no full secrets; no unhandled exception.
    Evidence: .sisyphus/evidence/task-9-doctor-missing-config.json

  Scenario: Doctor does not call live provider by default
    Tool: Bash
    Steps: Run doctor test with provider client mocked to throw if called.
    Expected: Test passes; provider client is not invoked unless explicit provider-check flag exists and is used.
    Evidence: .sisyphus/evidence/task-9-doctor-no-network.txt
  ```

  **Commit**: YES | Message: `feat(doctor): add cnm diagnostics` | Files: [`src/commands/doctor.ts`, `src/doctor/**`, `tests/doctor/**`]

- [x] 10. Build comprehensive automated test suite and hardening fixtures

  **What to do**: Expand tests into a complete suite across units, CLI subprocess tests, temp git repo integration tests, config security tests, JSON contract tests, and provider adapter tests. Use dependency injection or mocks for prompts/providers. Every test that touches config must use temporary `CNM_HOME`. Every git test must create an isolated temporary repo and configure test-local git identity. Cover edge cases from Metis: non-git repo, bare repo if feasible, subdirectory repo, initial commit, binary files, renamed/deleted files, spaces/unicode filenames, unstaged-only, untracked-only, staged+unstaged, merge/rebase/cherry-pick in progress, detached HEAD, hook failure, missing identity, diff truncation, secret warning/cancel, malformed provider output, Ctrl-C/cancel, non-TTY, config corruption, env override, project secret rejection.
  **Must NOT do**: Do not hit real AI APIs. Do not write real home config. Do not rely on test order. Do not require human input.

  **Recommended Agent Profile**:
  - Category: `deep` - Reason: comprehensive QA across subprocess, git, config, and provider seams.
  - Skills: [] - Test engineering task.
  - Omitted: [`playwright-cli`] - No browser UI.

  **Parallelization**: Can Parallel: NO | Wave 4 | Blocks: Task 11 | Blocked By: Tasks 2-9

  **References**:
  - Verification Strategy in this plan - Vitest with temp git repos and mocked providers.
  - Metis Review in this plan - exhaustive edge-case list and acceptance directives.
  - Requirement: `.sisyphus/drafts/commit-now-myfriend-cli.md:32-34` - no existing infra; tests must be added.

  **Acceptance Criteria**:
  - [ ] `pnpm test -- --run` exits `0`.
  - [ ] Test suite includes at least one subprocess test against `dist/index.js` after build.
  - [ ] Test suite includes temp git repo integration tests for successful commit, cancel, dry-run, stage-all prompt, hook failure.
  - [ ] Test suite includes provider adapter tests for all four provider types with no live network.
  - [ ] Test suite includes config redaction and project-secret rejection tests.
  - [ ] CI runs install, typecheck, test, build, and pack dry-run.

  **QA Scenarios**:
  ```
  Scenario: Full automated suite passes
    Tool: Bash
    Steps: Run `pnpm typecheck && pnpm test -- --run && pnpm build`.
    Expected: All commands exit 0; tests do not prompt for user input or call live provider APIs.
    Evidence: .sisyphus/evidence/task-10-full-suite.txt

  Scenario: Tests are isolated from real user config
    Tool: Bash
    Steps: Run config tests with HOME unchanged but `CNM_HOME` temp override; inspect test logs and temp paths.
    Expected: No writes under real `~/.cnm`; all config writes occur under temp CNM_HOME; secrets are redacted in snapshots/logs.
    Evidence: .sisyphus/evidence/task-10-config-isolation.txt
  ```

  **Commit**: YES | Message: `test: harden cnm cli workflows` | Files: [`tests/**`, `src/**`, `.github/workflows/ci.yml`]

- [x] 11. Complete README, examples, and npm publish readiness

  **What to do**: Write production README and release hygiene. README must include installation (`npm install -g commit-now-myfriend`, `npx commit-now-myfriend`, and npm exec fallback), quick start (`cnm init`, `cnm`), default staged-only behavior, no-staged prompt behavior, confirmation requirement, no `--yes`, `--dry-run`, `--json`, config directory `~/.cnm`, project config rules, API key plaintext warning, provider examples for all four provider types, custom prompt configuration, security warning that diffs are sent to configured provider, troubleshooting (`cnm doctor`, global `cnm` bin conflict because npm package `cnm` exists), and scope boundaries. Ensure package `files` whitelist, LICENSE, keywords, repository metadata placeholder if available, npm provenance/2FA guidance, `npm pack --dry-run` clean output, and final help examples are consistent.
  **Must NOT do**: Do not publish to npm unless user explicitly requests execution/publishing later. Do not include real API keys. Do not document unsupported push/amend/rebase/TUI/plugin features.

  **Recommended Agent Profile**:
  - Category: `writing` - Reason: user-facing documentation and release notes need clarity.
  - Skills: [`cli-design-framework`] - README must explain human-primary CLI surfaces and safety boundaries.
  - Omitted: [`frontend-design`] - No web UI.

  **Parallelization**: Can Parallel: NO | Wave 4 | Blocks: Final Verification | Blocked By: Tasks 1-10

  **References**:
  - Requirement: `.sisyphus/drafts/commit-now-myfriend-cli.md:9-11,19-29,39,44-46` - public distribution, v1 scope, package/bin finding, in/out scope.
  - External: npm CLI docs for `npm pack --dry-run` and provenance should be referenced from README if adding release section.

  **Acceptance Criteria**:
  - [ ] `README.md` includes install, quick start, provider config, safety model, staged behavior, config path, dry-run/json, doctor, custom prompt, troubleshooting, and v1 non-goals.
  - [ ] README examples match actual `node dist/index.js --help` output.
  - [ ] `npm pack --dry-run` excludes `.sisyphus/`, tests unless intentionally included, `.env`, `.cnm`, and any secrets.
  - [ ] Package metadata includes package name `commit-now-myfriend` and bin `cnm`.
  - [ ] No docs claim support for Vertex, push, amend, rebase, TUI, plugins, or `--yes`.

  **QA Scenarios**:
  ```
  Scenario: README quick start matches CLI
    Tool: Bash
    Steps: Run `pnpm build && node dist/index.js --help`; compare documented commands in README with help output.
    Expected: README mentions only implemented commands/options; help output includes all README quick-start commands.
    Evidence: .sisyphus/evidence/task-11-readme-help-consistency.txt

  Scenario: Package dry-run is release-clean
    Tool: Bash
    Steps: Run `npm pack --dry-run`.
    Expected: Exit code 0; package includes dist, README, LICENSE, package.json; excludes `.sisyphus/`, tests unless chosen, config files, and secrets.
    Evidence: .sisyphus/evidence/task-11-pack-dry-run.txt
  ```

  **Commit**: YES | Message: `docs: document cnm v1 usage and release checks` | Files: [`README.md`, `LICENSE`, `package.json`, `.npmignore` or package `files`, `.github/workflows/ci.yml`]

## Final Verification Wave (MANDATORY — after ALL implementation tasks)
> 4 review agents run in PARALLEL. ALL must APPROVE. Present consolidated results to user and get explicit "okay" before completing.
> **Do NOT auto-proceed after verification. Wait for user's explicit approval before marking work complete.**
> **Never mark F1-F4 as checked before getting user's okay.** Rejection or user feedback -> fix -> re-run -> present again -> wait for okay.
- [x] F1. Plan Compliance Audit — oracle
- [x] F2. Code Quality Review — unspecified-high
- [x] F3. Real Manual QA — unspecified-high (+ interactive_bash for CLI; playwright not needed because no browser UI)
- [x] F4. Scope Fidelity Check — deep

## Commit Strategy
- One implementation commit per completed task where file sets are coherent.
- Commit message style: `type(scope): concise description`.
- Do not commit `.sisyphus/evidence/` unless user explicitly wants evidence tracked.
- Do not commit real `~/.cnm` configs, `.env`, API keys, npm tokens, or provider secrets.
- Suggested final release-prep commit: `chore(release): prepare cnm v1 package`.

## Success Criteria
- A developer can install or run the package, execute `cnm init`, configure a default provider, run `cnm` inside a git repo, preview an AI Conventional Commit message, confirm, and get a real git commit.
- Safety defaults are proven by automated tests: no true commit without confirmation, no `--json` commit, no project secrets, no real home config writes in tests.
- All provider families are covered by mocked adapter tests and documented configuration examples.
- Package is publish-ready via `npm pack --dry-run` and README explains install, onboarding, safety, provider setup, and bin conflict troubleshooting.
