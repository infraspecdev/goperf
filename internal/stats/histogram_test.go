package stats

import (
	"testing"
	"time"
)

func TestNewHistogramRecorder(t *testing.T) {
	timeout := 10 * time.Second
	recorder := NewHistogramRecorder(timeout)

	if recorder == nil {
		t.Fatal("expected non-nil HistogramRecorder")
	}
}

func TestHistogramRecorder_RecordSingle(t *testing.T) {
	recorder := NewHistogramRecorder(10 * time.Second)

	recorder.Record(5 * time.Millisecond)

	if recorder.Count() != 1 {
		t.Errorf("expected count 1, got %d", recorder.Count())
	}
}

func TestHistogramRecorder_Min(t *testing.T) {
	recorder := NewHistogramRecorder(10 * time.Second)

	recorder.Record(10 * time.Millisecond)
	recorder.Record(20 * time.Millisecond)
	recorder.Record(30 * time.Millisecond)

	min := recorder.Min()
	expected := 10 * time.Millisecond

	if min < expected-time.Millisecond || min > expected+time.Millisecond {
		t.Errorf("expected min ~%v, got %v", expected, min)
	}
}

func TestHistogramRecorder_Max(t *testing.T) {
	recorder := NewHistogramRecorder(10 * time.Second)

	recorder.Record(10 * time.Millisecond)
	recorder.Record(20 * time.Millisecond)
	recorder.Record(30 * time.Millisecond)

	max := recorder.Max()
	expected := 30 * time.Millisecond

	if max < expected-time.Millisecond || max > expected+time.Millisecond {
		t.Errorf("expected max ~%v, got %v", expected, max)
	}
}

func TestHistogramRecorder_Avg(t *testing.T) {
	recorder := NewHistogramRecorder(10 * time.Second)

	recorder.Record(10 * time.Millisecond)
	recorder.Record(20 * time.Millisecond)
	recorder.Record(30 * time.Millisecond)

	avg := recorder.Avg()
	expected := 20 * time.Millisecond

	if avg < expected-time.Millisecond || avg > expected+time.Millisecond {
		t.Errorf("expected avg ~%v, got %v", expected, avg)
	}
}

func TestHistogramRecorder_Percentiles(t *testing.T) {
	recorder := NewHistogramRecorder(10 * time.Second)

	for i := 1; i <= 100; i++ {
		recorder.Record(time.Duration(i) * time.Millisecond)
	}

	tests := []struct {
		name       string
		percentile float64
		expected   time.Duration
	}{
		{"P50", 50, 50 * time.Millisecond},
		{"P90", 90, 90 * time.Millisecond},
		{"P99", 99, 99 * time.Millisecond},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := recorder.Percentile(tt.percentile)

			if got < tt.expected-2*time.Millisecond || got > tt.expected+2*time.Millisecond {
				t.Errorf("expected %s ~%v, got %v", tt.name, tt.expected, got)
			}
		})
	}
}

func TestHistogramRecorder_RecordFailure(t *testing.T) {
	recorder := NewHistogramRecorder(10 * time.Second)

	recorder.RecordFailure()
	recorder.RecordFailure()

	if recorder.FailedCount() != 2 {
		t.Errorf("expected 2 failed requests, got %d", recorder.FailedCount())
	}
}

func TestHistogramRecorder_TotalRequests(t *testing.T) {
	recorder := NewHistogramRecorder(10 * time.Second)

	recorder.Record(5 * time.Millisecond)
	recorder.Record(10 * time.Millisecond)
	recorder.RecordFailure()
	recorder.RecordFailure()

	if recorder.TotalRequests() != 4 {
		t.Errorf("expected 4 total requests, got %d", recorder.TotalRequests())
	}
	if recorder.Count() != 2 {
		t.Errorf("expected 2 successful requests, got %d", recorder.Count())
	}
	if recorder.FailedCount() != 2 {
		t.Errorf("expected 2 failed requests, got %d", recorder.FailedCount())
	}
}

func TestHistogramRecorder_NearTimeoutValue(t *testing.T) {
	timeout := 100 * time.Millisecond
	recorder := NewHistogramRecorder(timeout)
	recorder.Record(105 * time.Millisecond)
	if recorder.Count() != 1 {
		t.Errorf("expected value near timeout to be recorded, got count %d", recorder.Count())
	}
	if recorder.FailedCount() != 0 {
		t.Errorf("expected 0 failures, got %d", recorder.FailedCount())
	}
}

func BenchmarkRecord(b *testing.B) {
	recorder := NewHistogramRecorder(10 * time.Second)
	d := 5 * time.Millisecond

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		recorder.Record(d)
	}
}

func TestHistogramRecorder_RecordStatusCode(t *testing.T) {
	recorder := NewHistogramRecorder(10 * time.Second)
	recorder.RecordStatusCode(200)
	recorder.RecordStatusCode(200)
	recorder.RecordStatusCode(500)

	codes := recorder.StatusCodes()
	if codes[200] != 2 {
		t.Errorf("expected 2 times 200, got %d", codes[200])
	}
	if codes[500] != 1 {
		t.Errorf("expected 1 times 500, got %d", codes[500])
	}
}

func TestHistogramRecorder_RecordErrorResult(t *testing.T) {
	recorder := NewHistogramRecorder(10 * time.Second)
	recorder.RecordErrorResult(0, "timeout")
	recorder.RecordErrorResult(500, "connection refused")
	recorder.RecordErrorResult(429, "timeout")

	errs := recorder.Errors()
	if errs["timeout"] != 2 {
		t.Errorf("expected 2 times timeout, got %d", errs["timeout"])
	}
	if errs["connection refused"] != 1 {
		t.Errorf("expected 1 times connection refused, got %d", errs["connection refused"])
	}
	if recorder.FailedCount() != 3 {
		t.Errorf("expected 3 failures, got %d", recorder.FailedCount())
	}
	codes := recorder.StatusCodes()
	if codes[500] != 1 {
		t.Errorf("expected 1 times 500, got %d", codes[500])
	}
	if codes[429] != 1 {
		t.Errorf("expected 1 times 429, got %d", codes[429])
	}
}

func TestHistogramRecorder_Distribution(t *testing.T) {
	recorder := NewHistogramRecorder(10 * time.Second)
	recorder.Record(10 * time.Millisecond)
	recorder.Record(20 * time.Millisecond)

	dist := recorder.Distribution()
	if len(dist) == 0 {
		t.Fatal("expected non-empty distribution")
	}

	var totalCount int64
	for _, b := range dist {
		if b.Count == 0 {
			t.Errorf("expected no zero-count buckets, got one: %+v", b)
		}
		totalCount += b.Count
	}

	if totalCount != 2 {
		t.Errorf("expected total count 2 in distribution, got %d", totalCount)
	}
}
