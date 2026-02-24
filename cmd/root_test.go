package cmd

import "testing"

func TestRootCmdInitialization(t *testing.T) {

	if rootCmd == nil {
		t.Fatal("rootCmd is nil")
	}

	if rootCmd.Use == "" {
		t.Error("rootCmd Use field is empty")
	}

	if rootCmd.Use != "goperf" {
		t.Errorf("expected rootCmd Use='goperf', got '%s'", rootCmd.Use)
	}
}
