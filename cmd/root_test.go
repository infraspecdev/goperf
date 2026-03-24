package cmd

import (
	"bytes"
	"testing"
)

func TestRootCmdInitialization(t *testing.T) {

	cmd := NewRootCmd("dev (test)")
	if cmd == nil {
		t.Fatal("NewRootCmd() returned nil")
		return
	}

	if cmd.Use == "" {
		t.Error("rootCmd Use field is empty")
	}

	if cmd.Use != "goperf" {
		t.Errorf("expected rootCmd Use='goperf', got '%s'", cmd.Use)
	}
}

func TestVersionFlag(t *testing.T) {
	var out bytes.Buffer
	cmd := NewRootCmd("dev (test)")
	cmd.SetOut(&out)
	cmd.SetArgs([]string{"--version"})

	err := cmd.Execute()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	output := out.String()
	if output == "" {
		t.Fatal("expected version output, got empty string")
	}

	if !bytes.Contains([]byte(output), []byte("goperf")) {
		t.Errorf("version output should contain 'goperf', got %q", output)
	}
}

func TestRootCmdNoArgsShowHelp(t *testing.T) {
	var out bytes.Buffer
	cmd := NewRootCmd("dev (test)")
	cmd.SetOut(&out)
	cmd.SetArgs([]string{})

	err := cmd.Execute()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	output := out.String()
	if output == "" {
		t.Fatal("expected help output, got empty string")
	}

	if !bytes.Contains([]byte(output), []byte("Usage:")) {
		t.Errorf("expected output to contain 'Usage:', got %q", output)
	}

	if !bytes.Contains([]byte(output), []byte("goperf [flags]")) {
		t.Errorf("expected output to contain usage info for goperf, got %q", output)
	}
}
