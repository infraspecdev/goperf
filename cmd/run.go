package cmd

import (
	"fmt"
	"io"
	"net/http"
	"net/url"

	httpsclient "github.com/infraspecdev/goperf/internal/httpclient"
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
	RunE: func(cmd *cobra.Command, args []string) error {
		requests, err := cmd.Flags().GetInt("requests")
		if err != nil {
			return fmt.Errorf("error getting requests flag: %w", err)
		}

		if err := validateRequests(requests); err != nil {
			return fmt.Errorf("invalid requests value: %w", err)
		}

		u, err := validateTarget(args[0])
		if err != nil {
			return fmt.Errorf("invalid URL: %w", err)
		}

		fmt.Println("Parsed URL:", u)
		fmt.Printf("Making %d requests to %s\n", requests, u)
		if requests > 1 {
           return runCommandMultiple(args[0], requests, cmd.OutOrStdout())
       }
		return runCommand(args[0], cmd.OutOrStdout())
	},
}

func runCommand(url string, out io.Writer) error {
	statusCode, duration, err := httpsclient.MakeRequest(url)
	if err != nil {
		return err
	}

	statusText := http.StatusText(statusCode)

	fmt.Fprintf(out, "Status: %d %s\n", statusCode, statusText)
	fmt.Fprintf(out, "Time: %dms\n", duration.Milliseconds())

	return nil
}

func runCommandMultiple(url string, n int, out io.Writer) error {
   results := httpsclient.RunMultiple(nil, url, n)
   for _, res := range results {
       if res.Error != nil {
           return res.Error
       }
       statusText := http.StatusText(res.StatusCode)
       fmt.Fprintf(out, "Status: %d %s\n", res.StatusCode, statusText)
       fmt.Fprintf(out, "Time: %dms\n", res.Duration.Milliseconds())
   }
   return nil
}


func init() {
	runCmd.Flags().IntP("requests", "n", 1, "Number of requests to execute")
	rootCmd.AddCommand(runCmd)
}
