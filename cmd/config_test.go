package cmd

import (
	"testing"
	"time"

	"github.com/infraspecdev/goperf/internal/httpclient"
)

func TestRunConfig_Validate(t *testing.T) {
	validConfig := func() RunConfig {
		return RunConfig{
			Target:      "https://example.com/api",
			Requests:    10,
			Concurrency: 2,
			Timeout:     10 * time.Second,
			Duration:    0,
			Method:      "POST",
			Headers:     []string{"Content-Type: application/json"},
		}
	}

	tests := []struct {
		name    string
		mutate  func(*RunConfig)
		wantErr bool
		errMsg  string
	}{
		{
			name:    "Valid configuration",
			mutate:  func(c *RunConfig) {},
			wantErr: false,
		},
		{
			name: "Invalid Target - Missing Scheme",
			mutate: func(c *RunConfig) {
				c.Target = "example.com"
			},
			wantErr: true,
			errMsg:  "invalid target URL provided: parse error: parse \"example.com\": invalid URI for request",
		},
		{
			name: "Invalid Target - No Host",
			mutate: func(c *RunConfig) {
				c.Target = "/api/v1"
			},
			wantErr: true,
			errMsg:  "invalid target URL provided: missing scheme (e.g., http:// or https://) or host",
		},
		{
			name: "Concurrency zero",
			mutate: func(c *RunConfig) {
				c.Concurrency = 0
			},
			wantErr: true,
			errMsg:  "concurrency must be positive, got 0",
		},
		{
			name: "Concurrency negative",
			mutate: func(c *RunConfig) {
				c.Concurrency = -5
			},
			wantErr: true,
			errMsg:  "concurrency must be positive, got -5",
		},
		{
			name: "Timeout zero",
			mutate: func(c *RunConfig) {
				c.Timeout = 0
			},
			wantErr: true,
			errMsg:  "timeout must be positive, got 0s",
		},
		{
			name: "Duration negative",
			mutate: func(c *RunConfig) {
				c.Duration = -1 * time.Second
			},
			wantErr: true,
			errMsg:  "duration must not be negative, got -1s",
		},
		{
			name: "Invalid Method",
			mutate: func(c *RunConfig) {
				c.Method = "TRACE"
			},
			wantErr: true,
			errMsg:  "invalid HTTP method \"TRACE\", supported methods: DELETE, GET, HEAD, OPTIONS, PATCH, POST, PUT",
		},
		{
			name: "Invalid Header - missing colon",
			mutate: func(c *RunConfig) {
				c.Headers = []string{"InvalidHeaderValue"}
			},
			wantErr: true,
			errMsg:  "invalid header format \"InvalidHeaderValue\", expected 'Key: Value' without spaces in the key",
		},
		{
			name: "Invalid Header - space in key",
			mutate: func(c *RunConfig) {
				c.Headers = []string{"Content Type: application/json"}
			},
			wantErr: true,
			errMsg:  "invalid header format \"Content Type: application/json\", expected 'Key: Value' without spaces in the key",
		},
		{
			name: "Both duration and multiple requests set",
			mutate: func(c *RunConfig) {
				c.Duration = 10 * time.Second
				c.Requests = 50
			},
			wantErr: false,
		},
		{
			name: "Zero duration but invalid requests",
			mutate: func(c *RunConfig) {
				c.Duration = 0
				c.Requests = 0
			},
			wantErr: true,
			errMsg:  "number of requests must be positive, got 0",
		},
		{
			name: "Valid duration suppresses requests check",
			mutate: func(c *RunConfig) {
				c.Duration = 10 * time.Second
				c.Requests = 0
			},
			wantErr: false,
		},

		{
			name: "URL - empty string",
			mutate: func(c *RunConfig) {
				c.Target = ""
			},
			wantErr: true,
			errMsg:  "missing target URL: must be provided via CLI argument or config file",
		},
		{
			name: "URL - has fragment",
			mutate: func(c *RunConfig) {
				c.Target = "http://example.com/path#frag"
			},
			wantErr: false,
		},
		{
			name: "URL - has spaces",
			mutate: func(c *RunConfig) {
				c.Target = "http://example .com"
			},
			wantErr: true,
			errMsg:  `invalid target URL provided: parse error: parse "http://example .com": invalid character " " in host name`,
		},
		{
			name: "URL - valid with port and path",
			mutate: func(c *RunConfig) {
				c.Target = "https://example.com:8080/api/v1"
			},
			wantErr: false,
		},
		{
			name: "URL - valid IP address target",
			mutate: func(c *RunConfig) {
				c.Target = "http://192.168.1.1:3000/test"
			},
			wantErr: false,
		},

		{
			name:    "Method - GET",
			mutate:  func(c *RunConfig) { c.Method = "GET" },
			wantErr: false,
		},
		{
			name:    "Method - POST",
			mutate:  func(c *RunConfig) { c.Method = "POST" },
			wantErr: false,
		},
		{
			name:    "Method - PUT",
			mutate:  func(c *RunConfig) { c.Method = "PUT" },
			wantErr: false,
		},
		{
			name:    "Method - DELETE",
			mutate:  func(c *RunConfig) { c.Method = "DELETE" },
			wantErr: false,
		},
		{
			name:    "Method - PATCH",
			mutate:  func(c *RunConfig) { c.Method = "PATCH" },
			wantErr: false,
		},
		{
			name:    "Method - OPTIONS",
			mutate:  func(c *RunConfig) { c.Method = "OPTIONS" },
			wantErr: false,
		},
		{
			name:    "Method - HEAD",
			mutate:  func(c *RunConfig) { c.Method = "HEAD" },
			wantErr: false,
		},

		{
			name: "Header - colon in value",
			mutate: func(c *RunConfig) {
				c.Headers = []string{"X-Custom: value:with:colons"}
			},
			wantErr: false,
		},
		{
			name: "Header - empty value after colon",
			mutate: func(c *RunConfig) {
				c.Headers = []string{"X-Custom: "}
			},
			wantErr: false,
		},
		{
			name: "Header - empty key before colon",
			mutate: func(c *RunConfig) {
				c.Headers = []string{": some-value"}
			},
			wantErr: true,
			errMsg:  `invalid header format ": some-value", expected 'Key: Value' without spaces in the key`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := validConfig()
			tt.mutate(&cfg)

			err := cfg.Validate()
			if tt.wantErr {
				if err == nil {
					t.Errorf("expected error but got nil")
				} else if tt.errMsg != "" && err.Error() != tt.errMsg {
					t.Errorf("expected error containing %q, got %q", tt.errMsg, err.Error())
				}
			} else {
				if err != nil {
					t.Errorf("unexpected error: %v", err)
				}
			}
		})
	}
}

