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
