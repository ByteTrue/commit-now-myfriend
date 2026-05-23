# No remote telemetry in first version

The CLI handles local repository content and sends selected context to the user's configured AI provider, so adding a separate telemetry channel would weaken trust. We will not collect remote telemetry in the first version and will limit diagnostics to explicit local debug output.

## Consequences

Debug logging must be opt-in and avoid secrets, full diffs, prompts, and provider responses by default. Any future telemetry proposal should be a separate product decision with clear consent and data boundaries.
