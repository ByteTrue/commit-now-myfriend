## 2026-05-04 Task: startup-baseline
- Workspace baseline is effectively empty except `.sisyphus/plans/commit-now-myfriend-v1.md` and `.sisyphus/boulder.json`.
- The working directory is not currently a git repository, so Task 1 must initialize package files without assuming git metadata exists.
- Preserve `.sisyphus/` planning and boulder state; package publish config must exclude `.sisyphus/` from npm package contents.

## 2026-05-04 Task: scaffold-research
- Use ESM with `"type": "module"` and Node `>=20`.
- Use `commander` 14.x for Node 20 compatibility; commander 15 is ESM-only and requires Node 22.12+.
- Preserve CLI shebang in the TypeScript entry (`#!/usr/bin/env node`); configure tsup so `dist/index.js` remains executable/runnable.
- Prefer package `files` whitelist over broad publish inclusion; `npm pack --dry-run` must exclude `.sisyphus/`, `.env`, `.cnm`, tests, and fixture secrets.
- Use `vitest.config.ts` with `environment: "node"`.
- Use `packageManager` in `package.json`, commit `pnpm-lock.yaml`, and CI should use `pnpm install --frozen-lockfile`.

## 2026-05-04 Task: package-initialization
- `@clack/prompts` must use the current 1.x line; the older `^0.13.0` range is no longer installable from npm.
- Keeping the shebang in `src/index.ts` is sufficient for `tsup`; adding a `banner` duplicates the shebang and breaks `node dist/index.js` with a syntax error.
- `npm pack --dry-run` writes most listing output to stderr, so evidence capture must redirect both stdout and stderr.

## 2026-05-04 Task: cli-shell-and-output-routing
- The `cnm` CLI is human-primary with a secondary JSON surface, so help/version stay human-readable while placeholder command handlers explicitly route human diagnostics to stderr and JSON payloads to stdout only.
- Defining `--dry-run` and `--json` on both the root command and current placeholder subcommands keeps `cnm init --json --dry-run` working today without forcing later tasks to rewrite the command shell.
- Attaching a root commander action caused unknown subcommands to degrade into a generic "too many arguments" error; a small pre-parse unknown-command check restores the intended `unknown command '...'` contract.
- Rejecting `--json --help` keeps the JSON contract unambiguous until a deliberate machine-readable help surface exists.

## 2026-05-04 Task: provider-adapters
- Keep provider-facing business API normalized around `generateCommitMessage(input)` and hide SDK request/response shapes behind adapter modules.
- SDK clients are mockable through `ProviderClientFactories`; unit tests inject structural fake clients and therefore do not require API keys or live network access.
- Sanitization should run after every provider response and before returning metadata, so empty/malformed model output fails as a typed `ProviderError`.

## 2026-05-04 Task: git-service-safety
- Git inspection lives in `src/git/**` and uses `execa("git", args, { reject: false })` with argument arrays for every subprocess.
- Staged-first semantics are enforced with `git diff --cached`; unstaged and untracked files only appear as metadata warnings unless `stageAllChanges` is called with confirmed TTY intent.
- Git integration tests use temp repos under the OS temp directory and configure local test identity only where commits are needed.
- Secret scanning and diff truncation are exposed as metadata on the inspection result so later workflow code can block, warn, or ask for confirmation before provider calls.

## 2026-05-04 Task: config-onboarding-redaction
- User config now resolves to `CNM_HOME/config.json` when `CNM_HOME` is set, otherwise `~/.cnm/config.json`; config tests should always inject a temp `CNM_HOME`.
- Effective config precedence is implemented per key as flags > env (`CNM_*`) > project `.cnmrc.json` > user config > defaults.
- Project config may provide non-secret defaults, but any `apiKey` entry is warned and ignored; redacted outputs render API keys as `[redacted]` in both human and JSON surfaces.
- `cnm init` writes a concrete starter user config with resolved provider/model/promptStyle defaults, but never creates project config.

## 2026-05-04 Task: main-commit-workflow
- Root `cnm` now routes through `src/commands/commit.ts` into a dedicated workflow state machine so commander global options and existing subcommands remain intact while tests can inject fake prompts/providers/commit runners.
- The workflow blocks on git repository blockers and secret-scan hits before any provider call, warns on staged-vs-unstaged drift, and only stages all changes after an explicit TTY-confirmed prompt with default No behavior.
- Safe commit execution is implemented with `execa("git", ["commit", "-F", tempFile])`; commit messages never pass through shell interpolation and hooks are allowed to run normally.
- Manual CLI QA can use an OpenAI-compatible local mock endpoint through `CNM_BASE_URL`, which exercises the real root command without live AI credentials.

## 2026-05-04 Task: terminal-interaction-details
- Preview rendering is clearer when file rows preserve combined status markers such as `staged:modified, unstaged:modified`; collapsing to a single status hides exactly the drift this workflow is trying to make visible.
- Showing `Operation: git commit`, an explicit interaction mode, and the generation attempt count keeps the preview human-primary without implying any push/amend/rebase side effects.
- Edited message safety needs validation in two places: inline Clack `text()` validation for the real terminal UX, and a workflow-level re-check so mocked prompts or future alternate prompt seams cannot slip an invalid Conventional Commit subject into `git commit`.
- A 20s timeout is currently needed for the workflow integration test file on this machine because temp git repository setup plus real commit/hook coverage regularly exceeds Vitest's default 5s test timeout.

