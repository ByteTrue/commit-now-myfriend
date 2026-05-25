# Go-first distribution

This repository ships a Go-native runtime while preserving npm as an installation surface.

## Goals

- Ship `cnm` as a native Go binary for macOS, Linux, and Windows.
- Keep `npm install -g commit-now-myfriend` working.
- Make npm a binary delivery wrapper, not the long-term runtime owner.
- Publish release archives and checksums for direct download users.

## Release artifacts

`goreleaser` is configured to build these targets:

- macOS amd64
- macOS arm64
- Linux amd64
- Linux arm64
- Windows amd64
- Windows arm64

Generated archives follow this pattern:

- `commit-now-myfriend_<version>_darwin_amd64.tar.gz`
- `commit-now-myfriend_<version>_darwin_arm64.tar.gz`
- `commit-now-myfriend_<version>_linux_amd64.tar.gz`
- `commit-now-myfriend_<version>_linux_arm64.tar.gz`
- `commit-now-myfriend_<version>_windows_amd64.zip`
- `commit-now-myfriend_<version>_windows_arm64.zip`

A `checksums.txt` file is emitted alongside the archives.

## Local release rehearsal

Build the Go binary:

```bash
npm run build:go
```

Package the current platform into a local archive:

```bash
npm run build:release-local
```

Run a full release snapshot with GoReleaser:

```bash
make go-release-snapshot
```

For the current manual publish sequence, see `docs/release-runbook.md`.

## npm wrapper layout

The npm package now includes:

- `scripts/cnm.js` — tiny launcher that execs the native binary
- `scripts/npm-install.js` — postinstall downloader for the matching platform binary
- `bin/` — destination for the installed native binary

At install time, `postinstall` downloads the archive matching the host platform and extracts `cnm` into `bin/`.

The launcher script fails fast with a helpful message if the binary was not downloaded successfully.

## Release assumptions

By default, the installer downloads from GitHub Releases using:

- owner: `ByteTrue`
- repo: `commit-now-myfriend`
- tag: `v<package.json version>`

These can be overridden for testing with:

- `CNM_RELEASE_OWNER`
- `CNM_RELEASE_REPO`
- `CNM_RELEASE_BASE_URL`

## Trust model

> Current state: `scripts/npm-install.js` downloads the matching GitHub Release archive, verifies it against `checksums.txt`, and then extracts `cnm`.

That means release hardening now guarantees the downloaded archive matches the published checksum before installation. Remaining trust still depends on correct GitHub Release asset naming and post-publish smoke checks.

## Notes

- The npm package is a thin installer/launcher around the native `cnm` binary.
- Direct release archives are intended for users who do not want to install through npm.
- The repository's main runtime path is Go; legacy TypeScript runtime code is not part of the main branch execution path.
