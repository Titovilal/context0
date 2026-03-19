# Project Overview

## What It Does
MDM (The Middleman) is a CLI tool that orchestrates multiple AI coding agents (Claude Code, Gemini, OpenCode, Codex) from a single manager. It externalizes context management and task delegation so a "Middleman" agent can spawn, delegate to, rewind, and monitor multiple AI agents working in parallel on the same codebase.

## Main Files
- `main.go` - Entry point, calls cmd.Execute()
- `agent/` - Domain types (Agent, TaskRecord, CheckpointRecord) and in-memory registry
- `config/` - Runtime configuration and path resolution
- `connector/` - Pluggable interface for AI CLIs with four implementations (Claude, Gemini, OpenCode, Codex)
- `orchestrator/` - Business logic for spawn (with inline delegation), rewind, remove, and agent listing
- `store/` - JSON file persistence with atomic writes
- `cmd/` - CLI commands via Cobra (spawn, result, status, rewind, remove, launch, update, sync-docs)

## Flow
1. User (or a Middleman agent) runs `mdm spawn <name> --briefing '...' 'task'` to create an agent and delegate a task in one call; if the agent already exists, the task is queued
2. Tasks run asynchronously in background processes; results are retrieved with `mdm result`
3. Sessions can be rewound to any checkpoint via `mdm rewind`, which forks the AI CLI session at the checkpoint's native reference point (note: only Claude supports true session forking; Gemini and OpenCode have degraded rewind with limitations)

## Documentation available in `.mdm/docs/`
- **`agent_registry.md`** — Agent domain types and in-memory registry
- **`config_connector_orchestrator.md`** — Configuration, connector interface, all four AI CLI connectors, and orchestrator business logic
- **`cli_commands.md`** — All CLI commands (spawn, result, status, rewind, remove, launch, update, sync-docs)
- **`store_persistence.md`** — JSON file persistence, atomic writes, and .mdm/ directory initialization
