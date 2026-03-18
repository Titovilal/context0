# MDM - The Middleman

A CLI tool for orchestrating multiple AI agent instances (Claude Code, Gemini CLI, OpenCode) from a single manager.

## The problem

When working on complex projects with AI coding assistants, you end up with several open instances:
- Each one has a different context (files read, decisions made, errors seen)
- You have to mentally track "this Claude knows about the auth module", "this one has the refactor context"
- You manually rewind sessions when a recent change contaminates an instance's context
- You decide yourself which instance to send each question to

MDM externalizes that cognitive load.

## How it works

The **Middleman** is a pure orchestrator. It never reads files or runs code directly. It only:
- Keeps a registry of active agents and their known context
- Delegates tasks to the right agent
- Creates checkpoints after each task
- Rewinds agents to a previous checkpoint when their context gets contaminated

Each agent is a subprocess running a real AI CLI. The Middleman only sees the **final response** — never the internal stream of tool calls, file reads, or intermediate reasoning. Each agent is a black box: task in, response out.

Rewinds are not a sign that something went wrong. They are a deliberate strategy to preserve clean context before a bad direction infects an agent.

## Connectors

MDM uses a pluggable connector interface. Each connector wraps a different AI CLI:

| Connector | Status |
|---|---|
| `claude` | Implemented (Claude Code CLI) |
| `gemini` | Stub |
| `opencode` | Stub |

Adding a new connector only requires implementing the `AgentConnector` interface in `connector/<name>/`.

## Installation

Requires Go 1.23+.

```bash
git clone https://github.com/Titovilal/middleman
cd mdm
go build -o mdm .
```

Add to your PATH or move the binary to `/usr/local/bin`.

## Usage

### For AI agents

Run `mdm agent-prompt` to get the full system prompt that tells an AI agent how to act as the Middleman. Pipe it into your agent's system prompt or CLAUDE.md:

```bash
# Add to CLAUDE.md
mdm agent-prompt > CLAUDE.md

# Or use directly with Claude Code
claude --append-system-prompt "$(mdm agent-prompt)"
```

### Spawn an agent

```bash
mdm spawn agent-auth --briefing "You are working on the authentication module of this project."
mdm spawn agent-db --connector gemini --briefing "You handle database migrations."
```

### Delegate a task

```bash
mdm delegate --to agent-auth "refactor the JWT validation logic"
mdm delegate --to agent-auth --timeout 10m "run a full analysis of the codebase"
```

The response is printed to stdout. The agent's internal work is not shown. Default timeout is 5 minutes.

### Check agent status

```bash
mdm status
mdm status --all   # includes discarded agents
```

### Inspect an agent

```bash
mdm inspect agent-auth
mdm inspect agent-auth --json
```

### List checkpoints and rewind

```bash
# Show available checkpoints
mdm rewind agent-auth --list

# Rewind to the most recent checkpoint (undo last task)
mdm rewind agent-auth

# Rewind to a specific checkpoint
mdm rewind agent-auth --to pre-task-20260318-110000
```

Rewinds fork the session — the original is never deleted.

### Task history

```bash
mdm history agent-auth
mdm history agent-auth --tail 5
```

### Update the Middleman's notes about an agent

```bash
mdm context agent-auth "has read auth/handler.go and auth/middleware.go, decided to use JWT with RS256"
```

This is the Middleman's own notes about what an agent knows — separate from the agent's internal context.

### Discard an agent

```bash
mdm discard agent-auth
```

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
│   └── opencode/
├── orchestrator/  # Business logic: Spawn, Delegate, Rewind, Inspect
├── store/         # JSON persistence with atomic writes
└── config/        # Runtime config, paths
```

The `orchestrator` package has no CLI or I/O concerns — it only depends on the `connector` interface and the `store`. This makes it independently testable with a mock connector.
