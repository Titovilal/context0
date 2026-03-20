# Connectors

## What It Does
Provides a thin abstraction over external AI CLI tools so that MDM commands (`sync-docs`, `open`) can run prompts through any supported AI CLI without knowing the details of each tool's invocation.

## Main Files
- `cmd/connector.go` - Defines the `connector` struct (Name + Run function), a global registry map, and five implementations: Claude, Copilot, Gemini, Codex, OpenCode
- `cmd/root.go` - Reads `config.json` to determine the default CLI; commands fall back to this when no `--connector` flag is provided

## Flow
1. A command (e.g. `sync-docs`) looks up the connector by name from the registry
2. The connector's `Run` function builds an `exec.Command` with the appropriate flags for that AI CLI
3. The subprocess runs the prompt against the codebase and returns the text result; Claude's output is JSON-parsed, others return raw stdout
