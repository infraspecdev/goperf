package httpclient

import (
	"context"
	"errors"
	"io"
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

	status, duration, err := MakeRequest(context.Background(), server.URL, testTimeout, "GET", "", nil)

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

			_, _, err := MakeRequest(context.Background(), targetURL, tt.timeout, "GET", "", nil)

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

func TestRunMultipleConcurrent_UsesConcurrency(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(100 * time.Millisecond)
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	n := 4
	concurrency := 4
	timeout := 2 * time.Second

	start := time.Now()
	recorder := RunMultipleConcurrent(context.Background(), server.URL, n, concurrency, timeout, "GET", "", nil)

	if recorder == nil {
		t.Fatal("expected non-nil recorder returned")
	}

	elapsed := time.Since(start)

	if recorder.Count() != int64(n) {
		t.Fatalf("expected %d results, got %d", n, recorder.Count())
	}

	if elapsed > 250*time.Millisecond {
		t.Fatalf("expected concurrent execution, took %v", elapsed)
	}
}

func TestRunForDuration_ReturnsHistogram(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	duration := 1 * time.Second
	concurrency := 2
	timeout := 2 * time.Second

	start := time.Now()
	recorder := RunForDuration(context.Background(), server.URL, concurrency, timeout, duration, "GET", "", nil)
	elapsed := time.Since(start)

	if recorder == nil {
		t.Fatal("expected non-nil recorder")
	}
	if recorder.Count() == 0 {
		t.Fatal("expected at least one recorded request")
	}
	if recorder.Min() <= 0 {
		t.Errorf("expected positive Min, got %v", recorder.Min())
	}
	if recorder.Max() <= 0 {
		t.Errorf("expected positive Max, got %v", recorder.Max())
	}
	if recorder.Avg() <= 0 {
		t.Errorf("expected positive Avg, got %v", recorder.Avg())
	}
	if elapsed < duration {
		t.Errorf("expected to run for at least %v, ran for %v", duration, elapsed)
	}
}

func TestRunForDuration_RespectsContext(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	ctx, cancel := context.WithCancel(context.Background())

	go func() {
		time.Sleep(200 * time.Millisecond)
		cancel()
	}()

	start := time.Now()
	recorder := RunForDuration(ctx, server.URL, 2, 2*time.Second, 5*time.Second, "GET", "", nil)
	elapsed := time.Since(start)

	if recorder == nil {
		t.Fatal("expected non-nil recorder")
	}
	if elapsed > 1*time.Second {
		t.Errorf("expected early stop due to context cancellation, took %v", elapsed)
	}
}

func TestMakeRequest_Methods(t *testing.T) {
	tests := []struct {
		method string
	}{
		{"GET"},
		{"POST"},
		{"PUT"},
		{"DELETE"},
	}

	for _, tt := range tests {
		t.Run(tt.method, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.Method != tt.method {
					t.Errorf("expected %s method, got %s", tt.method, r.Method)
					w.WriteHeader(http.StatusMethodNotAllowed)
					return
				}
				w.WriteHeader(http.StatusOK)
			}))
			defer server.Close()

			status, _, err := MakeRequest(context.Background(), server.URL, testTimeout, tt.method, "", nil)
			if err != nil {
				t.Fatalf("expected no error, got %v", err)
			}
			if status != http.StatusOK {
				t.Fatalf("expected status 200, got %d", status)
			}
		})
	}
}

func TestMakeRequestWithBody(t *testing.T) {
	expectedBody := `{"key":"value"}`
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, err := io.ReadAll(r.Body)
		if err != nil {
			t.Fatalf("failed to read request body: %v", err)
		}
		if string(body) != expectedBody {
			t.Errorf("expected body %q, got %q", expectedBody, string(body))
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	status, _, err := MakeRequest(context.Background(), server.URL, testTimeout, "POST", expectedBody, nil)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if status != http.StatusOK {
		t.Fatalf("expected status 200, got %d", status)
	}
}

func TestMakeRequestGetNoBody(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, err := io.ReadAll(r.Body)
		if err != nil {
			t.Fatalf("failed to read request body: %v", err)
		}
		if len(body) != 0 {
			t.Errorf("expected empty body for GET, got %q", string(body))
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	status, _, err := MakeRequest(context.Background(), server.URL, testTimeout, "GET", "", nil)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if status != http.StatusOK {
		t.Fatalf("expected status 200, got %d", status)
	}
}

func TestMakeRequestWithHeaders(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Bearer test-token" {
			t.Errorf("expected Authorization header 'Bearer test-token', got %q", r.Header.Get("Authorization"))
		}
		if r.Header.Get("X-Custom") != "my-value" {
			t.Errorf("expected X-Custom header 'my-value', got %q", r.Header.Get("X-Custom"))
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	status, _, err := MakeRequest(context.Background(), server.URL, testTimeout, "GET", "", []string{
		"Authorization: Bearer test-token",
		"X-Custom: my-value",
	})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if status != http.StatusOK {
		t.Fatalf("expected status 200, got %d", status)
	}
}

func TestRunMultipleConcurrent_WithHeaders(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("X-Test") != "hello" {
			t.Errorf("expected X-Test header 'hello', got %q", r.Header.Get("X-Test"))
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	recorder := RunMultipleConcurrent(context.Background(), server.URL, 3, 2, testTimeout, "GET", "", []string{"X-Test: hello"})
	if recorder == nil {
		t.Fatal("expected non-nil recorder")
	}
	if recorder.Count() != 3 {
		t.Errorf("expected 3 successful requests, got %d", recorder.Count())
	}
}

func TestRunForDuration_WithHeaders(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("X-Duration-Test") != "yes" {
			t.Errorf("expected X-Duration-Test header 'yes', got %q", r.Header.Get("X-Duration-Test"))
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	recorder := RunForDuration(context.Background(), server.URL, 2, testTimeout, 500*time.Millisecond, "GET", "", []string{"X-Duration-Test: yes"})
	if recorder == nil {
		t.Fatal("expected non-nil recorder")
	}
	if recorder.Count() == 0 {
		t.Error("expected at least one recorded request")
	}
}
