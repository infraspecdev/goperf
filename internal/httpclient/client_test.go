package httpclient

import (
	"context"
	"errors"
	"net"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

const testTimeout = 2 * time.Second

func TestMakeRequestSuccess(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	status, duration, err := MakeRequest(context.Background(), server.URL, testTimeout)

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
	_, _, err := MakeRequest(context.Background(), "http://127.0.0.1:9999", testTimeout)

	if err == nil {
		t.Fatalf("expected error, got nil")
	}

	if err.Error() != "connection refused" {
		t.Fatalf("expected 'connection refused', got %q", err.Error())
	}
}

func TestMakeRequestNoSuchHost(t *testing.T) {
	_, _, err := MakeRequest(context.Background(), "http://this-host-does-not-exist-12345", testTimeout)

	if err == nil {
		t.Fatalf("expected error, got nil")
	}

	var dnsErr *net.DNSError
	if !errors.As(err, &dnsErr) {
		t.Fatalf("expected DNS error, got %v", err)
	}
}

func TestRunMultipleExecutesNTimes(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	results := RunMultiple(context.Background(), server.URL, 3, testTimeout)

	if len(results) != 3 {
		t.Fatalf("expected 3 results, got %d", len(results))
	}
}

func TestRunMultipleCollectsResults(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	results := RunMultiple(context.Background(), server.URL, 2, testTimeout)

	for i, result := range results {
		if result.StatusCode != http.StatusOK {
			t.Errorf("result %d: expected status 200, got %d", i, result.StatusCode)
		}
	}
}

func TestRunMultipleEachRequestGetsOwnTimeout(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(50 * time.Millisecond)
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	results := RunMultiple(context.Background(), server.URL, 5, testTimeout)

	for i, result := range results {
		if result.Error != nil {
			t.Errorf("request %d failed: %v", i, result.Error)
		}
		if result.StatusCode != http.StatusOK {
			t.Errorf("request %d: expected status 200, got %d", i, result.StatusCode)
		}
	}
}
