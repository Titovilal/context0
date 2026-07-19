# CLI and Connectors

## What It Does
Context0 is a Cobra-based CLI that manages AI-powered project documentation. It initializes a `.ctx/` directory in any project, generates documentation by delegating to AI CLI tools (connectors), and handles self-updating from GitHub releases. It also provides platform installers and a CI pipeline for multi-architecture builds.

## Main Files
- `main.go` - Entry point; embeds the `defaults/` directory and starts Cobra
- `cmd/root.go` - Root command, `init` command, CLI integration selection, config management, and ASCII banner
- `cmd/sync_docs.go` - `sync-docs` command; three-phase pipeline that generates `project_overview.md`, then all other docs in parallel, then writes a sync log
- `cmd/connector.go` - Connector definitions for Claude, Copilot, Gemini, Codex, and OpenCode; each wraps its respective CLI with the right flags
- `cmd/update.go` - `update` and `version` commands; downloads latest binary from GitHub, handles cross-platform install and automatic migration from the old `mdm` binary name
- `cmd/populate.go` - `populate` command; copies default guides/templates into an existing `.ctx/`, adding only missing files by default (`--force` overwrites existing ones)
- `cmd/style.go` - Shared ANSI styling helpers (colors, step/done/skip indicators) used across all commands
- `defaults/` - Embedded default files (guides, templates, AGENTS.md, CLAUDE.md, GEMINI.md) copied on `ctx init`
- `install.sh` - Shell installer for Linux/macOS; detects OS and architecture, downloads from GitHub releases, installs to `/usr/local/bin`
- `install.ps1` - PowerShell installer for Windows; downloads to `%LOCALAPPDATA%\ctx` and adds it to user PATH
- `.github/workflows/release.yml` - CI workflow triggered by version tags; builds binaries for 5 platform/arch combinations, generates a changelog, and creates a GitHub release

## Flow
1. **`ctx init`** scaffolds `.ctx/` with guides, templates, and an empty docs directory. It offers three modes when `.ctx/` already exists (overwrite, fresh, keep), lets the user pick which AI CLIs to integrate, copies their config files (AGENTS.md plus CLI-specific files like CLAUDE.md), and saves the default connector choice to `.ctx/config.json`
2. **`ctx sync-docs`** generates documentation in three phases: Phase 1 sends the guide and overview template to the chosen connector to produce `project_overview.md`; Phase 2 parses the doc list from the overview, then generates each doc in parallel (up to `--workers` concurrent agents, default 5); Phase 3 writes a sync log in `.ctx/logs/` following the sync log template
3. **`ctx update`** checks GitHub for the latest release, downloads the platform-specific binary, and replaces the current executable. It handles Windows-specific locking, system-directory-to-user-directory migration, and automatic migration from the old `mdm` binary name
4. **`ctx populate`** brings new default guides and templates into an existing `.ctx/`. After a `ctx update` ships new defaults, running this in a project seeds the missing files without touching the user's existing ones; `--force` overwrites existing files with the shipped defaults. Root files (AGENTS.md, CLAUDE.md, GEMINI.md) are left untouched since they depend on the project's CLI selection
5. **`ctx version`** prints the current version, which is injected at build time via `-ldflags`

---

**Best practices:** keep it short, focus on the big picture, use plain language. Avoid code snippets, implementation details, and complex jargon. You can link to other docs in `.ctx/docs/` for related context.
