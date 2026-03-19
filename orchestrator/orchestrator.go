package orchestrator

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/Titovilal/middleman/agent"
	"github.com/Titovilal/middleman/connector"
	"github.com/Titovilal/middleman/store"
	"github.com/google/uuid"
)

// Orchestrator is the business logic layer. It has no CLI or I/O concerns.
type Orchestrator struct {
	store      *store.Store
	connectors *connector.ConnectorRegistry
	workDir    string
}

func New(s *store.Store, connectors *connector.ConnectorRegistry, workDir string) *Orchestrator {
	return &Orchestrator{store: s, connectors: connectors, workDir: workDir}
}

// Spawn creates a new agent with the given briefing and runs an initial probe.
// If the agent already exists, it delegates the task to the existing agent instead.
// The agent behavior guide is automatically prepended to the briefing.
func (o *Orchestrator) Spawn(ctx context.Context, id, briefing, connectorName, task string, timeout time.Duration) (*agent.Agent, *agent.TaskRecord, error) {
	// If agent already exists, delegate the task to it.
	reg, err := o.store.Load()
	if err != nil {
		return nil, nil, err
	}
	if existing, getErr := reg.Get(id); getErr == nil {
		if task == "" {
			return existing, nil, nil
		}
		taskRec, err := o.DelegateAsync(ctx, id, task, timeout)
		if err != nil {
			return nil, nil, err
		}
		// Reload agent after delegate updated it.
		_, a, _ := o.loadAgentAndConnector(id)
		return a, taskRec, nil
	}

	conn, ok := o.connectors.Get(connectorName)
	if !ok {
		return nil, nil, fmt.Errorf("connector %q not registered", connectorName)
	}

	// Build full briefing with guides, project overview, and docs list.
	fullBriefing := o.buildBriefing(briefing)

	req := connector.RunRequest{
		Prompt:             "ok",
		WorkDir:            o.workDir,
		SystemPromptAppend: fullBriefing,
		Timeout:            2 * time.Minute,
	}

	startedAt := time.Now()
	result, err := conn.Run(ctx, req)
	if err != nil {
		return nil, nil, fmt.Errorf("spawn run failed: %w", err)
	}

	now := time.Now()
	a := &agent.Agent{
		ID:            id,
		ConnectorName: connectorName,
		SessionID:     result.SessionID,
		WorkDir:       o.workDir,
		Status:        agent.StatusIdle,
		Briefing:      briefing,
		Checkpoints:   make([]agent.CheckpointRecord, 0),
		TaskLog: []agent.TaskRecord{{
			TaskID:      uuid.NewString(),
			Prompt:      "__spawn__",
			Response:    result.FinalText,
			IsError:     result.IsError,
			ErrorDetail: result.ErrorDetail,
			Status:      agent.TaskCompleted,
			StartedAt:   startedAt,
			CompletedAt: now,
		}},
		CreatedAt:    now,
		LastActiveAt: now,
	}

	// Create initial checkpoint after briefing.
	if err := o.appendCheckpoint(ctx, a, conn, "spawn"); err != nil {
		// Non-fatal: agent is still usable without a checkpoint.
		fmt.Printf("warning: could not create initial checkpoint: %v\n", err)
	}

	if err := o.store.WithLock(func(reg *agent.Registry) error {
		return reg.Add(a)
	}); err != nil {
		return nil, nil, fmt.Errorf("save agent: %w", err)
	}

	// If a task was provided, delegate it immediately.
	if task != "" {
		taskRec, err := o.DelegateAsync(ctx, id, task, timeout)
		if err != nil {
			return a, nil, fmt.Errorf("agent spawned but delegate failed: %w", err)
		}
		// Reload agent after delegate updated it.
		_, a, _ = o.loadAgentAndConnector(id)
		return a, taskRec, nil
	}

	return a, nil, nil
}

