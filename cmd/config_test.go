package cmd

import (
	"testing"
	"time"
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
			wantErr: true,
			errMsg:  "cannot use both --requests (-n) and --duration (-d) at the same time",
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
