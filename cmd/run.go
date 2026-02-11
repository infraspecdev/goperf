package cmd

import (
	"fmt"
	"net/url"
	"os"

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

func validateRequests(n int) error {
	if n <= 0 {
		return fmt.Errorf("number of requests must be positive, got %d", n)
	}
	return nil
}

var runCmd = &cobra.Command{
	Use:   "run <url>",
	Short: "Command to give input URL",
	Args: func(cmd *cobra.Command, args []string) error {
		if len(args) == 0 {
			return fmt.Errorf("missing required argument: URL")
		}
		return nil
	},

	Run: func(cmd *cobra.Command, args []string) {
		requests, err := cmd.Flags().GetInt("requests")
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error getting requests flag: %v\n", err)
			os.Exit(1)
		}

		if err := validateRequests(requests); err != nil {
			fmt.Fprintf(os.Stderr, "Invalid requests value: %v\n", err)
			os.Exit(1)
		}
		u, err := validateTarget(args[0])
		if err != nil {
			fmt.Fprintln(os.Stderr, "Invalid URL:", err)
			os.Exit(1)
		}
		fmt.Println("Parsed URL:", u)
		fmt.Printf("Making %d requests to %s\n", requests, u)
	},
}

func init() {
	runCmd.Flags().IntP("requests", "n", 1, "Number of requests to execute")
	rootCmd.AddCommand(runCmd)
}
