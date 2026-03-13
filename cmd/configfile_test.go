package cmd

import (
	"os"
	"path/filepath"
	"testing"
)

func writeTempFile(t *testing.T, name, content string) string {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, name)
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatalf("failed to write temp file: %v", err)
	}
	return path
}

func TestLoadConfig_ValidJSON_AllFields(t *testing.T) {
	content := `{
		"target": "https://example.com",
		"requests": 100,
		"concurrency": 10,
		"timeout": "5s",
		"duration": "30s",
		"method": "POST",
		"body": "{\"key\":\"value\"}",
		"headers": ["Content-Type: application/json", "Authorization: Bearer token"]
	}`
	path := writeTempFile(t, "config.json", content)

	cfg, err := loadConfig(path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if cfg.Target == nil || *cfg.Target != "https://example.com" {
		t.Errorf("Target: got %v, want %q", cfg.Target, "https://example.com")
	}
	if cfg.Requests == nil || *cfg.Requests != 100 {
		t.Errorf("Requests: got %v, want 100", cfg.Requests)
	}
	if cfg.Concurrency == nil || *cfg.Concurrency != 10 {
		t.Errorf("Concurrency: got %v, want 10", cfg.Concurrency)
	}
	if cfg.Timeout == nil || *cfg.Timeout != "5s" {
		t.Errorf("Timeout: got %v, want %q", cfg.Timeout, "5s")
	}
	if cfg.Duration == nil || *cfg.Duration != "30s" {
		t.Errorf("Duration: got %v, want %q", cfg.Duration, "30s")
	}
	if cfg.Method == nil || *cfg.Method != "POST" {
		t.Errorf("Method: got %v, want %q", cfg.Method, "POST")
	}
	if cfg.Body == nil || *cfg.Body != `{"key":"value"}` {
		t.Errorf("Body: got %v, want %q", cfg.Body, `{"key":"value"}`)
	}
	if len(cfg.Headers) != 2 {
		t.Fatalf("Headers: got %d items, want 2", len(cfg.Headers))
	}
	if cfg.Headers[0] != "Content-Type: application/json" {
		t.Errorf("Headers[0]: got %q, want %q", cfg.Headers[0], "Content-Type: application/json")
	}
}

func TestLoadConfig_ValidJSON_PartialFields(t *testing.T) {
	content := `{"target": "https://example.com", "requests": 50}`
	path := writeTempFile(t, "partial.json", content)

	cfg, err := loadConfig(path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if cfg.Target == nil || *cfg.Target != "https://example.com" {
		t.Errorf("Target: got %v, want %q", cfg.Target, "https://example.com")
	}
	if cfg.Requests == nil || *cfg.Requests != 50 {
		t.Errorf("Requests: got %v, want 50", cfg.Requests)
	}
	if cfg.Concurrency != nil {
		t.Errorf("Concurrency: expected nil, got %v", *cfg.Concurrency)
	}
	if cfg.Timeout != nil {
		t.Errorf("Timeout: expected nil, got %v", *cfg.Timeout)
	}
	if cfg.Method != nil {
		t.Errorf("Method: expected nil, got %v", *cfg.Method)
	}
}

func TestLoadConfig_MalformedJSON(t *testing.T) {
	path := writeTempFile(t, "bad.json", `{invalid json}`)

	_, err := loadConfig(path)
	if err == nil {
		t.Fatal("expected error for malformed JSON, got nil")
	}
}

func TestLoadConfig_FileNotFound(t *testing.T) {
	_, err := loadConfig("/nonexistent/path/config.json")
	if err == nil {
		t.Fatal("expected error for missing file, got nil")
	}
}

func TestLoadConfig_UnsupportedExtension(t *testing.T) {
	path := writeTempFile(t, "config.txt", `{"target": "https://example.com"}`)

	_, err := loadConfig(path)
	if err == nil {
		t.Fatal("expected error for unsupported extension, got nil")
	}
}

func TestLoadConfig_EmptyFile(t *testing.T) {
	path := writeTempFile(t, "empty.json", "")

	cfg, err := loadConfig(path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.Target != nil {
		t.Errorf("expected nil Target for empty file, got %v", *cfg.Target)
	}
}

func TestLoadConfig_ValidYAML_AllFields(t *testing.T) {
	content := `target: https://example.com
requests: 100
concurrency: 10
timeout: "5s"
duration: "30s"
method: POST
body: '{"key":"value"}'
headers:
  - "Content-Type: application/json"
  - "Authorization: Bearer token"
`
	path := writeTempFile(t, "config.yaml", content)

	cfg, err := loadConfig(path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if cfg.Target == nil || *cfg.Target != "https://example.com" {
		t.Errorf("Target: got %v, want %q", cfg.Target, "https://example.com")
	}
	if cfg.Requests == nil || *cfg.Requests != 100 {
		t.Errorf("Requests: got %v, want 100", cfg.Requests)
	}
	if cfg.Concurrency == nil || *cfg.Concurrency != 10 {
		t.Errorf("Concurrency: got %v, want 10", cfg.Concurrency)
	}
	if cfg.Timeout == nil || *cfg.Timeout != "5s" {
		t.Errorf("Timeout: got %v, want %q", cfg.Timeout, "5s")
	}
	if cfg.Duration == nil || *cfg.Duration != "30s" {
		t.Errorf("Duration: got %v, want %q", cfg.Duration, "30s")
	}
	if cfg.Method == nil || *cfg.Method != "POST" {
		t.Errorf("Method: got %v, want %q", cfg.Method, "POST")
	}
	if cfg.Body == nil || *cfg.Body != `{"key":"value"}` {
		t.Errorf("Body: got %v, want %q", cfg.Body, `{"key":"value"}`)
	}
	if len(cfg.Headers) != 2 {
		t.Fatalf("Headers: got %d items, want 2", len(cfg.Headers))
	}
}

func TestLoadConfig_ValidYAML_PartialFields(t *testing.T) {
	content := `target: https://example.com
requests: 50
`
	path := writeTempFile(t, "partial.yaml", content)

	cfg, err := loadConfig(path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if cfg.Target == nil || *cfg.Target != "https://example.com" {
		t.Errorf("Target: got %v, want %q", cfg.Target, "https://example.com")
	}
	if cfg.Requests == nil || *cfg.Requests != 50 {
		t.Errorf("Requests: got %v, want 50", cfg.Requests)
	}
	if cfg.Concurrency != nil {
		t.Errorf("Concurrency: expected nil, got %v", *cfg.Concurrency)
	}
}

func TestLoadConfig_MalformedYAML(t *testing.T) {
	path := writeTempFile(t, "bad.yaml", ":\n  :\n    - :\n  invalid: [")

	_, err := loadConfig(path)
	if err == nil {
		t.Fatal("expected error for malformed YAML, got nil")
	}
}

func TestLoadConfig_YMLExtension(t *testing.T) {
	content := `target: https://example.com
requests: 25
`
	path := writeTempFile(t, "config.yml", content)

	cfg, err := loadConfig(path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if cfg.Target == nil || *cfg.Target != "https://example.com" {
		t.Errorf("Target: got %v, want %q", cfg.Target, "https://example.com")
	}
	if cfg.Requests == nil || *cfg.Requests != 25 {
		t.Errorf("Requests: got %v, want 25", cfg.Requests)
	}
}

func TestLoadConfig_CaseInsensitiveExtension(t *testing.T) {
	content := `{"target": "https://example.com"}`
	path := writeTempFile(t, "config.JSON", content)

	cfg, err := loadConfig(path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if cfg.Target == nil || *cfg.Target != "https://example.com" {
		t.Errorf("Target: got %v, want %q", cfg.Target, "https://example.com")
	}
}

func TestLoadConfig_EmptyYAMLFile(t *testing.T) {
	path := writeTempFile(t, "empty.yaml", "")

	cfg, err := loadConfig(path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.Target != nil {
		t.Errorf("expected nil Target for empty YAML file, got %v", *cfg.Target)
	}
}
