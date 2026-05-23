# Replace split command with primary commit flows

The project treats the Go rewrite and TUI redesign as a new product shape rather than a compatibility migration. We will remove `cnm split` as a primary or compatibility command and make commit splitting part of the main `cnm` Interactive Commit and `cnm auto` Autonomous Commit flows, because preserving a separate split command would keep old workflow debt in the redesigned CLI.

## Consequences

Existing scripts or habits that call `cnm split` will need to migrate to the new primary flows. Documentation and tests should describe split decisions as part of commit planning rather than as a standalone command.
