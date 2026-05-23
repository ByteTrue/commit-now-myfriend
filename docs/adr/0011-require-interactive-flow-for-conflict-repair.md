# Require interactive flow for conflict repair

Merge conflict resolution can change source semantics and should not happen in a fully non-interactive run. We will require conflict repair to happen through the Full-screen TUI, while non-interactive Autonomous Commit reports conflicts and fails instead of editing conflicted files.

## Consequences

`cnm auto` does not silently repair conflicts. If `--tui` is present, a conflicted repository may hand off to Interactive Commit; otherwise the run exits with a clear conflict report and leaves resolution to the developer.
