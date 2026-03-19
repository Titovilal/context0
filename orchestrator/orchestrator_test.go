package orchestrator

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/Titovilal/middleman/agent"
	"github.com/Titovilal/middleman/connector"
	"github.com/Titovilal/middleman/store"
)

// fakeConnector is a minimal connector for testing orchestrator logic.
type fakeConnector struct {
	runResult connector.RunResult
	runErr    error
}

func (f *fakeConnector) Name() string { return "fake" }
func (f *fakeConnector) Run(ctx context.Context, req connector.RunRequest) (connector.RunResult, error) {
	return f.runResult, f.runErr
}
func (f *fakeConnector) Fork(ctx context.Context, src string, cp connector.Checkpoint) (string, error) {
	return "forked-session", nil
}
func (f *fakeConnector) SupportsFork() bool                                           { return true }
func (f *fakeConnector) TurnCount(ctx context.Context, sessionID string) (int, error) { return 1, nil }

func setupOrch(t *testing.T, fc *fakeConnector) (*Orchestrator, string) {
	t.Helper()
	dir := t.TempDir()
	s, err := store.New(dir, nil)
	if err != nil {
		t.Fatalf("store.New: %v", err)
	}

	reg := connector.NewConnectorRegistry()
	reg.Register(fc)

	orch := New(s, reg, dir)
	return orch, dir
}

func TestSpawn_NewAgent(t *testing.T) {
	fc := &fakeConnector{
		runResult: connector.RunResult{SessionID: "sess-1", FinalText: "ready"},
	}
	orch, _ := setupOrch(t, fc)

	a, _, err := orch.Spawn(context.Background(), "test-agent", "do stuff", "fake", "", 0)
	if err != nil {
		t.Fatalf("Spawn error: %v", err)
	}
	if a.ID != "test-agent" {
		t.Fatalf("expected ID test-agent, got %s", a.ID)
	}
	if a.Status != agent.StatusIdle {
		t.Fatalf("expected idle status, got %s", a.Status)
	}
	if a.SessionID != "sess-1" {
		t.Fatalf("expected session sess-1, got %s", a.SessionID)
	}
}

func TestSpawn_ExistingAgent_NoTask(t *testing.T) {
	fc := &fakeConnector{
		runResult: connector.RunResult{SessionID: "sess-1", FinalText: "ready"},
	}
	orch, _ := setupOrch(t, fc)

	// Spawn first.
	_, _, _ = orch.Spawn(context.Background(), "test-agent", "do stuff", "fake", "", 0)

	// Spawn again with no task — should return existing agent.
	a, taskRec, err := orch.Spawn(context.Background(), "test-agent", "do stuff", "fake", "", 0)
	if err != nil {
		t.Fatalf("Spawn error: %v", err)
	}
	if a == nil {
		t.Fatal("expected non-nil agent")
	}
	if taskRec != nil {
		t.Fatal("expected nil task record when no task given")
	}
}

func TestSpawn_ConnectorNotFound(t *testing.T) {
	fc := &fakeConnector{}
	orch, _ := setupOrch(t, fc)

	_, _, err := orch.Spawn(context.Background(), "test-agent", "do stuff", "nonexistent", "", 0)
	if err == nil {
		t.Fatal("expected error for missing connector")
	}
}

func TestInspect(t *testing.T) {
	fc := &fakeConnector{
		runResult: connector.RunResult{SessionID: "sess-1", FinalText: "ok"},
	}
	orch, _ := setupOrch(t, fc)

	_, _, _ = orch.Spawn(context.Background(), "my-agent", "brief", "fake", "", 0)

	a, err := orch.Inspect(context.Background(), "my-agent")
	if err != nil {
		t.Fatalf("Inspect error: %v", err)
	}
	if a.ID != "my-agent" {
		t.Fatalf("expected my-agent, got %s", a.ID)
	}
}

func TestInspect_NotFound(t *testing.T) {
	fc := &fakeConnector{}
	orch, _ := setupOrch(t, fc)

	_, err := orch.Inspect(context.Background(), "nonexistent")
	if err == nil {
		t.Fatal("expected error for nonexistent agent")
	}
}

func TestRemove(t *testing.T) {
	fc := &fakeConnector{
		runResult: connector.RunResult{SessionID: "sess-1", FinalText: "ok"},
	}
	orch, _ := setupOrch(t, fc)

	_, _, _ = orch.Spawn(context.Background(), "to-remove", "brief", "fake", "", 0)

	if err := orch.Remove(context.Background(), "to-remove"); err != nil {
		t.Fatalf("Remove error: %v", err)
	}

	_, err := orch.Inspect(context.Background(), "to-remove")
	if err == nil {
		t.Fatal("expected error after removal")
	}
}

