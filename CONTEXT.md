# Commit Now Myfriend

Commit Now Myfriend is a local AI-assisted Git commit tool. Its language distinguishes interactive human-guided commits from autonomous local commit execution.

## Language

**Autonomous Commit**:
An AI-led local commit flow that can create one or more local Git commits without step-by-step human confirmation. It may modify the index and working tree only within the selected commit workflow, and it never pushes to a remote.
_Avoid_: auto mode, hands-free commit, agent commit

**Interactive Repair**:
An explicitly enabled AI-assisted repair flow inside the Full-screen TUI that may edit working tree files to resolve Git commit blockers such as merge conflicts before committing. It does not run in non-interactive Autonomous Commit, does not repair project validation or Git hook failures, and still never pushes to a remote.
_Avoid_: autonomous repair, auto fix, silent repair, background coding

**Interactive Commit**:
A human-guided commit flow where the developer confirms each meaningful step before the tool creates commits or changes commit grouping.
_Avoid_: manual mode, normal commit

**Full-screen TUI**:
The primary terminal interface for Interactive Commit, presenting commit planning, review, confirmation, and developer feedback as one cohesive screen-based workflow.
_Avoid_: prompt flow, wizard, question sequence

**Focused TUI**:
The visual design standard for the Full-screen TUI: polished, readable, and workflow-dense without decorative complexity. It supports theme and terminal capability differences while prioritizing commit review and control.
_Avoid_: flashy TUI, decorative terminal UI, prompt skin

**Compact Run Output**:
The default human-readable output for non-interactive runs, showing the terminal result and created commits without full prompts, diffs, or provider internals. Detailed execution belongs to verbose or JSON output.
_Avoid_: quiet mode, verbose log, raw trace

**Machine Output Contract**:
The stable versioned JSON output schema for non-interactive commands. It is intended for scripts and must not be mixed with Full-screen TUI rendering.
_Avoid_: json mode, raw output, debug dump

**Onboarding**:
The guided setup flow that collects the minimum configuration needed before the tool can request AI help, such as provider, model, API key, and commit style. It belongs primarily to the interactive experience rather than to autonomous runs.
_Avoid_: init script, setup wizard, config prompts

**Message Language**:
The configured natural language for generated commit messages, separate from commit style. It may be inferred from recent commits or set explicitly during Onboarding.
_Avoid_: language prompt, custom language, commit locale

**Standing Instruction**:
A persistent developer preference included in each Tool Call Loop, used for personal or repository-specific commit behavior that is not covered by structured settings. It is distinct from an Agent Instruction, which affects only the active workflow.
_Avoid_: custom prompt, default prompt, global chat instruction

**Shared Preference**:
A repository-level configuration preference that can be committed for the team, such as message style, message language, or repository-specific commit guidance. Shared Preferences can influence AI behavior and must be visible to the developer.
_Avoid_: project config, team setting

**Private Preference**:
A developer-local configuration preference that should not be committed for the team, such as API credentials, provider selection, model choice, theme, keybindings, or personal Standing Instructions.
_Avoid_: user config, local setting

**Provider Recommendation**:
A repository-level suggestion for provider or model shown during Onboarding and diagnostics. It does not override the developer's Private Preference.
_Avoid_: project provider, enforced model, shared provider

**Secret Store**:
The system-backed credential storage used for API keys by default. It is preferred over plaintext config, while environment variables remain supported for explicit temporary overrides.
_Avoid_: config api key, saved key, plaintext secret

**Selected Changes Clean**:
The desired end state after a commit flow: the selected working tree changes have been committed, no commit workflow is left half-applied, and no remote push has occurred. Unselected paths and ignored files are outside this cleanliness guarantee.
_Avoid_: clean repo, completely clean repository, finished state

**Commit Scope**:
The set of working tree changes selected for a commit flow. It is defined by the default whole-working-tree behavior, `--staged`, and optional Git pathspecs after `--`.
_Avoid_: file filter, target files, include list

**Secret Blocker**:
A suspected credential or sensitive value detected inside the Commit Scope. It blocks the selected commit flow instead of being silently skipped.
_Avoid_: secret warning, skipped secret, sensitive file notice

