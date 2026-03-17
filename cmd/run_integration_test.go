package cmd

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func TestRunCommand_RequestCountMode(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(10 * time.Millisecond)
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	var out bytes.Buffer
	requests := "3"
	concurrency := "2"

	cmd := NewRootCmd()
	cmd.SetOut(&out)
	cmd.SetArgs([]string{"run", server.URL, "-n", requests, "-c", concurrency})

	err := cmd.Execute()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	output := out.String()
	expectedIntro := fmt.Sprintf("Making %s requests to %s with concurrency %s\n", requests, server.URL, concurrency)
	if !strings.Contains(output, expectedIntro) {
		t.Errorf("Expected intro %q, got: %s", expectedIntro, output)
	}

	if !strings.Contains(output, "Requests:   3 total (3 succeeded, 0 failed)") {
		t.Errorf("Expected 'Requests:   3 total (3 succeeded, 0 failed)', got: %s", output)
	}
	expectedStats := []string{"Fastest:", "Slowest:", "Average:", "p50:", "p90:", "p99:", "Throughput:"}
	for _, stat := range expectedStats {
		if !strings.Contains(output, stat) {
			t.Errorf("expected %s statistic, got: %s", stat, output)
		}
	}
}

func TestRunCommand_ConnectionError(t *testing.T) {
	var out bytes.Buffer

	cmd := NewRootCmd()
	cmd.SetOut(&out)
	cmd.SetArgs([]string{"run", "http://127.0.0.1:12345", "-n", "2", "-c", "1"})

	err := cmd.Execute()
	if err != nil {
		t.Fatalf("expected graceful handle but got error: %v", err)
	}

	output := out.String()
	if !strings.Contains(output, "Requests:   2 total (0 succeeded, 2 failed)") {
		t.Errorf("Expected 'Requests:   2 total (0 succeeded, 2 failed)', got: %s", output)
	}
}

func TestRunCommand_Concurrency(t *testing.T) {
	var currentConcurrency int32
	var maxConcurrency int32

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		count := atomic.AddInt32(&currentConcurrency, 1)
		defer atomic.AddInt32(&currentConcurrency, -1)

		for {
			currentMax := atomic.LoadInt32(&maxConcurrency)
			if count <= currentMax {
				break
			}
			if atomic.CompareAndSwapInt32(&maxConcurrency, currentMax, count) {
				break
			}
		}

		time.Sleep(100 * time.Millisecond)
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	var out bytes.Buffer
	requests := "10"
	concurrency := "5"
	cmd := NewRootCmd()
	cmd.SetOut(&out)
	cmd.SetArgs([]string{"run", server.URL, "-n", requests, "-c", concurrency})

	start := time.Now()
	err := cmd.Execute()
	duration := time.Since(start)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if duration > 500*time.Millisecond {
		t.Errorf("Concurrency test took too long: %v (expected < 500ms)", duration)
	}

	maxSeen := atomic.LoadInt32(&maxConcurrency)
	if maxSeen < 5 {
		t.Errorf("Expected max concurrency 5, but server only saw %d", maxSeen)
	}

	output := out.String()
	if !strings.Contains(output, "Latency:") {
		t.Errorf("Expected output to contain Latency header, but it didn't")
	}

	t.Logf("Test finished in %v, max concurrency seen by server: %d", duration, maxSeen)
}

func TestRunCommand_DurationMode(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	var out bytes.Buffer

	cmd := NewRootCmd()
	cmd.SetOut(&out)
	cmd.SetArgs([]string{"run", server.URL, "--duration", "1s", "-c", "2"})

	start := time.Now()
	err := cmd.Execute()
	elapsed := time.Since(start)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if elapsed < 1*time.Second {
		t.Errorf("expected to run for at least 1s, ran for %v", elapsed)
	}

	output := out.String()

	expectedSubstrings := []string{"Latency:", "Target:", "Duration:", "Requests:", "Fastest:", "Slowest:", "Average:", "p50:", "p90:", "p99:", "Throughput:"}
	for _, sub := range expectedSubstrings {
		if !strings.Contains(output, sub) {
			t.Errorf("expected output to contain %q, got:\n%s", sub, output)
		}
	}

}

func TestRunCommand_MethodFlag(t *testing.T) {
	methods := []string{"GET", "POST", "PUT", "DELETE", "PATCH", "OPTIONS", "HEAD"}

	for _, method := range methods {
		t.Run(method, func(t *testing.T) {
			var receivedMethod string
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				receivedMethod = r.Method
				w.WriteHeader(http.StatusOK)
			}))
			defer server.Close()

			var out bytes.Buffer
			cmd := NewRootCmd()
			cmd.SetOut(&out)
			cmd.SetArgs([]string{"run", server.URL, "-n", "1", "-m", method})

			err := cmd.Execute()
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if receivedMethod != method {
				t.Errorf("expected server to receive %s, got %s", method, receivedMethod)
			}

			output := out.String()
			if !strings.Contains(output, "Requests:   1 total (1 succeeded, 0 failed)") {
				t.Errorf("expected successful request output, got: %s", output)
			}
		})
	}
}

func TestRunCommand_InvalidMethod(t *testing.T) {
	var out bytes.Buffer
	cmd := NewRootCmd()
	cmd.SetOut(&out)
	cmd.SetErr(&out)
	cmd.SetArgs([]string{"run", "http://example.com", "-n", "1", "-m", "TRACE"})

	err := cmd.Execute()
	if err == nil {
		t.Fatal("expected error for invalid method, got nil")
	}

	if !strings.Contains(err.Error(), "supported methods: DELETE, GET, HEAD, OPTIONS, PATCH, POST, PUT") {
		t.Errorf("expected error to list supported methods, got: %v", err)
	}
}

