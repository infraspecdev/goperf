package httpclient

import (
	"context"
	"errors"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

const testTimeout = 2 * time.Second

func TestNewHTTPClient(t *testing.T) {
	client := NewHTTPClient(50)

	tr, ok := client.Transport.(*http.Transport)
	if !ok {
		t.Fatal("expected *http.Transport")
	}
	if tr.MaxIdleConnsPerHost != 50 {
		t.Errorf("expected MaxIdleConnsPerHost=50, got %d", tr.MaxIdleConnsPerHost)
	}
	if !tr.DisableCompression {
		t.Error("expected DisableCompression=true")
	}
}

func TestMakeRequestSuccess(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	status, duration, err := MakeRequest(context.Background(), &http.Client{}, Config{
		Target:  server.URL,
		Timeout: testTimeout,
		Method:  "GET",
	})

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

			_, _, err := MakeRequest(context.Background(), &http.Client{}, Config{
				Target:  targetURL,
				Timeout: tt.timeout,
				Method:  "GET",
			})

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
	cfg := Config{
		Target:      server.URL,
		Requests:    n,
		Concurrency: concurrency,
		Timeout:     timeout,
		Method:      "GET",
	}
	recorder := RunMultipleConcurrent(context.Background(), cfg)

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
	cfg := Config{
		Target:      server.URL,
		Concurrency: concurrency,
		Timeout:     timeout,
		Duration:    duration,
		Method:      "GET",
	}
	recorder := RunForDuration(context.Background(), cfg)
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
	cfg := Config{
		Target:      server.URL,
		Concurrency: 2,
		Timeout:     2 * time.Second,
		Duration:    5 * time.Second,
		Method:      "GET",
	}
	recorder := RunForDuration(ctx, cfg)
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
		{"PATCH"},
		{"OPTIONS"},
		{"HEAD"},
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

			status, _, err := MakeRequest(context.Background(), &http.Client{}, Config{
				Target:  server.URL,
				Timeout: testTimeout,
				Method:  tt.method,
			})
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
	methods := []string{"POST", "PUT", "PATCH"}

	for _, method := range methods {
		t.Run(method, func(t *testing.T) {
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

			status, _, err := MakeRequest(context.Background(), &http.Client{}, Config{
				Target:  server.URL,
				Timeout: testTimeout,
				Method:  method,
				Body:    expectedBody,
			})
			if err != nil {
				t.Fatalf("expected no error, got %v", err)
			}
			if status != http.StatusOK {
				t.Fatalf("expected status 200, got %d", status)
			}
		})
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

	status, _, err := MakeRequest(context.Background(), &http.Client{}, Config{
		Target:  server.URL,
		Timeout: testTimeout,
		Method:  "GET",
	})
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

		acceptHeaders := r.Header.Values("Accept")
		if len(acceptHeaders) != 2 || acceptHeaders[0] != "text/plain" || acceptHeaders[1] != "application/json" {
			t.Errorf("expected Accept header to have values ['text/plain', 'application/json'], got %v", acceptHeaders)
		}

		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	status, _, err := MakeRequest(context.Background(), &http.Client{}, Config{
		Target:  server.URL,
		Timeout: testTimeout,
		Method:  "GET",
		Headers: []string{
			"Authorization: Bearer test-token",
			"X-Custom: my-value",
			"Accept: text/plain",
			"Accept: application/json",
		},
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

	cfg := Config{
		Target:      server.URL,
		Requests:    3,
		Concurrency: 2,
		Timeout:     testTimeout,
		Method:      "GET",
		Headers:     []string{"X-Test: hello"},
	}
	recorder := RunMultipleConcurrent(context.Background(), cfg)
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

	cfg := Config{
		Target:      server.URL,
		Concurrency: 2,
		Timeout:     testTimeout,
		Duration:    500 * time.Millisecond,
		Method:      "GET",
		Headers:     []string{"X-Duration-Test: yes"},
	}
	recorder := RunForDuration(context.Background(), cfg)
	if recorder == nil {
		t.Fatal("expected non-nil recorder")
	}
	if recorder.Count() == 0 {
		t.Error("expected at least one recorded request")
	}
}

func TestRunForDuration_ServerErrors(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	cfg := Config{
		Target:      server.URL,
		Concurrency: 2,
		Timeout:     testTimeout,
		Duration:    500 * time.Millisecond,
		Method:      "GET",
	}
	recorder := RunForDuration(context.Background(), cfg)

	if recorder.Count() != 0 {
		t.Errorf("expected 0 successful requests, got %d", recorder.Count())
	}
	if recorder.FailedCount() == 0 {
		t.Error("expected at least one failed request")
	}
	if recorder.TotalRequests() != recorder.FailedCount() {
		t.Errorf("expected all requests to be failures: total=%d, failed=%d", recorder.TotalRequests(), recorder.FailedCount())
	}
}

func TestRunMultipleConcurrent_MixedStatusCodes(t *testing.T) {
	var reqCount int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		count := atomic.AddInt32(&reqCount, 1)
		if count%2 == 0 {
			w.WriteHeader(http.StatusInternalServerError)
		} else {
			w.WriteHeader(http.StatusOK)
		}
	}))
	defer server.Close()

	cfg := Config{
		Target:      server.URL,
		Requests:    4,
		Concurrency: 1,
		Timeout:     testTimeout,
		Method:      "GET",
	}
	recorder := RunMultipleConcurrent(context.Background(), cfg)

	if recorder.Count() != 2 {
		t.Errorf("expected 2 successful requests, got %d", recorder.Count())
	}
	if recorder.FailedCount() != 2 {
		t.Errorf("expected 2 failed requests, got %d", recorder.FailedCount())
	}
	if recorder.TotalRequests() != 4 {
		t.Errorf("expected 4 total requests, got %d", recorder.TotalRequests())
	}
}

func TestRunMultipleConcurrent_NonServerErrorCodes(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusTooManyRequests)
	}))
	defer server.Close()

	cfg := Config{
		Target:      server.URL,
		Requests:    3,
		Concurrency: 1,
		Timeout:     testTimeout,
		Method:      "GET",
	}
	recorder := RunMultipleConcurrent(context.Background(), cfg)

	if recorder.Count() != 0 {
		t.Errorf("expected 0 successful requests for 429 responses, got %d", recorder.Count())
	}
	if recorder.FailedCount() != 3 {
		t.Errorf("expected 3 failed requests for 429 responses, got %d", recorder.FailedCount())
	}
}

