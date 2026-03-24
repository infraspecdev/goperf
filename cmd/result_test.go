package cmd

import (
	"bytes"
	"encoding/json"
	"math"
	"testing"
	"time"

	"github.com/infraspecdev/goperf/internal/stats"
)

func newTestRecorder(latencies ...time.Duration) *stats.HistogramRecorder {
	r := stats.NewHistogramRecorder(10 * time.Second)
	for _, d := range latencies {
		r.Record(d)
	}
	return r
}

func TestNewResult(t *testing.T) {
	recorder := newTestRecorder(10*time.Millisecond, 20*time.Millisecond, 30*time.Millisecond)

	r := newResult(recorder, "http://example.com", 1*time.Second)

	if r.Target != "http://example.com" {
		t.Errorf("expected Target %q, got %q", "http://example.com", r.Target)
	}
	if r.Elapsed != 1*time.Second {
		t.Errorf("expected Elapsed %v, got %v", 1*time.Second, r.Elapsed)
	}
	if r.Total != 3 {
		t.Errorf("expected Total 3, got %d", r.Total)
	}
	if r.Succeeded != 3 {
		t.Errorf("expected Succeeded 3, got %d", r.Succeeded)
	}
	if r.Failed != 0 {
		t.Errorf("expected Failed 0, got %d", r.Failed)
	}
	if r.Min != 10*time.Millisecond {
		t.Errorf("expected Min 10ms, got %v", r.Min)
	}
	if r.Max != 30*time.Millisecond {
		t.Errorf("expected Max 30ms, got %v", r.Max)
	}
	if r.Avg.Milliseconds() != 20 {
		t.Errorf("expected Avg ~20ms, got %v", r.Avg)
	}
	if r.P50.Milliseconds() != 20 {
		t.Errorf("expected P50 ~20ms, got %v", r.P50)
	}
	if r.P90.Milliseconds() != 30 {
		t.Errorf("expected P90 ~30ms, got %v", r.P90)
	}
	if r.P99.Milliseconds() != 30 {
		t.Errorf("expected P99 ~30ms, got %v", r.P99)
	}
	if math.Abs(r.Throughput-3.0) > 1e-9 {
		t.Errorf("expected Throughput ~3.0, got %f", r.Throughput)
	}
}

func TestNewResult_WithFailures(t *testing.T) {
	recorder := newTestRecorder(10 * time.Millisecond)
	recorder.RecordFailure()
	recorder.RecordFailure()

	r := newResult(recorder, "http://example.com", 1*time.Second)

	if r.Total != 3 {
		t.Errorf("expected Total 3, got %d", r.Total)
	}
	if r.Succeeded != 1 {
		t.Errorf("expected Succeeded 1, got %d", r.Succeeded)
	}
	if r.Failed != 2 {
		t.Errorf("expected Failed 2, got %d", r.Failed)
	}
}

func TestNewResult_AllFailed(t *testing.T) {
	recorder := stats.NewHistogramRecorder(10 * time.Second)
	recorder.RecordFailure()
	recorder.RecordFailure()

	r := newResult(recorder, "http://example.com", 1*time.Second)

	if r.Total != 2 {
		t.Errorf("expected Total 2, got %d", r.Total)
	}
	if r.Succeeded != 0 {
		t.Errorf("expected Succeeded 0, got %d", r.Succeeded)
	}
	if r.Min != 0 {
		t.Errorf("expected Min 0 when no successes, got %v", r.Min)
	}
	if r.Max != 0 {
		t.Errorf("expected Max 0 when no successes, got %v", r.Max)
	}
}

func TestNewResult_ZeroElapsed(t *testing.T) {
	recorder := newTestRecorder(10 * time.Millisecond)

	r := newResult(recorder, "http://example.com", 0)

	if math.Abs(r.Throughput) > 1e-9 {
		t.Errorf("expected Throughput ~0.0 for zero elapsed, got %f", r.Throughput)
	}
}

func TestNewResult_NilRecorder(t *testing.T) {
	r := newResult(nil, "http://example.com", 1*time.Second)

	if r.Target != "http://example.com" {
		t.Errorf("expected Target %q, got %q", "http://example.com", r.Target)
	}
	if r.Elapsed != 1*time.Second {
		t.Errorf("expected Elapsed %v, got %v", 1*time.Second, r.Elapsed)
	}
	if r.Total != 0 || r.Succeeded != 0 || r.Failed != 0 {
		t.Errorf("expected zero counts, got total=%d succ=%d fail=%d", r.Total, r.Succeeded, r.Failed)
	}
}

