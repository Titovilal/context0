package cmd

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"

	"github.com/spf13/cobra"
)

var launchFlags struct {
	workDir string
}

// launchSpec defines how to start an interactive session for a given connector.
type launchSpec struct {
	bin       string
	buildArgs func(promptFile string) []string
}

var launchSpecs = map[string]launchSpec{
	"claude": {
		bin: "claude",
		buildArgs: func(promptFile string) []string {
			return []string{
				"--dangerously-skip-permissions",
				"You are now the Middleman. Always use the mdm CLI — never create or manage agents directly. Read this and act accordingly: @" + promptFile,
			}
		},
	},
	"gemini": {
		bin: "gemini",
		buildArgs: func(promptFile string) []string {
			data, _ := os.ReadFile(promptFile)
			return []string{"--system-prompt", string(data)}
		},
	},
	"codex": {
		bin: "codex",
		buildArgs: func(promptFile string) []string {
			data, _ := os.ReadFile(promptFile)
			return []string{"--instructions", string(data)}
		},
	},
}

var launchCmd = &cobra.Command{
	Use:         "launch <connector>",
	Short:       "Launch an interactive AI CLI session with the Middleman prompt injected",
	Annotations: map[string]string{"skip_init": "true"},
	Long: `Launches the specified AI CLI tool (claude, gemini, codex) in interactive mode
with the Middleman orchestrator prompt automatically injected. The user gets
a fully interactive session where the AI is already acting as the Middleman.

Examples:
  mdm launch claude
  mdm launch gemini
  mdm launch codex`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		connName := args[0]

		spec, ok := launchSpecs[connName]
		if !ok {
			names := make([]string, 0, len(launchSpecs))
			for n := range launchSpecs {
				names = append(names, n)
			}
			sort.Strings(names)
			return fmt.Errorf("unknown connector %q (available: %s)", connName, strings.Join(names, ", "))
		}

		// Resolve mdm binary path for the prompt.
		mdmBin, err := os.Executable()
		if err != nil {
			mdmBin = "mdm"
		} else {
			mdmBin, _ = filepath.Abs(mdmBin)
		}

		workDir := launchFlags.workDir
		if workDir == "" {
			workDir, _ = os.Getwd()
		}

		// Build prompt: binary/workdir + guide + project overview.
		guidePath := filepath.Join(workDir, ".mdm", "guides", "how_mdm_works.md")
		guideContent, err := os.ReadFile(guidePath)
		if err != nil {
			return fmt.Errorf(".mdm/ is not initialized. Run 'mdm sync-docs' first")
		}
		prompt := fmt.Sprintf("## MDM binary\n\n  %s\n\n## Working directory\n\n  %s\n\n%s", mdmBin, workDir, string(guideContent))

		// Append project overview if available, warn if docs are not populated.
		overviewPath := filepath.Join(workDir, ".mdm", "docs", "project_overview.md")
		if overview, err := os.ReadFile(overviewPath); err == nil {
			prompt += "\n\n" + string(overview)
		} else {
			fmt.Fprintf(os.Stderr, "Warning: .mdm/docs/ is not populated. Run 'mdm sync-docs' to generate project documentation.\n")
		}

		// Write combined prompt to .mdm/.prompt_cache.md inside the project.
		promptFile := filepath.Join(workDir, ".mdm", ".prompt_cache.md")
		if err := os.MkdirAll(filepath.Dir(promptFile), 0o755); err != nil {
			return fmt.Errorf("create .mdm dir: %w", err)
		}
		if err := os.WriteFile(promptFile, []byte(prompt), 0o644); err != nil {
			return fmt.Errorf("write prompt file: %w", err)
		}
		defer os.Remove(promptFile)

		// Build the command.
		cliArgs := spec.buildArgs(promptFile)
		cliPath, err := exec.LookPath(spec.bin)
		if err != nil {
			return fmt.Errorf("%s CLI not found in PATH: %w", spec.bin, err)
		}

		c := exec.Command(cliPath, cliArgs...)
		c.Dir = workDir
		c.Stdin = os.Stdin
		c.Stdout = os.Stdout
		c.Stderr = os.Stderr

		fmt.Fprintf(os.Stderr, "Launching %s as Middleman orchestrator...\n", connName)

		if err := c.Run(); err != nil {
			// If the CLI exited with a non-zero code, propagate it.
			if exitErr, ok := err.(*exec.ExitError); ok {
				os.Exit(exitErr.ExitCode())
			}
			return fmt.Errorf("failed to run %s: %w", spec.bin, err)
		}
		return nil
	},
}

func init() {
	launchCmd.Flags().StringVarP(&launchFlags.workDir, "workdir", "w", "", "project directory (default: current dir)")
	rootCmd.AddCommand(launchCmd)
}