func TestVerboseLogging(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	var stderrBuf strings.Builder

	cfg := Config{
		Target:      server.URL,
		Requests:    4,
		Concurrency: 2,
		Timeout:     testTimeout,
		Method:      "GET",
		Verbose:     true,
		Stderr:      &stderrBuf,
	}

	RunMultipleConcurrent(context.Background(), cfg)

	output := stderrBuf.String()
	if !strings.Contains(output, "Request") {
		t.Errorf("expected verbose output to contain 'Request', got %q", output)
	}

	lines := strings.Split(strings.TrimSpace(output), "\n")
	if len(lines) != 4 {
		t.Errorf("expected 4 lines of output, got %d", len(lines))
	}
}

func TestMakeRequestLatency(t *testing.T) {
	tests := []struct {
		name  string
		delay time.Duration
	}{
		{"Short delay", 50 * time.Millisecond},
		{"Medium delay", 100 * time.Millisecond},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				time.Sleep(tt.delay)
				w.WriteHeader(http.StatusOK)
			}))
			defer server.Close()

			cfg := Config{
				Target:  server.URL,
				Timeout: testTimeout,
				Method:  "GET",
			}

			_, duration, err := MakeRequest(context.Background(), &http.Client{}, cfg)
			if err != nil {
				t.Fatalf("expected no error, got %v", err)
			}

			minDuration := tt.delay - 5*time.Millisecond
			maxDuration := tt.delay + 50*time.Millisecond

			if duration < minDuration || duration > maxDuration {
				t.Errorf("%s: expected latency between %v and %v, got %v", tt.name, minDuration, maxDuration, duration)
			}
		})
	}
}

func BenchmarkMakeRequest(b *testing.B) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	cfg := Config{
		Target:  server.URL,
		Timeout: 2 * time.Second,
		Method:  "GET",
	}
	client := &http.Client{}

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _, err := MakeRequest(context.Background(), client, cfg)
		if err != nil {
			b.Fatalf("MakeRequest failed: %v", err)
		}
	}
}
