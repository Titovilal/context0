package cmd

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/Titovilal/middleman/agent"
	"github.com/spf13/cobra"
)

var spawnFlags struct {
	briefing  string
	connector string
	timeout   time.Duration
}

var spawnCmd = &cobra.Command{
	Use:   "spawn <name> [task]",
	Short: "Create agent and optionally delegate a task",
	Long: `Create a new agent with a briefing. If a task is provided, it runs in the background.
If the agent already exists, the task is queued into it.`,
	Args: cobra.MinimumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		id := args[0]
		connName := spawnFlags.connector
		if connName == "" {
			connName = cfg.DefaultConnector
		}

		task := ""
		if len(args) > 1 {
			task = strings.Join(args[1:], " ")
		}

		a, taskRec, err := orch.Spawn(context.Background(), id, spawnFlags.briefing, connName, task, spawnFlags.timeout)
		if err != nil {
			return err
		}

		fmt.Printf("agent: %s\n", a.ID)
		fmt.Printf("  connector: %s\n", a.ConnectorName)
		fmt.Printf("  session:   %s\n", a.SessionID)
		fmt.Printf("  status:    %s\n", a.Status)

		if taskRec != nil {
			// Launch background process if task is pending (not queued).
			if taskRec.Status == agent.TaskPending {
				exe, err := os.Executable()
				if err != nil {
					return fmt.Errorf("resolve executable: %w", err)
				}

				bgArgs := []string{"_run-task", id, taskRec.TaskID}
				if spawnFlags.timeout > 0 {
					bgArgs = append(bgArgs, "--timeout", spawnFlags.timeout.String())
				}

				bgCmd := exec.Command(exe, bgArgs...)
				bgCmd.Stdout = nil
				bgCmd.Stderr = nil
				bgCmd.Stdin = nil
				if err := bgCmd.Start(); err != nil {
					return fmt.Errorf("start background task: %w", err)
				}
				_ = bgCmd.Process.Release()
			}

			fmt.Printf("  task_id:   %s\n", taskRec.TaskID)
			fmt.Printf("  task:      %s\n", taskRec.Status)
			if taskRec.Status == agent.TaskQueued {
				fmt.Printf("\nAgent is busy. Task queued — it will run when the current task finishes.\n")
			}
		}
		return nil
	},
}

func init() {
	spawnCmd.Flags().StringVarP(&spawnFlags.briefing, "briefing", "b", "", "initial context for the agent")
	spawnCmd.Flags().StringVar(&spawnFlags.connector, "connector", "", "connector to use (overrides default)")
	spawnCmd.Flags().DurationVar(&spawnFlags.timeout, "timeout", 0, "task timeout (default 5m, e.g. 10m, 2m30s)")
	rootCmd.AddCommand(spawnCmd)
}
