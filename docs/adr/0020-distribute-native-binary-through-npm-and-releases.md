# Distribute native binary through npm and releases

The CLI should run as a native Go binary while remaining easy for JavaScript ecosystem users to install. We will publish native release binaries and keep npm as a distribution wrapper for those binaries rather than a TypeScript or Node.js runtime path.

## Consequences

Runtime behavior must not depend on Node.js. Packaging work should focus on selecting or installing the correct platform binary, while direct release downloads remain a first-class installation path.
