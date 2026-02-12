package httpclient

import (
	"context"
	"errors"
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

func MakeRequest(url string) (statusCode int, duration time.Duration, err error) {

	start := time.Now()

	resp, err := http.Get(url)

	duration = time.Since(start)

	if err != nil {

		var netErr *net.OpError

		if errors.As(err, &netErr) {

			if netErr.Op == "dial" {
				if strings.Contains(netErr.Err.Error(), "refused") {
					return 0, duration, errors.New("connection refused")
				}
				if strings.Contains(netErr.Err.Error(), "no such host") {
					return 0, duration, errors.New("no such host")
				}
			}
		}

		return 0, duration, err
	}

	defer resp.Body.Close()

	return resp.StatusCode, duration, nil
}

func RunMultiple(ctx context.Context, url string, n int) []RequestResult {
	results := make([]RequestResult, n)
	for i := 0; i < n; i++ {
		statusCode, duration, err := MakeRequest(url)
		results[i] = RequestResult{
			StatusCode: statusCode,
			Duration:   duration,
			Error:      err,
		}
	}
	return results
}
