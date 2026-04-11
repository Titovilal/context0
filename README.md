```
   ██████╗ ██████╗ ███╗   ██╗████████╗███████╗██╗  ██╗████████╗ ██████╗
  ██╔════╝██╔═══██╗████╗  ██║╚══██╔══╝██╔════╝╚██╗██╔╝╚══██╔══╝██╔═████╗
  ██║     ██║   ██║██╔██╗ ██║   ██║   █████╗   ╚███╔╝    ██║   ██║██╔██║
  ██║     ██║   ██║██║╚██╗██║   ██║   ██╔══╝   ██╔██╗    ██║   ████╔╝██║
  ╚██████╗╚██████╔╝██║ ╚████║   ██║   ███████╗██╔╝ ██╗   ██║   ╚██████╔╝
   ╚═════╝ ╚═════╝ ╚═╝  ╚═══╝  ╚═╝   ╚══════╝╚═╝  ╚═╝   ╚═╝    ╚═════╝
```

# Context0

AI agents code better when they understand your project first.

Context0 creates a `.ctx/docs/` folder with plain text files that describe your codebase. No background processes, no magic — just a self-maintained directory of docs that agents read before they write.

Every doc is surgical: short, dense, and optimized for any model intelligence level. As if you were going to the moon and only had 16KB of space to explain your entire project.

Works with any AI coding CLI: Claude Code, Codex, Copilot, Gemini CLI, OpenCode.

## Installation

### Linux / macOS

```bash
curl -sL https://raw.githubusercontent.com/Titovilal/context0/main/install.sh | sh
```

### Windows (PowerShell)

```powershell
irm https://raw.githubusercontent.com/Titovilal/context0/main/install.ps1 | iex
```

## Quick start

```bash
cd your-project
ctx init                  # scaffold .ctx/, pick your CLIs
ctx sync-docs             # generate documentation
```

## How agents read the docs

```
CLAUDE.md / GEMINI.md          "read AGENTS.md"
        │
        ▼
    AGENTS.md                  "read project_overview.md first"
        │
        ▼
  project_overview.md          high-level summary + list of docs
        │
        ▼
  [feature_doc].md             context for a specific area
        │
        ▼
    source files               the agent starts coding with context
```

## Fully customizable

Everything Context0 generates is meant to be changed. Edit the templates to control what gets documented, rewrite `AGENTS.md` to change how agents behave, tweak the guides to match your workflow. The defaults ship with patterns refined over years of daily AI-assisted coding — from the first day GPT-3.5 launched through the agentic era — but they're a starting point, not a constraint.

## Commands

| Command         | Description                                                 |
| --------------- | ----------------------------------------------------------- |
| `ctx init`      | Scaffold `.ctx/` and copy instruction files to project root |
| `ctx sync-docs` | Generate/update `.ctx/docs/` (2-phase parallel pipeline)    |
| `ctx update`    | Self-update to the latest version                           |
| `ctx version`   | Print current version                                       |

Run `ctx <command> --help` for all available flags.

## Project structure

```
your-project/
├── AGENTS.md              — agent behavior rules (read-before-code)
├── CLAUDE.md              — Claude instructions (if selected)
├── GEMINI.md              — Gemini instructions (if selected)
└── .ctx/
    ├── config.json        — settings
    ├── guides/            — concept docs
    ├── templates/         — doc structure templates (editable)
    └── docs/              — generated documentation
```

## Supported CLIs

| CLI         | Instruction file          | Status   |
| ----------- | ------------------------- | -------- |
| Claude Code | `CLAUDE.md` + `AGENTS.md` | Tested   |
| Codex       | `AGENTS.md`               | Tested   |
| Copilot     | `AGENTS.md`               | Tested   |
| Gemini CLI  | `GEMINI.md` + `AGENTS.md` | Untested |
| OpenCode    | `AGENTS.md`               | Tested   |
