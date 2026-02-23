package httpclient

import (
	"context"
	"errors"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

const testTimeout = 2 * time.Second

func TestMakeRequestSuccess(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	status, duration, err := MakeRequest(context.Background(), server.URL, testTimeout)

	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if status != http.StatusOK {
		t.Fatalf("expected status 200, got %d", status)
	}

	if duration <= 0 {
		t.Fatalf("expected positive duration")
	}
}

func TestMakeRequest_Errors(t *testing.T) {
	tests := []struct {
		name          string
		url           string
		timeout       time.Duration
		expectedError string
		validateErr   func(*testing.T, error)
	}{
		{
			name:          "Connection Refused",
			url:           "http://127.0.0.1:9999",
			timeout:       testTimeout,
			expectedError: "connection refused",
		},
		{
			name:    "No Such Host",
			url:     "http://this-host-does-not-exist-12345",
			timeout: testTimeout,
			validateErr: func(t *testing.T, err error) {
				var dnsErr *net.DNSError
				if !errors.As(err, &dnsErr) {
					t.Fatalf("expected DNS error, got %v", err)
				}
			},
		},
		{
			name:          "Timeout Exceeded",
			url:           "",
			timeout:       50 * time.Millisecond,
			expectedError: "context deadline exceeded",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			targetURL := tt.url
			if tt.name == "Timeout Exceeded" {
				server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					time.Sleep(500 * time.Millisecond)
					w.WriteHeader(http.StatusOK)
				}))
				defer server.Close()
				targetURL = server.URL
			}

			_, _, err := MakeRequest(context.Background(), targetURL, tt.timeout)

			if err == nil {
				t.Fatalf("expected error, got nil")
			}

			if tt.expectedError != "" && !strings.Contains(strings.ToLower(err.Error()), tt.expectedError) {
				t.Fatalf("expected error to contain %q, got %q", tt.expectedError, err.Error())
			}

			if tt.validateErr != nil {
				tt.validateErr(t, err)
			}
		})
	}
}

func TestRunMultiple_Success(t *testing.T) {
	tests := []struct {
		name     string
		count    int
		handler  http.HandlerFunc
		validate func(*testing.T, []RequestResult)
	}{
		{
			name:  "ExecutesNTimes",
			count: 3,
			handler: func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusOK)
			},
			validate: func(t *testing.T, results []RequestResult) {
				if len(results) != 3 {
					t.Fatalf("expected 3 results, got %d", len(results))
				}
			},
		},
		{
			name:  "CollectsResults",
			count: 2,
			handler: func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusOK)
			},
			validate: func(t *testing.T, results []RequestResult) {
				for i, result := range results {
					if result.StatusCode != http.StatusOK {
						t.Errorf("result %d: expected status 200, got %d", i, result.StatusCode)
					}
				}
			},
		},
		{
			name:  "EachRequestGetsOwnTimeout",
			count: 5,
			handler: func(w http.ResponseWriter, r *http.Request) {
				time.Sleep(10 * time.Millisecond)
				w.WriteHeader(http.StatusOK)
			},
			validate: func(t *testing.T, results []RequestResult) {
				for i, result := range results {
					if result.Error != nil {
						t.Errorf("request %d failed: %v", i, result.Error)
					}
					if result.StatusCode != http.StatusOK {
						t.Errorf("request %d: expected status 200, got %d", i, result.StatusCode)
					}
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(tt.handler)
			defer server.Close()

			results := RunMultiple(context.Background(), server.URL, tt.count, testTimeout)
			tt.validate(t, results)
		})
	}
}

func TestRunMultiple_TimeoutExceeded(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(500 * time.Millisecond)
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	results := RunMultiple(context.Background(), server.URL, 3, 50*time.Millisecond)

	for i, result := range results {
		if result.Error == nil {
			t.Errorf("request %d: expected timeout error, got nil", i)
		}
		if result.StatusCode != 0 {
			t.Errorf("request %d: expected status 0, got %d", i, result.StatusCode)
		}
	}
}
