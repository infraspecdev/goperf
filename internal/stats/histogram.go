package stats

import (
	"sync"
	"time"

	hdrhistogram "github.com/HdrHistogram/hdrhistogram-go"
)

type HistogramRecorder struct {
	mu        sync.RWMutex
	histogram *hdrhistogram.Histogram
	failed    int64
}

func NewHistogramRecorder(timeout time.Duration) *HistogramRecorder {
	return &HistogramRecorder{
		histogram: hdrhistogram.New(1000, timeout.Nanoseconds(), 3),
	}
}

func (h *HistogramRecorder) Record(d time.Duration) {
	h.mu.Lock()
	defer h.mu.Unlock()
	err := h.histogram.RecordValue(d.Nanoseconds())
	if err != nil {
		h.failed++
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
	return h.histogram.TotalCount()
}

func (h *HistogramRecorder) FailedCount() int64 {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return h.failed
}

func (h *HistogramRecorder) TotalRequests() int64 {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return h.histogram.TotalCount() + h.failed
}

func (h *HistogramRecorder) Min() time.Duration {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return time.Duration(h.histogram.Min()) * time.Nanosecond
}

func (h *HistogramRecorder) Max() time.Duration {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return time.Duration(h.histogram.Max()) * time.Nanosecond
}

func (h *HistogramRecorder) Avg() time.Duration {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return time.Duration(h.histogram.Mean()) * time.Nanosecond
}

func (h *HistogramRecorder) Percentile(p float64) time.Duration {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return time.Duration(h.histogram.ValueAtQuantile(p)) * time.Nanosecond
}
