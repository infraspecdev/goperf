package cmd

import (
	"context"
	"fmt"
	"io"
	"os"
	"os/signal"
	"strings"
	"time"

	"github.com/infraspecdev/goperf/internal/httpclient"
	"github.com/infraspecdev/goperf/internal/stats"
	"github.com/spf13/cobra"
)

var runCmd = &cobra.Command{
	Use:   "run <url>",
	Short: "Run a load test against an HTTP endpoint",
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
		duration, _ := f.GetDuration("duration")
		method, _ := f.GetString("method")
		body, _ := f.GetString("body")
		headers, _ := f.GetStringArray("header")
		method = strings.ToUpper(method)

		config := RunConfig{
			Target:      args[0],
			Requests:    requests,
			Concurrency: concurrency,
			Timeout:     timeout,
			Duration:    duration,
			Method:      method,
			Body:        body,
			Headers:     headers,
		}

		err := config.Validate()
		if err != nil {
			return err
		}

		u := config.ParsedTarget

		if config.Duration > 0 {
			fmt.Fprintf(cmd.OutOrStdout(), "Running for %v against %s with concurrency %d\n", config.Duration, u, config.Concurrency)
			return runCommandDuration(args[0], config.Concurrency, config.Timeout, config.Duration, config.Method, config.Body, config.Headers, cmd.OutOrStdout())
		}

		fmt.Fprintf(cmd.OutOrStdout(), "Making %d requests to %v with concurrency %d\n", config.Requests, u, config.Concurrency)

		return runCommandMultipleConcurrent(args[0], config.Requests, config.Concurrency, config.Timeout, config.Method, config.Body, config.Headers, cmd.OutOrStdout())
	},
}

func runCommandDuration(target string, concurrency int, timeout time.Duration, duration time.Duration, method string, body string, headers []string, out io.Writer) error {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()

	cfg := httpclient.Config{
		Target:      target,
		Concurrency: concurrency,
		Timeout:     timeout,
		Duration:    duration,
		Method:      method,
		Body:        body,
		Headers:     headers,
	}

	start := time.Now()
	recorder := httpclient.RunForDuration(ctx, cfg)
	elapsed := time.Since(start)

	return printHistogramStatistics(out, recorder, target, elapsed)
}

func runCommandMultipleConcurrent(target string, n int, concurrency int, timeout time.Duration, method string, body string, headers []string, out io.Writer) error {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()

	cfg := httpclient.Config{
		Target:      target,
		Requests:    n,
		Concurrency: concurrency,
		Timeout:     timeout,
		Method:      method,
		Body:        body,
		Headers:     headers,
	}

	start := time.Now()
	recorder := httpclient.RunMultipleConcurrent(ctx, cfg)
	elapsed := time.Since(start)

	if err := printHistogramStatistics(out, recorder, target, elapsed); err != nil {
		return err
	}

	return nil
}

func printHistogramStatistics(out io.Writer, recorder *stats.HistogramRecorder, target string, elapsed time.Duration) error {
	totalReqs := recorder.TotalRequests()
	successReqs := recorder.Count()
	failedReqs := recorder.FailedCount()

	throughput := 0.0
	if elapsed.Seconds() > 0 {
		throughput = float64(totalReqs) / elapsed.Seconds()
	}

	if _, err := fmt.Fprintf(out, "\nTarget:     %s\n", target); err != nil {
		return err
	}
	if _, err := fmt.Fprintf(out, "Duration:   %.1fs\n", elapsed.Seconds()); err != nil {
		return err
	}
	if _, err := fmt.Fprintf(out, "Requests:   %d total (%d succeeded, %d failed)\n\n", totalReqs, successReqs, failedReqs); err != nil {
		return err
	}
	if successReqs == 0 {
		if _, err := fmt.Fprintf(out, "Throughput: %.1f requests/sec\n", throughput); err != nil {
			return err
		}
		return nil
	}

	if _, err := fmt.Fprintf(out, "Latency:\n"); err != nil {
		return err
	}
	if _, err := fmt.Fprintf(out, "  Fastest:  %dms\n", recorder.Min().Milliseconds()); err != nil {
		return err
	}
	if _, err := fmt.Fprintf(out, "  Slowest:  %dms\n", recorder.Max().Milliseconds()); err != nil {
		return err
	}
	if _, err := fmt.Fprintf(out, "  Average:  %dms\n", recorder.Avg().Milliseconds()); err != nil {
		return err
	}
	if _, err := fmt.Fprintf(out, "  p50:      %dms\n", recorder.Percentile(50).Milliseconds()); err != nil {
		return err
	}
	if _, err := fmt.Fprintf(out, "  p90:      %dms\n", recorder.Percentile(90).Milliseconds()); err != nil {
		return err
	}
	if _, err := fmt.Fprintf(out, "  p99:      %dms\n\n", recorder.Percentile(99).Milliseconds()); err != nil {
		return err
	}

	if _, err := fmt.Fprintf(out, "Throughput: %.1f requests/sec\n", throughput); err != nil {
		return err
	}

	return nil
}

func init() {
	runCmd.Flags().IntP("requests", "n", 1, "Number of requests to execute")
	runCmd.Flags().DurationP("timeout", "t", 10*time.Second, "Timeout per request")
	runCmd.Flags().IntP("concurrency", "c", 1, "Number of concurrent workers")
	runCmd.Flags().DurationP("duration", "d", 0, "Duration to run the test (e.g., 10s, 1m)")
	runCmd.Flags().StringP("method", "m", "GET", "HTTP method to use")
	runCmd.Flags().StringP("body", "b", "", "Request body content")
	runCmd.Flags().StringArrayP("header", "H", []string{}, "HTTP header in 'Key: Value' format (can be repeated)")
	rootCmd.AddCommand(runCmd)
}