// DelegateAsync registers a task for an agent and returns the task record.
// If the agent is idle, the task is set to "pending" and a background process should be launched.
// If the agent is already working, the task is "queued" and will run automatically when the current task finishes.
func (o *Orchestrator) DelegateAsync(ctx context.Context, agentID, prompt string, timeout time.Duration) (*agent.TaskRecord, error) {
	if timeout == 0 {
		timeout = 5 * time.Minute
	}

	task := &agent.TaskRecord{
		TaskID:    uuid.NewString(),
		Prompt:    prompt,
		StartedAt: time.Now(),
	}

	if err := o.store.WithLock(func(reg *agent.Registry) error {
		a, err := reg.Get(agentID)
		if err != nil {
			return err
		}

		// If agent is busy, queue the task instead of running immediately.
		queued := a.Status == agent.StatusWorking
		if queued && len(a.QueuedTasks()) >= 2 {
			return fmt.Errorf("agent %s already has 2 queued tasks, wait or use another agent", agentID)
		}

		if queued {
			task.Status = agent.TaskQueued
		} else {
			task.Status = agent.TaskPending
			a.Status = agent.StatusWorking
		}
		a.LastActiveAt = time.Now()
		a.TaskLog = append(a.TaskLog, *task)
		return nil
	}); err != nil {
		return nil, fmt.Errorf("save agent after delegate: %w", err)
	}

	return task, nil
}

// RunTask executes a pending task synchronously. Called by the background process.
// After completing, it processes any queued tasks on the same agent.
func (o *Orchestrator) RunTask(ctx context.Context, agentID, taskID string, timeout time.Duration) error {
	if err := o.runSingleTask(ctx, agentID, taskID, timeout); err != nil {
		return err
	}

	// Process queued tasks.
	for {
		var nextTaskID string
		if err := o.store.WithLock(func(reg *agent.Registry) error {
			a, err := reg.Get(agentID)
			if err != nil {
				return err
			}
			queued := a.QueuedTasks()
			if len(queued) == 0 {
				return nil
			}
			// Promote first queued task to pending.
			next := a.TaskByID(queued[0].TaskID)
			next.Status = agent.TaskPending
			next.StartedAt = time.Now()
			a.Status = agent.StatusWorking
			nextTaskID = next.TaskID
			return nil
		}); err != nil {
			return err
		}

		if nextTaskID == "" {
			return nil
		}

		if err := o.runSingleTask(ctx, agentID, nextTaskID, timeout); err != nil {
			return err
		}
	}
}

func (o *Orchestrator) runSingleTask(ctx context.Context, agentID, taskID string, timeout time.Duration) error {
	// Read agent state for the connector call (read-only snapshot).
	conn, snap, err := o.loadAgentAndConnector(agentID)
	if err != nil {
		return err
	}

	task := snap.TaskByID(taskID)
	if task == nil {
		return fmt.Errorf("task %s not found on agent %s", taskID, agentID)
	}

	// Checkpoint before running (mutates snap in memory only for the checkpoint append).
	preLabel := fmt.Sprintf("pre-task-%s", time.Now().Format("20060102-150405"))
	preCP := o.buildCheckpoint(ctx, snap, conn, preLabel)

	// Save pre-task checkpoint atomically.
	if preCP != nil {
		if err := o.store.WithLock(func(reg *agent.Registry) error {
			a, err := reg.Get(agentID)
			if err != nil {
				return err
			}
			a.Checkpoints = append(a.Checkpoints, *preCP)
			return nil
		}); err != nil {
			fmt.Printf("warning: could not save pre-task checkpoint: %v\n", err)
		}
	}

	if timeout == 0 {
		timeout = 5 * time.Minute
	}
	req := connector.RunRequest{
		SessionID: snap.SessionID,
		Prompt:    task.Prompt,
		WorkDir:   snap.WorkDir,
		Timeout:   timeout,
	}

	// Execute the AI CLI call (long-running, outside any lock).
	result, runErr := conn.Run(ctx, req)
	now := time.Now()

	// Build post-task checkpoint before saving (needs the possibly-new session).
	sessionID := snap.SessionID
	if result.SessionID != "" {
		sessionID = result.SessionID
	}
	snapForPost := &agent.Agent{SessionID: sessionID, ConnectorName: snap.ConnectorName}
	postLabel := fmt.Sprintf("post-task-%s", now.Format("20060102-150405"))
	postCP := o.buildCheckpoint(ctx, snapForPost, conn, postLabel)

	// Apply all mutations atomically on the fresh registry state.
	return o.store.WithLock(func(reg *agent.Registry) error {
		a, err := reg.Get(agentID)
		if err != nil {
			return err
		}

		t := a.TaskByID(taskID)
		if t == nil {
			return fmt.Errorf("task %s not found on agent %s", taskID, agentID)
		}

		if result.SessionID != "" {
			a.SessionID = result.SessionID
		}

		t.Response = result.FinalText
		t.CompletedAt = now
		t.IsError = result.IsError || runErr != nil
		if runErr != nil {
			t.ErrorDetail = runErr.Error()
		} else {
			t.ErrorDetail = result.ErrorDetail
		}

		if t.IsError {
			t.Status = agent.TaskFailed
		} else {
			t.Status = agent.TaskCompleted
		}

		a.Status = agent.StatusIdle
		a.LastActiveAt = now

		if postCP != nil {
			a.Checkpoints = append(a.Checkpoints, *postCP)
		}

		return nil
	})
}

