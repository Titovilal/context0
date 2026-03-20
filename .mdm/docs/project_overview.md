# Project Overview

## What It Does
MDM (The Middleman) is a CLI tool that manages project documentation and orchestrates AI coding agents. It initializes a `.mdm/` directory in any project, uses AI CLIs (Claude, Gemini, Codex, Copilot, OpenCode) to generate and maintain documentation, and can launch a "Middleman" agent session that delegates work to subagents.

## Main Files
- `main.go` - Entry point; embeds the `defaults/` directory and calls `cmd.Execute()`
- `cmd/root.go` - Root command, `mdm init` (scaffolds `.mdm/`, CLI wizard, config), banner, global flags
- `cmd/connector.go` - AI CLI connector abstraction with five implementations (Claude, Copilot, Gemini, Codex, OpenCode)
- `cmd/sync_docs.go` - `mdm sync-docs` generates/updates docs by prompting an AI CLI with guides and templates
- `cmd/open.go` - `mdm open` launches a Middleman agent session through an AI CLI
- `cmd/update.go` - `mdm update` self-updates from GitHub releases; `mdm version` prints version
- `defaults/` - Embedded default files (guides, templates, agent instructions) copied during init
- `.github/workflows/release.yml` - CI/CD: cross-platform builds and GitHub releases on tag push

## Flow
1. User runs `mdm init` to scaffold `.mdm/` with guides, templates, config, and agent instruction files
2. User runs `mdm sync-docs` to generate project documentation — an AI CLI reads the codebase and writes docs into `.mdm/docs/`
3. User runs `mdm open <request>` to start a Middleman session that delegates work to AI subagents

## Documentation available in `.mdm/docs/`
- **`cli_commands.md`** — All CLI commands (init, sync-docs, open, update, version) and the Cobra command structure
- **`connectors.md`** — AI CLI connector abstraction and the five implementations
- **`init_and_defaults.md`** — The init command, embedded defaults, config, and CLI selection wizard
- **`release_and_install.md`** — GitHub Actions release workflow, install scripts, and self-update mechanism
