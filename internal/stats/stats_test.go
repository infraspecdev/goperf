package stats

import (
	"testing"
	"time"
)

func TestMinResponseTime_NormalCase(t *testing.T) {
	input := []time.Duration{
		30 * time.Millisecond,
		10 * time.Millisecond,
		20 * time.Millisecond,
	}

	min := MinResponseTime(input)

	if min != 10*time.Millisecond {
		t.Errorf("expected min 10ms, got %v", min)
	}
}

func TestMinResponseTime_SingleValue(t *testing.T) {
	input := []time.Duration{
		15 * time.Millisecond,
	}

	min := MinResponseTime(input)

	if min != 15*time.Millisecond {
		t.Errorf("expected min 15ms, got %v", min)
	}
}

func TestMinResponseTime_EmptySlice(t *testing.T) {
	var input []time.Duration

	min := MinResponseTime(input)

	if min != 0 {
		t.Errorf("expected min 0 for empty slice, got %v", min)
	}
}

func TestMaxResponseTime_NormalCase(t *testing.T) {
	input := []time.Duration{
		30 * time.Millisecond,
		10 * time.Millisecond,
		20 * time.Millisecond,
	}

	max := MaxResponseTime(input)

	if max != 30*time.Millisecond {
		t.Errorf("expected max 30ms, got %v", max)
	}
}

func TestMaxResponseTime_SingleValue(t *testing.T) {
	input := []time.Duration{
		15 * time.Millisecond,
	}

	max := MaxResponseTime(input)

	if max != 15*time.Millisecond {
		t.Errorf("expected max 15ms, got %v", max)
	}
}

func TestMaxResponseTime_EmptySlice(t *testing.T) {
	var input []time.Duration

	max := MaxResponseTime(input)

	if max != 0 {
		t.Errorf("expected max 0 for empty slice, got %v", max)
	}
}

func TestAverageResponseTime_NormalCase(t *testing.T) {
	input := []time.Duration{
		10 * time.Millisecond,
		20 * time.Millisecond,
		30 * time.Millisecond,
	}

	avg := AverageResponseTime(input)

	if avg != 20*time.Millisecond {
		t.Errorf("expected avg 20ms, got %v", avg)
	}
}

func TestAverageResponseTime_SingleValue(t *testing.T) {
	input := []time.Duration{
		25 * time.Millisecond,
	}

	avg := AverageResponseTime(input)

	if avg != 25*time.Millisecond {
		t.Errorf("expected avg 25ms, got %v", avg)
	}
}

func TestAverageResponseTime_EmptySlice(t *testing.T) {
	var input []time.Duration

	avg := AverageResponseTime(input)

	if avg != 0 {
		t.Errorf("expected avg 0 for empty slice, got %v", avg)
	}
}
