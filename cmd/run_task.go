package cmd

import (
	"context"
	"time"

	"github.com/spf13/cobra"
)

var runTaskFlags struct {
	timeout time.Duration
}

// _run-task is an internal command launched by 'spawn' as a background process.
// It is not shown in help.
var runTaskCmd = &cobra.Command{
	Use:    "_run-task <agent_id> <task_id>",
	Short:  "Internal: execute a pending task",
	Hidden: true,
	Args:   cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		agentID := args[0]
		taskID := args[1]
		return orch.RunTask(context.Background(), agentID, taskID, runTaskFlags.timeout)
	},
}

func init() {
	runTaskCmd.Flags().DurationVar(&runTaskFlags.timeout, "timeout", 0, "task timeout (default 5m)")
	rootCmd.AddCommand(runTaskCmd)
}
