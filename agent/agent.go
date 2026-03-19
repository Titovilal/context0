package agent

import "time"

type Status string

const (
	StatusActive  Status = "active"
	StatusIdle    Status = "idle"
	StatusWorking Status = "working"
)

// Agent is the registry record for a single AI agent instance.
type Agent struct {
	ID            string `json:"id"`
	ConnectorName string `json:"connector_name"`
	SessionID     string `json:"session_id"`
	WorkDir       string `json:"work_dir"`
	Status        Status `json:"status"`

	// Briefing is the initial context given at spawn time.
	Briefing string `json:"briefing"`

	// Checkpoints is an ordered list of save points (oldest first).
	Checkpoints []CheckpointRecord `json:"checkpoints"`

	// TaskLog is an append-only record of every delegation.
	TaskLog []TaskRecord `json:"task_log"`

	CreatedAt    time.Time `json:"created_at"`
	LastActiveAt time.Time `json:"last_active_at"`
}

// CheckpointRecord is a persisted snapshot of a session state.
type CheckpointRecord struct {
	Label string `json:"label"`

	// TurnIndex is the assistant turn count at checkpoint time.
	TurnIndex int `json:"turn_index"`

	// NativeRef is connector-specific:
	// Claude: UUID of the last assistant message.
	// Gemini: numeric session index as string.
	NativeRef string `json:"native_ref"`

	CreatedAt time.Time `json:"created_at"`
}

// TaskStatus tracks whether a background task is still running.
type TaskStatus string

const (
	TaskQueued    TaskStatus = "queued"
	TaskPending   TaskStatus = "pending"
	TaskCompleted TaskStatus = "completed"
	TaskFailed    TaskStatus = "failed"
)

// TaskRecord is one entry in the agent's task log.
type TaskRecord struct {
	TaskID      string     `json:"task_id"`
	Prompt      string     `json:"prompt"`
	Response    string     `json:"response"`
	IsError     bool       `json:"is_error"`
	ErrorDetail string     `json:"error_detail,omitempty"`
	Status      TaskStatus `json:"status"`
	StartedAt   time.Time  `json:"started_at"`
	CompletedAt time.Time  `json:"completed_at"`
}

// LatestTask returns the most recent task, or nil if none.
func (a *Agent) LatestTask() *TaskRecord {
	if len(a.TaskLog) == 0 {
		return nil
	}
	t := a.TaskLog[len(a.TaskLog)-1]
	return &t
}

// TaskByID returns the task with the given ID, or nil.
func (a *Agent) TaskByID(taskID string) *TaskRecord {
	for i := range a.TaskLog {
		if a.TaskLog[i].TaskID == taskID {
			return &a.TaskLog[i]
		}
	}
	return nil
}

// QueuedTasks returns all tasks with status "queued", in order.
func (a *Agent) QueuedTasks() []TaskRecord {
	var queued []TaskRecord
	for _, t := range a.TaskLog {
		if t.Status == TaskQueued {
			queued = append(queued, t)
		}
	}
	return queued
}

// HasPendingWork returns true if the agent has a pending or queued task.
func (a *Agent) HasPendingWork() bool {
	for _, t := range a.TaskLog {
		if t.Status == TaskPending || t.Status == TaskQueued {
			return true
		}
	}
	return false
}

// LatestCheckpoint returns the most recent checkpoint, or nil if none.
func (a *Agent) LatestCheckpoint() *CheckpointRecord {
	if len(a.Checkpoints) == 0 {
		return nil
	}
	cp := a.Checkpoints[len(a.Checkpoints)-1]
	return &cp
}

// CheckpointByLabel returns the checkpoint with the given label, or nil.
func (a *Agent) CheckpointByLabel(label string) *CheckpointRecord {
	for i := range a.Checkpoints {
		if a.Checkpoints[i].Label == label {
			return &a.Checkpoints[i]
		}
	}
	return nil
}
