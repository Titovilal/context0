# CLI Commands

## What It Does
Implements the full MDM command-line interface using Cobra. Each command handles its own logic directly — there is no separate orchestrator or domain layer; all behavior lives in the `cmd/` package.

## Main Files
- `cmd/root.go` - Root command, `init` command, global `--workdir` flag, `.mdm/` initialization from embedded defaults, config load/save, CLI selection wizard, banner display
- `cmd/connector.go` - Connector abstraction and five AI CLI implementations (Claude, Copilot, Gemini, Codex, OpenCode) that execute prompts via subprocesses
- `cmd/sync_docs.go` - `mdm sync-docs` reads guides and templates then delegates doc generation to an AI CLI
- `cmd/open.go` - `mdm open` launches a Middleman session by injecting the middleman guide and user request into an AI CLI
- `cmd/update.go` - `mdm update` self-updates from GitHub releases; `mdm version` prints the current version

## Flow
1. User runs `mdm init` to scaffold `.mdm/` with guides, templates, config, and root-level agent instruction files (AGENTS.md, CLAUDE.md, etc.)
2. User runs `mdm sync-docs` to generate project documentation — the command builds a prompt from guides/templates and sends it to the configured AI CLI
3. User runs `mdm open <request>` to start a Middleman session — the command injects the middleman guide and fires the request through the AI CLI
