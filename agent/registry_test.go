package agent

import (
	"errors"
	"testing"
)

func TestNewRegistry(t *testing.T) {
	reg := NewRegistry()
	if reg.Agents == nil || len(reg.Agents) != 0 {
		t.Fatal("expected empty initialized agent list")
	}
}

func TestRegistry_AddAndGet(t *testing.T) {
	reg := NewRegistry()
	a := newTestAgent("agent-1")

	if err := reg.Add(a); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	got, err := reg.Get("agent-1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.ID != "agent-1" {
		t.Fatalf("expected agent-1, got %s", got.ID)
	}
}

func TestRegistry_AddDuplicate(t *testing.T) {
	reg := NewRegistry()
	a := newTestAgent("agent-1")

	_ = reg.Add(a)
	err := reg.Add(a)
	if !errors.Is(err, ErrAgentExists) {
		t.Fatalf("expected ErrAgentExists, got %v", err)
	}
}

func TestRegistry_GetNotFound(t *testing.T) {
	reg := NewRegistry()
	_, err := reg.Get("nonexistent")
	if !errors.Is(err, ErrAgentNotFound) {
		t.Fatalf("expected ErrAgentNotFound, got %v", err)
	}
}

func TestRegistry_Update(t *testing.T) {
	reg := NewRegistry()
	a := newTestAgent("agent-1")
	_ = reg.Add(a)

	a.Status = StatusWorking
	if err := reg.Update(a); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	got, _ := reg.Get("agent-1")
	if got.Status != StatusWorking {
		t.Fatal("expected status to be updated")
	}
}

func TestRegistry_UpdateNotFound(t *testing.T) {
	reg := NewRegistry()
	a := newTestAgent("agent-1")
	err := reg.Update(a)
	if !errors.Is(err, ErrAgentNotFound) {
		t.Fatalf("expected ErrAgentNotFound, got %v", err)
	}
}

func TestRegistry_Delete(t *testing.T) {
	reg := NewRegistry()
	_ = reg.Add(newTestAgent("agent-1"))
	_ = reg.Add(newTestAgent("agent-2"))

	if err := reg.Delete("agent-1"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(reg.Agents) != 1 {
		t.Fatalf("expected 1 agent, got %d", len(reg.Agents))
	}
	if reg.Agents[0].ID != "agent-2" {
		t.Fatal("wrong agent remained")
	}
}

func TestRegistry_DeleteNotFound(t *testing.T) {
	reg := NewRegistry()
	err := reg.Delete("nonexistent")
	if !errors.Is(err, ErrAgentNotFound) {
		t.Fatalf("expected ErrAgentNotFound, got %v", err)
	}
}

func TestRegistry_List_All(t *testing.T) {
	reg := NewRegistry()
	a1 := newTestAgent("a1")
	a1.Status = StatusIdle
	a2 := newTestAgent("a2")
	a2.Status = StatusWorking
	_ = reg.Add(a1)
	_ = reg.Add(a2)

	all := reg.List()
	if len(all) != 2 {
		t.Fatalf("expected 2, got %d", len(all))
	}
}

func TestRegistry_List_Filtered(t *testing.T) {
	reg := NewRegistry()
	a1 := newTestAgent("a1")
	a1.Status = StatusIdle
	a2 := newTestAgent("a2")
	a2.Status = StatusWorking
	a3 := newTestAgent("a3")
	a3.Status = StatusIdle
	_ = reg.Add(a1)
	_ = reg.Add(a2)
	_ = reg.Add(a3)

	idle := reg.List(StatusIdle)
	if len(idle) != 2 {
		t.Fatalf("expected 2 idle, got %d", len(idle))
	}

	working := reg.List(StatusWorking)
	if len(working) != 1 {
		t.Fatalf("expected 1 working, got %d", len(working))
	}
}

func TestRegistry_List_IsCopy(t *testing.T) {
	reg := NewRegistry()
	_ = reg.Add(newTestAgent("a1"))

	list := reg.List()
	list[0] = nil // mutate the copy
	if reg.Agents[0] == nil {
		t.Fatal("List should return a copy, not the original slice")
	}
}
