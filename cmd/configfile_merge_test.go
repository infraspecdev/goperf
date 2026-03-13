package cmd

import (
	"strings"
	"testing"
	"time"
)

func strPtr(s string) *string { return &s }
func intPtr(i int) *int       { return &i }

func TestMergeConfig_NilFileConfig(t *testing.T) {
	cli := RunConfig{
		Target:      "https://cli.example.com",
		Requests:    1,
		Concurrency: 1,
		Timeout:     10 * time.Second,
		Method:      "GET",
	}

	got, err := mergeConfig(nil, cli, map[string]bool{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if got.Target != cli.Target {
		t.Errorf("Target: got %q, want %q", got.Target, cli.Target)
	}
	if got.Requests != cli.Requests {
		t.Errorf("Requests: got %d, want %d", got.Requests, cli.Requests)
	}
}

func TestMergeConfig_FileValuesUsedWhenCLIUnchanged(t *testing.T) {
	file := &fileConfig{
		Target:      strPtr("https://file.example.com"),
		Requests:    intPtr(200),
		Concurrency: intPtr(20),
		Timeout:     strPtr("30s"),
		Duration:    strPtr("1m"),
		Method:      strPtr("POST"),
		Body:        strPtr(`{"data":"test"}`),
		Headers:     []string{"X-From: file"},
	}

	cli := RunConfig{
		Target:      "",
		Requests:    1,
		Concurrency: 1,
		Timeout:     10 * time.Second,
		Method:      "GET",
	}

	got, err := mergeConfig(file, cli, map[string]bool{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if got.Target != "https://file.example.com" {
		t.Errorf("Target: got %q, want %q", got.Target, "https://file.example.com")
	}
	if got.Requests != 200 {
		t.Errorf("Requests: got %d, want 200", got.Requests)
	}
	if got.Concurrency != 20 {
		t.Errorf("Concurrency: got %d, want 20", got.Concurrency)
	}
	if got.Timeout != 30*time.Second {
		t.Errorf("Timeout: got %v, want 30s", got.Timeout)
	}
	if got.Duration != 1*time.Minute {
		t.Errorf("Duration: got %v, want 1m", got.Duration)
	}
	if got.Method != "POST" {
		t.Errorf("Method: got %q, want %q", got.Method, "POST")
	}
	if got.Body != `{"data":"test"}` {
		t.Errorf("Body: got %q, want %q", got.Body, `{"data":"test"}`)
	}
	if len(got.Headers) != 1 || got.Headers[0] != "X-From: file" {
		t.Errorf("Headers: got %v, want [X-From: file]", got.Headers)
	}
}

func TestMergeConfig_CLIOverridesFileValues(t *testing.T) {
	file := &fileConfig{
		Target:      strPtr("https://file.example.com"),
		Requests:    intPtr(200),
		Concurrency: intPtr(20),
		Timeout:     strPtr("30s"),
		Method:      strPtr("POST"),
		Body:        strPtr(`{"data":"file"}`),
		Headers:     []string{"X-From: file"},
	}

	cli := RunConfig{
		Target:      "https://cli.example.com",
		Requests:    500,
		Concurrency: 50,
		Timeout:     5 * time.Second,
		Method:      "PUT",
		Body:        `{"data":"cli"}`,
		Headers:     []string{"X-From: cli"},
	}

	changed := map[string]bool{
		"requests":    true,
		"concurrency": true,
		"timeout":     true,
		"method":      true,
		"body":        true,
		"header":      true,
	}

	got, err := mergeConfig(file, cli, changed)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if got.Target != "https://file.example.com" {
		t.Errorf("Target: got %q, want %q (file value since URL not in changed)", got.Target, "https://file.example.com")
	}
	if got.Requests != 500 {
		t.Errorf("Requests: got %d, want 500 (CLI override)", got.Requests)
	}
	if got.Concurrency != 50 {
		t.Errorf("Concurrency: got %d, want 50 (CLI override)", got.Concurrency)
	}
	if got.Timeout != 5*time.Second {
		t.Errorf("Timeout: got %v, want 5s (CLI override)", got.Timeout)
	}
	if got.Method != "PUT" {
		t.Errorf("Method: got %q, want %q (CLI override)", got.Method, "PUT")
	}
	if got.Body != `{"data":"cli"}` {
		t.Errorf("Body: got %q, want %q (CLI override)", got.Body, `{"data":"cli"}`)
	}
	if len(got.Headers) != 1 || got.Headers[0] != "X-From: cli" {
		t.Errorf("Headers: got %v, want [X-From: cli] (CLI override)", got.Headers)
	}
}

func TestMergeConfig_PartialFileConfig(t *testing.T) {
	file := &fileConfig{
		Target:   strPtr("https://file.example.com"),
		Requests: intPtr(50),
	}

	cli := RunConfig{
		Target:      "",
		Requests:    1,
		Concurrency: 1,
		Timeout:     10 * time.Second,
		Method:      "GET",
	}

	got, err := mergeConfig(file, cli, map[string]bool{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if got.Target != "https://file.example.com" {
		t.Errorf("Target: got %q, want %q", got.Target, "https://file.example.com")
	}
	if got.Requests != 50 {
		t.Errorf("Requests: got %d, want 50", got.Requests)
	}
	if got.Concurrency != 1 {
		t.Errorf("Concurrency: got %d, want 1 (CLI default)", got.Concurrency)
	}
	if got.Timeout != 10*time.Second {
		t.Errorf("Timeout: got %v, want 10s (CLI default)", got.Timeout)
	}
	if got.Method != "GET" {
		t.Errorf("Method: got %q, want %q (CLI default)", got.Method, "GET")
	}
}

func TestMergeConfig_MethodNormalization(t *testing.T) {
	file := &fileConfig{
		Method: strPtr("post"),
	}

	cli := RunConfig{
		Target:      "https://example.com",
		Requests:    1,
		Concurrency: 1,
		Timeout:     10 * time.Second,
		Method:      "GET",
	}

	got, err := mergeConfig(file, cli, map[string]bool{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if got.Method != "POST" {
		t.Errorf("Method: got %q, want %q (normalized to uppercase)", got.Method, "POST")
	}
}

func TestMergeConfig_InvalidTimeoutString(t *testing.T) {
	file := &fileConfig{
		Timeout: strPtr("not-a-duration"),
	}

	cli := RunConfig{
		Target:      "https://example.com",
		Requests:    1,
		Concurrency: 1,
		Timeout:     10 * time.Second,
		Method:      "GET",
	}

	_, err := mergeConfig(file, cli, map[string]bool{})

	if err == nil {
		t.Fatal("expected error for invalid timeout string, got nil")
	}

	expectedErr := "invalid timeout format in config file"
	if err != nil && !strings.Contains(err.Error(), expectedErr) {
		t.Errorf("expected error to contain %q, got: %v", expectedErr, err)
	}
}

func TestMergeConfig_CLITargetOverridesFileTarget(t *testing.T) {
	file := &fileConfig{
		Target: strPtr("https://file.example.com"),
	}

	cli := RunConfig{
		Target:      "https://cli.example.com",
		Requests:    1,
		Concurrency: 1,
		Timeout:     10 * time.Second,
		Method:      "GET",
	}

	got, err := mergeConfig(file, cli, map[string]bool{"target": true})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if got.Target != "https://cli.example.com" {
		t.Errorf("Target: got %q, want %q (CLI override)", got.Target, "https://cli.example.com")
	}
}
