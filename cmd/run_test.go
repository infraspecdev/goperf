package cmd

import "testing"

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
