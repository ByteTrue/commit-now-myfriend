# Use full-screen TUI for interactive commit

The redesigned CLI needs Interactive Commit to support commit planning, diff review, message editing, replanning, and natural-language developer feedback in one flow. We will build `cnm` around a Full-screen TUI rather than a sequence of prompts, because the interaction model is closer to a terminal application than to a confirmation wizard.

## Consequences

`cnm auto` remains a compact non-full-screen flow for fast and scriptable execution. The Full-screen TUI decision mainly shapes `cnm`, where richer review and confirmation are part of the product value.
