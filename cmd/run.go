package cmd

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"os/signal"
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
		f := cmd.Flags()

		concurrency, _ := f.GetInt("concurrency")
		requests, _ := f.GetInt("requests")
		timeout, _ := f.GetDuration("timeout")

		if err := validateConcurrency(concurrency); err != nil {
			return err
		}
		if err := validateRequests(requests); err != nil {
			return err
		}
		if err := validateTimeout(timeout); err != nil {
			return err
		}

		u, err := validateTarget(args[0])
		if err != nil {
			return err
		}

		fmt.Println("Parsed URL:", u)
		fmt.Printf("Making %d requests to %s with concurrency %d\n", requests, u, concurrency)

		return runCommandMultipleConcurrent(args[0], requests, concurrency, timeout, cmd.OutOrStdout())
	},
}

func runCommandMultipleConcurrent(target string, n int, concurrency int, timeout time.Duration, out io.Writer) error {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()

	results := httpclient.RunMultipleConcurrent(ctx, target, n, concurrency, timeout)

	durations := make([]time.Duration, 0, len(results))
	for _, res := range results {
		if err := printResult(out, res); err != nil {
			return err
		}
		if res.Error == nil {
			durations = append(durations, res.Duration)
		}
	}

	if err := printStatistics(out, durations); err != nil {
		return err
	}

	return nil
}

func printResult(out io.Writer, res httpclient.RequestResult) error {
	if res.Error != nil {
		if _, err := fmt.Fprintf(out, "Status: Error\n"); err != nil {
			return err
		}
		if _, err := fmt.Fprintf(out, "Time: %dms\n", res.Duration.Milliseconds()); err != nil {
			return err
		}
		if _, err := fmt.Fprintf(out, "Error: %v\n", res.Error); err != nil {
			return err
		}
		return nil
	}

	if _, err := fmt.Fprintf(out, "Status: %d %s\n", res.StatusCode, http.StatusText(res.StatusCode)); err != nil {
		return err
	}
	if _, err := fmt.Fprintf(out, "Time: %dms\n", res.Duration.Milliseconds()); err != nil {
		return err
	}

	return nil
}

func printStatistics(out io.Writer, durations []time.Duration) error {
	if len(durations) == 0 {
		return nil
	}

	min := stats.MinResponseTime(durations)
	max := stats.MaxResponseTime(durations)
	avg := stats.AverageResponseTime(durations)

	if _, err := fmt.Fprintf(out, "\nStatistics:\n"); err != nil {
		return err
	}
	if _, err := fmt.Fprintf(out, "Min: %dms\n", min.Milliseconds()); err != nil {
		return err
	}
	if _, err := fmt.Fprintf(out, "Max: %dms\n", max.Milliseconds()); err != nil {
		return err
	}
	if _, err := fmt.Fprintf(out, "Avg: %dms\n", avg.Milliseconds()); err != nil {
		return err
	}

	return nil
}

func init() {
	runCmd.Flags().IntP("requests", "n", 1, "Number of requests to execute")
	runCmd.Flags().DurationP("timeout", "t", 10*time.Second, "Timeout per request")
	runCmd.Flags().IntP("concurrency", "c", 1, "Number of concurrent workers")
	rootCmd.AddCommand(runCmd)
}
