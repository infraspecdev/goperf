package cmd

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestRunCommand_PrintsStatus(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	var out bytes.Buffer

	err := runCommand(server.URL, 10*time.Second, &out)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	output := out.String()

	if !strings.Contains(output, "Status: 200 OK") {
		t.Fatalf("expected status line, got: %s", output)
	}
}

func TestRunCommand_Prints404Status(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	var out bytes.Buffer

	err := runCommand(server.URL, 10*time.Second, &out)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	output := out.String()

	if !strings.Contains(output, "Status: 404 Not Found") {
		t.Fatalf("expected status line, got: %s", output)
	}

	if !strings.Contains(output, "Time:") {
		t.Fatalf("expected time output, got: %s", output)
	}
}

func TestRunCommand_Prints500Status(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	var out bytes.Buffer

	err := runCommand(server.URL, 10*time.Second, &out)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	output := out.String()

	if !strings.Contains(output, "Status: 500 Internal Server Error") {
		t.Fatalf("expected 500 status line, got: %s", output)
	}

	if !strings.Contains(output, "Time:") {
		t.Fatalf("expected time output, got: %s", output)
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
}
