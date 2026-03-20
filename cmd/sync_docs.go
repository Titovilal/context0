package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
)

var syncDocsFlags struct {
	connector string
}

var syncDocsCmd = &cobra.Command{
	Use:   "sync-docs",
	Short: "Create or update .mdm/docs/ using an AI CLI",
	Long: fmt.Sprintf(`Reads the codebase and creates or updates the documentation in .mdm/docs/
following the guide in .mdm/guides/how_to_sync_docs.md and the templates
in .mdm/templates/. Blocks until finished.

Supported connectors: %s`, strings.Join(connectorNames(), ", ")),
	RunE: func(cmd *cobra.Command, args []string) error {
		wd := workDir

		connName := syncDocsFlags.connector
		if connName == "" {
			cfg := loadConfig(filepath.Join(wd, ".mdm"))
			connName = cfg.DefaultCLI
		}
		conn, ok := connectors[connName]
		if !ok {
			return fmt.Errorf("unknown connector %q (available: %s)", connName, strings.Join(connectorNames(), ", "))
		}

		guide, err := os.ReadFile(filepath.Join(wd, ".mdm", "guides", "how_to_sync_docs.md"))
		if err != nil {
			return fmt.Errorf("read how_to_sync_docs.md: %w (run 'mdm init' first)", err)
		}
		docTemplate, _ := os.ReadFile(filepath.Join(wd, ".mdm", "templates", "doc_template.md"))
		overviewTemplate, _ := os.ReadFile(filepath.Join(wd, ".mdm", "templates", "project_overview_template.md"))

		prompt := fmt.Sprintf(`You are a documentation agent. Your only job is to read the codebase and create/update .mdm/docs/ following the guide and templates provided.

## Guide
%s

## Doc template
%s

## Project overview template
%s`, string(guide), string(docTemplate), string(overviewTemplate))

		fmt.Fprintln(os.Stderr)
		fmt.Fprintln(os.Stderr, cDim+"  ▸ "+cReset+"Syncing docs with "+stValue(conn.Name)+stDim("..."))

		result, err := conn.Run(wd, prompt)
		if err != nil {
			return err
		}

		fmt.Println(result)
		fmt.Fprintln(os.Stderr, stOk("  ✓ ")+"Done.")
		return nil
	},
}

func connectorNames() []string {
	names := make([]string, 0, len(connectors))
	for k := range connectors {
		names = append(names, k)
	}
	return names
}

func init() {
	syncDocsCmd.Flags().StringVarP(&syncDocsFlags.connector, "connector", "c", "", "AI CLI to use (default: claude)")
	rootCmd.AddCommand(syncDocsCmd)
}