func TestResultWriteText(t *testing.T) {
	r := &result{
		Target:     "http://example.com",
		Elapsed:    1 * time.Second,
		Total:      3,
		Succeeded:  3,
		Failed:     0,
		Min:        10 * time.Millisecond,
		Max:        30 * time.Millisecond,
		Avg:        20 * time.Millisecond,
		P50:        20 * time.Millisecond,
		P90:        30 * time.Millisecond,
		P99:        30 * time.Millisecond,
		Throughput: 3.0,
	}

	var buf bytes.Buffer
	err := r.WriteText(&buf)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	output := buf.String()

	expected := `
Target:     http://example.com
Duration:   1.000s
Requests:   3 total (3 succeeded, 0 failed)

Latency:
  Fastest:  10.00ms
  Slowest:  30.00ms
  Average:  20.00ms
  p50:      20.00ms
  p90:      30.00ms
  p99:      30.00ms

Throughput: 3.0 requests/sec
`
	if output != expected {
		t.Errorf("expected output:\n%s\n\ngot:\n%s\n", expected, output)
	}
}

func TestResultWriteText_SubMillisecond(t *testing.T) {
	r := &result{
		Target:     "http://example.com",
		Elapsed:    1 * time.Second,
		Total:      1,
		Succeeded:  1,
		Failed:     0,
		Min:        500 * time.Microsecond,
		Max:        500 * time.Microsecond,
		Avg:        500 * time.Microsecond,
		P50:        500 * time.Microsecond,
		P90:        500 * time.Microsecond,
		P99:        500 * time.Microsecond,
		Throughput: 1.0,
	}

	var buf bytes.Buffer
	err := r.WriteText(&buf)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	output := buf.String()

	expected := `
Target:     http://example.com
Duration:   1.000s
Requests:   1 total (1 succeeded, 0 failed)

Latency:
  Fastest:  0.50ms
  Slowest:  0.50ms
  Average:  0.50ms
  p50:      0.50ms
  p90:      0.50ms
  p99:      0.50ms

Throughput: 1.0 requests/sec
`
	if output != expected {
		t.Errorf("expected output:\n%s\n\ngot:\n%s\n", expected, output)
	}
}

func TestResultWriteText_AllFailed(t *testing.T) {
	recorder := stats.NewHistogramRecorder(10 * time.Second)
	recorder.RecordFailure()

	r := newResult(recorder, "http://example.com", 1*time.Second)

	var buf bytes.Buffer
	err := r.WriteText(&buf)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	output := buf.String()
	expected := `
Target:     http://example.com
Duration:   1.000s
Requests:   1 total (0 succeeded, 1 failed)

Throughput: 1.0 requests/sec
`

	if output != expected {
		t.Errorf("expected output:\n%q\n\ngot:\n%q\n", expected, output)
	}
}

func TestResultWriteJSON_Fields(t *testing.T) {
	r := &result{
		Target:     "http://test.com",
		Elapsed:    2500 * time.Millisecond,
		Total:      10,
		Succeeded:  8,
		Failed:     2,
		Min:        10 * time.Millisecond,
		Max:        100 * time.Millisecond,
		Avg:        50 * time.Millisecond,
		P50:        45 * time.Millisecond,
		P90:        80 * time.Millisecond,
		P99:        95 * time.Millisecond,
		Throughput: 125.5,
	}

	var buf bytes.Buffer
	if err := r.WriteJSON(&buf); err != nil {
		t.Fatalf("WriteJSON failed: %v", err)
	}

	var output map[string]interface{}
	if err := json.Unmarshal(buf.Bytes(), &output); err != nil {
		t.Fatalf("failed to parse JSON: %v", err)
	}

	tests := []struct {
		key  string
		want interface{}
	}{
		{"target", "http://test.com"},
		{"elapsed_sec", 2.5},
		{"total", 10.0},
		{"succeeded", 8.0},
		{"failed", 2.0},
		{"min_ms", 10.0},
		{"max_ms", 100.0},
		{"avg_ms", 50.0},
		{"p50_ms", 45.0},
		{"p90_ms", 80.0},
		{"p99_ms", 95.0},
		{"throughput", 125.5},
	}

	for _, tt := range tests {
		t.Run(tt.key, func(t *testing.T) {
			got := output[tt.key]
			if got != tt.want {
				t.Errorf("field %q: expected %v, got %v", tt.key, tt.want, got)
			}
		})
	}
}
