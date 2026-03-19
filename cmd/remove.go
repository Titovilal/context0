package cmd

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"
)

var removeCmd = &cobra.Command{
	Use:   "remove <agent_id>",
	Short: "Remove an agent from the registry",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		if err := orch.Remove(context.Background(), args[0]); err != nil {
			return err
		}
		fmt.Printf("agent %s removed\n", args[0])
		return nil
	},
}

func init() {
	rootCmd.AddCommand(removeCmd)
}
