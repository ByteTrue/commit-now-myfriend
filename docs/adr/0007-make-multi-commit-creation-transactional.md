# Make multi-commit creation transactional

Autonomous Commit can create several local commits through one Domain Tool call, but a half-applied sequence would violate the expected Clean Repository Outcome. We will treat multi-commit creation as a Commit Transaction: record the starting HEAD, index state, and working tree fingerprint, then roll back only the commits and staging changes created by that tool call if a later commit fails and no concurrent user changes are detected.

## Consequences

Rollback is an internal recovery operation, not an AI-exposed tool. If the repository changes concurrently or rollback cannot be proven safe, the tool must stop and report the partial state instead of risking unrelated user work.
