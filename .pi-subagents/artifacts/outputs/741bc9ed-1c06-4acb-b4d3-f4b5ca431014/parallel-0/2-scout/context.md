# Code Context

## Files Retrieved
1. `cmd/cnm/main.go` (lines 1-11) - current native entry point; already minimal and reusable.
2. `internal/cli/cli.go` (lines 28-129, 436-634, 1005-1172) - command dispatch and the existing end-to-end inspect/provider/tool/commit path; most product complexity is concentrated here.
3. `internal/git/service.go` (lines 15-129, 131-216, 317-359) - reusable git runner, repository/scope inspection, truncation path, and recent-commit helper.
4. `internal/git/types.go` (lines 31-58, 117-179) - current diff/status/scope contracts; useful but substantially larger than a message-only contract.
5. `internal/providers/tool_call_provider.go` (lines 14-80, 147-185, 235-269) - reusable authenticated multi-provider HTTP request flow.
6. `internal/providers/tool_protocol.go` (lines 32-61, 63-153, 155-251) - four protocol adapters and response parsing.
7. `internal/providers/http.go` (lines 11-41) - small, reusable stdlib HTTP safety helpers including response cap.
8. `internal/providers/helpers.go` (lines 10-63) - provider defaults, validation, URL and client helpers.
9. `internal/runtime/runtime.go` (lines 11-104, 110-205) - current multi-turn tool-call runtime; unnecessary for a single diff-to-message request.
10. `internal/runtime/types.go` (lines 9-115, 123-177) - tool/runtime and commit-execution types; only `DiffResult` resembles the reduced product need.
11. `go.mod` (lines 1-36) - six direct and 21 indirect non-stdlib modules, dominated by TUI/keyring.
12. `package.json` (lines 1-41) - npm remains a thin binary installer/launcher and is separable from core behavior.
13. `README.md` (lines 19-41, 86-105, 115-151) - documents the much broader current product contract.

## Key Code

### Existing reusable seams

```go
// internal/git/service.go:18-50
func DefaultCommandRunner(cwd string, args []string, env map[string]string) (CommandResult, error)
```

This is the cleanest reusable primitive: it executes `git` in a selected directory, preserves an injected environment, captures output, and normalizes process exit errors. A minimal implementation can call it with one explicitly chosen diff contract.

```go
// internal/providers/tool_call_provider.go:36-52
func CreateToolCallProvider(options ToolCallProviderOptions) (runtimex.ToolCallProvider, error)
```

This is reusable only if retaining all four providers and the tool-call protocol. For the reduced product, direct text completion is simpler than preserving a tool loop just to return one string.

```go
// internal/cli/cli.go:1129-1138
GetDiff: func() (runtimex.DiffResult, error) {
    result, err := gitpkg.DefaultCommandRunner(runtime.CWD,
        scopedDiffArgs(commitScopePaths(scope)), runtime.Env)
    // ...
}
```

This is the current closest implementation of “read diff”, but `scopedDiffArgs` is `git diff -- <paths>` (`internal/cli/cli.go:865-869`). It reads unstaged tracked changes only: it omits staged content (`--cached`) and cannot represent untracked files. It therefore must not be lifted unchanged until “git diff” semantics are fixed explicitly.

### Current data flow

`cmd/cnm/main.go` → `cli.Execute` → root/`auto` dispatch → config resolution → `git.InspectCommitScope` → provider construction → `runtime.ToolCallRuntime.Run` → provider calls tools (`inspect_commit_scope`, `get_diff`, optional `read_file`, `create_commits`) → commit planning/execution.

The desired data flow is only:

`main` → parse minimal flags/env → run one git diff command with a byte cap → reject empty/error → make one provider request with prompt + diff → parse/trim one message → print it.

No staging, commit execution, transaction rollback, repair, secret store UI, doctor, TUI, file reads, multi-commit plans, or tool-call loop is needed.

## Architecture

### What can be reused safely

