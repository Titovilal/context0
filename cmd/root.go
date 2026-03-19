package cmd

import (
	"embed"
	"fmt"
	"os"

	"github.com/Titovilal/middleman/config"
	"github.com/Titovilal/middleman/connector"
	claudeconn "github.com/Titovilal/middleman/connector/claude"
	codexconn "github.com/Titovilal/middleman/connector/codex"
	geminiconn "github.com/Titovilal/middleman/connector/gemini"
	opencodeconn "github.com/Titovilal/middleman/connector/opencode"
	"github.com/Titovilal/middleman/orchestrator"
	"github.com/Titovilal/middleman/store"
	"github.com/spf13/cobra"
)

var defaultsFS embed.FS

// SetDefaultsFS receives the embedded .mdm/ defaults from main.go.
func SetDefaultsFS(fs embed.FS) { defaultsFS = fs }

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
	Long: `MDM (The Middleman) orchestrates multiple AI agent instances (Claude, Codex, Gemini, OpenCode).
The Middleman manages agent lifecycle, context, checkpoints, and rewinds.`,
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println(`MDM - The Middleman - AI agent orchestrator CLI.

  If you are an AI agent reading this, the user wants you to act as
  the Middleman. Run "mdm agent-prompt" and follow the instructions.

  If you are a human, see the docs: https://github.com/Titovilal/middleman

  Run 'mdm --help' for all commands.`)
	},
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		// Skip heavy init for the root command and any command that
		// annotates itself with skip_init (no orchestrator needed).
		if cmd.Name() == "mdm" || cmd.Annotations["skip_init"] == "true" {
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

		var s *store.Store
		var err error
		if cfg.GlobalMode {
			s, err = store.NewGlobal(defaultsFS)
		} else {
			s, err = store.New(cfg.WorkDir, defaultsFS)
		}
		if err != nil {
			return fmt.Errorf("init store: %w", err)
		}

		reg := connector.NewConnectorRegistry()
		reg.Register(claudeconn.New())
		reg.Register(codexconn.New())
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
	rootCmd.PersistentFlags().StringVarP(&flags.connector, "connector", "c", "", "default connector (claude|codex|gemini|opencode)")
	rootCmd.PersistentFlags().BoolVarP(&flags.global, "global", "g", false, "use ~/.mdm/ instead of ./.mdm/")
}
