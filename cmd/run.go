package cmd

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"

	"github.com/infraspecdev/goperf/internal/httpclient"
<<<<<<< HEAD
=======
	"github.com/infraspecdev/goperf/internal/stats"
>>>>>>> feature/response-time-clean
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

func validateTimeout(d time.Duration) error {
	if d <= 0 {
		return fmt.Errorf("timeout must be positive, got %v", d)
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

		timeout, err := cmd.Flags().GetDuration("timeout")
		if err != nil {
			return fmt.Errorf("error getting timeout flag: %w", err)
		}

		if err := validateTimeout(timeout); err != nil {
			return fmt.Errorf("invalid timeout value: %w", err)
		}

		u, err := validateTarget(args[0])
		if err != nil {
			return fmt.Errorf("invalid URL: %w", err)
		}

		fmt.Println("Parsed URL:", u)
		fmt.Printf("Making %d requests to %s\n", requests, u)

		if requests > 1 {
<<<<<<< HEAD
			return runCommandMultiple(args[0], requests, timeout, cmd.OutOrStdout())
		}
		return runCommand(args[0], timeout, cmd.OutOrStdout())
	},
}

func runCommand(target string, timeout time.Duration, out io.Writer) error {
	statusCode, duration, err := httpclient.MakeRequest(context.Background(), target, timeout)
=======
			return runCommandMultiple(args[0], requests, cmd.OutOrStdout())
		}
		return runCommand(args[0], cmd.OutOrStdout())
	},
}

func runCommand(url string, out io.Writer) error {
	statusCode, duration, err := httpclient.MakeRequest(url)
>>>>>>> feature/response-time-clean
	if err != nil {
		fmt.Fprintf(out, "Status: N/A\n")
		fmt.Fprintf(out, "Time: N/A\n")
		return nil
	}

	statusText := http.StatusText(statusCode)

	_, err = fmt.Fprintf(out, "Status: %d %s\n", statusCode, statusText)
	if err != nil {
		return fmt.Errorf("error writing status: %w", err)
	}
	_, err = fmt.Fprintf(out, "Time: %dms\n", duration.Milliseconds())
	if err != nil {
		return fmt.Errorf("error writing duration: %w", err)
	}

	return nil
}

<<<<<<< HEAD
func runCommandMultiple(target string, n int, timeout time.Duration, out io.Writer) error {
	results := httpclient.RunMultiple(context.Background(), target, n, timeout)

	for _, res := range results {
		if res.Error != nil {
			return res.Error
		}
		statusText := http.StatusText(res.StatusCode)
		_, err := fmt.Fprintf(out, "Status: %d %s\n", res.StatusCode, statusText)
		if err != nil {
			return fmt.Errorf("error writing status: %w", err)
		}
		_, err = fmt.Fprintf(out, "Time: %dms\n", res.Duration.Milliseconds())
		if err != nil {
			return fmt.Errorf("error writing duration: %w", err)
		}
	}
=======
func runCommandMultiple(url string, n int, out io.Writer) error {
	results := httpclient.RunMultiple(nil, url, n)

	var successfulDurations []time.Duration

	for _, res := range results {
		if res.Error != nil {
			continue
		}

		successfulDurations = append(successfulDurations, res.Duration)

		statusText := http.StatusText(res.StatusCode)
		fmt.Fprintf(out, "Status: %d %s\n", res.StatusCode, statusText)
		fmt.Fprintf(out, "Time: %dms\n", res.Duration.Milliseconds())
	}

	// If zero successful
	if len(successfulDurations) == 0 {
		fmt.Fprintf(out, "\nMin: N/A\n")
		fmt.Fprintf(out, "Max: N/A\n")
		fmt.Fprintf(out, "Avg: N/A\n")
		return nil
	}

	min := stats.MinResponseTime(successfulDurations)
	max := stats.MaxResponseTime(successfulDurations)
	avg := stats.AverageResponseTime(successfulDurations)

	fmt.Fprintf(out, "\nMin: %dms\n", min.Milliseconds())
	fmt.Fprintf(out, "Max: %dms\n", max.Milliseconds())
	fmt.Fprintf(out, "Avg: %dms\n", avg.Milliseconds())

>>>>>>> feature/response-time-clean
	return nil
}

func init() {
	runCmd.Flags().IntP("requests", "n", 1, "Number of requests to execute")
	runCmd.Flags().DurationP("timeout", "t", 10*time.Second, "Timeout per request")
	rootCmd.AddCommand(runCmd)
}
