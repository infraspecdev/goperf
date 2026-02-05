package httpsclient

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestMakeRequestSuccess(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	status, duration, err := MakeRequest(server.URL)

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

func TestMakeRequestConnectionRefused(t *testing.T) {
	// No server running on this port → connection refused
	_, _, err := MakeRequest("http://127.0.0.1:9999")

	if err == nil {
		t.Fatalf("expected error, got nil")
	}

	if err.Error() != "connection refused" {
		t.Fatalf("expected 'connection refused', got %q", err.Error())
	}
}

func TestMakeRequestNoSuchHost(t *testing.T) {
	_, _, err := MakeRequest("http://this-host-does-not-exist-12345")

	if err == nil {
		t.Fatalf("expected error, got nil")
	}

	if err.Error() != "no such host" {
		t.Fatalf("expected 'no such host', got %q", err.Error())
	}
}
