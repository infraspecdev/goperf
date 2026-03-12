package cmd

import (
	"bytes"
	"strings"
	"testing"
	"time"

	"github.com/infraspecdev/goperf/internal/stats"
)

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

func TestConcurrencyFlagExists(t *testing.T) {
	cmd := runCmd
	flag := cmd.Flags().Lookup("concurrency")

	if flag == nil {
		t.Fatal("expected concurrency flag to exist")
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
