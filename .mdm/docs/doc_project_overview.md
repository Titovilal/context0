# MDM - The Middleman

## What It Does
MDM is a CLI tool that orchestrates multiple AI coding agents (Claude, Codex, Gemini, OpenCode). It manages agent lifecycle, context preservation, checkpointing, and task delegation without ever touching project code directly.

## Main Files
- `main.go` - entry point
- `cmd/root.go` - CLI setup, connector registration, flag handling
- `cmd/spawn.go` - creates new agents with briefings
- `cmd/delegate.go` - sends tasks to agents (background execution)
- `cmd/run_task.go` - internal background task executor
- `cmd/result.go` - fetches task results
- `cmd/rewind.go` - restores agents to previous checkpoints
- `cmd/agent_prompt.go` - generates system prompt for Middleman agents
- `orchestrator/orchestrator.go` - business logic (spawn, delegate, rewind, inspect)
- `agent/agent.go` - data types (Agent, TaskRecord, CheckpointRecord)
- `agent/registry.go` - in-memory agent registry
- `connector/connector.go` - abstract interface for AI CLI backends
- `connector/claude/claude.go` - Claude Code CLI connector
- `connector/codex/codex.go` - OpenAI Codex CLI connector
- `connector/gemini/gemini.go` - Google Gemini CLI connector
- `connector/opencode/opencode.go` - OpenCode CLI connector
- `store/store.go` - JSON file persistence with atomic writes
- `config/config.go` - runtime configuration

## Flow
1. User runs `mdm spawn <name> --briefing "..."` to create an agent backed by an AI CLI
2. User runs `mdm delegate --to <name> "task"` which queues tasks that run in background processes
3. User checks results with `mdm result <name>`, can rewind with `mdm rewind`, and manages context with `mdm context`
