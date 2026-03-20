package cmd

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
)

// connector defines how to call an AI CLI to run a prompt.
type connector struct {
	Name string
	// Run executes the prompt and returns the text response.
	Run func(workDir, prompt string) (string, error)
}

var connectors = map[string]connector{
	"claude":  {Name: "claude", Run: runClaude},
	"copilot": {Name: "copilot", Run: runCopilot},
	"gemini":  {Name: "gemini", Run: runGemini},
	"codex":   {Name: "codex", Run: runCodex},
	"opencode": {Name: "opencode", Run: runOpenCode},
}

// --- Claude ---

type claudeOutput struct {
	Result  string `json:"result"`
	IsError bool   `json:"is_error"`
}

func runClaude(workDir, prompt string) (string, error) {
	c := exec.Command("claude", "--print", "--output-format", "json", "--dangerously-skip-permissions", prompt)
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
	c := exec.Command("copilot", "--prompt", prompt, "--silent")
	c.Dir = workDir
	c.Stderr = os.Stderr

	out, err := c.Output()
	if err != nil {
		return "", fmt.Errorf("copilot failed: %w", err)
	}
	return string(out), nil
}

// --- Gemini ---

func runGemini(workDir, prompt string) (string, error) {
	c := exec.Command("gemini", "--noinput", prompt)
	c.Dir = workDir
	c.Stderr = os.Stderr

	out, err := c.Output()
	if err != nil {
		return "", fmt.Errorf("gemini failed: %w", err)
	}
	return string(out), nil
}

// --- Codex ---

func runCodex(workDir, prompt string) (string, error) {
	c := exec.Command("codex", "--approval-mode", "full-auto", "--quiet", prompt)
	c.Dir = workDir
	c.Stderr = os.Stderr

	out, err := c.Output()
	if err != nil {
		return "", fmt.Errorf("codex failed: %w", err)
	}
	return string(out), nil
}

// --- OpenCode ---

func runOpenCode(workDir, prompt string) (string, error) {
	c := exec.Command("opencode", "--non-interactive", prompt)
	c.Dir = workDir
	c.Stderr = os.Stderr

	out, err := c.Output()
	if err != nil {
		return "", fmt.Errorf("opencode failed: %w", err)
	}
	return string(out), nil
}
