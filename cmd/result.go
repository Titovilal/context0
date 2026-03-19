package cmd

import (
	"context"
	"fmt"

	"github.com/Titovilal/middleman/agent"
	"github.com/spf13/cobra"
)

var resultFlags struct {
	taskID string
}

var resultCmd = &cobra.Command{
	Use:   "result <agent_id>",
	Short: "Check the result of an agent's task",
	Long:  `Shows the result of the latest task (or a specific --task-id). If the task is still running, it says so.`,
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		agentID := args[0]

		a, err := orch.Inspect(context.Background(), agentID)
		if err != nil {
			return err
		}

		var task = a.LatestTask()
		if resultFlags.taskID != "" {
			task = a.TaskByID(resultFlags.taskID)
		}
		if task == nil {
			return fmt.Errorf("no task found")
		}

		fmt.Printf("agent:   %s\n", agentID)
		fmt.Printf("task_id: %s\n", task.TaskID)
		fmt.Printf("status:  %s\n", task.Status)
		fmt.Printf("prompt:  %s\n", task.Prompt)

		switch task.Status {
		case agent.TaskPending:
			fmt.Println("\nTask is still running. Check again later.")
		case agent.TaskCompleted:
			fmt.Printf("\n%s\n", task.Response)
		case agent.TaskFailed:
			fmt.Printf("\n[error] %s\n", task.ErrorDetail)
			if task.Response != "" {
				fmt.Printf("\n%s\n", task.Response)
			}
		default:
			// Legacy tasks without status field.
			if task.IsError {
				fmt.Printf("\n[error] %s\n", task.ErrorDetail)
			} else {
				fmt.Printf("\n%s\n", task.Response)
			}
		}
		return nil
	},
}

func init() {
	resultCmd.Flags().StringVar(&resultFlags.taskID, "task-id", "", "specific task ID (default: latest task)")
	rootCmd.AddCommand(resultCmd)
}
