package stats

import (
	"math"
	"sync"
	"time"

	hdrhistogram "github.com/HdrHistogram/hdrhistogram-go"
)

type HistogramRecorder struct {
	mu        sync.RWMutex
	histogram *hdrhistogram.Histogram
	failed    int64
	count     int64
	sum       int64
	min       int64
	max       int64
}

func NewHistogramRecorder(timeout time.Duration) *HistogramRecorder {
	return &HistogramRecorder{
		histogram: hdrhistogram.New(1, timeout.Nanoseconds()*10, 3),
		min:       math.MaxInt64,
	}
}

func (h *HistogramRecorder) Record(d time.Duration) {
	h.mu.Lock()
	defer h.mu.Unlock()
	ns := d.Nanoseconds()
	if ns < 1 {
		ns = 1
	}
	err := h.histogram.RecordValue(ns)
	if err != nil {
		h.failed++
		return
	}
	h.count++
	h.sum += ns
	if ns < h.min {
		h.min = ns
	}
	if ns > h.max {
		h.max = ns
	}
}

func (h *HistogramRecorder) RecordFailure() {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.failed++
}

func (h *HistogramRecorder) Count() int64 {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return h.count
}

func (h *HistogramRecorder) FailedCount() int64 {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return h.failed
}

func (h *HistogramRecorder) TotalRequests() int64 {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return h.count + h.failed
}

func (h *HistogramRecorder) Min() time.Duration {
	h.mu.RLock()
	defer h.mu.RUnlock()
	if h.count == 0 {
		return 0
	}
	return time.Duration(h.min) * time.Nanosecond
}

func (h *HistogramRecorder) Max() time.Duration {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return time.Duration(h.max) * time.Nanosecond
}

func (h *HistogramRecorder) Avg() time.Duration {
	h.mu.RLock()
	defer h.mu.RUnlock()
	if h.count == 0 {
		return 0
	}
	return time.Duration(h.sum/h.count) * time.Nanosecond
}

func (h *HistogramRecorder) Percentile(p float64) time.Duration {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return time.Duration(h.histogram.ValueAtQuantile(p)) * time.Nanosecond
}
