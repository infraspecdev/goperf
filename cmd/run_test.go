package cmd

import (
	"bytes"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/infraspecdev/goperf/internal/stats"
)

func TestValidateTarget(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		wantErr bool
	}{
		{"Valid HTTPS", "https://google.com", false},
		{"Valid HTTP with Port", "http://localhost:8080", false},
		{"Valid with Path", "https://example.com/api/v1", false},
		{"Error: Missing Scheme", "google.com", true},
		{"Error: Relative Path", "/api/login", true},
		{"Error: Just Fragment", "#top", true},
		{"Error: Empty String", "", true},
		{"Error: Invalid Characters", "https://exa mple.com", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := validateTarget(tt.input)

			if tt.wantErr {
				if err == nil {
					t.Errorf("Test %s: We expected an error but didn't get one", tt.name)
				}
			} else {
				if err != nil {
					t.Errorf("Test %s: We got an unexpected error: %v", tt.name, err)
				}
			}
		})
	}
}

func TestRunCmdHasNFlag(t *testing.T) {
	cmd := runCmd
	flag := cmd.Flags().Lookup("requests")
	if flag == nil {
		t.Error("Expected --n flag to exist, but it doesn't")
	}
}

func TestNFlagDefaultValue(t *testing.T) {
	cmd := runCmd
	requests, err := cmd.Flags().GetInt("requests")
	if err != nil {
		t.Errorf("Error getting requests flag: %v", err)
	}
	if requests != 1 {
		t.Errorf("Expected default requests value to be 1, got %d", requests)
	}
}

func TestNFlagPositiveValidation(t *testing.T) {
	tests := []struct {
		name    string
		input   int
		wantErr bool
	}{
		{"Valid: positive number", 5, false},
		{"Valid: exactly 1", 1, false},
		{"Invalid: zero", 0, true},
		{"Invalid: negative number", -5, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd := runCmd
			err := cmd.Flags().Set("requests", fmt.Sprintf("%d", tt.input))
			if err != nil {
				t.Fatalf("Error setting flag: %v", err)
			}
			requests, err := cmd.Flags().GetInt("requests")
			if err != nil {
				t.Fatalf("Error getting flag value: %v", err)
			}
			err = validateRequests(requests)

			if tt.wantErr {
				if err == nil {
					t.Errorf("Expected error but got none for input %d", tt.input)
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected error for input %d: %v", tt.input, err)
				}
			}
		})
	}
}

func TestRunCmdHasTimeoutFlag(t *testing.T) {
	cmd := runCmd
	flag := cmd.Flags().Lookup("timeout")
	if flag == nil {
		t.Fatal("Expected --timeout flag to exist")
	}
}

func TestTimeoutFlagDefaultValue(t *testing.T) {
	cmd := runCmd
	timeout, err := cmd.Flags().GetDuration("timeout")
	if err != nil {
		t.Fatalf("Error getting timeout flag: %v", err)
	}
	if timeout != 10*time.Second {
		t.Errorf("Expected default timeout to be 10s, got %v", timeout)
	}
}

func TestValidateTimeout(t *testing.T) {
	tests := []struct {
		name    string
		input   time.Duration
		wantErr bool
	}{
		{"Valid: 5 seconds", 5 * time.Second, false},
		{"Valid: 10 seconds", 10 * time.Second, false},
		{"Valid: 500 milliseconds", 500 * time.Millisecond, false},
		{"Invalid: zero", 0, true},
		{"Invalid: negative", -1 * time.Second, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateTimeout(tt.input)

			if tt.wantErr {
				if err == nil {
					t.Errorf("Expected error but got none for input %v", tt.input)
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected error for input %v: %v", tt.input, err)
				}
			}
		})
	}
}

func TestConcurrencyFlagExists(t *testing.T) {
	cmd := runCmd
	flag := cmd.Flags().Lookup("concurrency")

	if flag == nil {
		t.Fatal("expected concurrency flag to exist")
	}
}

func TestValidateConcurrency(t *testing.T) {
	tests := []struct {
		name    string
		input   int
		wantErr bool
	}{
		{"Valid: positive number", 5, false},
		{"Valid: exactly 1", 1, false},
		{"Invalid: zero", 0, true},
		{"Invalid: negative number", -10, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateConcurrency(tt.input)

			if tt.wantErr {
				if err == nil {
					t.Errorf("Expected error but got none for input %d", tt.input)
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected error for input %d: %v", tt.input, err)
				}
			}
		})
	}
}

func TestDurationFlagExists(t *testing.T) {
	cmd := runCmd
	flag := cmd.Flags().Lookup("duration")

	if flag == nil {
		t.Fatal("expected --duration flag to exist")
	}

	if flag.Shorthand != "d" {
		t.Errorf("expected shorthand -d, got -%s", flag.Shorthand)
	}
}

func TestDurationFlagDefault(t *testing.T) {
	cmd := runCmd
	duration, err := cmd.Flags().GetDuration("duration")
	if err != nil {
		t.Fatalf("Error getting duration flag: %v", err)
	}
	if duration != 0 {
		t.Errorf("expected default duration to be 0s, got %v", duration)
	}
}

