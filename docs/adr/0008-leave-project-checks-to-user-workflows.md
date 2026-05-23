# Leave project checks to user workflows

The CLI should focus on AI-assisted local commit planning, messaging, splitting, and optional repair, not on becoming a project test runner. We will not add `--check` or a check-running Domain Tool; developers who need validation before commits can use their own commands or Git hooks.

## Consequences

`cnm` may still surface Git hook failures from `git commit`, but it does not own discovering, configuring, or running project checks. This keeps the Tool Call Runtime smaller and avoids surprising long-running or side-effectful commands.
