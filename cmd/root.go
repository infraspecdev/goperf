package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "goperf",
	Short: "GoPerf is a HTTP load testing tool",
	Run: func(cmd *cobra.Command, args []string) {
		cmd.Println("Welcome to GoPerf")
		//fmt.Println(args[0])
	},
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
