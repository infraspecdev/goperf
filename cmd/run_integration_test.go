package cmd

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestRunCommand_PrintsStatus(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	var out bytes.Buffer

	err := runCommand(server.URL, &out)
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

	err := runCommand(server.URL, &out)
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

	err := runCommand(server.URL, &out)
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
