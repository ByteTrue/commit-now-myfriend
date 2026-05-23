# Implementation TODO

This TODO tracks the end-to-end work required to deliver the redesigned `cnm` product described in PRD #15 and issues #16-#29.

## Phase 1: Command Surface And Core Entry Points

- [x] #16 Establish the redesigned command surface and run modes.
- [x] Support `cnm` as Interactive Commit entry point.
- [x] Support `cnm auto` as Autonomous Commit entry point.
- [x] Keep `cnm init`, `cnm config`, `cnm doctor`, and version output.
- [x] Remove standalone `cnm split`, `cnm repair`, `cnm check`, and `cnm onboard` from the product surface.
- [x] Update help output to describe only the redesigned command surface.
- [x] Add CLI smoke tests for supported and removed commands.

## Phase 2: Preferences, Secret Store, And Onboarding

- [x] #17 Implement Shared Preference and Private Preference resolution.
- [x] Add Provider Recommendation without letting project config force provider/model.
- [x] Add first-class Message Language.
- [x] Replace product-level `customPrompt` with Standing Instruction.
- [x] Resolve API keys from env, Secret Store, or explicit plaintext opt-in.
- [x] Make Secret Store the default credential save target.
- [x] Add first-run Onboarding for `cnm`.
- [x] Keep `cnm auto` predictable when required configuration is missing.
- [x] Extend `cnm config` around the redesigned preference model.

## Phase 3: Working Tree Commit And Commit Scope

- [x] #18 Inspect staged, unstaged tracked, and untracked non-ignored files by default.
- [x] Support `--staged` for staged-only Commit Scope.
- [x] Support `--no-untracked` to disable Untracked Inclusion.
- [x] Support Git pathspecs after `--`.
- [x] Report no selected changes without provider calls.
- [x] Capture enough index state to support later Index Snapshot recovery.

## Phase 4: Safety, Privacy, And Exposure Controls

- [x] #19 Implement Secret Blocker with redacted output.
- [x] Represent Opaque Changes without pretending to inspect hidden content.
- [x] Enforce Diff Budget.
- [x] Enforce Read Budget.
- [x] Add Context Policy with bounded file reads and diff-only mode.
- [x] Produce AI Exposure Summary for TUI, verbose output, and Machine Output Contract.
- [x] Keep debug logging local, explicit, and conservative.

## Phase 5: Tool Call Runtime

- [x] #20 Implement provider-native Tool Call Loop with a fake provider tracer.
- [x] Expose Domain Tools instead of raw shell or low-level Git commands.
- [x] Validate every tool call before side effects.
- [x] Return structured invalid-call feedback so the AI can continue.
- [x] Enforce Loop Limits for calls, duration, provider retries, and commit retries.
- [x] Enforce Read-before-write Guardrail for repair writes.
- [x] Keep workflow/runtime independent from TUI rendering.
- [x] Drive Autonomous Commit planning and commit creation through provider-native Tool Call Loop.
- [x] Drive Interactive Commit AI Activity through provider-native Tool Call Loop instead of local heuristic planning.
- [x] Remove unreachable staged-first workflow and one-shot commit-message provider compatibility code.

## Phase 6: Autonomous Commit

- [x] #21 Implement single-commit Autonomous Commit.
- [x] Support Compact Run Output.
- [x] Support versioned Machine Output Contract for `--json`.
- [x] Support Commit Plan Preview through `--dry-run`.
- [x] Respect Git hooks by default.
- [x] Support explicit `--no-verify`.
- [x] Add one Message Retry for commit message rejection.
- [x] Reject empty commits.
- [x] Fail non-interactively on conflicts.

## Phase 7: File-level Commit Split And Recovery

- [x] #22 Implement File-level Commit Split.
- [x] Use Conservative Split by default.
- [x] Report Split Limitation for same-file split needs.
- [x] Fall back to one truthful commit when a Split Limitation is safe.
- [x] Defer Hunk-level Commit Split.
- [x] Restore Index Snapshot after failure where safe.
- [x] Wrap multi-commit creation in Commit Transaction.
- [x] Detect concurrent repository changes before rollback.

## Phase 8: Provider Protocols

- [x] #23 Adapt OpenAI Responses native tool calls.
- [x] Adapt OpenAI-compatible chat completions native tool calls.
- [x] Adapt Anthropic Messages native tool calls.
- [x] Adapt Google Gemini native tool calls.
- [x] Expose Provider Capability metadata.
- [x] Keep provider-specific protocol behavior out of workflow code.
- [x] Cover adapters with fake HTTP tests.

## Phase 9: Interactive Commit TUI

- [x] #24 Build Bubble Tea Full-screen TUI shell.
- [x] Follow Focused TUI visual standard through reviewable render structure and focused TUI tests.
- [x] Show Commit Scope first and allow basic scope adjustment.
- [x] Show Commit Scope first.
- [x] Show AI activity.
- [x] Accept Agent Instructions.
- [x] Show AI Exposure Summary and configuration source indicators.
- [x] Show commit grouping and message review state.
- [x] Support narrow terminal and no-color fallback at a basic level.

