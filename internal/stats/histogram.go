package stats

import (
	"time"

	hdrhistogram "github.com/HdrHistogram/hdrhistogram-go"
)

type HistogramRecorder struct {
	histogram *hdrhistogram.Histogram
}

func NewHistogramRecorder(timeout time.Duration) *HistogramRecorder {
	return &HistogramRecorder{
		histogram: hdrhistogram.New(1000, timeout.Nanoseconds(), 3),
	}
}

func (h *HistogramRecorder) Record(d time.Duration) {
	_ = h.histogram.RecordValue(d.Nanoseconds())
}

func (h *HistogramRecorder) Count() int64 {
	return h.histogram.TotalCount()
}

func (h *HistogramRecorder) Min() time.Duration {
	return time.Duration(h.histogram.Min()) * time.Nanosecond
}
