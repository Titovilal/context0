package opencode

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"time"

	"github.com/Titovilal/middleman/connector"
)

const Name = "opencode"

// opencodeEvent is one JSON line from opencode run --format json.
type opencodeEvent struct {
	Type      string `json:"type,omitempty"`
	SessionID string `json:"session_id,omitempty"`
	Result    string `json:"result,omitempty"`
	Content   string `json:"content,omitempty"`
	Message   string `json:"message,omitempty"`
	IsError   bool   `json:"is_error,omitempty"`
}

// Connector implements connector.AgentConnector for OpenCode CLI.
// Uses 'opencode run' for non-interactive execution.
type Connector struct {
	opencodeBin string
}

func New() *Connector {
	return &Connector{opencodeBin: "opencode"}
}

func (c *Connector) Name() string { return Name }

func (c *Connector) Run(ctx context.Context, req connector.RunRequest) (connector.RunResult, error) {
	args := []string{"run", "--format", "json"}

	// Session resume.
	if req.SessionID != "" {
		args = append(args, "--session", req.SessionID)
	}
	if req.ForkFromSessionID != "" {
		args = append(args, "--session", req.ForkFromSessionID, "--fork")
	}

	if req.WorkDir != "" {
		args = append(args, "--dir", req.WorkDir)
	}

	// Prompt as positional argument.
	args = append(args, req.Prompt)

	if req.Timeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, req.Timeout)
		defer cancel()
	}

	cmd := exec.CommandContext(ctx, c.opencodeBin, args...)
	// Don't set cmd.Dir since we use --dir flag.

	var stdout bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		if stdout.Len() == 0 {
			return connector.RunResult{IsError: true, ErrorDetail: err.Error()}, nil
		}
	}

	// Parse JSON output — try JSONL stream first.
	var finalText string
	var sessionID string
	var isError bool

	scanner := bufio.NewScanner(&stdout)
	lineCount := 0
	for scanner.Scan() {
		line := scanner.Bytes()
		if len(line) == 0 {
			continue
		}
		lineCount++
		var evt opencodeEvent
		if err := json.Unmarshal(line, &evt); err != nil {
			continue
		}
		if evt.SessionID != "" {
			sessionID = evt.SessionID
		}
		if evt.Result != "" {
			finalText = evt.Result
		}
		if evt.Content != "" {
			finalText = evt.Content
		}
		if evt.Message != "" {
			finalText = evt.Message
		}
		isError = evt.IsError
	}

	// Fallback: single JSON object.
	if finalText == "" && lineCount <= 1 {
		raw := stdout.Bytes()
		var single opencodeEvent
		if err := json.Unmarshal(raw, &single); err == nil {
			sessionID = single.SessionID
			finalText = single.Result
			if finalText == "" {
				finalText = single.Content
			}
			if finalText == "" {
				finalText = single.Message
			}
			isError = single.IsError
		} else if len(raw) > 0 {
			finalText = string(raw)
		}
	}

	return connector.RunResult{
		SessionID: sessionID,
		FinalText: finalText,
		IsError:   isError,
	}, nil
}

// Fork for OpenCode uses --session with --fork to branch a session.
func (c *Connector) Fork(ctx context.Context, sourceSessionID string, checkpoint connector.Checkpoint) (string, error) {
	resumeID := sourceSessionID
	if checkpoint.NativeRef != "" {
		resumeID = checkpoint.NativeRef
	}
	req := connector.RunRequest{
		ForkFromSessionID: resumeID,
		Prompt:            "ok",
		Timeout:           45 * time.Second,
	}
	result, err := c.Run(ctx, req)
	if err != nil {
		return "", fmt.Errorf("opencode fork failed: %w", err)
	}
	if result.IsError {
		return "", fmt.Errorf("opencode fork failed: %s", result.ErrorDetail)
	}
	return result.SessionID, nil
}

func (c *Connector) SupportsFork() bool { return true }

// TurnCount for OpenCode returns 0 to allow checkpoints (degraded).
func (c *Connector) TurnCount(ctx context.Context, sessionID string) (int, error) {
	return 0, nil
}
