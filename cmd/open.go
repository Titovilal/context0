package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
)

var openFlags struct {
	connector string
}

var openCmd = &cobra.Command{
	Use:   "open [request]",
	Short: "Open a Middleman session with a request",
	Long: fmt.Sprintf(`Opens an AI CLI as a Middleman agent. The Middleman receives
the guide from .mdm/guides/the_middleman.md, is instructed to create
subagents in the background without asking, and processes your request.

Supported connectors: %s`, strings.Join(connectorNames(), ", ")),
	Args: cobra.MinimumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		wd := workDir

		connName := openFlags.connector
		if connName == "" {
			cfg := loadConfig(filepath.Join(wd, ".mdm"))
			connName = cfg.DefaultCLI
		}
		conn, ok := connectors[connName]
		if !ok {
			return fmt.Errorf("unknown connector %q (available: %s)", connName, strings.Join(connectorNames(), ", "))
		}

		middlemanGuide, err := os.ReadFile(filepath.Join(wd, ".mdm", "guides", "the_middleman.md"))
		if err != nil {
			return fmt.Errorf("read the_middleman.md: %w (run 'mdm init' first)", err)
		}

		userRequest := strings.Join(args, " ")

		prompt := fmt.Sprintf(`You are the Middleman Agent. Follow the guide below strictly.
Create subagents in the background. Do not ask for confirmation, just act.

## Middleman Guide
%s

## User Request
%s`, string(middlemanGuide), userRequest)

		fmt.Fprintln(os.Stderr)
		fmt.Fprintln(os.Stderr, cDim+"  ▸ "+cReset+"Opening middleman with "+stValue(conn.Name)+stDim("..."))

		result, err := conn.Run(wd, prompt)
		if err != nil {
			return err
		}

		fmt.Println(result)
		return nil
	},
}

func init() {
	openCmd.Flags().StringVarP(&openFlags.connector, "connector", "c", "", "AI CLI to use (default: from config)")
	rootCmd.AddCommand(openCmd)
}
