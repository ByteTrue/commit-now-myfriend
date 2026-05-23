# Plan commits from working tree by default

The previous CLI centered on staged diffs, but the redesigned product aims to let a developer run one command and have AI plan the commit work. We will make `cnm` and `cnm auto` plan from the working tree by default, considering staged, unstaged tracked, and untracked files while preserving safety blockers and providing explicit staged-only or path-limited modes.

## Consequences

The Git index becomes an implementation detail of commit planning rather than the main user input boundary. Existing staged-first workflow code and documentation should be replaced, not treated as the default behavior for the new product shape.
