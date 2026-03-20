# Init & Defaults

## What It Does
Bootstraps the `.mdm/` directory in a project with guides, templates, and configuration. Also copies root-level agent instruction files (AGENTS.md, CLAUDE.md, GEMINI.md) so AI CLIs know how to behave.

## Main Files
- `cmd/root.go` - `mdm init` command implementation: scaffolds `.mdm/`, runs the CLI selection wizard, saves `config.json`, optionally triggers `sync-docs`
- `main.go` - Embeds the `defaults/` directory via `go:embed` and passes it to `cmd`
- `defaults/` - Embedded default files: guides (`the_middleman.md`, `how_to_sync_docs.md`), templates (`doc_template.md`, `project_overview_template.md`), and root files (`AGENTS.md`, `CLAUDE.md`, `GEMINI.md`)
- `.mdm/config.json` - Stores the user's chosen default CLI

## Flow
1. User runs `mdm init`; the command creates `.mdm/` and writes embedded defaults (guides, templates) into it
2. An interactive wizard asks which AI CLIs to integrate and which is the default; the selection is saved to `.mdm/config.json`
3. Root-level instruction files (AGENTS.md, plus CLI-specific files like CLAUDE.md) are copied to the project root so AI agents can read them
