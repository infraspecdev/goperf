package cmd

import (
	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "goperf",
	Short: "GoPerf is a HTTP load testing tool",
	RunE: func(cmd *cobra.Command, args []string) error {
		cmd.Println("Welcome to GoPerf")
		return nil
	},
}

func Execute() error {
	return rootCmd.Execute()
}

func init() {
	rootCmd.AddCommand(runCmd)
}
