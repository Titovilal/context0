package cmd

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/Titovilal/middleman/agent"
	"github.com/spf13/cobra"
)

var syncDocsFlags struct {
	connector string
}

var syncDocsCmd = &cobra.Command{
	Use:   "sync-docs",
	Short: "Spawn an agent to create or update .mdm/docs/",
	Long: `Spawns a dedicated agent that reads the codebase and creates or updates
the documentation in .mdm/docs/ following the guide in .mdm/guides/how_to_manage_docs.md
and the templates in .mdm/templates/.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		connName := syncDocsFlags.connector
		if connName == "" {
			connName = cfg.DefaultConnector
		}

		wd := cfg.WorkDir

		// Read the guide and templates to build the task prompt.
		guide, err := os.ReadFile(filepath.Join(wd, ".mdm", "guides", "how_to_manage_docs.md"))
		if err != nil {
			return fmt.Errorf("read how_to_manage_docs.md: %w (run any mdm command first to initialize .mdm/)", err)
		}
		docTemplate, _ := os.ReadFile(filepath.Join(wd, ".mdm", "templates", "doc_template.md"))
		overviewTemplate, _ := os.ReadFile(filepath.Join(wd, ".mdm", "templates", "project_overview_template.md"))

		task := fmt.Sprintf(`Follow these instructions to create or update the project documentation.

## Guide
%s

## Doc template
%s

## Project overview template
%s`, string(guide), string(docTemplate), string(overviewTemplate))

		briefing := "You are a documentation agent. Your only job is to read the codebase and create/update .mdm/docs/ following the guide and templates provided."

		a, taskRec, err := orch.Spawn(context.Background(), "sync-docs", briefing, connName, task, 0)
		if err != nil {
			return err
		}

		fmt.Printf("agent: %s\n", a.ID)
		fmt.Printf("  connector: %s\n", a.ConnectorName)
		fmt.Printf("  session:   %s\n", a.SessionID)

		if taskRec != nil && taskRec.Status == agent.TaskPending {
			exe, err := os.Executable()
			if err != nil {
				return fmt.Errorf("resolve executable: %w", err)
			}
			bgCmd := exec.Command(exe, "_run-task", a.ID, taskRec.TaskID)
			bgCmd.Stdout = nil
			bgCmd.Stderr = nil
			bgCmd.Stdin = nil
			if err := bgCmd.Start(); err != nil {
				return fmt.Errorf("start background task: %w", err)
			}
			_ = bgCmd.Process.Release()
			fmt.Printf("  task_id:   %s\n", taskRec.TaskID)
			fmt.Printf("  status:    running in background\n")
			fmt.Printf("\nCheck results with: mdm result sync-docs\n")
		}

		return nil
	},
}

func init() {
	syncDocsCmd.Flags().StringVar(&syncDocsFlags.connector, "connector", "", "connector to use (overrides default)")
	rootCmd.AddCommand(syncDocsCmd)
}
