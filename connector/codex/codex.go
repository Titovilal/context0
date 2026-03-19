package codex

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"time"

	"github.com/Titovilal/middleman/connector"
)

const Name = "codex"

// codexEvent is one JSON line from codex exec --json output.
type codexEvent struct {
	Type string `json:"type"`
	// item.completed events contain the final message
	Item *codexItem `json:"item,omitempty"`
}

type codexItem struct {
	Type    string `json:"type"` // "message", "reasoning", "command", etc.
	Content string `json:"content,omitempty"`
}

// Connector implements connector.AgentConnector for OpenAI Codex CLI.
type Connector struct {
	codexBin    string
	sessionsDir string
}

func New() *Connector {
	home, _ := os.UserHomeDir()
	return &Connector{
		codexBin:    "codex",
		sessionsDir: filepath.Join(home, ".codex", "sessions"),
	}
}

func (c *Connector) Name() string { return Name }

func (c *Connector) Run(ctx context.Context, req connector.RunRequest) (connector.RunResult, error) {
	args := []string{"exec"}

	// Bypass approvals and sandbox for non-interactive use.
	args = append(args, "--dangerously-bypass-approvals-and-sandbox")
	// JSON output for machine consumption.
	args = append(args, "--json")

	if req.WorkDir != "" {
		args = append(args, "--cd", req.WorkDir)
	}

	// Resume session if provided.
	if req.SessionID != "" || req.ForkFromSessionID != "" {
		resumeID := req.SessionID
		if req.ForkFromSessionID != "" {
			resumeID = req.ForkFromSessionID
		}
		args = append(args, "resume", resumeID)
		// Follow-up prompt after resume.
		if req.Prompt != "" {
			args = append(args, req.Prompt)
		}
	} else {
		args = append(args, req.Prompt)
	}

	if req.Timeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, req.Timeout)
		defer cancel()
	}

	cmd := exec.CommandContext(ctx, c.codexBin, args...)
	// Don't set cmd.Dir since we use --cd flag.

	var stdout bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		if stdout.Len() == 0 {
			return connector.RunResult{IsError: true, ErrorDetail: err.Error()}, nil
		}
	}

	// Parse JSONL output. Extract the last message content as the final text.
	var finalText string
	var sessionID string
	scanner := bufio.NewScanner(&stdout)
	for scanner.Scan() {
		line := scanner.Bytes()
		if len(line) == 0 {
			continue
		}
		var evt codexEvent
		if err := json.Unmarshal(line, &evt); err != nil {
			continue
		}

		// Extract session ID from thread.started event.
		if evt.Type == "thread.started" {
			var raw map[string]interface{}
			if err := json.Unmarshal(line, &raw); err == nil {
				if sid, ok := raw["session_id"].(string); ok {
					sessionID = sid
				}
			}
		}

		// Collect final message from item.completed events.
		if evt.Type == "item.completed" && evt.Item != nil && evt.Item.Type == "message" {
			finalText = evt.Item.Content
		}
	}

	// If no JSONL parsed, use raw output as text.
	if finalText == "" && stdout.Len() > 0 {
		finalText = stdout.String()
	}

	return connector.RunResult{
		SessionID: sessionID,
		FinalText: finalText,
		IsError:   false,
	}, nil
}

// Fork for Codex resumes from a previous session.
// Codex does not have a --fork-session equivalent, so this resumes in place.
func (c *Connector) Fork(ctx context.Context, sourceSessionID string, checkpoint connector.Checkpoint) (string, error) {
	resumeID := sourceSessionID
	if checkpoint.NativeRef != "" {
		resumeID = checkpoint.NativeRef
	}
	req := connector.RunRequest{
		SessionID: resumeID,
		Prompt:    "ok",
		Timeout:   45 * time.Second,
	}
	result, err := c.Run(ctx, req)
	if err != nil {
		return "", fmt.Errorf("codex fork failed: %w", err)
	}
	if result.IsError {
		return "", fmt.Errorf("codex fork failed: %s", result.ErrorDetail)
	}
	return result.SessionID, nil
}

func (c *Connector) SupportsFork() bool { return false }

// TurnCount reads session JSONL files and counts turn.completed events.
func (c *Connector) TurnCount(ctx context.Context, sessionID string) (int, error) {
	path := c.sessionFilePath(sessionID)
	if path == "" {
		return 0, fmt.Errorf("session file not found for session %s", sessionID)
	}

	f, err := os.Open(path)
	if err != nil {
		return 0, fmt.Errorf("open session file: %w", err)
	}
	defer f.Close()

	count := 0
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := scanner.Bytes()
		if len(line) == 0 {
			continue
		}
		var evt codexEvent
		if err := json.Unmarshal(line, &evt); err != nil {
			continue
		}
		if evt.Type == "turn.completed" {
			count++
		}
	}
	return count, scanner.Err()
}

// sessionFilePath tries to locate the session file for a given session ID.
func (c *Connector) sessionFilePath(sessionID string) string {
	// Try direct path first.
	candidate := filepath.Join(c.sessionsDir, sessionID+".jsonl")
	if _, err := os.Stat(candidate); err == nil {
		return candidate
	}

	// Scan subdirectories.
	entries, err := os.ReadDir(c.sessionsDir)
	if err != nil {
		return ""
	}
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		candidate := filepath.Join(c.sessionsDir, entry.Name(), sessionID+".jsonl")
		if _, err := os.Stat(candidate); err == nil {
			return candidate
		}
	}
	return ""
}
