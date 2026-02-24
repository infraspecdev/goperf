package cmd

import (
	"fmt"
	"testing"
	"time"
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
