# Release runbook

This document records the current manual release sequence for the Go-native `cnm` binary and the npm wrapper.

## Current assumptions

- Version source of truth: `package.json`
- GoReleaser reads the same semantic version when creating release archives
- GitHub Releases hosts the native archives consumed by `scripts/npm-install.js`
- GitHub Actions release automation lives in `.github/workflows/release.yml`; this runbook explains the same sequence in human terms and serves as the fallback checklist

## Release order

1. **Confirm the working tree is clean**
   ```bash
   git status --short
   ```

2. **Run the normal validation suite**
   ```bash
   npm test
   npm run fmt
   npm run build
   npm pack --dry-run
   ```

3. **Run a local release rehearsal**
   ```bash
   npm run build:release-local
   make go-release-snapshot
   ```

4. **Smoke-check the built binary and wrapper**
   ```bash
   ./dist/go/cnm --help
   ./dist/go/cnm doctor --json
   node scripts/cnm.js --help
   ```

5. **Create/push the release tag**
   - Tag format: `v<package.json version>`
   - Example: `v0.1.4`

6. **Publish GitHub Release artifacts first**
   - Upload GoReleaser archives
   - Upload `checksums.txt`
   - Verify naming matches `.goreleaser.yml`
     - macOS/Linux: `.tar.gz`
     - Windows: `.zip`

7. **Publish the npm wrapper after GitHub Release assets exist**
   - `scripts/npm-install.js` downloads by `package.json` version from GitHub Releases
   - Publishing npm before release assets exist will break `postinstall`
  - The release workflow enforces this ordering by running npm publish only after GitHub Release artifacts are created

8. **Post-publish smoke checks**
   ```bash
   npm install -g commit-now-myfriend@<version>
   cnm --help
   cnm doctor --json
   npx commit-now-myfriend --help
   ```

## Trust model / known gap

The npm installer now downloads the matching release archive, verifies it against `checksums.txt`, and then extracts the binary. Release hardening still depends on:

- correct GitHub Release asset naming
- transport integrity from GitHub Releases
- manual release smoke checks after publish

## Follow-up candidates

- Extend the release workflow with stronger smoke coverage or environment-specific checks
- Continue improving installer and wrapper smoke coverage, especially on Windows
- Add automated Windows installer smoke coverage
