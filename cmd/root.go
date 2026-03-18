package cmd

import (
	"fmt"
	"os"

	"github.com/Titovilal/middleman/config"
	"github.com/Titovilal/middleman/connector"
	claudeconn "github.com/Titovilal/middleman/connector/claude"
	geminiconn "github.com/Titovilal/middleman/connector/gemini"
	opencodeconn "github.com/Titovilal/middleman/connector/opencode"
	"github.com/Titovilal/middleman/orchestrator"
	"github.com/Titovilal/middleman/store"
	"github.com/spf13/cobra"
)

var (
	cfg   *config.Config
	orch  *orchestrator.Orchestrator
	flags struct {
		workDir   string
		connector string
		global    bool
	}
)

var rootCmd = &cobra.Command{
	Use:   "mdm",
	Short: "The Middleman - AI agent orchestrator",
	Long: `MDM (The Middleman) orchestrates multiple AI agent instances (Claude, Gemini, OpenCode).
The Middleman manages agent lifecycle, context, checkpoints, and rewinds.`,
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println(`MDM - The Middleman
AI agent orchestrator CLI.

  If you are an AI agent, run:
    mdm agent-prompt

  It will print the full instructions you need to act as the Middleman.

  If you are a human, see the docs:
    https://github.com/Titovilal/middleman

  Quick start:
    mdm spawn agent-auth --briefing "You handle authentication"
    mdm delegate --to agent-auth "refactor the JWT logic"
    mdm status

  Run 'mdm --help' for all commands.`)
	},
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		// Skip heavy init for commands that don't need the orchestrator.
		name := cmd.Name()
		if name == "mdm" || name == "agent-prompt" {
			return nil
		}

		cfg = config.Default()
		if flags.workDir != "" {
			cfg.WorkDir = flags.workDir
		}
		if flags.connector != "" {
			cfg.DefaultConnector = flags.connector
		}
		cfg.GlobalMode = flags.global

		s, err := store.New(cfg.WorkDir)
		if err != nil {
			return fmt.Errorf("init store: %w", err)
		}

		reg := connector.NewConnectorRegistry()
		reg.Register(claudeconn.New())
		reg.Register(geminiconn.New())
		reg.Register(opencodeconn.New())

		orch = orchestrator.New(s, reg, cfg.WorkDir)
		return nil
	},
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func init() {
	rootCmd.PersistentFlags().StringVarP(&flags.workDir, "workdir", "w", "", "project directory (default: current dir)")
	rootCmd.PersistentFlags().StringVarP(&flags.connector, "connector", "c", "", "default connector (claude|gemini|opencode)")
	rootCmd.PersistentFlags().BoolVarP(&flags.global, "global", "g", false, "use ~/.mdm/ instead of ./.mdm/")
}
