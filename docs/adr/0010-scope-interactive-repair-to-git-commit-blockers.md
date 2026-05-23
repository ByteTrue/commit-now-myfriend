# Scope interactive repair to git commit blockers

Interactive Repair should not become a general project test, lint, or hook-failure fixer. We will scope it to Git commit blockers such as merge conflicts, while Git hook failures stop the flow and are surfaced to the developer.

## Consequences

If hooks modify files or reject a commit, cnm reports the failure and restores or rolls back where safe rather than asking AI to infer a project-specific fix. Interactive Commit may let the developer edit a rejected commit message and retry, but repair does not own arbitrary project validation failures.
