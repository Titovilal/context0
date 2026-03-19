# Agent & Registry

## What It Does
Defines the core domain types (Agent, TaskRecord, CheckpointRecord) and provides an in-memory registry for managing agent lifecycle. Every other package depends on these types.

## Main Files
- `agent/agent.go` - Agent struct, TaskRecord, CheckpointRecord, status enums, and query methods (LatestTask, QueuedTasks, HasPendingWork, etc.)
- `agent/registry.go` - In-memory Registry with CRUD operations (Get, Add, Update, Delete) and status-based filtering (List)

## Flow
1. Orchestrator creates an Agent struct with a unique ID, connector name, briefing, and initial status
2. Agent accumulates TaskRecords (delegated work) and CheckpointRecords (session snapshots) over its lifetime
3. Registry holds all agents in memory; the store package handles persistence to disk
