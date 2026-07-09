package cmd

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
)

// connector defines how to call an AI CLI.
type connector struct {
	Name string
	// Run executes the prompt synchronously and returns the text response.
	Run func(workDir, prompt string) (string, error)
}

var connectors = map[string]connector{
	"claude":   {Name: "claude", Run: runClaude},
	"copilot":  {Name: "copilot", Run: runCopilot},
	"gemini":   {Name: "gemini", Run: runGemini},
	"codex":    {Name: "codex", Run: runCodex},
	"opencode": {Name: "opencode", Run: runOpenCode},
}

// runSync launches a CLI command and captures its stdout output.
func runSync(workDir string, name string, args ...string) (string, error) {
	c := exec.Command(name, args...)
	c.Dir = workDir
	c.Stderr = os.Stderr
	out, err := c.Output()
	if err != nil {
		return "", fmt.Errorf("%s failed: %w", name, err)
	}
	return string(out), nil
}

// --- Claude ---

type claudeOutput struct {
	Result  string `json:"result"`
	IsError bool   `json:"is_error"`
}

func runClaude(workDir, prompt string) (string, error) {
	c := exec.Command("claude", "--print", "--output-format", "json", "--model", "sonnet", "--dangerously-skip-permissions", prompt)
	c.Dir = workDir
	c.Stderr = os.Stderr

	var stdout bytes.Buffer
	c.Stdout = &stdout

	if err := c.Run(); err != nil {
		if stdout.Len() == 0 {
			return "", fmt.Errorf("claude failed: %w", err)
		}
	}

	var out claudeOutput
	if err := json.Unmarshal(stdout.Bytes(), &out); err != nil {
		return "", fmt.Errorf("failed to parse claude output: %v\nraw: %s", err, stdout.String())
	}
	if out.IsError {
		return "", fmt.Errorf("claude returned error: %s", out.Result)
	}
	return out.Result, nil
}

// --- Copilot (GitHub Copilot CLI) ---

func runCopilot(workDir, prompt string) (string, error) {
	return runSync(workDir, "copilot", "--prompt", prompt, "--silent")
}

// --- Gemini ---

func runGemini(workDir, prompt string) (string, error) {
	return runSync(workDir, "gemini", "--noinput", prompt)
}

// --- Codex ---

func runCodex(workDir, prompt string) (string, error) {
	// The current Codex CLI (Rust rewrite) runs non-interactively via the
	// `exec` subcommand. `--sandbox` and `--ask-for-approval` are GLOBAL flags
	// that must precede the subcommand — `codex exec --ask-for-approval` is
	// rejected as an unexpected argument. So the order is:
	//   codex --ask-for-approval never --sandbox workspace-write exec <prompt>
	return runSync(workDir, "codex", "--ask-for-approval", "never", "--sandbox", "workspace-write", "exec", prompt)
}

// --- OpenCode ---

func runOpenCode(workDir, prompt string) (string, error) {
	return runSync(workDir, "opencode", "--non-interactive", prompt)
}
