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
		_, err := validateTarget(args[0])
		if err != nil {
			return err
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
	fmt.Fprintf(out, "Time: %d ms\n", duration.Milliseconds())

	return nil
}

func init() {
	rootCmd.AddCommand(runCmd)
}