**Commit Plan Preview**:
A no-side-effect preview of the commits the tool intends to create, including grouping and commit messages. It does not edit the index or run project checks.
_Avoid_: dry execution, simulated commit, test run

**Working Tree Commit**:
A commit flow that plans from the repository's current working tree rather than only from already staged changes. It can consider staged files, unstaged tracked files, and untracked files while still respecting ignore rules and safety blockers.
_Avoid_: staged commit, add-all commit, dirty tree commit

**Untracked Inclusion**:
The default rule that untracked, non-ignored files inside the Commit Scope can be considered for commit creation. It must be visible to the developer and can be disabled for a run.
_Avoid_: add new files, git add all, untracked staging

**Index Snapshot**:
A record of the Git index state captured before an autonomous flow temporarily stages or unstages files. It lets the tool restore the developer's staging state when automation fails before completing its commits.
_Avoid_: staging backup, git restore point, index checkpoint

**Commit Transaction**:
The transaction boundary around one commit Domain Tool call, especially multi-commit creation. It lets the tool either complete the requested local commits or roll back only the commits and index changes created by that tool call when strict safety checks allow it.
_Avoid_: rollback, git transaction, batch commit

**Hook Respect**:
The rule that cnm lets configured Git hooks run during commit creation and treats hook failures as commit failures. Bypassing hooks requires an explicit developer choice.
_Avoid_: hook bypass, no-verify default, silent hook handling

**Message Retry**:
A narrow retry after a commit message rejection where the AI may generate a replacement commit message without changing files. It is the only automatic hook-failure retry allowed in Autonomous Commit.
_Avoid_: hook repair, validation fix, retry loop

**TUI Handoff**:
An explicit transition from Autonomous Commit into the Full-screen TUI when automation cannot safely continue without developer judgment. It is opt-in so automated runs remain predictable.
_Avoid_: auto fallback, surprise prompt, interactive fallback

**Tool Call Runtime**:
The controlled execution layer that exposes repository-specific tools through native provider tool calls and validates every requested action before applying it locally. It is not an arbitrary shell and only offers capabilities allowed by the active flow.
_Avoid_: shell access, agent terminal, raw command execution

**Tool Call Loop**:
The iterative exchange where the AI requests native tool calls, the local runtime executes or rejects them, and the AI continues until the commit workflow reaches a terminal outcome. It is the core control model for Autonomous Commit and Interactive Commit.
_Avoid_: JSON plan, generated plan, command script

**Loop Limit**:
A hard bound on Tool Call Loop execution, such as maximum tool calls, duration, provider retries, or commit retries. It protects local repository state, API cost, and user time.
_Avoid_: timeout, retry setting, safety cap

**Domain Tool**:
A workflow-level tool exposed to the AI, such as creating commits, previewing changes, or repairing files. Domain Tools hide low-level Git command sequencing behind validated local behavior.
_Avoid_: git command, shell command, raw tool

**Read-before-write Guardrail**:
A tool contract requiring the AI to inspect current file content or repository state before requesting a write or repair action. It exists to reduce stale edits, accidental overwrites, and unsupported tool-call sequences.
_Avoid_: edit rule, read first, prompt reminder

**Diff Budget**:
The maximum diff content a repository tool returns in one response. It keeps provider context, cost, and latency bounded while allowing the AI to request narrower diffs as needed.
_Avoid_: diff truncation, max diff size, context limit

**Read Budget**:
The maximum file content a repository tool returns during a run. It allows bounded context beyond diffs while keeping provider exposure, cost, and latency visible and controlled.
_Avoid_: file read limit, context budget, privacy cap

**Context Policy**:
The privacy and context rule that controls what repository content AI tools may read, such as bounded file reads or diff-only operation. It changes AI quality and data exposure and must be visible in the run.
_Avoid_: privacy mode, context mode, data setting

**AI Exposure Summary**:
A user-visible summary of what repository content, configuration sources, and preferences were exposed to the provider during a run. It supports transparency without printing secret values.
_Avoid_: privacy log, prompt dump, telemetry summary