## Phase 10: TUI Commit Execution

- [x] #25 Require TUI confirmation before side effects.
- [x] Let the developer edit commit messages.
- [x] Execute single-commit and multi-commit flows from TUI.
- [x] Reuse Hook Respect, Message Retry, Index Snapshot, and Commit Transaction behavior.
- [x] Surface hook failures without turning them into project check repair.

## Phase 11: Interactive Repair

- [x] #26 Fail `cnm auto` clearly on conflicts without TUI handoff.
- [x] Support `cnm auto --tui` handoff into conflict context.
- [x] Provide Interactive Repair execution only inside Full-screen TUI.
- [x] Allow repair writes only for eligible conflicted files.
- [x] Require read-before-write for repair writes.
- [x] Require developer confirmation before applying repair writes.
- [x] Do not repair hook failures or project validation failures.

## Phase 12: Doctor And Provider Probing

- [x] #27 Keep `cnm doctor` local-only by default.
- [x] Report Git, config, Secret Store, credential source, provider/model, and Provider Capability metadata.
- [x] Add explicit `cnm doctor --probe-provider`.
- [x] Probe providers with fixed non-repository content.
- [x] Include provider probe status in Machine Output Contract.

## Phase 13: Documentation And Distribution

- [x] #28 Update README and CLI reference around the redesigned command surface.
- [x] Document privacy, Secret Store, Context Policy, AI Exposure Summary, Secret Blocker, and no telemetry.
- [x] Document migration away from staged-first and standalone split workflows.
- [x] #29 Package native binaries for releases.
- [x] Keep npm as a native binary distribution wrapper only.
- [x] Verify runtime behavior does not depend on Node.js.

## Phase 14: Completion Audit And Product Semantics Cleanup

- [x] Re-audit checked phases against `CONTEXT.md`, ADRs, README, CLI behavior, and tests before declaring the redesigned product complete.
- [x] Remove legacy `customPrompt` compatibility from config schema, environment resolution, `cnm init`, `cnm config`, human output, and JSON output.
- [x] Remove local heuristic commit plan/message fallback outside the provider-native Tool Call Loop.
- [x] Align `cnm --help` with the full Interactive Commit flag surface implemented by the parser and documented in README.
- [x] Make no-argument TTY `cnm init` a real interactive Onboarding flow that collects provider, model, style, message language, and API key, saving the key to Secret Store by default.
- [x] Include optional Standing Instruction collection in `cnm init` and first-run `cnm` Onboarding.
- [x] Reject non-TTY no-argument `cnm init` instead of writing incomplete default configuration.
- [x] Verify removed product surfaces (`cnm split`, `cnm repair`, `cnm check`, `cnm onboard`) remain rejected and absent from help output.
- [x] Verify Autonomous Commit and Interactive Commit planning are both driven by provider-native Tool Call Loop behavior.
- [x] Run final release-path verification: `go test ./...`, `go vet ./...`, `make go-build`, `npm pack --dry-run`, CLI help smoke tests, and npm wrapper smoke test.

## Phase 15: Full-screen Charm TUI

- [x] Adopt Bubble Tea + Lip Gloss + Bubbles (`viewport`, `spinner`) and require Go 1.24+.
- [x] Enable alternate screen, mouse cell motion, and `WindowSizeMsg`-driven layout for the Interactive Commit TUI.
- [x] Render the Interactive Commit screen as a two-pane Charm layout: scope list on the left, active screen + diff/commit-plan/repair on the right, with a header strip, AI Exposure summary, and contextual key hint footer.
- [x] Wire a real diff provider so the right pane shows working-tree diffs (including synthetic untracked-file diffs) for the highlighted file.
- [x] Wire a real file reader so Interactive Repair can show conflict files with conflict markers highlighted.
- [x] Add a Charm-style spinner during AI activity in both the Interactive Commit TUI and the header indicator.
- [x] Build a Full-screen `cnm config` panel with editable values, choice pickers, and Secret Store-aware writes; keep non-TTY `cnm config get/list/set/unset` unchanged for scripts.
- [x] Build a Full-screen `cnm init` Onboarding wizard (provider → model → optional base URL → style → language → standing instruction → API key) that masks secrets and saves to the Secret Store by default; legacy stdio prompter still serves test injection.
- [x] Render `cnm doctor` as a bordered Charm dashboard in TTY (status badge, checks panel, effective configuration, optional probe, issues) while preserving the existing line-based and JSON outputs for scripts.
- [x] Inject `cnm --version` from `package.json` at build time so npm and direct binary installs report the same version.
- [x] Drive a real provider-native Tool Call Loop end-to-end against the user-supplied AI gateway: produce a multi-commit Commit Plan Preview via `cnm auto --dry-run --json` and create real local commits via `cnm auto`.
- [x] Make Tool Call Loop continuations resilient: send tool definitions on every request, force tool use where the provider supports it, and inject a structured reminder turn when the model returns text-only responses.
- [x] Add JSON Schemas (with `required` fields) for every Domain Tool so models know how to call `create_commits`, `repair_file`, `read_file`, `preview_commit`, `finish`, and `abort`.