func TestRunConfig_ToHTTPConfig(t *testing.T) {
	rc := RunConfig{
		Target:      "https://example.com/api",
		Requests:    100,
		Concurrency: 10,
		Timeout:     5 * time.Second,
		Duration:    30 * time.Second,
		Method:      "POST",
		Body:        `{"key":"value"}`,
		Headers:     []string{"Authorization: Bearer token", "X-Custom: val:with:colons"},
		Verbose:     true,
	}

	got := rc.ToHTTPConfig()

	want := httpclient.Config{
		Target:      "https://example.com/api",
		Requests:    100,
		Concurrency: 10,
		Timeout:     5 * time.Second,
		Duration:    30 * time.Second,
		Method:      "POST",
		Body:        `{"key":"value"}`,
		Headers:     []string{"Authorization: Bearer token", "X-Custom: val:with:colons"},
		Verbose:     true,
	}

	if got.Target != want.Target {
		t.Errorf("Target: got %q, want %q", got.Target, want.Target)
	}
	if got.Verbose != want.Verbose {
		t.Errorf("Verbose: got %v, want %v", got.Verbose, want.Verbose)
	}
	if got.Requests != want.Requests {
		t.Errorf("Requests: got %d, want %d", got.Requests, want.Requests)
	}
	if got.Concurrency != want.Concurrency {
		t.Errorf("Concurrency: got %d, want %d", got.Concurrency, want.Concurrency)
	}
	if got.Timeout != want.Timeout {
		t.Errorf("Timeout: got %v, want %v", got.Timeout, want.Timeout)
	}
	if got.Duration != want.Duration {
		t.Errorf("Duration: got %v, want %v", got.Duration, want.Duration)
	}
	if got.Method != want.Method {
		t.Errorf("Method: got %q, want %q", got.Method, want.Method)
	}
	if got.Body != want.Body {
		t.Errorf("Body: got %q, want %q", got.Body, want.Body)
	}
	if len(got.Headers) != len(want.Headers) {
		t.Fatalf("Headers length: got %d, want %d", len(got.Headers), len(want.Headers))
	}
	for i := range want.Headers {
		if got.Headers[i] != want.Headers[i] {
			t.Errorf("Headers[%d]: got %q, want %q", i, got.Headers[i], want.Headers[i])
		}
	}
}
