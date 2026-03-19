package httpclient

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/infraspecdev/goperf/internal/stats"
)

func NewHTTPClient(concurrency int) *http.Client {
	return &http.Client{
		Transport: &http.Transport{
			MaxIdleConnsPerHost: concurrency,
			DisableCompression:  true,
		},
	}
}

type Config struct {
	Target      string
	Requests    int
	Concurrency int
	Timeout     time.Duration
	Duration    time.Duration
	Method      string
	Body        string
	Headers     []string
	Verbose     bool
	Stderr      io.Writer
}

type HTTPDoer interface {
	Do(req *http.Request) (*http.Response, error)
}

type syncWriter struct {
	mu sync.Mutex
	w  io.Writer
}

func (sw *syncWriter) Write(p []byte) (int, error) {
	sw.mu.Lock()
	defer sw.mu.Unlock()
	return sw.w.Write(p)
}

func MakeRequest(ctx context.Context, client HTTPDoer, cfg Config) (statusCode int, duration time.Duration, err error) {
	reqCtx, cancel := context.WithTimeout(ctx, cfg.Timeout)
	defer cancel()

	var reqBody io.Reader
	if cfg.Body != "" {
		reqBody = strings.NewReader(cfg.Body)
	}

	req, err := http.NewRequestWithContext(reqCtx, cfg.Method, cfg.Target, reqBody)
	if err != nil {
		return 0, 0, err
	}

	for _, h := range cfg.Headers {
		parts := strings.SplitN(h, ":", 2)
		if len(parts) == 2 {
			req.Header.Add(strings.TrimSpace(parts[0]), strings.TrimSpace(parts[1]))
		}
	}

	if cfg.Body != "" && req.Header.Get("Content-Type") == "" {
		req.Header.Set("Content-Type", "application/json")
	}

	start := time.Now()
	resp, err := client.Do(req)
	duration = time.Since(start)

	if err != nil {

		var netErr *net.OpError

		if errors.As(err, &netErr) {

			if netErr.Op == "dial" {
				if strings.Contains(netErr.Err.Error(), "refused") {
					return 0, duration, fmt.Errorf("connection refused: %w", err)
				}
				if strings.Contains(netErr.Err.Error(), "no such host") {
					return 0, duration, fmt.Errorf("no such host: %w", err)
				}
			}
		}

		return 0, duration, err
	}

	defer func() { _ = resp.Body.Close() }()
	_, _ = io.Copy(io.Discard, resp.Body)

	return resp.StatusCode, duration, nil
}

func recordResult(ctx context.Context, recorder *stats.HistogramRecorder, verboseWriter io.Writer, statusCode int, latency time.Duration, err error) {
	if err != nil && ctx.Err() != nil && errors.Is(err, ctx.Err()) {
		return
	}
	if verboseWriter != nil {
		if err != nil {
			_, _ = fmt.Fprintf(verboseWriter, "Request error: %v\n", err)
		} else {
			_, _ = fmt.Fprintf(verboseWriter, "Request [%d]: %8.2fms\n", statusCode, float64(latency.Microseconds())/1000.0)
		}
	}
	if err != nil {
		recorder.RecordFailure()
	} else if statusCode >= 200 && statusCode < 300 {
		recorder.Record(latency)
	} else {
		recorder.RecordFailure()
	}
}

func Run(ctx context.Context, cfg Config) *stats.HistogramRecorder {
	client := NewHTTPClient(cfg.Concurrency)
	recorder := stats.NewHistogramRecorder(cfg.Timeout)

	var verboseWriter io.Writer
	if cfg.Verbose && cfg.Stderr != nil {
		verboseWriter = &syncWriter{w: cfg.Stderr}
	}

	var wg sync.WaitGroup
	wg.Add(cfg.Concurrency)

	var reqCtx context.Context
	var cancel context.CancelFunc

	if cfg.Duration > 0 {
		reqCtx, cancel = context.WithTimeout(ctx, cfg.Duration)
		defer cancel()
	} else {
		reqCtx = ctx
	}

	var count int64

	for w := 0; w < cfg.Concurrency; w++ {
		go func() {
			defer wg.Done()
			for {
				if reqCtx.Err() != nil {
					return
				}
				if cfg.Duration == 0 {
					if atomic.AddInt64(&count, 1) > int64(cfg.Requests) {
						return
					}
				}
				statusCode, d, err := MakeRequest(reqCtx, client, cfg)
				recordResult(reqCtx, recorder, verboseWriter, statusCode, d, err)
			}
		}()
	}

	wg.Wait()
	return recorder
}

func RunMultipleConcurrent(ctx context.Context, cfg Config) *stats.HistogramRecorder {
	cfg.Duration = 0
	return Run(ctx, cfg)
}

func RunForDuration(ctx context.Context, cfg Config) *stats.HistogramRecorder {
	return Run(ctx, cfg)
}