- **Keep unchanged initially:** `cmd/cnm/main.go`; `internal/git.DefaultCommandRunner`; provider HTTP body limit/request helpers.
- **Reuse by extraction/simplification:** provider endpoint/auth/response parsing from `internal/providers/tool_call_provider.go` and `tool_protocol.go`; config provider/model/base URL/API-key types if four-provider compatibility is a hard compatibility requirement.
- **Do not reuse as the reduced orchestration:** `runAuto`, `executeAutoCommit`, `ToolCallRuntime`, `autonomousCommitDomainTools`. They encode side effects and multi-turn planning that the reduced product explicitly does not need.
- **Do not reuse unchanged:** current `GetDiff` closure, because it excludes staged changes and untracked files.

### Directories that become deletable

For the strict single responsibility (“read git diff, generate one message”), these entire runtime directories can go:

- `internal/tui/` (3,150 LOC): all full-screen/config/onboarding/doctor UI.
- `internal/commands/` (862 LOC): init/config/doctor command surface.
- `internal/doctor/` (579 LOC): setup/probe reporting.
- `internal/interactive/` (142 LOC): confirmation IO.
- `internal/security/` (102 LOC): scope secret scanning; valuable safety, but outside the stated sole responsibility unless explicitly retained as a trust-boundary requirement.
- `internal/runtime/` (1,021 LOC): tool loop, repair and create-commit tools.
- `internal/output/` (77 LOC): optional; plain stdout/stderr and `encoding/json` suffice.

`internal/config/` (1,547 LOC) can also be deleted if configuration becomes environment/flags only. If preserving current config-file precedence and system keyring is required, retain it temporarily; that choice alone keeps `github.com/zalando/go-keyring` and its platform transitives.

`internal/git/` and `internal/providers/` should be **shrunk**, not deleted wholesale. Their current production sizes are 1,104 and 1,089 LOC respectively, but the needed subsets are small.

Repository-support directories are separate from runtime scope: `.codestable/`, `docs/adr/`, release/npm scripts, and `.github/` do not affect the binary. Delete or rewrite product docs/ADRs only after the product decision; keep `scripts/` if npm/native distribution remains required.

### LOC and dependency estimate

Measured source baseline (line count, including blanks/comments):

- Go production: **10,454 LOC / 42 files**.
- Go tests: **5,897 LOC / 10 files**.
- Node production distribution scripts: **349 LOC / 4 files**.
- Node tests: **171 LOC / 2 files**.
- Current Go modules: **6 direct + 21 indirect** non-stdlib requirements.

Estimated strict minimal retained implementation:

- One provider/protocol, env flags, tracked `git diff`: **~250-450 production Go LOC**, **~100-200 test LOC**, 0 non-stdlib Go dependencies. Reduction: roughly **96-98% production Go LOC** and **27 → 0 module requirements**.
- Preserve all four provider protocols but remove tool calls/config UI/keyring: **~700-1,200 production Go LOC**, **~300-600 test LOC**, still 0 non-stdlib dependencies because HTTP/JSON are stdlib. Reduction: roughly **89-93% production Go LOC**.
- Low-risk compatibility bridge retaining current config + four providers while replacing only orchestration: likely **~2,200-3,000 production Go LOC** after pruning, with keyring as the only direct dependency family. Reduction: roughly **71-79%**, then simplify config later.

These are planning ranges, not a compiled deletion diff. Exact LOC depends primarily on three unresolved contract choices: staged vs unstaged vs `HEAD`, handling untracked files, and whether four providers/current config formats remain compatible.

### Lower-risk path than rewrite

Yes. A staged strangler path is lower risk than a clean rewrite:

