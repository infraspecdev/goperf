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
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

func newRunCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "run <url>",
		Short: "Run a load test against an HTTP endpoint",
		Long: `Run a load test against an HTTP endpoint and report latency statistics.

Latency Percentiles:
  Fastest:  The minimum latency recorded.
  Slowest:  The maximum latency recorded.
  Average:  The arithmetic mean of all recorded latencies.
  p50:      50th percentile (median) - 50% of requests were faster than this value.
  p90:      90th percentile - 90% of requests were faster than this value.
  p99:      99th percentile - 99% of requests were faster than this value.`,
		Args: func(cmd *cobra.Command, args []string) error {
			configPath, _ := cmd.Flags().GetString("config")
			if len(args) == 0 && configPath == "" {
				return fmt.Errorf("missing required argument: URL (or provide via --config)")
			}
			return nil
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			f := cmd.Flags()

			configPath, _ := f.GetString("config")
			var fileCfg *fileConfig
			var err error

			if configPath != "" {
				fileCfg, err = loadConfig(configPath)
				if err != nil {
					return err
				}
			}

			concurrency, _ := f.GetInt("concurrency")
			requests, _ := f.GetInt("requests")
			timeout, _ := f.GetDuration("timeout")
			duration, _ := f.GetDuration("duration")
			method, _ := f.GetString("method")
			body, _ := f.GetString("body")
			headers, _ := f.GetStringArray("header")
			verbose, _ := f.GetBool("verbose")
			outputFormat, _ := f.GetString("output")

			if outputFormat != "text" && outputFormat != "json" {
				return fmt.Errorf("invalid output format: %q. Must be 'text' or 'json'", outputFormat)
			}

			target := ""
			if len(args) > 0 {
				target = args[0]
			}

			cliConfig := RunConfig{
				Target:      target,
				Requests:    requests,
				Concurrency: concurrency,
				Timeout:     timeout,
				Duration:    duration,
				Method:      strings.ToUpper(method),
				Body:        body,
				Headers:     headers,
				Verbose:     verbose,
			}

			changed := make(map[string]bool)
			f.Visit(func(flag *pflag.Flag) {
				changed[flag.Name] = true
			})
			if len(args) > 0 {
				changed["target"] = true
			}

			config, err := mergeConfig(fileCfg, cliConfig, changed)
			if err != nil {
				return err
			}

			if config.Duration > 0 {
				config.Requests = 0
			}

			err = config.Validate()
			if err != nil {
				return err
			}

			u := config.ParsedTarget

			httpCfg := config.ToHTTPConfig()
			if outputFormat != "json" {
				httpCfg.Stderr = cmd.ErrOrStderr()
			}

			if config.Duration > 0 {
				if outputFormat != "json" {
					_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Running for %v against %s with concurrency %d\n", config.Duration, u, config.Concurrency)
				}
			} else {
				if outputFormat != "json" {
					requestWord := "requests"
					if config.Requests == 1 {
						requestWord = "request"
					}
					_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Making %d %s to %v with concurrency %d\n", config.Requests, requestWord, u, config.Concurrency)
				}
			}

			return runCommand(httpCfg, outputFormat, cmd.OutOrStdout())
		},
	}

	cmd.Flags().IntP("requests", "n", 1, "Number of requests to execute")
	cmd.Flags().DurationP("timeout", "t", 10*time.Second, "Timeout per request")
	cmd.Flags().IntP("concurrency", "c", 1, "Number of concurrent workers")
	cmd.Flags().DurationP("duration", "d", 0, "Duration to run the test. Overrides -n when set (e.g., 10s, 1m)")
	cmd.Flags().StringP("method", "m", "GET", "HTTP method to use")
	cmd.Flags().StringP("body", "b", "", "Request body content")
	cmd.Flags().StringArrayP("header", "H", []string{}, "HTTP header in 'Key: Value' format (can be repeated)")
	cmd.Flags().StringP("config", "f", "", "Path to configuration file (JSON/YAML)")
	cmd.Flags().BoolP("verbose", "v", false, "Enable verbose output")
	cmd.Flags().StringP("output", "o", "text", "Output format (text or json)")

	return cmd
}

func runCommand(cfg httpclient.Config, outputFormat string, out io.Writer) error {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()

	start := time.Now()
	recorder := httpclient.Run(ctx, cfg)
	elapsed := time.Since(start)

	res := newResult(recorder, cfg.Target, elapsed)

	if outputFormat == "json" {
		return res.WriteJSON(out)
	}
	return res.WriteText(out)
}
