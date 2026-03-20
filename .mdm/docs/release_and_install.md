# Release & Install

## What It Does
Handles building, releasing, installing, and self-updating the MDM binary across Linux, macOS, and Windows.

## Main Files
- `.github/workflows/release.yml` - GitHub Actions workflow: builds cross-platform binaries on tag push, generates a changelog, and creates a GitHub release with all artifacts
- `cmd/update.go` - `mdm update` checks GitHub for the latest release, downloads the correct binary for the current OS/arch, and replaces itself; `mdm version` prints the current version
- `install.sh` - Curl-based installer for Linux/macOS that downloads the latest release binary to `/usr/local/bin`
- `install.ps1` - PowerShell installer for Windows

## Flow
1. A version tag is pushed to GitHub; the release workflow builds binaries for 5 OS/arch combos and publishes them as a GitHub release
2. New users install via `install.sh` or `install.ps1`, which download the latest binary from GitHub releases
3. Existing users run `mdm update` to self-update; the command handles platform-specific quirks like Windows exe locking and PATH management
