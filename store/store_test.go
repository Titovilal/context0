package store

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/Titovilal/middleman/agent"
)

func TestNew_CreatesDir(t *testing.T) {
	dir := t.TempDir()
	s, err := New(dir, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	mdmDir := filepath.Join(dir, ".mdm")
	if _, err := os.Stat(mdmDir); os.IsNotExist(err) {
		t.Fatal(".mdm directory should exist")
	}

	expected := filepath.Join(mdmDir, "registry.json")
	if s.Path() != expected {
		t.Fatalf("expected path %s, got %s", expected, s.Path())
	}
}

func TestLoad_EmptyReturnsNewRegistry(t *testing.T) {
	dir := t.TempDir()
	s, _ := New(dir, nil)

	reg, err := s.Load()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if reg == nil || reg.Agents == nil {
		t.Fatal("expected non-nil registry with initialized agents")
	}
	if len(reg.Agents) != 0 {
		t.Fatal("expected empty agent list")
	}
}

func TestSaveAndLoad(t *testing.T) {
	dir := t.TempDir()
	s, _ := New(dir, nil)

	reg := agent.NewRegistry()
	a := &agent.Agent{
		ID:            "test-agent",
		ConnectorName: "claude",
		SessionID:     "sess-1",
		Status:        agent.StatusIdle,
		Checkpoints:   make([]agent.CheckpointRecord, 0),
		TaskLog:       make([]agent.TaskRecord, 0),
		CreatedAt:     time.Now(),
		LastActiveAt:  time.Now(),
	}
	_ = reg.Add(a)

	if err := s.Save(reg); err != nil {
		t.Fatalf("save error: %v", err)
	}

	loaded, err := s.Load()
	if err != nil {
		t.Fatalf("load error: %v", err)
	}
	if len(loaded.Agents) != 1 {
		t.Fatalf("expected 1 agent, got %d", len(loaded.Agents))
	}
	if loaded.Agents[0].ID != "test-agent" {
		t.Fatalf("expected test-agent, got %s", loaded.Agents[0].ID)
	}
}

func TestWithLock(t *testing.T) {
	dir := t.TempDir()
	s, _ := New(dir, nil)

	// Add an agent via WithLock.
	err := s.WithLock(func(reg *agent.Registry) error {
		return reg.Add(&agent.Agent{
			ID:          "locked-agent",
			Status:      agent.StatusIdle,
			Checkpoints: make([]agent.CheckpointRecord, 0),
			TaskLog:     make([]agent.TaskRecord, 0),
		})
	})
	if err != nil {
		t.Fatalf("WithLock error: %v", err)
	}

	// Verify it was persisted.
	reg, _ := s.Load()
	if len(reg.Agents) != 1 || reg.Agents[0].ID != "locked-agent" {
		t.Fatal("agent not persisted via WithLock")
	}
}

func TestWithLock_RollbackOnError(t *testing.T) {
	dir := t.TempDir()
	s, _ := New(dir, nil)

	// First, add an agent.
	_ = s.WithLock(func(reg *agent.Registry) error {
		return reg.Add(&agent.Agent{
			ID:          "existing",
			Status:      agent.StatusIdle,
			Checkpoints: make([]agent.CheckpointRecord, 0),
			TaskLog:     make([]agent.TaskRecord, 0),
		})
	})

	// Try to add a duplicate — should fail and not corrupt state.
	err := s.WithLock(func(reg *agent.Registry) error {
		return reg.Add(&agent.Agent{
			ID:          "existing",
			Checkpoints: make([]agent.CheckpointRecord, 0),
			TaskLog:     make([]agent.TaskRecord, 0),
		})
	})
	if err == nil {
		t.Fatal("expected error for duplicate")
	}

	// Original agent should still be there.
	reg, _ := s.Load()
	if len(reg.Agents) != 1 {
		t.Fatalf("expected 1 agent, got %d", len(reg.Agents))
	}
}

func TestLoad_CorruptFile(t *testing.T) {
	dir := t.TempDir()
	s, _ := New(dir, nil)

	// Write garbage to the registry file.
	_ = os.WriteFile(s.Path(), []byte("not json"), 0o644)

	_, err := s.Load()
	if err == nil {
		t.Fatal("expected error for corrupt file")
	}
}
