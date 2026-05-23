# Support four provider protocols as first class

The redesigned product should not narrow the CLI to a single AI provider even though native tool calling differs across APIs. We will support OpenAI Responses, OpenAI-compatible chat completions, Anthropic Messages, and Google Gemini as first-class Provider Protocols by adapting each one into the same local Tool Call Runtime contract.

## Consequences

Provider integration becomes a core architecture concern rather than a thin HTTP detail. Each Provider Protocol needs native tool-call capability tests so `cnm auto` and `cnm` do not depend on provider-specific behavior leaking into the workflow layer.
