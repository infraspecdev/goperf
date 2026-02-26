package cmd

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

func TestRunCommand_StatusCodes(t *testing.T) {
	tests := []struct {
		name       string
		statusCode int
		expected   string
	}{
		{"200 OK", http.StatusOK, "Status: 200 OK"},
		{"404 Not Found", http.StatusNotFound, "Status: 404 Not Found"},
		{"500 Internal Server Error", http.StatusInternalServerError, "Status: 500 Internal Server Error"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(tt.statusCode)
			}))
			defer server.Close()

			var out bytes.Buffer

			err := runCommandMultipleConcurrent(server.URL, 1, 1, 10*time.Second, &out)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			output := out.String()

			if !strings.Contains(output, tt.expected) {
				t.Fatalf("expected status line %q, got: %s", tt.expected, output)
			}

			if !strings.Contains(output, "Time:") {
				t.Fatalf("expected time output, got: %s", output)
			}
		})
	}
}

func TestRunCommand_ConnectionError(t *testing.T) {
	var out bytes.Buffer

	err := runCommandMultipleConcurrent("http://localhost:9999", 1, 1, 500*time.Millisecond, &out)
	if err != nil {
		t.Fatalf("runCommand should handle connection errors gracefully and not return error, got: %v", err)
	}

	output := out.String()
	if !strings.Contains(output, "Status: Error") {
		t.Fatalf("expected Status: Error for connection error, got: %s", output)
	}
	if !strings.Contains(output, "Time:") || !strings.Contains(output, "ms") {
		t.Fatalf("expected Time in ms for connection error, got: %s", output)
	}
}

func TestRunCommand_MultipleRequests(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusCreated)
	}))
	defer server.Close()

	var out bytes.Buffer
	requests := 3

	err := runCommandMultipleConcurrent(server.URL, requests, 1, 10*time.Second, &out)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	output := out.String()
	count := strings.Count(output, "Status: 201 Created")
	if count != requests {
		t.Fatalf("expected %d status lines, got %d. Output: %s", requests, count, output)
	}

	if strings.Count(output, "Time:") != requests {
		t.Fatalf("expected %d time outputs, got: %s", requests, output)
	}

	if !strings.Contains(output, "Statistics:") {
		t.Fatalf("expected Statistics header, got: %s", output)
	}
	if !strings.Contains(output, "Min:") {
		t.Fatalf("expected Min statistic, got: %s", output)
	}
	if !strings.Contains(output, "Max:") {
		t.Fatalf("expected Max statistic, got: %s", output)
	}
	if !strings.Contains(output, "Avg:") {
		t.Fatalf("expected Avg statistic, got: %s", output)
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
	rootCmd.SetOut(&out)
	rootCmd.SetArgs([]string{"run", server.URL, "-n", requests, "-c", concurrency})

	defer func() {
		_ = runCmd.Flags().Set("requests", "1")
		_ = runCmd.Flags().Set("concurrency", "1")
		_ = runCmd.Flags().Set("timeout", "10s")
	}()

	start := time.Now()
	err := rootCmd.Execute()
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
	if !strings.Contains(output, "Statistics:") {
		t.Errorf("Expected output to contain statistics, but it didn't")
	}

	t.Logf("Test finished in %v, max concurrency seen by server: %d", duration, maxSeen)
}

func TestRunCommand_DurationMode(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	var out bytes.Buffer
	rootCmd.SetOut(&out)
	rootCmd.SetArgs([]string{"run", server.URL, "--duration", "1s", "-c", "2"})

	defer func() {
		_ = runCmd.Flags().Set("requests", "1")
		_ = runCmd.Flags().Set("concurrency", "1")
		_ = runCmd.Flags().Set("timeout", "10s")
		_ = runCmd.Flags().Set("duration", "0s")
	}()

	start := time.Now()
	err := rootCmd.Execute()
	elapsed := time.Since(start)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if elapsed < 1*time.Second {
		t.Errorf("expected to run for at least 1s, ran for %v", elapsed)
	}

	output := out.String()

	expectedSubstrings := []string{"Statistics:", "Total:", "Min:", "Max:", "Avg:", "P50:", "P90:", "P99:"}
	for _, sub := range expectedSubstrings {
		if !strings.Contains(output, sub) {
			t.Errorf("expected output to contain %q, got:\n%s", sub, output)
		}
	}

}