func TestRunCommand_MethodWithBody(t *testing.T) {
	expectedBody := `{"key":"value"}`
	methods := []string{"POST", "PUT", "PATCH"}

	for _, method := range methods {
		t.Run(method, func(t *testing.T) {
			var receivedMethod string
			var receivedBody string

			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				receivedMethod = r.Method
				body, _ := io.ReadAll(r.Body)
				receivedBody = string(body)
				w.WriteHeader(http.StatusOK)
			}))
			defer server.Close()

			var out bytes.Buffer
			cmd := NewRootCmd()
			cmd.SetOut(&out)
			cmd.SetArgs([]string{"run", server.URL, "-n", "1", "-m", method, "-b", expectedBody})

			err := cmd.Execute()
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if receivedMethod != method {
				t.Errorf("expected %s, got %s", method, receivedMethod)
			}
			if receivedBody != expectedBody {
				t.Errorf("expected body %q, got %q", expectedBody, receivedBody)
			}
		})
	}
}

func TestRunCommand_HeaderFlag(t *testing.T) {
	var receivedAuth string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedAuth = r.Header.Get("Authorization")
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	var out bytes.Buffer
	cmd := NewRootCmd()
	cmd.SetOut(&out)
	cmd.SetArgs([]string{"run", server.URL, "-n", "1", "-H", "Authorization: Bearer test-token"})

	err := cmd.Execute()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if receivedAuth != "Bearer test-token" {
		t.Errorf("expected Authorization header 'Bearer test-token', got %q", receivedAuth)
	}
}

func TestRunCommand_MultipleHeaders(t *testing.T) {
	var receivedAuth string
	var receivedContentType string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedAuth = r.Header.Get("Authorization")
		receivedContentType = r.Header.Get("Content-Type")
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	var out bytes.Buffer
	cmd := NewRootCmd()
	cmd.SetOut(&out)
	cmd.SetArgs([]string{
		"run", server.URL, "-n", "1",
		"-H", "Authorization: Bearer multi-token",
		"-H", "Content-Type: application/json",
	})

	err := cmd.Execute()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if receivedAuth != "Bearer multi-token" {
		t.Errorf("expected Authorization header 'Bearer multi-token', got %q", receivedAuth)
	}
	if receivedContentType != "application/json" {
		t.Errorf("expected Content-Type header 'application/json', got %q", receivedContentType)
	}
}

func TestRunCommand_ConfigFile(t *testing.T) {
	var receivedMethod string
	var receivedAuth string
	var receivedConcurrency int32
	var mu sync.Mutex

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&receivedConcurrency, 1)

		mu.Lock()
		receivedMethod = r.Method
		receivedAuth = r.Header.Get("Authorization")
		mu.Unlock()

		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	configContent := fmt.Sprintf(`
target: %s
requests: 5
concurrency: 3
method: POST
headers:
  - "Authorization: Bearer from-file"
`, server.URL)

	configPath := writeTempFile(t, "test-config.yaml", configContent)

	var out bytes.Buffer
	cmd := NewRootCmd()
	cmd.SetOut(&out)

	cmd.SetArgs([]string{
		"run",
		"--config", configPath,
		"-m", "PUT",
		"-H", "Authorization: Bearer from-cli",
	})

	err := cmd.Execute()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	output := out.String()

	if receivedMethod != "PUT" {
		t.Errorf("expected CLI method PUT to override file POST, got %q", receivedMethod)
	}
	if receivedAuth != "Bearer from-cli" {
		t.Errorf("expected CLI header to override file header, got %q", receivedAuth)
	}

	if !strings.Contains(output, "Making 5 requests") {
		t.Errorf("expected 5 requests from config file, got output: %s", output)
	}
	if !strings.Contains(output, "concurrency 3") {
		t.Errorf("expected concurrency 3 from config file, got output: %s", output)
	}
}

func TestRunCommand_ConfigFileMissingURL(t *testing.T) {
	configContent := `
requests: 5
`
	configPath := writeTempFile(t, "missing-target.yaml", configContent)

	var out bytes.Buffer
	cmd := NewRootCmd()
	cmd.SetOut(&out)
	cmd.SetErr(&out)

	cmd.SetArgs([]string{"run", "--config", configPath})

	err := cmd.Execute()
	if err == nil {
		t.Fatal("expected error for missing target URL, got nil")
	}

	if !strings.Contains(err.Error(), "missing target URL") {
		t.Errorf("expected missing target URL error, got: %v", err)
	}
}

func TestVerboseE2EExecution(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	var out bytes.Buffer
	var errOut bytes.Buffer

	cmd := NewRootCmd()
	cmd.SetOut(&out)
	cmd.SetErr(&errOut)

	cmd.SetArgs([]string{"run", server.URL, "-n", "4", "-c", "2", "-v"})

	err := cmd.Execute()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	stdErrOutput := errOut.String()
	if !strings.Contains(stdErrOutput, "Request") {
		t.Errorf("expected verbose logging on Stderr, got %q", stdErrOutput)
	}

	lines := strings.Split(strings.TrimSpace(stdErrOutput), "\n")
	if len(lines) != 4 {
		t.Errorf("expected 4 request logs, got %d", len(lines))
	}
}