// Rewind forks an agent back to the given checkpoint (or latest if empty).
func (o *Orchestrator) Rewind(ctx context.Context, agentID, checkpointLabel string) (*agent.Agent, error) {
	// Read snapshot to get checkpoint data and connector for the fork call.
	conn, snap, err := o.loadAgentAndConnector(agentID)
	if err != nil {
		return nil, err
	}

	var cp *agent.CheckpointRecord
	if checkpointLabel == "" {
		cp = snap.LatestCheckpoint()
	} else {
		cp = snap.CheckpointByLabel(checkpointLabel)
	}
	if cp == nil {
		return nil, fmt.Errorf("no checkpoint found for agent %s (label: %q)", agentID, checkpointLabel)
	}

	if !conn.SupportsFork() {
		fmt.Printf("warning: connector %q does not support true fork — rewind will modify the original session\n", snap.ConnectorName)
	}

	// Fork happens outside lock (external process call).
	newSessionID, err := conn.Fork(ctx, snap.SessionID, connector.Checkpoint{
		Label:     cp.Label,
		TurnIndex: cp.TurnIndex,
		NativeRef: cp.NativeRef,
		CreatedAt: cp.CreatedAt,
	})
	if err != nil {
		return nil, fmt.Errorf("fork failed: %w", err)
	}

	// Apply mutations atomically on the fresh registry state.
	var result *agent.Agent
	cpLabel := cp.Label
	cpTurnIndex := cp.TurnIndex
	if err := o.store.WithLock(func(reg *agent.Registry) error {
		a, err := reg.Get(agentID)
		if err != nil {
			return err
		}

		// Truncate checkpoints to the rewound point.
		newCheckpoints := make([]agent.CheckpointRecord, 0)
		for _, c := range a.Checkpoints {
			newCheckpoints = append(newCheckpoints, c)
			if c.Label == cpLabel {
				break
			}
		}

		a.SessionID = newSessionID
		a.Checkpoints = newCheckpoints
		a.LastActiveAt = time.Now()
		a.TaskLog = append(a.TaskLog, agent.TaskRecord{
			TaskID:      uuid.NewString(),
			Prompt:      fmt.Sprintf("__rewind_to:%s__", cpLabel),
			Response:    fmt.Sprintf("rewound to checkpoint %q (turn %d)", cpLabel, cpTurnIndex),
			Status:      agent.TaskCompleted,
			StartedAt:   time.Now(),
			CompletedAt: time.Now(),
		})

		result = a
		return nil
	}); err != nil {
		return nil, fmt.Errorf("save agent after rewind: %w", err)
	}

	return result, nil
}

// Inspect returns a copy of the agent record.
func (o *Orchestrator) Inspect(ctx context.Context, agentID string) (*agent.Agent, error) {
	reg, err := o.store.Load()
	if err != nil {
		return nil, err
	}
	return reg.Get(agentID)
}

// Remove deletes an agent from the registry entirely.
func (o *Orchestrator) Remove(ctx context.Context, agentID string) error {
	return o.store.WithLock(func(reg *agent.Registry) error {
		return reg.Delete(agentID)
	})
}

