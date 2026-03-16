package httpclient

import (
	"context"
	"errors"
	"net/http"
	"strings"
	"testing"
	"time"
)

func TestMakeRequest_WithMock(t *testing.T) {
	t.Run("Success", func(t *testing.T) {
		mockClient := &MockHTTPDoer{
			DoFunc: func(req *http.Request) (*http.Response, error) {
				return &http.Response{
					StatusCode: http.StatusCreated,
					Body:       http.NoBody,
				}, nil
			},
		}

		cfg := Config{
			Target:  "http://example.com/api",
			Timeout: 5 * time.Second,
			Method:  "POST",
		}

		statusCode, _, err := MakeRequest(context.Background(), mockClient, cfg)
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
		if statusCode != http.StatusCreated {
			t.Errorf("expected status %d, got %d", http.StatusCreated, statusCode)
		}
	})

	t.Run("Failure", func(t *testing.T) {
		expectedErr := errors.New("network error")
		mockClient := &MockHTTPDoer{
			DoFunc: func(req *http.Request) (*http.Response, error) {
				return nil, expectedErr
			},
		}

		cfg := Config{
			Target:  "http://example.com/api",
			Timeout: 5 * time.Second,
			Method:  "GET",
		}

		_, _, err := MakeRequest(context.Background(), mockClient, cfg)
		if err == nil {
			t.Fatal("expected error, got nil")
		}
		if !strings.Contains(err.Error(), "network error") {
			t.Errorf("expected error to contain %q, got %v", "network error", err)
		}
	})
}
