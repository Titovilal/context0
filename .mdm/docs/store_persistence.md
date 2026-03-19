# Store & Persistence

## What It Does
Provides thread-safe JSON file persistence for the agent registry, with atomic writes and mutex-protected access. Also initializes the `.mdm/` directory structure from embedded defaults (guides, templates).

## Main Files
- `store/store.go` - Store struct with Load, Save, WithLock methods; atomic writes via temp file + rename; initializes `.mdm/` from embedded defaults on first use
- `.mdm/registry.json` - JSON file holding all agent records (version field for forward compatibility)

## Flow
1. Store is created with a directory path and an embedded defaults FS; it ensures `.mdm/` exists and writes default files (guides, templates) if missing
2. Orchestrator calls WithLock to atomically load the registry, apply mutations, and save back
3. Save writes to a temp file first, then atomically renames it to `registry.json` to prevent corruption