**Opaque Change**:
A selected change whose contents are unavailable or impractical for AI inspection, such as binary files or oversized files beyond the Diff Budget. It can be committed, but messages must describe only known metadata and avoid pretending to understand hidden content.
_Avoid_: binary diff, unknown file, large file warning

**Provider Protocol**:
The provider-specific API shape used to request native tool calls, such as OpenAI Responses, OpenAI-compatible chat completions, Anthropic Messages, or Google Gemini. Provider Protocols are adapters into the same local Tool Call Runtime concepts.
_Avoid_: model format, backend format, provider mode

**Provider Capability**:
The set of product flows a Provider Protocol can support after local adaptation, such as native tool calling, streaming progress, or repair assistance. A provider is available for a flow only when its native tool-call capability contract is implemented and tested.
_Avoid_: provider support, model support, API support

**Agent Instruction**:
A natural-language developer instruction that can change the commit plan, message style, or repair direction within the active flow. It is workflow control, not casual chat.
_Avoid_: chat message, prompt text, user note

**File-level Commit Split**:
A commit split where every changed file belongs to exactly one planned commit. It is the first supported automatic split boundary for Autonomous Commit.
_Avoid_: simple split, file grouping

**Conservative Split**:
The default commit splitting strategy: split only when changed files clearly represent independent intentions, and keep code, tests, and docs together when they support the same change.
_Avoid_: safe split, minimal split, cautious split

**Hunk-level Commit Split**:
A commit split where different diff hunks from the same file may belong to different planned commits. It is more precise than File-level Commit Split and is treated as an advanced or interactive capability.
_Avoid_: line split, partial split

**Split Limitation**:
A reported case where the selected changes would ideally need Hunk-level Commit Split, but the current flow can only commit them together or ask the developer to adjust scope. It is a warning when a truthful single commit is still acceptable, and a blocker only when the selected changes cannot be safely committed together.
_Avoid_: split failure, hunk error, grouping bug

## Example Dialogue

Developer: "I want `cnm auto` to run an Autonomous Commit."

Domain expert: "Then the tool may create several local commits without asking you, but it must not push."

Developer: "Does success mean the whole repository has no changes?"

Domain expert: "No. Success means Selected Changes Clean; unselected paths and ignored files are outside the guarantee."

Developer: "How do I limit what cnm commits?"

Domain expert: "Use Commit Scope: pass Git pathspecs after `--`, optionally combined with `--staged`."

Developer: "Can cnm just skip files that look like secrets?"

Domain expert: "No. A Secret Blocker stops the selected flow; the developer must narrow the Commit Scope or resolve the secret."

Developer: "Does Interactive Commit start from a different scope than Autonomous Commit?"

Domain expert: "No. Both start from the same default working tree scope, but Interactive Commit lets the developer adjust it in the Full-screen TUI before planning."

Developer: "Can it resolve conflicts too?"

Domain expert: "Only through Interactive Repair in the Full-screen TUI. Non-interactive Autonomous Commit fails when conflicts are present."

Developer: "Does the AI get a terminal?"

Domain expert: "No. It uses the Tool Call Runtime, which exposes validated Domain Tools according to the active flow."

Developer: "Does the AI output a JSON plan?"

Domain expert: "No. The core control model is a Tool Call Loop; the AI uses native tool calls and receives tool results until the workflow is done."

Developer: "Can the AI loop forever trying tools?"

Domain expert: "No. Loop Limits stop the flow and trigger recovery when the run exceeds bounded execution."

Developer: "Can the AI edit files without looking first?"

Domain expert: "No. Repair tools follow the Read-before-write Guardrail so writes are based on current repository state."

Developer: "Does cnm send the whole diff at once?"

Domain expert: "No. Repository tools enforce a Diff Budget and the AI requests narrower diffs when needed."

Developer: "Can AI read files beyond the diff?"

Domain expert: "Yes, within the Read Budget and Commit Scope. Privacy-focused runs can use a diff-only context policy."

Developer: "How do I stop AI from reading full files?"

Domain expert: "Use the diff-only Context Policy for that run or as a preference."

Developer: "How do I know what AI saw?"

Domain expert: "The TUI shows an AI Exposure Summary, and verbose or machine output can include the same summary for non-interactive runs."

