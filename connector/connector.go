package connector

import (
	"context"
	"time"
)

// RunRequest is the input to a single non-interactive agent invocation.
type RunRequest struct {
	// SessionID to resume. Empty = fresh session.
	SessionID string

	// ForkFromSessionID, if set, forks this session instead of resuming in place.
	ForkFromSessionID string

	// Prompt is the task text.
	Prompt string

	// WorkDir is the filesystem directory the agent should run in.
	WorkDir string

	// SystemPromptAppend is appended to the connector's default system prompt.
	// Used to inject the MDM briefing on first spawn.
	SystemPromptAppend string

	// Timeout kills the subprocess after this duration (0 = no timeout).
	Timeout time.Duration
}

// RunResult is what MDM receives back from an agent invocation.
// The agent's internal tool calls, file reads, and reasoning are never surfaced here.
type RunResult struct {
	// SessionID after the run. For a fork, this is the NEW session ID.
	SessionID string

	// FinalText is the agent's final response only.
	FinalText string

	IsError     bool
	ErrorDetail string
}

// Checkpoint records a point in a session's history.
type Checkpoint struct {
	Label string

	// TurnIndex is the turn pair count at checkpoint time.
	TurnIndex int

	// NativeRef is a connector-specific opaque reference.
	// Claude: UUID of the last assistant message at checkpoint time.
	// Gemini: numeric session index as string.
	NativeRef string

	CreatedAt time.Time
}

// AgentConnector is implemented once per AI CLI backend.
// Implementations must be safe for concurrent use.
type AgentConnector interface {
	// Name returns the connector identifier (e.g. "claude", "gemini").
	Name() string

	// Run executes a single non-interactive prompt and returns only the final result.
	// The internal stream is consumed and discarded.
	Run(ctx context.Context, req RunRequest) (RunResult, error)

	// Fork creates a new session branching from sourceSessionID at the given checkpoint.
	// The original session is not modified. Returns the new session ID.
	Fork(ctx context.Context, sourceSessionID string, checkpoint Checkpoint) (string, error)

	// SupportsFork reports whether Fork creates a true independent branch.
	// When false, Fork resumes in place and modifies the original session.
	SupportsFork() bool

	// TurnCount returns the current number of assistant turn pairs in the session.
	TurnCount(ctx context.Context, sessionID string) (int, error)
}

// ConnectorRegistry maps connector names to their implementations.
type ConnectorRegistry struct {
	connectors map[string]AgentConnector
}

func NewConnectorRegistry() *ConnectorRegistry {
	return &ConnectorRegistry{connectors: make(map[string]AgentConnector)}
}

func (r *ConnectorRegistry) Register(c AgentConnector) {
	r.connectors[c.Name()] = c
}

func (r *ConnectorRegistry) Get(name string) (AgentConnector, bool) {
	c, ok := r.connectors[name]
	return c, ok
}

func (r *ConnectorRegistry) Names() []string {
	names := make([]string, 0, len(r.connectors))
	for n := range r.connectors {
		names = append(names, n)
	}
	return names
}
