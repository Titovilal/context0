package gemini

import (
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

type geminiOutput struct {
	SessionID string `json:"session_id"`
	Result    string `json:"result"`
	IsError   bool   `json:"is_error"`
}

// Connector implements connector.AgentConnector for Gemini CLI.
// NOTE: Gemini CLI session management differs from Claude.
// Rewinds are destructive (no --fork-session equivalent).
type Connector struct {
	geminiBin string
}

func New() *Connector {
	return &Connector{geminiBin: "gemini"}
}

func (c *Connector) Name() string { return Name }

func (c *Connector) Run(ctx context.Context, req connector.RunRequest) (connector.RunResult, error) {
	args := []string{"--output-format", "json"}

	if req.SessionID != "" {
		args = append(args, "--resume", req.SessionID)
	}
	if req.SystemPromptAppend != "" {
		args = append(args, "--system-prompt", req.SystemPromptAppend)
	}
	args = append(args, "--prompt", req.Prompt)

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

	var out geminiOutput
	if err := json.Unmarshal(stdout.Bytes(), &out); err != nil {
		return connector.RunResult{
			IsError:     true,
			ErrorDetail: fmt.Sprintf("failed to parse gemini output: %v\nraw: %s", err, stdout.String()),
		}, nil
	}

	return connector.RunResult{
		SessionID: out.SessionID,
		FinalText: out.Result,
		IsError:   out.IsError,
	}, nil
}

// Fork for Gemini is a degraded rewind: resumes from the checkpoint index in place.
// The original session is modified. This is a known limitation.
func (c *Connector) Fork(ctx context.Context, sourceSessionID string, checkpoint connector.Checkpoint) (string, error) {
	req := connector.RunRequest{
		SessionID: checkpoint.NativeRef, // numeric index for Gemini
		Prompt:    "__mdm_fork__",
		Timeout:   30 * time.Second,
	}
	result, err := c.Run(ctx, req)
	if err != nil {
		return "", fmt.Errorf("gemini rewind failed: %w", err)
	}
	return result.SessionID, nil
}

// TurnCount for Gemini is a stub — session file format TBD.
func (c *Connector) TurnCount(ctx context.Context, sessionID string) (int, error) {
	return 0, fmt.Errorf("TurnCount not yet implemented for gemini connector")
}
