# Store & Persistence

## What It Does
Provides thread-safe JSON file persistence for the agent registry, with atomic writes and mutex-protected access. Also initializes the `.mdm/` directory structure including docs templates and the agents.md briefing file.

## Main Files
- `store/store.go` - Store struct with Load, Save, WithLock methods; atomic writes via temp file + rename; initializes `.mdm/docs/` and `.mdm/agents.md` on first use
- `.mdm/registry.json` - JSON file holding all agent records (version field for forward compatibility)
- `.mdm/agents.md` - Mandatory instructions prepended to every agent briefing at spawn time
- `main.go` - Entry point; calls cmd.Execute()
- `go.mod` - Module declaration and dependencies (Cobra, UUID, pflag)

## Flow
1. Store is created with a directory path; it ensures `.mdm/`, `docs/`, `agents.md`, and templates exist
2. Orchestrator calls WithLock to atomically load the registry, apply mutations, and save back
3. Save writes to a temp file first, then atomically renames it to `registry.json` to prevent corruption
