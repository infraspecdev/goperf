package httpclient

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"strings"
	"time"
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

func RunMultiple(ctx context.Context, rawURL string, n int, timeout time.Duration) []RequestResult {
	results := make([]RequestResult, n)
	for i := 0; i < n; i++ {
		statusCode, duration, err := MakeRequest(ctx, rawURL, timeout)
		results[i] = RequestResult{
			StatusCode: statusCode,
			Duration:   duration,
			Error:      err,
		}
	}
	return results
}
