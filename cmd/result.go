package cmd

import (
	"encoding/json"
	"fmt"
	"io"
	"time"

	"github.com/infraspecdev/goperf/internal/stats"
)

type result struct {
	Target     string
	Elapsed    time.Duration
	Total      int64
	Succeeded  int64
	Failed     int64
	Min        time.Duration
	Max        time.Duration
	Avg        time.Duration
	P50        time.Duration
	P90        time.Duration
	P99        time.Duration
	Throughput float64
}

func newResult(recorder *stats.HistogramRecorder, target string, elapsed time.Duration) *result {
	if recorder == nil {
		return &result{
			Target:  target,
			Elapsed: elapsed,
		}
	}

	total := recorder.TotalRequests()
	succeeded := recorder.Count()

	throughput := 0.0
	if elapsed.Seconds() > 0 {
		throughput = float64(total) / elapsed.Seconds()
	}

	r := &result{
		Target:     target,
		Elapsed:    elapsed,
		Total:      total,
		Succeeded:  succeeded,
		Failed:     recorder.FailedCount(),
		Throughput: throughput,
	}

	if succeeded > 0 {
		r.Min = recorder.Min()
		r.Max = recorder.Max()
		r.Avg = recorder.Avg()
		r.P50 = recorder.Percentile(50)
		r.P90 = recorder.Percentile(90)
		r.P99 = recorder.Percentile(99)
	}

	return r
}

func (r *result) WriteText(w io.Writer) error {
	_, err := fmt.Fprintf(w, `
Target:     %s
Duration:   %.1fs
Requests:   %d total (%d succeeded, %d failed)

`, r.Target, r.Elapsed.Seconds(), r.Total, r.Succeeded, r.Failed)
	if err != nil {
		return err
	}

	if r.Succeeded == 0 {
		_, err = fmt.Fprintf(w, "Throughput: %.1f requests/sec\n", r.Throughput)
		return err
	}

	_, err = fmt.Fprintf(w, `Latency:
  Fastest:  %.2fms
  Slowest:  %.2fms
  Average:  %.2fms
  p50:      %.2fms
  p90:      %.2fms
  p99:      %.2fms

Throughput: %.1f requests/sec
`,
		float64(r.Min)/float64(time.Millisecond),
		float64(r.Max)/float64(time.Millisecond),
		float64(r.Avg)/float64(time.Millisecond),
		float64(r.P50)/float64(time.Millisecond),
		float64(r.P90)/float64(time.Millisecond),
		float64(r.P99)/float64(time.Millisecond),
		r.Throughput)
	return err
}

func (r *result) WriteJSON(w io.Writer) error {
	output := struct {
		Target     string  `json:"target"`
		ElapsedSec float64 `json:"elapsed_sec"`
		Total      int64   `json:"total"`
		Succeeded  int64   `json:"succeeded"`
		Failed     int64   `json:"failed"`
		MinMs      float64 `json:"min_ms"`
		MaxMs      float64 `json:"max_ms"`
		AvgMs      float64 `json:"avg_ms"`
		P50Ms      float64 `json:"p50_ms"`
		P90Ms      float64 `json:"p90_ms"`
		P99Ms      float64 `json:"p99_ms"`
		Throughput float64 `json:"throughput"`
	}{
		Target:     r.Target,
		ElapsedSec: r.Elapsed.Seconds(),
		Total:      r.Total,
		Succeeded:  r.Succeeded,
		Failed:     r.Failed,
		MinMs:      float64(r.Min) / float64(time.Millisecond),
		MaxMs:      float64(r.Max) / float64(time.Millisecond),
		AvgMs:      float64(r.Avg) / float64(time.Millisecond),
		P50Ms:      float64(r.P50) / float64(time.Millisecond),
		P90Ms:      float64(r.P90) / float64(time.Millisecond),
		P99Ms:      float64(r.P99) / float64(time.Millisecond),
		Throughput: r.Throughput,
	}

	encoder := json.NewEncoder(w)
	encoder.SetIndent("", "  ")
	return encoder.Encode(output)
}
