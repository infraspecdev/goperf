package cmd

import "testing"

func TestRootCmdInitialization(t *testing.T) {

	cmd := NewRootCmd()
	if cmd == nil {
		t.Fatal("NewRootCmd() returned nil")
	}

	if cmd.Use == "" {
		t.Error("rootCmd Use field is empty")
	}

	if cmd.Use != "goperf" {
		t.Errorf("expected rootCmd Use='goperf', got '%s'", cmd.Use)
	}
}
