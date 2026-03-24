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
	Target       string
	Elapsed      time.Duration
	Total        int64
	Succeeded    int64
	Failed       int64
	Min          time.Duration
	Max          time.Duration
	Avg          time.Duration
	P50          time.Duration
	P90          time.Duration
	P99          time.Duration
	Throughput   float64
	StatusCodes  map[int]int64
	Errors       map[string]int64
	Distribution []stats.DistributionBar
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
		r.Distribution = recorder.Distribution()
	}

	return r
}

func (r *result) WriteText(w io.Writer) error {
	_, err := fmt.Fprintf(w, `
Target:     %s
Duration:   %.3fs
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
  Fastest:  %.2fms
  Slowest:  %.2fms
  Average:  %.2fms
  p50:      %.2fms
  p90:      %.2fms
  p99:      %.2fms
`,
		float64(r.Min)/float64(time.Millisecond),
		float64(r.Max)/float64(time.Millisecond),
		float64(r.Avg)/float64(time.Millisecond),
		float64(r.P50)/float64(time.Millisecond),
		float64(r.P90)/float64(time.Millisecond),
		float64(r.P99)/float64(time.Millisecond))
	if err != nil {
		return err
	}

	if len(r.Distribution) > 0 {
		if _, err = fmt.Fprintf(w, "Response time histogram:\n"); err != nil {
			return err
		}

		minMs := float64(r.Min) / float64(time.Millisecond)
		maxMs := float64(r.Max) / float64(time.Millisecond)
		const numBuckets = 10
		bucketWidth := (maxMs - minMs) / numBuckets
		if bucketWidth <= 0 {
			bucketWidth = 0.1
		}

		buckets := make([]int64, numBuckets)

		for _, b := range r.Distribution {
			startIdx := int((b.FromMs - minMs) / bucketWidth)
			endIdx := int((b.ToMs - minMs) / bucketWidth)

			if startIdx < 0 {
				startIdx = 0
			}
			if endIdx >= numBuckets {
				endIdx = numBuckets - 1
			}

			span := endIdx - startIdx + 1
			if span <= 0 {
				span = 1
			}

			portion := b.Count / int64(span)
			remainder := b.Count % int64(span)

			for i := startIdx; i <= endIdx; i++ {
				buckets[i] += portion
				if remainder > 0 {
					buckets[i]++
					remainder--
				}
			}
		}

		var maxCount int64
		for _, c := range buckets {
			if c > maxCount {
				maxCount = c
			}
		}

		const maxBarWidth = 40
		for i, count := range buckets {
			if count == 0 {
				continue
			}
			edge := minMs + float64(i)*bucketWidth
			var barLen int
			if maxCount > 0 {
				barLen = int(float64(count) / float64(maxCount) * maxBarWidth)
			}
			bar := ""
			for j := 0; j < barLen; j++ {
				bar += "■"
			}
			if _, err = fmt.Fprintf(w, "  %.3f [%d]\t|%s\n", edge, count, bar); err != nil {
				return err
			}
		}
		if _, err = fmt.Fprintf(w, "\n"); err != nil {
			return err
		}
	}

	_, err = fmt.Fprintf(w, "\nThroughput: %.1f requests/sec\n", r.Throughput)
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
