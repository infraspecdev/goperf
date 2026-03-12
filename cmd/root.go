package cmd

import (
	"github.com/spf13/cobra"
)

func NewRootCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "goperf",
		Short: "GoPerf is a HTTP load testing tool",
		RunE: func(cmd *cobra.Command, args []string) error {
			cmd.Println("Welcome to GoPerf")
			return nil
		},
	}

	cmd.AddCommand(newRunCmd())
	return cmd
}
