# Remove non-interactive repair mode

Conflict repair belongs in the Full-screen TUI because it can change source semantics and needs developer judgment. We will not provide `cnm auto --repair`; non-interactive Autonomous Commit fails on conflicts, while `cnm auto --tui` may hand off to Interactive Commit for Interactive Repair.

## Consequences

Repair is a user-confirmed interactive action, not an autonomous flag. Command design should keep `cnm auto` focused on organizing and creating commits from non-conflicted working tree changes.
