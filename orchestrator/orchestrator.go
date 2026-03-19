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
	_, a, err := o.loadAgentAndConnector(agentID)
	if err != nil {
		return nil, err
	}

	if timeout == 0 {
		timeout = 5 * time.Minute
	}

	// If agent is busy, queue the task instead of running immediately.
	queued := a.Status == agent.StatusWorking
	if queued && len(a.QueuedTasks()) >= 2 {
		return nil, fmt.Errorf("agent %s already has 2 queued tasks, wait or use another agent", agentID)
	}
	status := agent.TaskPending
	if queued {
		status = agent.TaskQueued
	}

	task := &agent.TaskRecord{
		TaskID:    uuid.NewString(),
		Prompt:    prompt,
		Status:    status,
		StartedAt: time.Now(),
	}

	if !queued {
		a.Status = agent.StatusWorking
	}
	a.LastActiveAt = time.Now()
	a.TaskLog = append(a.TaskLog, *task)

	if err := o.store.WithLock(func(reg *agent.Registry) error {
		return reg.Update(a)
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
		_, a, err := o.loadAgentAndConnector(agentID)
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
		if err := o.store.WithLock(func(reg *agent.Registry) error {
			return reg.Update(a)
		}); err != nil {
			return err
		}

		if err := o.runSingleTask(ctx, agentID, next.TaskID, timeout); err != nil {
			return err
		}
	}
}

func (o *Orchestrator) runSingleTask(ctx context.Context, agentID, taskID string, timeout time.Duration) error {
	conn, a, err := o.loadAgentAndConnector(agentID)
	if err != nil {
		return err
	}

	task := a.TaskByID(taskID)
	if task == nil {
		return fmt.Errorf("task %s not found on agent %s", taskID, agentID)
	}

	// Checkpoint before running.
	label := fmt.Sprintf("pre-task-%s", time.Now().Format("20060102-150405"))
	if err := o.appendCheckpoint(ctx, a, conn, label); err != nil {
		fmt.Printf("warning: could not create pre-task checkpoint: %v\n", err)
	}

	if timeout == 0 {
		timeout = 5 * time.Minute
	}
	req := connector.RunRequest{
		SessionID: a.SessionID,
		Prompt:    task.Prompt,
		WorkDir:   a.WorkDir,
		Timeout:   timeout,
	}

	result, runErr := conn.Run(ctx, req)
	now := time.Now()

	if result.SessionID != "" {
		a.SessionID = result.SessionID
	}

	task.Response = result.FinalText
	task.CompletedAt = now
	task.IsError = result.IsError || runErr != nil
	if runErr != nil {
		task.ErrorDetail = runErr.Error()
	} else {
		task.ErrorDetail = result.ErrorDetail
	}

	if task.IsError {
		task.Status = agent.TaskFailed
	} else {
		task.Status = agent.TaskCompleted
	}

	a.Status = agent.StatusIdle
	a.LastActiveAt = now

	// Post-task checkpoint.
	postLabel := fmt.Sprintf("post-task-%s", now.Format("20060102-150405"))
	if err := o.appendCheckpoint(ctx, a, conn, postLabel); err != nil {
		fmt.Printf("warning: could not create post-task checkpoint: %v\n", err)
	}

	return o.store.WithLock(func(reg *agent.Registry) error {
		return reg.Update(a)
	})
}

// Rewind forks an agent back to the given checkpoint (or latest if empty).
func (o *Orchestrator) Rewind(ctx context.Context, agentID, checkpointLabel string) (*agent.Agent, error) {
	conn, a, err := o.loadAgentAndConnector(agentID)
	if err != nil {
		return nil, err
	}

	var cp *agent.CheckpointRecord
	if checkpointLabel == "" {
		cp = a.LatestCheckpoint()
	} else {
		cp = a.CheckpointByLabel(checkpointLabel)
	}
	if cp == nil {
		return nil, fmt.Errorf("no checkpoint found for agent %s (label: %q)", agentID, checkpointLabel)
	}

	if !conn.SupportsFork() {
		fmt.Printf("warning: connector %q does not support true fork — rewind will modify the original session\n", a.ConnectorName)
	}

	newSessionID, err := conn.Fork(ctx, a.SessionID, connector.Checkpoint{
		Label:     cp.Label,
		TurnIndex: cp.TurnIndex,
		NativeRef: cp.NativeRef,
		CreatedAt: cp.CreatedAt,
	})
	if err != nil {
		return nil, fmt.Errorf("fork failed: %w", err)
	}

	// Truncate checkpoints to the rewound point.
	newCheckpoints := make([]agent.CheckpointRecord, 0)
	for _, c := range a.Checkpoints {
		newCheckpoints = append(newCheckpoints, c)
		if c.Label == cp.Label {
			break
		}
	}

	a.SessionID = newSessionID
	a.Checkpoints = newCheckpoints
	a.LastActiveAt = time.Now()
	a.TaskLog = append(a.TaskLog, agent.TaskRecord{
		TaskID:      uuid.NewString(),
		Prompt:      fmt.Sprintf("__rewind_to:%s__", cp.Label),
		Response:    fmt.Sprintf("rewound to checkpoint %q (turn %d)", cp.Label, cp.TurnIndex),
		Status:      agent.TaskCompleted,
		StartedAt:   time.Now(),
		CompletedAt: time.Now(),
	})

	if err := o.store.WithLock(func(reg *agent.Registry) error {
		return reg.Update(a)
	}); err != nil {
		return nil, fmt.Errorf("save agent after rewind: %w", err)
	}

	return a, nil
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

func (o *Orchestrator) appendCheckpoint(ctx context.Context, a *agent.Agent, conn connector.AgentConnector, label string) error {
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
		return err
	}

	// For Claude, get the last assistant UUID as NativeRef.
	nativeRef := ""
	if cc, ok := conn.(interface{ LastAssistantUUID(string) (string, error) }); ok {
		nativeRef, _ = cc.LastAssistantUUID(a.SessionID)
	}

	a.Checkpoints = append(a.Checkpoints, agent.CheckpointRecord{
		Label:     label,
		TurnIndex: turnCount,
		NativeRef: nativeRef,
		CreatedAt: time.Now(),
	})
	return nil
}
