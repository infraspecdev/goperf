package cmd

import (
	"encoding/json"
	"fmt"
	"io"
	"sort"
	"time"

	"github.com/infraspecdev/goperf/internal/stats"
)

type result struct {
	Target      string
	Elapsed     time.Duration
	Total       int64
	Succeeded   int64
	Failed      int64
	Min         time.Duration
	Max         time.Duration
	Avg         time.Duration
	P50         time.Duration
	P90         time.Duration
	P99         time.Duration
	Throughput  float64
	StatusCodes map[int]int64
	Errors      map[string]int64
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
		Target:      target,
		Elapsed:     elapsed,
		Total:       total,
		Succeeded:   succeeded,
		Failed:      recorder.FailedCount(),
		Throughput:  throughput,
		StatusCodes: recorder.StatusCodes(),
		Errors:      recorder.Errors(),
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

	if len(r.StatusCodes) > 0 {
		if _, err = fmt.Fprintf(w, "\nStatus code distribution:\n"); err != nil {
			return err
		}
		var codes []int
		for c := range r.StatusCodes {
			codes = append(codes, c)
		}
		sort.Ints(codes)
		for _, code := range codes {
			if _, err = fmt.Fprintf(w, "  [%d] %d responses\n", code, r.StatusCodes[code]); err != nil {
				return err
			}
		}
	}

	if len(r.Errors) > 0 {
		if _, err = fmt.Fprintf(w, "\nError distribution:\n"); err != nil {
			return err
		}
		var errMsgs []string
		for e := range r.Errors {
			errMsgs = append(errMsgs, e)
		}
		sort.Strings(errMsgs)
		for _, errMsg := range errMsgs {
			if _, err = fmt.Fprintf(w, "  [%d] %s\n", r.Errors[errMsg], errMsg); err != nil {
				return err
			}
		}
	}

	if _, err = fmt.Fprintf(w, "\n"); err != nil {
		return err
	}

	if r.Succeeded == 0 {
		_, err = fmt.Fprintf(w, "Throughput: %.1f requests/sec\n", r.Throughput)
		return err
	}

	_, err = fmt.Fprintf(w, `Latency:
  Fastest:  %dms
  Slowest:  %dms
  Average:  %dms
  p50:      %dms
  p90:      %dms
  p99:      %dms

Throughput: %.1f requests/sec
`,
		r.Min.Milliseconds(), r.Max.Milliseconds(), r.Avg.Milliseconds(),
		r.P50.Milliseconds(), r.P90.Milliseconds(), r.P99.Milliseconds(),
		r.Throughput)
	return err
}

func (r *result) WriteJSON(w io.Writer) error {
	output := struct {
		Target      string           `json:"target"`
		ElapsedSec  float64          `json:"elapsed_sec"`
		Total       int64            `json:"total"`
		Succeeded   int64            `json:"succeeded"`
		Failed      int64            `json:"failed"`
		MinMs       float64          `json:"min_ms"`
		MaxMs       float64          `json:"max_ms"`
		AvgMs       float64          `json:"avg_ms"`
		P50Ms       float64          `json:"p50_ms"`
		P90Ms       float64          `json:"p90_ms"`
		P99Ms       float64          `json:"p99_ms"`
		Throughput  float64          `json:"throughput"`
		StatusCodes map[string]int64 `json:"status_codes,omitempty"`
		Errors      map[string]int64 `json:"errors,omitempty"`
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

	if len(r.StatusCodes) > 0 {
		output.StatusCodes = make(map[string]int64)
		for k, v := range r.StatusCodes {
			output.StatusCodes[fmt.Sprintf("%d", k)] = v
		}
	}
	if len(r.Errors) > 0 {
		output.Errors = r.Errors
	}

	encoder := json.NewEncoder(w)
	encoder.SetIndent("", "  ")
	return encoder.Encode(output)
}
