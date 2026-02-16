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
