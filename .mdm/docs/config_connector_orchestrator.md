# Config, Connectors & Orchestrator

## What It Does
Provides runtime configuration, a pluggable connector interface for AI CLIs (Claude, Gemini, OpenCode, Codex), and the orchestrator that implements all business logic for agent lifecycle management, task delegation, and session rewinding.

## Main Files
- `config/config.go` - Config struct with WorkDir, DefaultConnector, GlobalMode; resolves `.mdm/` or `~/.mdm/` paths
- `connector/connector.go` - AgentConnector interface (Run, Fork, TurnCount), RunRequest/RunResult types, ConnectorRegistry
- `connector/claude/claude.go` - Full Claude Code connector; parses JSON output, reads session JSONL for checkpoints using assistant message UUIDs
- `connector/gemini/gemini.go` - Gemini CLI connector (degraded: no true fork or turn counting)
- `connector/opencode/opencode.go` - OpenCode CLI connector (degraded: limited checkpoint precision)
- `connector/codex/codex.go` - Codex CLI connector; parses JSONL events, counts turn.completed events
- `orchestrator/orchestrator.go` - Spawn (with inline delegation), DelegateAsync, RunTask, Rewind, Remove, ListAgents; manages checkpoints and briefing injection

## Flow
1. CLI commands initialize Config and pass it to the Orchestrator along with a ConnectorRegistry containing all four connectors
2. Orchestrator receives high-level commands (spawn, rewind, remove) and translates them into connector calls (Run, Fork, TurnCount); spawn handles both agent creation and task delegation in one operation
3. Connectors execute AI CLI subprocesses, parse their output, and return only the final response to the orchestrator — internal tool calls and reasoning are never exposed
