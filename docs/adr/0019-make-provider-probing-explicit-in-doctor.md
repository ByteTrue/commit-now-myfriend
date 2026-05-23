# Make provider probing explicit in doctor

Doctor should diagnose local setup by default without sending data to an AI provider or consuming tokens. We will keep provider API and native tool-call capability probing behind an explicit `cnm doctor --probe-provider` option using fixed non-repository test content.

## Consequences

Default doctor output can report configured provider and known adapter capability metadata, but it must not claim the remote provider works unless probing was requested. Machine output should indicate whether provider probing ran.
