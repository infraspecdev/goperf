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

func TestRunCommand_RequestCountMode(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(10 * time.Millisecond)
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	var out bytes.Buffer
	requests := "3"
	concurrency := "2"

	rootCmd.SetOut(&out)
	rootCmd.SetArgs([]string{"run", server.URL, "-n", requests, "-c", concurrency})

	defer func() {
		_ = runCmd.Flags().Set("requests", "1")
		_ = runCmd.Flags().Set("concurrency", "1")
		_ = runCmd.Flags().Set("timeout", "10s")
	}()

	err := rootCmd.Execute()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	output := out.String()
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

	rootCmd.SetOut(&out)
	rootCmd.SetArgs([]string{"run", "http://127.0.0.1:12345", "-n", "2", "-c", "1"})

	defer func() {
		_ = runCmd.Flags().Set("requests", "1")
		_ = runCmd.Flags().Set("concurrency", "1")
		_ = runCmd.Flags().Set("timeout", "10s")
	}()

	err := rootCmd.Execute()
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

	_ = runCmd.Flags().Set("requests", "1")
	runCmd.Flags().Lookup("requests").Changed = false

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

	expectedSubstrings := []string{"Latency:", "Target:", "Duration:", "Requests:", "Fastest:", "Slowest:", "Average:", "p50:", "p90:", "p99:", "Throughput:"}
	for _, sub := range expectedSubstrings {
		if !strings.Contains(output, sub) {
			t.Errorf("expected output to contain %q, got:\n%s", sub, output)
		}
	}

}
