package cmd

import (
	"context"
	"fmt"
	"io"
	"net/url"
	"os"
	"os/signal"
	"sort"
	"strings"
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

func validateDuration(d time.Duration) error {
	if d < 0 {
		return fmt.Errorf("duration must not be negative, got %v", d)
	}
	return nil
}

var validMethods = map[string]bool{
	"GET":     true,
	"POST":    true,
	"PUT":     true,
	"DELETE":  true,
	"PATCH":   true,
	"OPTIONS": true,
	"HEAD":    true,
}

func validateMethod(method string) error {
	if !validMethods[method] {
		methods := make([]string, 0, len(validMethods))
		for m := range validMethods {
			methods = append(methods, m)
		}
		sort.Strings(methods)
		return fmt.Errorf("invalid HTTP method %q, supported methods: %s", method, strings.Join(methods, ", "))
	}
	return nil
}

func validateHeaders(headers []string) error {
	for _, h := range headers {
		parts := strings.SplitN(h, ":", 2)
		if len(parts) != 2 || strings.TrimSpace(parts[0]) == "" || strings.Contains(strings.TrimSpace(parts[0]), " ") {
			return fmt.Errorf("invalid header format %q, expected 'Key: Value' without spaces in the key", h)
		}
	}
	return nil
}

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

		if err := validateConcurrency(concurrency); err != nil {
			return err
		}
		if err := validateTimeout(timeout); err != nil {
			return err
		}
		if err := validateDuration(duration); err != nil {
			return err
		}
		if err := validateMethod(method); err != nil {
			return err
		}
		if err := validateHeaders(headers); err != nil {
			return err
		}

		u, err := validateTarget(args[0])
		if err != nil {
			return err
		}

		if f.Changed("requests") && f.Changed("duration") {
			return fmt.Errorf("cannot use both --requests (-n) and --duration (-d) at the same time")
		}

		if duration > 0 {
			fmt.Fprintf(cmd.OutOrStdout(), "Running for %v against %s with concurrency %d\n", duration, u, concurrency)
			return runCommandDuration(args[0], concurrency, timeout, duration, method, body, headers, cmd.OutOrStdout())
		}

		if err := validateRequests(requests); err != nil {
			return err
		}

		fmt.Fprintf(cmd.OutOrStdout(), "Making %d requests to %v with concurrency %d\n", requests, u, concurrency)

		return runCommandMultipleConcurrent(args[0], requests, concurrency, timeout, method, body, headers, cmd.OutOrStdout())
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
