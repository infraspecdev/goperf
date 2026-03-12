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

type runnerFunc func(ctx context.Context, cfg httpclient.Config) *stats.HistogramRecorder

func newRunCmd() *cobra.Command {
	cmd := &cobra.Command{
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

			httpCfg := config.ToHTTPConfig()

			if config.Duration > 0 {
				fmt.Fprintf(cmd.OutOrStdout(), "Running for %v against %s with concurrency %d\n", config.Duration, u, config.Concurrency)
				return runCommand(httpclient.RunForDuration, httpCfg, cmd.OutOrStdout())
			}

			fmt.Fprintf(cmd.OutOrStdout(), "Making %d requests to %v with concurrency %d\n", config.Requests, u, config.Concurrency)

			return runCommand(httpclient.RunMultipleConcurrent, httpCfg, cmd.OutOrStdout())
		},
	}

	cmd.Flags().IntP("requests", "n", 1, "Number of requests to execute")
	cmd.Flags().DurationP("timeout", "t", 10*time.Second, "Timeout per request")
	cmd.Flags().IntP("concurrency", "c", 1, "Number of concurrent workers")
	cmd.Flags().DurationP("duration", "d", 0, "Duration to run the test (e.g., 10s, 1m)")
	cmd.Flags().StringP("method", "m", "GET", "HTTP method to use")
	cmd.Flags().StringP("body", "b", "", "Request body content")
	cmd.Flags().StringArrayP("header", "H", []string{}, "HTTP header in 'Key: Value' format (can be repeated)")

	return cmd
}

func runCommand(runner runnerFunc, cfg httpclient.Config, out io.Writer) error {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()

	start := time.Now()
	recorder := runner(ctx, cfg)
	elapsed := time.Since(start)

	return newResult(recorder, cfg.Target, elapsed).WriteText(out)
}
