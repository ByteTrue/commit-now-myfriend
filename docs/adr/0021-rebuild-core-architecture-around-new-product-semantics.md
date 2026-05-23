# Rebuild core architecture around new product semantics

The current Go rewrite preserves much of the staged-first message-generation workflow, which conflicts with the redesigned product semantics. We will reuse reliable low-level pieces where useful, but rebuild the core workflow, provider abstraction, TUI, configuration schema, and command routing around Working Tree Commit, Tool Call Loop, and the simplified command surface.

## Consequences

Existing Go packages are implementation material, not compatibility constraints. Tests and documentation should be rewritten around the new domain language rather than preserving staged-first behavior or standalone split workflows.
