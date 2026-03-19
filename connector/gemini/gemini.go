package gemini

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

const Name = "gemini"

// geminiEvent is one JSON line from gemini CLI --output-format json.
type geminiEvent struct {
	Type      string `json:"type"`
	SessionID string `json:"session_id,omitempty"`
	Result    string `json:"result,omitempty"`
	Content   string `json:"content,omitempty"`
	IsError   bool   `json:"is_error,omitempty"`
}

// Connector implements connector.AgentConnector for Gemini CLI.
// Uses -p for non-interactive mode and --yolo to bypass approvals.
type Connector struct {
	geminiBin string
}

func New() *Connector {
	return &Connector{geminiBin: "gemini"}
}

func (c *Connector) Name() string { return Name }

func (c *Connector) Run(ctx context.Context, req connector.RunRequest) (connector.RunResult, error) {
	args := []string{"--yolo", "--output-format", "json"}

	if req.SessionID != "" {
		args = append(args, "--resume", req.SessionID)
	}
	if req.ForkFromSessionID != "" {
		args = append(args, "--resume", req.ForkFromSessionID)
	}
	if req.SystemPromptAppend != "" {
		args = append(args, "--system-prompt", req.SystemPromptAppend)
	}
	args = append(args, "-p", req.Prompt)

	if req.Timeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, req.Timeout)
		defer cancel()
	}

	cmd := exec.CommandContext(ctx, c.geminiBin, args...)
	cmd.Dir = req.WorkDir

	var stdout bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		if stdout.Len() == 0 {
			return connector.RunResult{IsError: true, ErrorDetail: err.Error()}, nil
		}
	}

	// Try to parse as JSONL stream first (multiple lines).
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
		var evt geminiEvent
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
		isError = evt.IsError
	}

	// Fallback: try single JSON object if only one line.
	if finalText == "" && lineCount <= 1 {
		raw := stdout.Bytes()
		var single geminiEvent
		if err := json.Unmarshal(raw, &single); err == nil {
			sessionID = single.SessionID
			finalText = single.Result
			if finalText == "" {
				finalText = single.Content
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

// Fork for Gemini is a degraded rewind: resumes from the checkpoint index in place.
// The original session is modified. This is a known limitation.
func (c *Connector) Fork(ctx context.Context, sourceSessionID string, checkpoint connector.Checkpoint) (string, error) {
	resumeID := sourceSessionID
	if checkpoint.NativeRef != "" {
		resumeID = checkpoint.NativeRef
	}
	req := connector.RunRequest{
		SessionID: resumeID,
		Prompt:    "ok",
		Timeout:   30 * time.Second,
	}
	result, err := c.Run(ctx, req)
	if err != nil {
		return "", fmt.Errorf("gemini fork failed: %w", err)
	}
	if result.IsError {
		return "", fmt.Errorf("gemini fork failed: %s", result.ErrorDetail)
	}
	return result.SessionID, nil
}

func (c *Connector) SupportsFork() bool { return false }

// TurnCount for Gemini returns 0 to allow checkpoints to work (degraded).
func (c *Connector) TurnCount(ctx context.Context, sessionID string) (int, error) {
	return 0, nil
}
