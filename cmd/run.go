package cmd

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"

	"github.com/infraspecdev/goperf/internal/httpclient"
	"github.com/infraspecdev/goperf/internal/stats"
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

func validateConcurrency(c int) error {
	if c <= 0 {
		return fmt.Errorf("concurrency must be positive, got %d", c)
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
		concurrency, err := cmd.Flags().GetInt("concurrency")
		if err != nil {
			return fmt.Errorf("error getting concurrency flag: %w", err)
		}
		if err := validateConcurrency(concurrency); err != nil {
			return fmt.Errorf("invalid concurrency value: %w", err)
		}
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

		return runCommandMultipleConcurrent(args[0], requests, concurrency, timeout, cmd.OutOrStdout())
	},
}

func runCommandMultipleConcurrent(target string, n int, concurrency int, timeout time.Duration, out io.Writer) error {
	results := httpclient.RunMultipleConcurrent(context.Background(), target, n, concurrency, timeout)

	durations := make([]time.Duration, 0, len(results))
	for _, res := range results {
		printResult(out, res)
		if res.Error == nil {
			durations = append(durations, res.Duration)
		}
	}

	printStatistics(out, durations)
	return nil
}

func printResult(out io.Writer, res httpclient.RequestResult) {
	if res.Error != nil {
		fmt.Fprintf(out, "Status: Error\n")
		fmt.Fprintf(out, "Time: %dms\n", res.Duration.Milliseconds())
		fmt.Fprintf(out, "Error: %v\n", res.Error)
		return
	}

	fmt.Fprintf(out, "Status: %d %s\n", res.StatusCode, http.StatusText(res.StatusCode))
	fmt.Fprintf(out, "Time: %dms\n", res.Duration.Milliseconds())
}

func printStatistics(out io.Writer, durations []time.Duration) {
	if len(durations) == 0 {
		return
	}

	min := stats.MinResponseTime(durations)
	max := stats.MaxResponseTime(durations)
	avg := stats.AverageResponseTime(durations)

	fmt.Fprintf(out, "\nStatistics:\n")
	fmt.Fprintf(out, "  Min: %dms\n", min.Milliseconds())
	fmt.Fprintf(out, "  Max: %dms\n", max.Milliseconds())
	fmt.Fprintf(out, "  Avg: %dms\n", avg.Milliseconds())
}

func init() {
	runCmd.Flags().IntP("requests", "n", 1, "Number of requests to execute")
	runCmd.Flags().DurationP("timeout", "t", 10*time.Second, "Timeout per request")
	runCmd.Flags().IntP("concurrency", "c", 1, "Number of concurrent workers")
	rootCmd.AddCommand(runCmd)
}
