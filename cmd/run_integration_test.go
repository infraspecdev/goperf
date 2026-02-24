package cmd

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"strings"
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

			err := runCommand(server.URL, 10*time.Second, &out)
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

	err := runCommand("http://localhost:9999", 500*time.Millisecond, &out)
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

	err := runCommandMultiple(server.URL, requests, 10*time.Second, &out)
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


func TestRunUsesConcurrentWhenCSet(t *testing.T) {
	var buf bytes.Buffer

	err := runCommandMultipleConcurrent("http://example.com", 3, 2, 2*time.Second, &buf)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}