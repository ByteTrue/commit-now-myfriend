# Store API keys in system secret store by default

The redesigned CLI should not normalize plaintext API keys in `~/.cnm/config.json`. We will store API keys in the platform Secret Store by default, keep environment variables as explicit overrides, and require a clear opt-in for plaintext storage.

## Consequences

Configuration loading must resolve API key source separately from ordinary config values. Onboarding and diagnostics must show whether the key comes from environment, Secret Store, plaintext config, or is missing, and project configuration must never store API keys.
