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