func TestListAgents(t *testing.T) {
	fc := &fakeConnector{
		runResult: connector.RunResult{SessionID: "sess-1", FinalText: "ok"},
	}
	orch, _ := setupOrch(t, fc)

	_, _, _ = orch.Spawn(context.Background(), "a1", "brief1", "fake", "", 0)
	_, _, _ = orch.Spawn(context.Background(), "a2", "brief2", "fake", "", 0)

	agents, err := orch.ListAgents(context.Background())
	if err != nil {
		t.Fatalf("ListAgents error: %v", err)
	}
	if len(agents) != 2 {
		t.Fatalf("expected 2 agents, got %d", len(agents))
	}
}

func TestListAgents_FilterByStatus(t *testing.T) {
	fc := &fakeConnector{
		runResult: connector.RunResult{SessionID: "sess-1", FinalText: "ok"},
	}
	orch, _ := setupOrch(t, fc)

	_, _, _ = orch.Spawn(context.Background(), "a1", "brief1", "fake", "", 0)

	idle, err := orch.ListAgents(context.Background(), agent.StatusIdle)
	if err != nil {
		t.Fatalf("error: %v", err)
	}
	if len(idle) != 1 {
		t.Fatalf("expected 1 idle, got %d", len(idle))
	}

	working, _ := orch.ListAgents(context.Background(), agent.StatusWorking)
	if len(working) != 0 {
		t.Fatalf("expected 0 working, got %d", len(working))
	}
}

func TestDelegateAsync(t *testing.T) {
	fc := &fakeConnector{
		runResult: connector.RunResult{SessionID: "sess-1", FinalText: "ok"},
	}
	orch, _ := setupOrch(t, fc)

	_, _, _ = orch.Spawn(context.Background(), "worker", "brief", "fake", "", 0)

	task, err := orch.DelegateAsync(context.Background(), "worker", "do something", 5*time.Minute)
	if err != nil {
		t.Fatalf("DelegateAsync error: %v", err)
	}
	if task.TaskID == "" {
		t.Fatal("expected non-empty task ID")
	}
	if task.Status != agent.TaskPending {
		t.Fatalf("expected pending status, got %s", task.Status)
	}

	// Agent should now be working.
	a, _ := orch.Inspect(context.Background(), "worker")
	if a.Status != agent.StatusWorking {
		t.Fatalf("expected working status, got %s", a.Status)
	}
}

func TestDelegateAsync_NotFound(t *testing.T) {
	fc := &fakeConnector{}
	orch, _ := setupOrch(t, fc)

	_, err := orch.DelegateAsync(context.Background(), "nonexistent", "task", 0)
	if err == nil {
		t.Fatal("expected error for nonexistent agent")
	}
}

func TestBuildBriefing(t *testing.T) {
	fc := &fakeConnector{}
	orch, dir := setupOrch(t, fc)

	// Create guide and overview files.
	guidesDir := filepath.Join(dir, ".mdm", "guides")
	docsDir := filepath.Join(dir, ".mdm", "docs")
	_ = os.MkdirAll(guidesDir, 0o755)
	_ = os.MkdirAll(docsDir, 0o755)

	_ = os.WriteFile(filepath.Join(guidesDir, "how_agents_should_behave.md"), []byte("be nice"), 0o644)
	_ = os.WriteFile(filepath.Join(docsDir, "project_overview.md"), []byte("overview content"), 0o644)
	_ = os.WriteFile(filepath.Join(docsDir, "extra_doc.md"), []byte("extra"), 0o644)

	result := orch.buildBriefing("my briefing")

	// Check that all sections are present.
	checks := []string{
		"# Current project state",
		"## .mdm/guides/how_agents_should_behave.md",
		"be nice",
		"## .mdm/docs/project_overview.md",
		"overview content",
		"## Available docs in .mdm/docs/",
		"- extra_doc.md",
		"## Briefing",
		"my briefing",
	}
	for _, check := range checks {
		if !contains(result, check) {
			t.Errorf("expected briefing to contain %q", check)
		}
	}
}

func TestBuildBriefing_EmptyBriefing(t *testing.T) {
	fc := &fakeConnector{}
	orch, _ := setupOrch(t, fc)

	result := orch.buildBriefing("")
	if contains(result, "## Briefing") {
		t.Error("empty briefing should not produce a Briefing section")
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && searchString(s, substr)
}

func searchString(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
