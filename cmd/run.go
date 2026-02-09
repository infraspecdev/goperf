/*
Copyright © 2026 NAME HERE <EMAIL ADDRESS>
*/
package cmd

import (
	"fmt"
	"net/url"

	"github.com/spf13/cobra"
)


func validateTarget(input string) (*url.URL, error) {
	u, err := url.ParseRequestURI(input)
	if err != nil {
		return nil, fmt.Errorf("parse error: %w", err)
	}
	if u.Scheme == "" || u.Host == "" {
		return nil, fmt.Errorf("invalid URL: missing scheme or host")
	}
	return u, nil
}

var runCmd = &cobra.Command{
	Use:   "run <url>",
	Short: "Command to give input URL",
	Run: func(cmd *cobra.Command, args []string) {
		u, err := validateTarget(args[0])
		if err != nil {
			fmt.Println("Invalid URL:", err)
			return
		}
		fmt.Println("Parsed URL:", u)
	},
}

func init() {
	rootCmd.AddCommand(runCmd)
}
