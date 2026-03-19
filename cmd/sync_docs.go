package cmd

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
)

var syncDocsFlags struct {
	connector string
}

var syncDocsCmd = &cobra.Command{
	Use:   "sync-docs",
	Short: "Create or update .mdm/docs/ (runs synchronously)",
	Long: `Spawns a dedicated agent that reads the codebase and creates or updates
the documentation in .mdm/docs/ following the guide in .mdm/guides/how_to_manage_docs.md
and the templates in .mdm/templates/. Blocks until the agent finishes.`,
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

		if taskRec == nil {
			fmt.Println("sync-docs agent already exists with no new task")
			return nil
		}

		fmt.Fprintf(os.Stderr, "Syncing docs with %s...\n", a.ConnectorName)

		// Run synchronously — block until done.
		if err := orch.RunTask(context.Background(), a.ID, taskRec.TaskID, 0); err != nil {
			return fmt.Errorf("sync-docs failed: %w", err)
		}

		// Reload to get the completed task.
		a, err = orch.Inspect(context.Background(), a.ID)
		if err != nil {
			return err
		}
		t := a.TaskByID(taskRec.TaskID)
		if t != nil && t.Response != "" {
			fmt.Println(t.Response)
		}

		// Clean up the agent.
		_ = orch.Remove(context.Background(), a.ID)
		fmt.Fprintf(os.Stderr, "Done.\n")
		return nil
	},
}

func init() {
	syncDocsCmd.Flags().StringVar(&syncDocsFlags.connector, "connector", "", "connector to use (overrides default)")
	rootCmd.AddCommand(syncDocsCmd)
}
