package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
)

var agentPromptFlags struct {
	workDir string
}

var agentPromptCmd = &cobra.Command{
	Use:         "agent-prompt",
	Short:       "Print the system prompt for an AI agent acting as the Middleman",
	Long:        `Outputs instructions that an AI agent should follow to act as the MDM Middleman orchestrator. Pipe this into your agent's system prompt or CLAUDE.md.`,
	Annotations: map[string]string{"skip_init": "true"},
	RunE: func(cmd *cobra.Command, args []string) error {
		mdmBin, err := os.Executable()
		if err != nil {
			mdmBin = "mdm"
		} else {
			mdmBin, _ = filepath.Abs(mdmBin)
		}

		workDir := agentPromptFlags.workDir
		if workDir == "" {
			workDir, _ = os.Getwd()
		}

		// Read the guide from .mdm/guides/how_mdm_works.md
		guidePath := filepath.Join(workDir, ".mdm", "guides", "how_mdm_works.md")
		guideContent, err := os.ReadFile(guidePath)
		if err != nil {
			return fmt.Errorf("could not read guide at %s: %w", guidePath, err)
		}

		fmt.Printf("## MDM binary\n\n  %s\n\n## Working directory\n\n  %s\n\n", mdmBin, workDir)
		fmt.Print(string(guideContent))

		// Append project overview if available.
		overviewPath := filepath.Join(workDir, ".mdm", "docs", "project_overview.md")
		if overview, err := os.ReadFile(overviewPath); err == nil {
			fmt.Print("\n\n" + string(overview))
		}
		return nil
	},
}

func init() {
	agentPromptCmd.Flags().StringVarP(&agentPromptFlags.workDir, "workdir", "w", "", "project directory to include in prompt (default: current dir)")
	rootCmd.AddCommand(agentPromptCmd)
}
