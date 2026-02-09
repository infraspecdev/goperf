package httpsclient

import (
	"errors"
	"net"
	"net/http"
	"strings"
	"time"
)

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
