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
	bin  string
	// buildArgs returns the CLI arguments to launch interactively with prompt injection.
	buildArgs func(prompt string) []string
}

var launchSpecs = map[string]launchSpec{
	"claude": {
		bin: "claude",
		buildArgs: func(prompt string) []string {
			return []string{"--append-system-prompt", prompt}
		},
	},
	"gemini": {
		bin: "gemini",
		buildArgs: func(prompt string) []string {
			return []string{"--system-prompt", prompt}
		},
	},
	"codex": {
		bin: "codex",
		buildArgs: func(prompt string) []string {
			return []string{"--instructions", prompt}
		},
	},
}

var launchCmd = &cobra.Command{
	Use:   "launch <connector>",
	Short: "Launch an interactive AI CLI session with the Middleman prompt injected",
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

		// Build prompt: binary/workdir info + guide file.
		guidePath := filepath.Join(workDir, ".mdm", "guides", "how_mdm_works.md")
		guideContent, err := os.ReadFile(guidePath)
		if err != nil {
			return fmt.Errorf("could not read guide at %s: %w", guidePath, err)
		}
		prompt := fmt.Sprintf("## MDM binary\n\n  %s\n\n## Working directory\n\n  %s\n\n%s", mdmBin, workDir, string(guideContent))

		// Build the command.
		cliArgs := spec.buildArgs(prompt)
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
