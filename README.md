# MDM - The Middleman

A CLI tool for orchestrating multiple AI agent instances (Claude Code, Gemini CLI, OpenCode, Codex) from a single manager.

## The problem

When working on complex projects with AI coding assistants, you end up with several open instances:
- Each one has a different context (files read, decisions made, errors seen)
- You have to mentally track "this Claude knows about the auth module", "this one has the refactor context"
- You manually rewind sessions when a recent change contaminates an instance's context
- You decide yourself which instance to send each question to

MDM externalizes that cognitive load.

## How it works

The **Middleman** is an orchestrator. It manages AI coding agents but doesn't write code itself — agents do that. The Middleman can run project commands (build, test, git, etc.) to verify work and gather information.

Each agent is a subprocess running a real AI CLI. The Middleman only sees the **final response** — never the internal stream of tool calls, file reads, or intermediate reasoning. Each agent is a black box: task in, response out. When an agent's context gets contaminated, the Middleman rewinds it to a clean checkpoint — this is a deliberate strategy, not an error recovery mechanism.

## Connectors

MDM uses a pluggable connector interface. Each connector wraps a different AI CLI:

| Connector | Status |
|---|---|
| `claude` | Tested (Claude Code CLI) |
| `gemini` | Vibe-coded, untested (Gemini CLI) |
| `codex` | Vibe-coded, untested (Codex CLI) |
| `opencode` | Vibe-coded, untested (OpenCode CLI) |

Only the Claude connector has been verified. The rest are implemented but waiting for feedback or usage — expect rough edges. Adding a new connector only requires implementing the `AgentConnector` interface in `connector/<name>/`.

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

## Usage

### Launch the Middleman

```bash
mdm launch claude    # or gemini, codex
```

### Spawn an agent and delegate a task

```bash
mdm spawn auth --briefing "Auth module" "Implement OAuth2 flow"
mdm spawn tests --briefing "Payment tests" --connector gemini "Write tests for payments/"
mdm spawn auth "Add refresh token support"  # agent exists → task is queued
```

Spawn creates the agent and runs the task in the background. If the agent already exists, the task is queued into it.

### Check results

```bash
mdm result auth                  # latest task
mdm result auth --task-id <id>   # specific task
```

### Check agent status

```bash
mdm status
mdm status --all
```

### Rewind to a checkpoint

```bash
mdm rewind auth --list                        # show checkpoints
mdm rewind auth                               # rewind to latest
mdm rewind auth --to pre-task-20260318-110000  # rewind to specific
```

Rewinds fork the session — the original is never deleted.

### Remove an agent

```bash
mdm remove auth
```

## Documentation system

Each project has `.mdm/` with:
- **`guides/how_agents_should_behave.md`** — mandatory steps for all agents (auto-injected into every briefing)
- **`docs/`** — project documentation that agents read before modifying code
- **`guides/`** — process guides (e.g. `the_middleman.md` — the Middleman's playbook)
- **`templates/`** — templates for creating docs

`mdm sync-docs` generates skeleton docs automatically by scanning source files.

## Registry

The agent registry is stored in `.mdm/registry.json` in the project directory. Use `--global` to use `~/.mdm/` instead.

```bash
mdm --global status
mdm --workdir /path/to/project status
```

## Architecture

```
mdm/
├── main.go
├── cmd/           # CLI commands (cobra)
├── agent/         # Agent, Checkpoint, TaskRecord types and in-memory registry
├── connector/     # AgentConnector interface + per-CLI implementations
│   ├── claude/
│   ├── gemini/
│   ├── codex/
│   └── opencode/
├── orchestrator/  # Business logic: Spawn, Rewind, Remove, ListAgents
├── store/         # JSON persistence with atomic writes
└── config/        # Runtime config, paths
```

The `orchestrator` package has no CLI or I/O concerns — it only depends on the `connector` interface and the `store`. This makes it independently testable with a mock connector.