## 2026-05-04 Task: doctor-diagnostics
- `cnm doctor` now stays fully read-only: it reuses config loading and `inspectGitRepository`, checks `git --version` separately before repo inspection, and never touches provider clients or the network by default.
- Doctor JSON is stable and parseable with top-level `status`, `summary`, `issues`, `checks`, and `guidance`; stable issue codes include `provider_config_missing`, `api_key_missing`, `git_identity_missing`, and existing git issue codes like `not_git_repository`.
- Doctor output must always use redacted config snapshots (`[redacted]`) rather than raw config values; fixture secrets are acceptable in tests, but serialized doctor output should never contain them.
- Git-heavy doctor tests can exceed Vitest's default 5s timeout on this machine, so explicit per-test timeouts keep the suite stable without weakening assertions.

## 2026-05-04 Task: dry-run-json-contracts
- Root cnm --json now walks the real inspection, config, and provider path, but v1 always returns a single stdout JSON object with stable fields and never calls git commit.
- --dry-run now reuses the provider generation path so it can preview a real commit message while still skipping prompts and commit execution; tests should assert provider-called and commit-runner-not-called.
- On this environment, temp-repo integration suites are reliable only when Vitest runs with serialized files and a longer timeout; fileParallelism false, maxWorkers 1, and testTimeout 20000 keep pnpm test -- --run stable.

## 2026-05-04 Task: comprehensive-test-hardening
- Task 10 audit found existing coverage for most git/config/provider/workflow edges; the meaningful missing gaps were built `dist/index.js` subprocess coverage, spaces/unicode path handling, untracked-only git inspection, non-TTY workflow blocking, and secret-scan workflow blocking before provider calls.
- A Vitest subprocess smoke can safely run `pnpm build` in `beforeAll`, then exercise `node dist/index.js --help` and `node dist/index.js doctor --json` with temp `CNM_HOME`; this keeps CI order unchanged while proving the built CLI artifact runs.
- Git patch output may quote Unicode paths by default, so tests should assert decoded paths through porcelain-derived `stagedFiles` and assert diff content separately instead of depending on Git's patch quoting format.

## 2026-05-04 Task: documentation-and-publish-readiness
- README.md is now comprehensive, covering installation, quick start, safety model, provider configuration, and v1 non-goals.
- Package metadata and files whitelist are verified; npm pack --dry-run confirms that only dist, README, LICENSE, and package.json are included.
- The cnm binary name conflict is documented with troubleshooting guidance (npx/pnpm exec).
- All verification steps (build, typecheck, test, pack) pass with 66 tests green.

## 2026-05-04 Task: documentation-fix-after-qa
- Updated package.json description to "AI-assisted Git commit workflow CLI."
- README.md now includes concrete provider configuration examples for OpenAI, Anthropic, Google Gemini, and OpenAI-compatible services.
- Added explicit security warning regarding plaintext API key storage in the local config file.
- Documented all supported environment variables (CNM_*) and their precedence over configuration files.
- Clarified cnm init behavior and ensured all examples align with actual CLI help output.
- Verified that npm pack --dry-run still produces a clean package with only necessary files.

## 2026-05-05 Task: final-verification-f1-f2-fixes
- Root `cnm` now supports non-secret one-shot config overrides (`--provider`, `--model`, `--base-url`, `--prompt-style`, `--custom-prompt`) and passes them into `resolveEffectiveConfig({ flagOverrides })`; do not add root `--api-key` because secrets belong in env/user config to avoid shell-history leaks.
- Commander root options with names also used by subcommands need `enablePositionalOptions()` so `cnm init --provider ...` remains a subcommand option instead of being consumed by the parent command.
- `sanitizeCommitMessage()` should strip text before the first valid Conventional subject, but should not treat `Why:` or `Explanation:` after the subject as provider chatter because those are valid commit body content.
- Git service command failures are safest as `GitInspection.blockingIssues` (`git_status_failed`, `git_diff_numstat_failed`, `git_diff_patch_failed`, `git_add_failed`) so existing workflow blocking paths prevent provider calls and commits.
- Doctor and config-touching tests must include temp `CNM_HOME`; changing only `HOME`/`XDG_CONFIG_HOME` does not isolate cnm user config.

## 2026-05-05 Task: temp-git-helper-retry
- `tests/helpers/temp-git-repo.ts` now retries only transient-looking test setup git failures up to 3 attempts with a short backoff: empty stdout/stderr failures, EAGAIN/resource-pressure output, or signal/exit anomalies without diagnostics.
- Real git failures that include diagnostic stdout/stderr still fail with command, exit code, signal, stderr, and stdout in the thrown error, preserving product failure assertions and injected `GitCommandRunner` tests.

## 2026-05-05 Task: final-verification-wave
- F1/F2 initially rejected and were fixed, then re-run to APPROVE; F3/F4 approved on first pass.
- Final verification checkboxes were marked complete after the continuation directive authorized proceeding.

## 2026-05-05 Task: secret-scan-warning-only
- User clarified that potential secret matches should warn but not block; staged diff still goes to the configured provider if the user continues.
- Secret scan findings now surface as workflow warnings rather than blocking issues, preserving user control while keeping visibility.

## 2026-05-05 Task: empty-provider-output-token-budget
- `gemini-3-flash-preview` behind an OpenAI-compatible endpoint can consume completion budget as hidden reasoning tokens and return empty content for large diffs; provider default max output tokens was raised from 512 to 4096 and prompt wording now asks for the final message directly.

## 2026-05-05 Task: config-panel
- Bare `cnm config` can stay human-primary without breaking automation by routing only interactive TTY-or-injected sessions into a prompt-driven panel and keeping `--json`, `--dry-run`, subcommands, and non-TTY bare usage on the existing read-only output path.
