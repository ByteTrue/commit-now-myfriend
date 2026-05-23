# Defer hunk-level commit split

Hunk-level Commit Split would allow different hunks in one file to belong to different commits, but it requires reliable patch application, index recovery, and TUI hunk selection. We will defer it from the first version and support File-level Commit Split only, reporting same-file split needs as a limitation rather than attempting unsafe partial staging.

## Consequences

`cnm auto` fails when selected changes require same-file splitting for correctness. Interactive Commit can explain the limitation and let the developer adjust the scope or files manually, but it does not provide hunk selection in the first version.
