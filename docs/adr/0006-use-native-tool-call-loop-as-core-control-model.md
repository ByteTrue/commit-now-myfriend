# Use native tool-call loop as core control model

The product goal is for AI to drive the commit workflow through provider-native tool calls, not by emitting a declarative JSON plan that local code later interprets. We will use a Tool Call Loop where each supported Provider Protocol maps native tool calls into the same local Tool Call Runtime, because provider-native tool feedback gives the model a chance to correct invalid calls without turning free-form JSON parsing into a workflow blocker.

The runtime exposes Domain Tools rather than low-level Git or shell commands. Tool contracts should include stability guardrails such as requiring reads before repair writes, so prompt instructions and local validation reinforce each other.

## Consequences

Every first-class provider must implement native tool-call adaptation for the flows it supports. The local runtime still validates and may reject individual tool calls, but the AI remains in the loop and can continue after receiving structured tool results. Low-level Git sequencing remains owned by local code behind Domain Tools.
