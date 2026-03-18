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
	"github.com/spf13/pflag"
)

type runnerFunc func(ctx context.Context, cfg httpclient.Config) *stats.HistogramRecorder

func newRunCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "run <url>",
		Short: "Run a load test against an HTTP endpoint",
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

			err = config.Validate()
			if err != nil {
				return err
			}

			u := config.ParsedTarget

			httpCfg := config.ToHTTPConfig()
			httpCfg.Stderr = cmd.ErrOrStderr()

			if config.Duration > 0 {
				_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Running for %v against %s with concurrency %d\n", config.Duration, u, config.Concurrency)
				return runCommand(httpclient.RunForDuration, httpCfg, cmd.OutOrStdout())
			}

			_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Making %d requests to %v with concurrency %d\n", config.Requests, u, config.Concurrency)

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
	cmd.Flags().StringP("config", "f", "", "Path to configuration file (JSON/YAML)")
	cmd.Flags().BoolP("verbose", "v", false, "Enable verbose output")

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
