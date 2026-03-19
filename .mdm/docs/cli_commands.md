# CLI Commands

## What It Does
Implements the full MDM command-line interface using Cobra. Each command maps to an orchestrator operation and handles argument parsing, output formatting, and background process management.

## Main Files
- `cmd/root.go` - Root command setup, global flags (--workdir, --connector, --global), connector and orchestrator initialization
- `cmd/spawn.go` - `mdm spawn` creates a new agent with a briefing and optionally delegates a task; launches background `_run-task` process for async execution. If the agent already exists, the task is queued into it
- `cmd/run_task.go` - `mdm _run-task` (hidden) executes a pending task synchronously, called as a background process by spawn
- `cmd/result.go` - `mdm result` shows the latest or specific task result for an agent
- `cmd/status.go` - `mdm status` lists agents in a table with connector, status, turns, checkpoints, and last active time
- `cmd/rewind.go` - `mdm rewind` lists checkpoints or rewinds an agent to a specific checkpoint via session forking
- `cmd/remove.go` - `mdm remove` deletes an agent from the registry
- `cmd/launch.go` - `mdm launch` starts an interactive AI CLI session with the Middleman system prompt pre-injected
- `cmd/agent_prompt.go` - `mdm agent-prompt` prints the full Middleman system prompt for injection into AI agents
- `cmd/update.go` - `mdm update` and `mdm version` for self-updating from GitHub releases
- `cmd/sync_docs.go` - `mdm sync-docs` auto-generates skeleton documentation by scanning source files

## Flow
1. User runs an `mdm` command; Cobra parses flags and routes to the appropriate command handler
2. `PersistentPreRunE` in root.go initializes the store, connectors, and orchestrator before each command runs
3. Command handler calls the orchestrator, formats the result, and prints it to stdout; spawn additionally spawns a background process for async task execution when a task is provided
