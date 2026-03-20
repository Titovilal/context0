# MDM — The Middleman

One agent to rule them all.

A CLI that turns any AI coding assistant into a Middleman — an orchestrator that delegates work to subagents without writing code itself.

## The problem

When working on complex projects with AI coding assistants, you end up managing multiple instances manually. MDM externalizes that cognitive load: you talk to one agent, and it decides how to split the work across subagents.

## How it works

MDM initializes a `.mdm/` directory in your project with guides, templates, and documentation. When you run `mdm open`, it launches your chosen AI CLI as a Middleman agent, injecting the orchestration guide so it knows how to behave: delegate in parallel, don't write code, return control immediately.

## Supported CLIs

| CLI | Instruction file | Status |
|---|---|---|
| Claude Code | `CLAUDE.md` + `AGENTS.md` | Tested |
| Codex | `AGENTS.md` | Untested |
| Copilot | `AGENTS.md` | Untested |
| Gemini CLI | `GEMINI.md` + `AGENTS.md` | Untested |
| OpenCode | `AGENTS.md` | Untested |

## Installation

### Linux / macOS

```bash
curl -sL https://raw.githubusercontent.com/Titovilal/middleman/main/install.sh | sh
```

### Windows (PowerShell)

```powershell
irm https://raw.githubusercontent.com/Titovilal/middleman/main/install.ps1 | iex
```

### From source

```bash
go install github.com/Titovilal/middleman@latest
mv $(go env GOPATH)/bin/middleman $(go env GOPATH)/bin/mdm
```

## Quick start

```bash
# Initialize .mdm/ in your project — asks which CLIs to integrate
mdm init

# Generate project documentation
mdm sync-docs

# Open a Middleman session with a request
mdm open "refactor the auth module and add tests"
```

## Commands

| Command | Description |
|---|---|
| `mdm init` | Initialize `.mdm/` and copy instruction files to project root |
| `mdm sync-docs` | Generate/update `.mdm/docs/` using an AI CLI |
| `mdm open [request]` | Open a Middleman session with a user request |
| `mdm update` | Self-update to the latest version |
| `mdm version` | Print current version |

### `mdm init` flags

| Flag | Description |
|---|---|
| `--force`, `-f` | Overwrite existing files without asking |
| `--clis` | Comma-separated CLIs to integrate (e.g. `claude,gemini`) |
| `--default` | Default CLI for sync-docs and open |
| `--sync` | Run sync-docs after init |

### `mdm open` / `mdm sync-docs` flags

| Flag | Description |
|---|---|
| `--connector`, `-c` | AI CLI to use (overrides default from config) |

## Project structure

```
your-project/
├── AGENTS.md              — subagent behavior (always created)
├── CLAUDE.md              — Claude-specific instructions (if selected)
├── GEMINI.md              — Gemini-specific instructions (if selected)
└── .mdm/
    ├── config.json        — default CLI and settings
    ├── guides/
    │   ├── the_middleman.md    — how the Middleman operates
    │   └── how_to_sync_docs.md — guide for doc generation
    ├── templates/         — templates for generating docs
    └── docs/              — generated project documentation
```

## Architecture

```
mdm/
├── main.go          # Entry point, embeds defaults/
├── cmd/
│   ├── root.go      # CLI setup, init command, banner
│   ├── open.go      # mdm open — launch Middleman session
│   ├── sync_docs.go # mdm sync-docs — doc generation
│   ├── connector.go # AI CLI connectors (claude, codex, gemini, copilot, opencode)
│   └── update.go    # Self-update
└── defaults/        # Embedded files copied on mdm init
    ├── AGENTS.md
    ├── CLAUDE.md
    ├── GEMINI.md
    ├── guides/
    └── templates/
```
