package orchestrator

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
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
// The agents.md file from the working directory is automatically prepended to the briefing.
func (o *Orchestrator) Spawn(ctx context.Context, id, briefing, connectorName string) (*agent.Agent, error) {
	conn, ok := o.connectors.Get(connectorName)
	if !ok {
		return nil, fmt.Errorf("connector %q not registered", connectorName)
	}

	// Prepend agents.md content to briefing so every agent reads it.
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
		return nil, fmt.Errorf("spawn run failed: %w", err)
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
		return nil, fmt.Errorf("save agent: %w", err)
	}

	return a, nil
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

// Discard marks an agent as discarded.
func (o *Orchestrator) Discard(ctx context.Context, agentID string) error {
	return o.store.WithLock(func(reg *agent.Registry) error {
		a, err := reg.Get(agentID)
		if err != nil {
			return err
		}
		a.Status = agent.StatusDiscarded
		return reg.Update(a)
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

// UpdateContext updates the KnownContext field of an agent (Middleman's notes).
func (o *Orchestrator) UpdateContext(ctx context.Context, agentID, knownContext string) error {
	return o.store.WithLock(func(reg *agent.Registry) error {
		a, err := reg.Get(agentID)
		if err != nil {
			return err
		}
		a.KnownContext = knownContext
		return reg.Update(a)
	})
}

// --- helpers ---

// buildBriefing prepends the .mdm/agents.md content to the user-provided briefing.
func (o *Orchestrator) buildBriefing(briefing string) string {
	agentsPath := filepath.Join(o.workDir, ".mdm", "agents.md")
	data, err := os.ReadFile(agentsPath)
	if err != nil {
		// No agents.md found, just use the original briefing.
		return briefing
	}

	agentsContent := string(data)
	if briefing == "" {
		return agentsContent
	}
	return agentsContent + "\n\n---\n\n" + briefing
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
