package cmd

import "testing"

func TestRootCmdInitialization(t *testing.T) {
	
	if rootCmd == nil { // Ensure rootCmd is initialized
		t.Fatal("rootCmd is nil")
	}

	if rootCmd.Use == "" { // Ensure the command name is set
		t.Error("rootCmd Use field is empty")
	}

	if rootCmd.Use != "goperf" { // Ensure the command name is exactly "goperf"
		t.Errorf("expected rootCmd Use='goperf', got '%s'", rootCmd.Use)
	}
}