Developer: "Can cnm commit binary files?"

Domain expert: "Yes, as Opaque Changes. The AI may use metadata but must not invent details about content it could not inspect."

Developer: "Do all AI providers use the same API format?"

Domain expert: "No. Each Provider Protocol is adapted locally, and its Provider Capability determines which flows it can run."

Developer: "Can Autonomous Commit split everything perfectly?"

Domain expert: "It can rely on File-level Commit Split first. Hunk-level Commit Split is a separate capability because it changes how precisely the tool edits the index."

Developer: "How eager is cnm to split commits?"

Domain expert: "The default is Conservative Split; developers can request finer splitting through configuration or Agent Instructions."

Developer: "What if one file contains two logical changes?"

Domain expert: "That is a Split Limitation. Autonomous Commit may commit it as one honest commit with a warning when safe, because Hunk-level Commit Split is not first-version behavior."

Developer: "Can I tell the TUI what I want in plain language?"

Domain expert: "Yes. That is an Agent Instruction; the AI can replan the workflow from it, while side effects in Interactive Commit still wait for confirmation."

Developer: "What happens when `cnm auto` gets stuck?"

Domain expert: "It fails predictably unless TUI Handoff was explicitly requested."

Developer: "What if the stuck state is a merge conflict?"

Domain expert: "Conflict repair always requires Interactive Commit; non-interactive runs report the conflict and stop."

Developer: "What does `--dry-run` show?"

Domain expert: "It produces a Commit Plan Preview. Project checks are outside cnm and can be handled by Git hooks or the developer's own workflow."

Developer: "Do I need to stage files before running `cnm auto`?"

Domain expert: "No. The redesigned flow is a Working Tree Commit by default, with explicit options to restrict it to staged changes or selected paths."

Developer: "Will new files be included?"

Domain expert: "Yes, through Untracked Inclusion, unless the run disables it or the files are ignored or blocked by safety rules."

Developer: "What if automation fails after changing what is staged?"

Domain expert: "Autonomous flows use an Index Snapshot so they can restore the developer's staging state when they cannot complete."

Developer: "What if a multi-commit tool call succeeds halfway?"

Domain expert: "It runs inside a Commit Transaction, so the tool tries to roll back only its own partial local commits when safety checks allow."

Developer: "Can cnm skip my Git hooks to make automation succeed?"

Domain expert: "No. Hook Respect means hooks run by default; bypassing them requires an explicit option."

Developer: "What if only the commit message is rejected?"

Domain expert: "Autonomous Commit may use one Message Retry, but other hook failures stop the flow."

Developer: "If I run plain `cnm`, I want to approve each step."

Domain expert: "That is an Interactive Commit; it uses the Full-screen TUI so the tool can suggest grouping, messages, and repairs while waiting for confirmation before applying them."

Developer: "How fancy should the terminal UI be?"

Domain expert: "It should be a Focused TUI: polished and dense enough for review, without visual complexity that distracts from committing."

Developer: "How much should `cnm auto` print?"

Domain expert: "Use Compact Run Output by default: enough to show what happened, not a full execution trace."

Developer: "Can scripts depend on JSON output?"

Domain expert: "Yes, through the Machine Output Contract on non-interactive commands."

Developer: "What happens before I configure a provider?"

Domain expert: "Interactive Commit can enter Onboarding, while Autonomous Commit fails predictably unless TUI Handoff was requested."

Developer: "Is Chinese commit output just a custom prompt?"

Domain expert: "No. Message Language is a first-class preference; custom prompts refine behavior beyond language."

Developer: "Where do I put my long-term preferences?"

Domain expert: "Use a Standing Instruction. Use an Agent Instruction for one-off directions during the active workflow."

Developer: "Can a repository configure AI behavior for everyone?"

Domain expert: "Yes, through Shared Preferences, but the TUI and diagnostics must show their source because they affect AI behavior."

Developer: "Can a repository force my AI provider?"

Domain expert: "No. It can provide a Provider Recommendation, but provider and model remain Private Preferences."

Developer: "Where does cnm save my API key?"

Domain expert: "By default in the Secret Store. Plaintext config requires an explicit opt-in and must be diagnosed as such."
