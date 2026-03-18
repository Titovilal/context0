package opencode

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"

	"github.com/Titovilal/middleman/connector"
)

const Name = "opencode"

type opencodeOutput struct {
	SessionID string `json:"session_id"`
	Result    string `json:"result"`
	IsError   bool   `json:"is_error"`
}

// Connector implements connector.AgentConnector for OpenCode CLI.
// Stub implementation — session/resume API to be confirmed.
type Connector struct {
	opencodeBin string
}

func New() *Connector {
	return &Connector{opencodeBin: "opencode"}
}

func (c *Connector) Name() string { return Name }

func (c *Connector) Run(ctx context.Context, req connector.RunRequest) (connector.RunResult, error) {
	args := []string{"--output-format", "json"}

	if req.SessionID != "" {
		args = append(args, "--resume", req.SessionID)
	}
	args = append(args, req.Prompt)

	cmd := exec.CommandContext(ctx, c.opencodeBin, args...)
	cmd.Dir = req.WorkDir

	var stdout bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		if stdout.Len() == 0 {
			return connector.RunResult{IsError: true, ErrorDetail: err.Error()}, nil
		}
	}

	var out opencodeOutput
	if err := json.Unmarshal(stdout.Bytes(), &out); err != nil {
		return connector.RunResult{
			IsError:     true,
			ErrorDetail: fmt.Sprintf("failed to parse opencode output: %v\nraw: %s", err, stdout.String()),
		}, nil
	}

	return connector.RunResult{
		SessionID: out.SessionID,
		FinalText: out.Result,
		IsError:   out.IsError,
	}, nil
}

func (c *Connector) Fork(ctx context.Context, sourceSessionID string, checkpoint connector.Checkpoint) (string, error) {
	return "", fmt.Errorf("Fork not yet implemented for opencode connector")
}

func (c *Connector) TurnCount(ctx context.Context, sessionID string) (int, error) {
	return 0, fmt.Errorf("TurnCount not yet implemented for opencode connector")
}