func TestValidateDuration(t *testing.T) {
	tests := []struct {
		name    string
		input   time.Duration
		wantErr bool
	}{
		{"Valid: 10 seconds", 10 * time.Second, false},
		{"Valid: 1 minute", 1 * time.Minute, false},
		{"Valid: zero (disabled)", 0, false},
		{"Invalid: negative", -1 * time.Second, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateDuration(tt.input)

			if tt.wantErr {
				if err == nil {
					t.Errorf("Expected error but got none for input %v", tt.input)
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected error for input %v: %v", tt.input, err)
				}
			}
		})
	}
}

func TestPrintHistogramStatistics(t *testing.T) {
	recorder := stats.NewHistogramRecorder(10 * time.Second)
	recorder.Record(10 * time.Millisecond)
	recorder.Record(20 * time.Millisecond)
	recorder.Record(30 * time.Millisecond)

	var buf bytes.Buffer
	err := printHistogramStatistics(&buf, recorder, "http://example.com", 1*time.Second)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	output := buf.String()

	expectedSubstrings := []string{"Target:", "Duration:", "Requests:", "Latency:", "Fastest:", "Slowest:", "Average:", "p50:", "p90:", "p99:", "Throughput:"}
	for _, sub := range expectedSubstrings {
		if !strings.Contains(output, sub) {
			t.Errorf("expected output to contain %q, got:\n%s", sub, output)
		}
	}
}

func TestMethodFlagExists(t *testing.T) {
	cmd := runCmd
	flag := cmd.Flags().Lookup("method")
	if flag == nil {
		t.Fatal("Expected --method flag to exist")
	}
}

func TestMethodFlagDefaultValue(t *testing.T) {
	cmd := runCmd
	method, err := cmd.Flags().GetString("method")
	if err != nil {
		t.Fatalf("Error getting method flag: %v", err)
	}
	if method != "GET" {
		t.Errorf("Expected default method to be GET, got %v", method)
	}
}

func TestValidateMethod(t *testing.T) {
	tests := []struct {
		name    string
		method  string
		wantErr bool
	}{
		{"Valid: GET", "GET", false},
		{"Valid: POST", "POST", false},
		{"Valid: PUT", "PUT", false},
		{"Valid: DELETE", "DELETE", false},
		{"Invalid: PATCH", "PATCH", true},
		{"Invalid: RANDOM", "INVALID", true},
		{"Invalid: Empty", "", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateMethod(tt.method)
			if tt.wantErr {
				if err == nil {
					t.Errorf("Expected error for method %q, but got none", tt.method)
				} else if !strings.Contains(err.Error(), "supported methods: GET, POST, PUT, DELETE") {
					t.Errorf("Error message %q should contain supported methods list", err.Error())
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected error for method %q: %v", tt.method, err)
				}
			}
		})
	}
}

func TestBodyFlagExists(t *testing.T) {
	cmd := runCmd
	flag := cmd.Flags().Lookup("body")
	if flag == nil {
		t.Fatal("Expected --body flag to exist")
	}
}

func TestBodyFlagDefaultValue(t *testing.T) {
	cmd := runCmd
	body, err := cmd.Flags().GetString("body")
	if err != nil {
		t.Fatalf("Error getting body flag: %v", err)
	}
	if body != "" {
		t.Errorf("Expected default body to be empty string, got %q", body)
	}
}

func TestValidateHeaders(t *testing.T) {
	tests := []struct {
		name    string
		headers []string
		wantErr bool
	}{
		{"Valid: single header", []string{"Content-Type: application/json"}, false},
		{"Valid: multiple headers", []string{"Authorization: Bearer token", "Accept: text/html"}, false},
		{"Valid: header with multiple colons", []string{"X-Custom: value:with:colons"}, false},
		{"Valid: empty list", []string{}, false},
		{"Invalid: missing colon", []string{"InvalidHeader"}, true},
		{"Invalid: empty key", []string{": value"}, true},
		{"Invalid: only colon", []string{":"}, true},
		{"Invalid: space in key", []string{"Invalid Key: value"}, true},
		{"Invalid: space in key with padding", []string{"  Invalid Key  : value"}, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateHeaders(tt.headers)
			if tt.wantErr {
				if err == nil {
					t.Errorf("Expected error but got none for headers %v", tt.headers)
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected error for headers %v: %v", tt.headers, err)
				}
			}
		})
	}
}

func TestHeaderFlagExists(t *testing.T) {
	cmd := runCmd
	flag := cmd.Flags().Lookup("header")
	if flag == nil {
		t.Fatal("Expected --header flag to exist")
	}
	if flag.Shorthand != "H" {
		t.Errorf("Expected shorthand -H, got -%s", flag.Shorthand)
	}
}

func TestHeaderFlagDefaultValue(t *testing.T) {
	cmd := runCmd
	headers, err := cmd.Flags().GetStringArray("header")
	if err != nil {
		t.Fatalf("Error getting header flag: %v", err)
	}
	if len(headers) != 0 {
		t.Errorf("Expected default headers to be empty, got %v", headers)
	}
}