1. Add a message-only execution path behind the existing `cli.Execute`/provider configuration, using `DefaultCommandRunner` and existing provider adapters; make it output-only and never expose `create_commits`.
2. Define and test diff semantics first. For tracked staged+unstaged, `git diff HEAD -- ...` is the simplest established-repository contract; initial repositories and untracked files need explicit behavior rather than accidental omission.
3. Replace the tool loop with one provider request/response after compatibility tests cover all retained providers.
4. Delete TUI/commit/repair/doctor packages only after the new path is the sole entry point and tests pass.
5. Finally collapse config/keyring if environment-only configuration is acceptable.

This preserves proven provider payload/auth parsing and git process handling while removing side effects first. A rewrite is justified only if the intended contract is one provider + environment-only config; otherwise it needlessly reintroduces protocol bugs already covered by provider tests.

### Constraints and risks

- “git diff” is underspecified. `git diff` omits staged; `git diff --cached` omits unstaged; `git diff HEAD` fails to cover an unborn `HEAD`; none includes untracked content without reading files or synthesizing diffs.
- Existing scope inspection records diff metadata but not the worktree diff content (`internal/git/service.go:169-173`), so it does not itself supply the desired prompt input.
- Removing secret scanning means arbitrary diff content is sent to the configured provider. That is a product/privacy decision, not merely dead-code cleanup.
- Provider outputs currently arrive as tool calls, not plain assistant text. Reusing `CreateToolCallProvider` unchanged forces the unnecessary runtime; its transport/auth logic is reusable, its response contract is not.
- npm installer/release code is independent of core simplification and should not be deleted merely to reduce binary responsibility.

## Start Here

Open `internal/cli/cli.go` at lines **1005-1172** first. It contains the existing provider construction, prompt, diff tool, and commit side effect in one contiguous flow, making it the safest seam for introducing an output-only message path before deleting packages.

## Acceptance Evidence

- Read-only exploration only; no project source files were modified.
- Commands measured LOC and inspected `git status`; no tests/builds were run because the task requested a read-only architecture map.
- Pre-existing worktree state observed: modified `.github/workflows/release.yml`; untracked `.pi-subagents/`. No staging operation was performed.

```acceptance-report
{
  "criteriaSatisfied": [
    {
      "id": "criterion-1",
      "status": "satisfied",
      "evidence": "Produced a read-only minimal-core map limited to diff-to-message responsibility; no source implementation changes were made."
    },
    {
      "id": "criterion-2",
      "status": "satisfied",
      "evidence": "Cites exact files/ranges, measured LOC and dependency baseline, identifies reusable functions/deletable directories, and documents residual semantic risks."
    }
  ],
  "changedFiles": [],
  "testsAddedOrUpdated": [],
  "commandsRun": [
    {
      "command": "Python source line-count scripts over cmd/internal/scripts",
      "result": "passed",
      "summary": "Measured 10,454 production Go LOC, 5,897 Go test LOC, 349 Node production LOC, and 171 Node test LOC."
    },
    {
      "command": "git status --short",
      "result": "passed",
      "summary": "Observed pre-existing modified .github/workflows/release.yml and untracked .pi-subagents/."
    }
  ],
  "validationOutput": [
    "go.mod contains 6 direct and 21 indirect non-stdlib requirements.",
    "No build/test was necessary or run for this read-only architecture assessment."
  ],
  "residualRisks": [
    "Diff scope semantics (staged/unstaged/untracked/unborn HEAD) require a product decision before implementation.",
    "Exact minimal LOC is an estimate until provider/config compatibility requirements are fixed.",
    "Removing secret scanning changes the privacy boundary for provider-bound diff content."
  ],
  "noStagedFiles": true,
  "diffSummary": "No project source diff; only the required scout artifact was written.",
  "reviewFindings": [
    "warning: internal/cli/cli.go:1129-1138 - current get_diff uses git diff without --cached and therefore omits staged changes.",
    "warning: internal/git/service.go:169-173 - scope inspection retains only diff metadata, not content.",
    "no blockers for the architecture recommendation"
  ],
  "manualNotes": "The safest migration is an output-only path through existing git/provider seams, followed by package deletion after compatibility tests."
}
```
