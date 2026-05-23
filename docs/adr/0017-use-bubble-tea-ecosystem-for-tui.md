# Use Bubble Tea ecosystem for TUI

Interactive Commit needs a polished Full-screen TUI while keeping the CLI Go-native and lightweight. We will build the TUI with Bubble Tea, Lip Gloss, and Bubbles, while keeping workflow and Tool Call Runtime code independent of the TUI framework.

## Consequences

The TUI can use Bubble Tea's model/update/view architecture for testable terminal state, but non-interactive `cnm auto` must not depend on full-screen rendering. Visual styling should follow the Focused TUI standard rather than decorative terminal effects.
