# Treat secret detection as selected scope blocker

Autonomous commit flows should not silently work around suspected credentials. If secret detection finds a suspected credential inside the Commit Scope, cnm blocks the selected flow and creates no commits rather than excluding that file and continuing.

## Consequences

Developers can narrow the Commit Scope or use the Full-screen TUI to exclude unrelated files, but cnm does not provide a first-version `--ignore-secrets` path. Output must identify affected files and detection categories without printing secret values.