// ListAgents returns agents filtered by status (no filter = all).
func (o *Orchestrator) ListAgents(ctx context.Context, statuses ...agent.Status) ([]*agent.Agent, error) {
	reg, err := o.store.Load()
	if err != nil {
		return nil, err
	}
	return reg.List(statuses...), nil
}

// --- helpers ---

// buildBriefing assembles the full briefing: agent behavior guide,
// project overview, available docs list, and the user's briefing.
func (o *Orchestrator) buildBriefing(briefing string) string {
	var parts []string

	parts = append(parts, "# Current project state")

	// Agent behavior guide.
	if data, err := os.ReadFile(filepath.Join(o.workDir, ".mdm", "guides", "how_agents_should_behave.md")); err == nil {
		parts = append(parts, fmt.Sprintf("## .mdm/guides/how_agents_should_behave.md\n\n%s", string(data)))
	}

	// project_overview.md — so the agent understands the codebase.
	if data, err := os.ReadFile(filepath.Join(o.workDir, ".mdm", "docs", "project_overview.md")); err == nil {
		parts = append(parts, fmt.Sprintf("## .mdm/docs/project_overview.md\n\n%s", string(data)))
	}

	// List available docs so the agent knows what to read without listing the dir.
	docsDir := filepath.Join(o.workDir, ".mdm", "docs")
	if entries, err := os.ReadDir(docsDir); err == nil {
		var names []string
		for _, e := range entries {
			if !e.IsDir() && e.Name() != "project_overview.md" {
				names = append(names, e.Name())
			}
		}
		if len(names) > 0 {
			list := "## Available docs in .mdm/docs/\n"
			for _, n := range names {
				list += "- " + n + "\n"
			}
			parts = append(parts, list)
		}
	}

	if briefing != "" {
		parts = append(parts, "## Briefing\n\n"+briefing)
	}

	return strings.Join(parts, "\n\n")
}

func (o *Orchestrator) loadAgentAndConnector(agentID string) (connector.AgentConnector, *agent.Agent, error) {
	reg, err := o.store.Load()
	if err != nil {
		return nil, nil, err
	}
	a, err := reg.Get(agentID)
	if err != nil {
		return nil, nil, err
	}
	conn, ok := o.connectors.Get(a.ConnectorName)
	if !ok {
		return nil, nil, fmt.Errorf("connector %q not registered", a.ConnectorName)
	}
	return conn, a, nil
}

// buildCheckpoint creates a checkpoint record without mutating the agent.
// Returns nil if the checkpoint could not be built (non-fatal).
func (o *Orchestrator) buildCheckpoint(ctx context.Context, a *agent.Agent, conn connector.AgentConnector, label string) *agent.CheckpointRecord {
	// Claude writes the session JSONL asynchronously after the CLI exits.
	// Retry a few times with short sleeps to let the file appear.
	var turnCount int
	var err error
	for i := 0; i < 5; i++ {
		turnCount, err = conn.TurnCount(ctx, a.SessionID)
		if err == nil {
			break
		}
		time.Sleep(time.Duration(i+1) * 300 * time.Millisecond)
	}
	if err != nil {
		fmt.Printf("warning: could not create checkpoint %q: %v\n", label, err)
		return nil
	}

	// For Claude, get the last assistant UUID as NativeRef.
	nativeRef := ""
	if cc, ok := conn.(interface{ LastAssistantUUID(string) (string, error) }); ok {
		nativeRef, _ = cc.LastAssistantUUID(a.SessionID)
	}

	return &agent.CheckpointRecord{
		Label:     label,
		TurnIndex: turnCount,
		NativeRef: nativeRef,
		CreatedAt: time.Now(),
	}
}

// appendCheckpoint builds and appends a checkpoint to the agent in memory.
// Used during Spawn where the agent hasn't been persisted yet.
func (o *Orchestrator) appendCheckpoint(ctx context.Context, a *agent.Agent, conn connector.AgentConnector, label string) error {
	cp := o.buildCheckpoint(ctx, a, conn, label)
	if cp == nil {
		return fmt.Errorf("could not build checkpoint %q", label)
	}
	a.Checkpoints = append(a.Checkpoints, *cp)
	return nil
}
