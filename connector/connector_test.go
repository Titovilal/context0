package connector

import (
	"context"
	"testing"
)

// mockConnector implements AgentConnector for testing.
type mockConnector struct {
	name string
}

func (m *mockConnector) Name() string { return m.name }
func (m *mockConnector) Run(ctx context.Context, req RunRequest) (RunResult, error) {
	return RunResult{}, nil
}
func (m *mockConnector) Fork(ctx context.Context, sourceSessionID string, checkpoint Checkpoint) (string, error) {
	return "", nil
}
func (m *mockConnector) SupportsFork() bool                                    { return false }
func (m *mockConnector) TurnCount(ctx context.Context, sessionID string) (int, error) { return 0, nil }

func TestConnectorRegistry_RegisterAndGet(t *testing.T) {
	reg := NewConnectorRegistry()
	reg.Register(&mockConnector{name: "claude"})
	reg.Register(&mockConnector{name: "gemini"})

	c, ok := reg.Get("claude")
	if !ok || c.Name() != "claude" {
		t.Fatal("expected to find claude connector")
	}

	_, ok = reg.Get("nonexistent")
	if ok {
		t.Fatal("expected not found for nonexistent connector")
	}
}

func TestConnectorRegistry_Names(t *testing.T) {
	reg := NewConnectorRegistry()
	reg.Register(&mockConnector{name: "claude"})
	reg.Register(&mockConnector{name: "gemini"})

	names := reg.Names()
	if len(names) != 2 {
		t.Fatalf("expected 2 names, got %d", len(names))
	}

	nameSet := map[string]bool{}
	for _, n := range names {
		nameSet[n] = true
	}
	if !nameSet["claude"] || !nameSet["gemini"] {
		t.Fatal("expected both claude and gemini in names")
	}
}

func TestConnectorRegistry_OverwriteExisting(t *testing.T) {
	reg := NewConnectorRegistry()
	reg.Register(&mockConnector{name: "claude"})
	reg.Register(&mockConnector{name: "claude"}) // overwrite

	names := reg.Names()
	if len(names) != 1 {
		t.Fatalf("expected 1 name after overwrite, got %d", len(names))
	}
}
