package cmd

import (
	"fmt"
	"testing"
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
		name      string
		input     int
		wantErr   bool
	}{
		{"Valid: positive number", 5, false},
		{"Valid: exactly 1", 1, false},
		{"Invalid: zero", 0, true},
		{"Invalid: negative number", -5, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd := runCmd
			cmd.Flags().Set("requests", fmt.Sprintf("%d", tt.input))
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