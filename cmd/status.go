package cmd

import (
	"context"
	"fmt"

	"github.com/Titovilal/middleman/agent"
	"github.com/spf13/cobra"
)

var statusFlags struct {
	all bool
}

var statusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show active agents",
	RunE: func(cmd *cobra.Command, args []string) error {
		var statuses []agent.Status
		if !statusFlags.all {
			statuses = []agent.Status{agent.StatusActive, agent.StatusIdle, agent.StatusWorking}
		}

		agents, err := orch.ListAgents(context.Background(), statuses...)
		if err != nil {
			return err
		}

		if len(agents) == 0 {
			fmt.Println("no agents")
			return nil
		}

		fmt.Printf("%-20s %-10s %-10s %-5s %-5s  %s\n",
			"ID", "CONNECTOR", "STATUS", "TURNS", "CPS", "LAST ACTIVE")
		fmt.Printf("%-20s %-10s %-10s %-5s %-5s  %s\n",
			"--------------------", "----------", "----------", "-----", "---", "-----------")
		for _, a := range agents {
			fmt.Printf("%-20s %-10s %-10s %-5d %-5d  %s\n",
				a.ID,
				a.ConnectorName,
				string(a.Status),
				len(a.TaskLog),
				len(a.Checkpoints),
				a.LastActiveAt.Format("2006-01-02 15:04"),
			)
		}
		return nil
	},
}

func init() {
	statusCmd.Flags().BoolVar(&statusFlags.all, "all", false, "include all agents")
	rootCmd.AddCommand(statusCmd)
}
