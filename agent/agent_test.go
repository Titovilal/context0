package agent

import (
	"testing"
	"time"
)

func newTestAgent(id string) *Agent {
	return &Agent{
		ID:            id,
		ConnectorName: "claude",
		SessionID:     "sess-1",
		WorkDir:       "/tmp",
		Status:        StatusIdle,
		Briefing:      "test briefing",
		Checkpoints:   make([]CheckpointRecord, 0),
		TaskLog:       make([]TaskRecord, 0),
		CreatedAt:     time.Now(),
		LastActiveAt:  time.Now(),
	}
}

func TestLatestTask_Empty(t *testing.T) {
	a := newTestAgent("a1")
	if a.LatestTask() != nil {
		t.Fatal("expected nil for empty task log")
	}
}

func TestLatestTask(t *testing.T) {
	a := newTestAgent("a1")
	a.TaskLog = []TaskRecord{
		{TaskID: "t1", Prompt: "first"},
		{TaskID: "t2", Prompt: "second"},
	}
	latest := a.LatestTask()
	if latest == nil || latest.TaskID != "t2" {
		t.Fatalf("expected t2, got %v", latest)
	}
}

func TestTaskByID(t *testing.T) {
	a := newTestAgent("a1")
	a.TaskLog = []TaskRecord{
		{TaskID: "t1", Prompt: "first"},
		{TaskID: "t2", Prompt: "second"},
	}

	if task := a.TaskByID("t1"); task == nil || task.Prompt != "first" {
		t.Fatal("expected to find t1")
	}
	if task := a.TaskByID("nonexistent"); task != nil {
		t.Fatal("expected nil for nonexistent task")
	}
}

func TestQueuedTasks(t *testing.T) {
	a := newTestAgent("a1")
	a.TaskLog = []TaskRecord{
		{TaskID: "t1", Status: TaskCompleted},
		{TaskID: "t2", Status: TaskQueued},
		{TaskID: "t3", Status: TaskPending},
		{TaskID: "t4", Status: TaskQueued},
	}

	queued := a.QueuedTasks()
	if len(queued) != 2 {
		t.Fatalf("expected 2 queued tasks, got %d", len(queued))
	}
	if queued[0].TaskID != "t2" || queued[1].TaskID != "t4" {
		t.Fatal("queued tasks not in expected order")
	}
}

func TestHasPendingWork(t *testing.T) {
	a := newTestAgent("a1")
	if a.HasPendingWork() {
		t.Fatal("expected no pending work")
	}

	a.TaskLog = []TaskRecord{{TaskID: "t1", Status: TaskCompleted}}
	if a.HasPendingWork() {
		t.Fatal("expected no pending work with only completed tasks")
	}

	a.TaskLog = append(a.TaskLog, TaskRecord{TaskID: "t2", Status: TaskPending})
	if !a.HasPendingWork() {
		t.Fatal("expected pending work")
	}
}

func TestLatestCheckpoint(t *testing.T) {
	a := newTestAgent("a1")
	if a.LatestCheckpoint() != nil {
		t.Fatal("expected nil for empty checkpoints")
	}

	a.Checkpoints = []CheckpointRecord{
		{Label: "cp1", TurnIndex: 1},
		{Label: "cp2", TurnIndex: 3},
	}
	cp := a.LatestCheckpoint()
	if cp == nil || cp.Label != "cp2" {
		t.Fatalf("expected cp2, got %v", cp)
	}
}

func TestCheckpointByLabel(t *testing.T) {
	a := newTestAgent("a1")
	a.Checkpoints = []CheckpointRecord{
		{Label: "cp1", TurnIndex: 1},
		{Label: "cp2", TurnIndex: 3},
	}

	cp := a.CheckpointByLabel("cp1")
	if cp == nil || cp.TurnIndex != 1 {
		t.Fatal("expected to find cp1")
	}

	if a.CheckpointByLabel("nonexistent") != nil {
		t.Fatal("expected nil for nonexistent label")
	}
}
