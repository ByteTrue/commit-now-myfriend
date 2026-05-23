# Respect git hooks by default

Project validation belongs to the developer's workflow, often through Git hooks, even though cnm does not run project checks itself. We will let hooks run during commit creation by default and treat hook failures as commit failures that trigger Index Snapshot restoration and Commit Transaction rollback where safe.

## Consequences

`--no-verify` may exist as an explicit developer override, but it must not be the default or be encouraged during Onboarding. Hook output should be surfaced clearly so the developer can fix their project workflow outside cnm or re-run with an explicit override.

A commit message rejection is the narrow exception: Autonomous Commit may perform one Message Retry by generating a replacement commit message without editing files. Other hook failures stop the flow.
