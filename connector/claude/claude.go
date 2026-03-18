package claude

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/Titovilal/middleman/connector"
)

const Name = "claude"

// claudeOutput is the JSON structure returned by claude --output-format json.
type claudeOutput struct {
	Type      string `json:"type"`
	SessionID string `json:"session_id"`
	Result    string `json:"result"`
	IsError   bool   `json:"is_error"`
}

// claudeMessage is one entry in a session JSONL file.
type claudeMessage struct {
	UUID       string `json:"uuid"`
	ParentUUID string `json:"parentUuid"`
	Type       string `json:"type"` // "user" | "assistant" | "summary"
}

// Connector implements connector.AgentConnector for Claude Code CLI.
type Connector struct {
	// claudeBin is the path to the claude binary. Defaults to "claude" (PATH lookup).
	claudeBin string

	// projectsDir is ~/.claude/projects by default.
	projectsDir string
}

func New() *Connector {
	home, _ := os.UserHomeDir()
	return &Connector{
		claudeBin:   "claude",
		projectsDir: filepath.Join(home, ".claude", "projects"),
	}
}

func (c *Connector) Name() string { return Name }

func (c *Connector) Run(ctx context.Context, req connector.RunRequest) (connector.RunResult, error) {
	args := []string{"--print", "--output-format", "json"}

	sessionID := req.SessionID
	if req.ForkFromSessionID != "" {
		sessionID = req.ForkFromSessionID
	}
	if sessionID != "" {
		args = append(args, "--resume", sessionID)
	}
	if req.ForkFromSessionID != "" {
		args = append(args, "--fork-session")
	}
	if req.SystemPromptAppend != "" {
		args = append(args, "--append-system-prompt", req.SystemPromptAppend)
	}

	args = append(args, req.Prompt)

	if req.Timeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, req.Timeout)
		defer cancel()
	}

	cmd := exec.CommandContext(ctx, c.claudeBin, args...)
	cmd.Dir = req.WorkDir

	var stdout bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = os.Stderr // debug: connector errors go to mdm stderr, never to result

	if err := cmd.Run(); err != nil {
		// Non-zero exit: try to parse output anyway (claude may include error JSON)
		if stdout.Len() == 0 {
			return connector.RunResult{IsError: true, ErrorDetail: err.Error()}, nil
		}
	}

	var out claudeOutput
	if err := json.Unmarshal(stdout.Bytes(), &out); err != nil {
		return connector.RunResult{
			IsError:     true,
			ErrorDetail: fmt.Sprintf("failed to parse claude output: %v\nraw: %s", err, stdout.String()),
		}, nil
	}

	return connector.RunResult{
		SessionID:   out.SessionID,
		FinalText:   out.Result,
		IsError:     out.IsError,
		ErrorDetail: "",
	}, nil
}

func (c *Connector) Fork(ctx context.Context, sourceSessionID string, checkpoint connector.Checkpoint) (string, error) {
	// Resume from the checkpoint's NativeRef (UUID of the last assistant message
	// at that point in the session), not from the current session tip.
	// This is what makes the rewind actually go back in time.
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
		return "", fmt.Errorf("fork failed: %w", err)
	}
	if result.IsError {
		return "", fmt.Errorf("fork failed: %s", result.ErrorDetail)
	}
	return result.SessionID, nil
}

func (c *Connector) TurnCount(ctx context.Context, sessionID string) (int, error) {
	path, err := c.sessionFilePath(sessionID)
	if err != nil {
		return 0, err
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
		var msg claudeMessage
		if err := json.Unmarshal(line, &msg); err != nil {
			continue
		}
		if msg.Type == "assistant" {
			count++
		}
	}
	return count, scanner.Err()
}

// LastAssistantUUID returns the UUID of the Nth assistant message (1-indexed).
// Used to create checkpoint NativeRef values.
func (c *Connector) LastAssistantUUID(sessionID string) (string, error) {
	path, err := c.sessionFilePath(sessionID)
	if err != nil {
		return "", err
	}

	f, err := os.Open(path)
	if err != nil {
		return "", fmt.Errorf("open session file: %w", err)
	}
	defer f.Close()

	var lastUUID string
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := scanner.Bytes()
		if len(line) == 0 {
			continue
		}
		var msg claudeMessage
		if err := json.Unmarshal(line, &msg); err != nil {
			continue
		}
		if msg.Type == "assistant" {
			lastUUID = msg.UUID
		}
	}
	if err := scanner.Err(); err != nil {
		return "", err
	}
	if lastUUID == "" {
		return "", fmt.Errorf("no assistant messages found in session %s", sessionID)
	}
	return lastUUID, nil
}

// sessionFilePath finds the JSONL file for a given session ID by scanning
// all project directories under ~/.claude/projects/.
func (c *Connector) sessionFilePath(sessionID string) (string, error) {
	entries, err := os.ReadDir(c.projectsDir)
	if err != nil {
		return "", fmt.Errorf("read projects dir %s: %w", c.projectsDir, err)
	}
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		candidate := filepath.Join(c.projectsDir, entry.Name(), sessionID+".jsonl")
		if _, err := os.Stat(candidate); err == nil {
			return candidate, nil
		}
	}
	return "", fmt.Errorf("session file not found for session %s in %s", sessionID, c.projectsDir)
}

// WorkDirSlug returns the directory slug Claude uses for a given workDir path.
// Claude slugifies paths by replacing / with - and stripping the leading -.
func WorkDirSlug(workDir string) string {
	slug := strings.ReplaceAll(workDir, "/", "-")
	slug = strings.TrimPrefix(slug, "-")
	return slug
}
