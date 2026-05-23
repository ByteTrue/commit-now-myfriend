# Simplify command surface around interactive and auto

The redesigned CLI should center on two commit entry points: `cnm` for Interactive Commit and `cnm auto` for Autonomous Commit. We will keep `init`, `config`, and `doctor` for setup and diagnostics, and remove standalone `split`, `repair`, `check`, and `onboard` commands because those concerns belong inside the primary flows or outside cnm.

## Consequences

Commit splitting is part of `cnm` and `cnm auto`, conflict repair is part of the Full-screen TUI, project checks stay in user workflows or Git hooks, and Onboarding is reached through `cnm` or `cnm init` rather than a separate command.
