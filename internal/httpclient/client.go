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
	"time"

	"github.com/infraspecdev/goperf/internal/stats"
)

type RequestResult struct {
	StatusCode int
	Duration   time.Duration
	Error      error
}

var client = &http.Client{}

func MakeRequest(ctx context.Context, rawURL string, timeout time.Duration) (statusCode int, duration time.Duration, err error) {
	reqCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	start := time.Now()

	req, err := http.NewRequestWithContext(reqCtx, http.MethodGet, rawURL, nil)
	if err != nil {
		return 0, 0, err
	}

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

func RunMultipleConcurrent(ctx context.Context, rawURL string, n, concurrency int, timeout time.Duration) []RequestResult {
	results := make([]RequestResult, n)
	jobs := make(chan int, concurrency)

	var wg sync.WaitGroup
	wg.Add(concurrency)

	for w := 0; w < concurrency; w++ {
		go func() {
			defer wg.Done()
			for i := range jobs {
				if ctx.Err() != nil {
					results[i] = RequestResult{Error: ctx.Err()}
					continue
				}
				statusCode, duration, err := MakeRequest(ctx, rawURL, timeout)
				results[i] = RequestResult{
					StatusCode: statusCode,
					Duration:   duration,
					Error:      err,
				}
			}
		}()
	}

	for i := 0; i < n; i++ {
		if ctx.Err() != nil {
			break
		}
		jobs <- i
	}
	close(jobs)

	wg.Wait()
	return results
}

func RunForDuration(ctx context.Context, rawURL string, concurrency int, timeout time.Duration, duration time.Duration) *stats.HistogramRecorder {
	recorder := stats.NewHistogramRecorder(timeout)

	ctx, cancel := context.WithTimeout(ctx, duration)
	defer cancel()

	var wg sync.WaitGroup
	wg.Add(concurrency)

	for w := 0; w < concurrency; w++ {
		go func() {
			defer wg.Done()
			for {
				if ctx.Err() != nil {
					return
				}
				_, d, err := MakeRequest(ctx, rawURL, timeout)
				if err == nil {
					recorder.Record(d)
				}
			}
		}()
	}

	wg.Wait()
	return recorder
}
