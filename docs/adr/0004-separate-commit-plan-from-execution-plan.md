# Separate commit plan from execution plan

Status: superseded by ADR-0006

The AI should decide commit grouping, messages, and developer-facing intent, but it should not directly control low-level Git command order. We will have the AI produce a declarative Commit Plan and let the Tool Call Runtime derive a validated Execution Plan, keeping local side effects predictable and testable.

## Consequences

Provider outputs must be validated as desired outcomes before execution. Git staging, committing, checking, repair, and recovery remain owned by local code rather than by arbitrary AI-generated command sequences.
