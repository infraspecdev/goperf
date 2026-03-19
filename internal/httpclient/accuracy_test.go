package httpclient

import (
	"context"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"
)

func TestAggregate_UniformLatencyAccuracy(t *testing.T) {
	const (
		delay       = 50 * time.Millisecond
		requests    = 20
		concurrency = 4
	)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(delay)
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	cfg := Config{
		Target:      server.URL,
		Requests:    requests,
		Concurrency: concurrency,
		Timeout:     5 * time.Second,
		Method:      "GET",
	}

	recorder := RunMultipleConcurrent(context.Background(), cfg)

	if recorder.Count() != requests {
		t.Fatalf("expected %d successful requests, got %d", requests, recorder.Count())
	}
	if recorder.FailedCount() != 0 {
		t.Fatalf("expected 0 failures, got %d", recorder.FailedCount())
	}

	lowerBound := delay * 8 / 10
	upperAvg := delay * 3
	upperP99 := delay * 4

	if avg := recorder.Avg(); avg < lowerBound || avg > upperAvg {
		t.Errorf("Avg() = %v; want between %v and %v", avg, lowerBound, upperAvg)
	}
	if min := recorder.Min(); min < lowerBound {
		t.Errorf("Min() = %v; want >= %v", min, lowerBound)
	}
	if p50 := recorder.Percentile(50); p50 < lowerBound || p50 > upperAvg {
		t.Errorf("P50() = %v; want between %v and %v", p50, lowerBound, upperAvg)
	}
	if p99 := recorder.Percentile(99); p99 < lowerBound || p99 > upperP99 {
		t.Errorf("P99() = %v; want between %v and %v", p99, lowerBound, upperP99)
	}
}

func TestAggregate_RequestCountExact(t *testing.T) {
	cases := []struct {
		name        string
		requests    int
		concurrency int
	}{
		{"N=10,c=1", 10, 1},
		{"N=50,c=5", 50, 5},
		{"N=100,c=10", 100, 10},
		{"N=500,c=20", 500, 20},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			var serverCount atomic.Int64
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				serverCount.Add(1)
				w.WriteHeader(http.StatusOK)
			}))
			defer server.Close()

			cfg := Config{
				Target:      server.URL,
				Requests:    tc.requests,
				Concurrency: tc.concurrency,
				Timeout:     10 * time.Second,
				Method:      "GET",
			}

			recorder := RunMultipleConcurrent(context.Background(), cfg)

			if got := recorder.Count(); got != int64(tc.requests) {
				t.Errorf("recorder.Count() = %d; want %d", got, tc.requests)
			}
			if got := recorder.FailedCount(); got != 0 {
				t.Errorf("recorder.FailedCount() = %d; want 0", got)
			}
			if got := serverCount.Load(); got != int64(tc.requests) {
				t.Errorf("server received %d requests; want %d", got, tc.requests)
			}
		})
	}
}

func TestAggregate_SuccessFailureSplit(t *testing.T) {
	var counter atomic.Int64
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		n := counter.Add(1)
		if n%2 == 0 {
			w.WriteHeader(http.StatusOK)
		} else {
			w.WriteHeader(http.StatusInternalServerError)
		}
	}))
	defer server.Close()

	cfg := Config{
		Target:      server.URL,
		Requests:    10,
		Concurrency: 1,
		Timeout:     5 * time.Second,
		Method:      "GET",
	}

	recorder := RunMultipleConcurrent(context.Background(), cfg)

	if got := recorder.Count(); got != 5 {
		t.Errorf("recorder.Count() = %d; want 5 (successes)", got)
	}
	if got := recorder.FailedCount(); got != 5 {
		t.Errorf("recorder.FailedCount() = %d; want 5 (failures)", got)
	}
	if got := recorder.TotalRequests(); got != 10 {
		t.Errorf("recorder.TotalRequests() = %d; want 10", got)
	}
}

func TestAggregate_SuccessFailureSplit_Concurrent(t *testing.T) {
	var counter atomic.Int64
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		n := counter.Add(1)
		if n%2 == 0 {
			w.WriteHeader(http.StatusOK)
		} else {
			w.WriteHeader(http.StatusInternalServerError)
		}
	}))
	defer server.Close()

	cfg := Config{
		Target:      server.URL,
		Requests:    100,
		Concurrency: 5,
		Timeout:     10 * time.Second,
		Method:      "GET",
	}

	recorder := RunMultipleConcurrent(context.Background(), cfg)

	if got := recorder.TotalRequests(); got != 100 {
		t.Fatalf("recorder.TotalRequests() = %d; want 100", got)
	}
	total := recorder.Count() + recorder.FailedCount()
	if total != 100 {
		t.Fatalf("Count()+FailedCount() = %d; want 100", total)
	}
	if s := recorder.Count(); s < 40 || s > 60 {
		t.Errorf("recorder.Count() = %d; want approximately 50 (between 40-60)", s)
	}
	if f := recorder.FailedCount(); f < 40 || f > 60 {
		t.Errorf("recorder.FailedCount() = %d; want approximately 50 (between 40-60)", f)
	}
}

func TestAggregate_MixedErrorTypes(t *testing.T) {
	var counter atomic.Int64
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		n := counter.Add(1)
		switch {
		case n%3 == 0:
			w.WriteHeader(http.StatusInternalServerError)
		case n%5 == 0:
			w.WriteHeader(http.StatusTooManyRequests)
		default:
			w.WriteHeader(http.StatusOK)
		}
	}))
	defer server.Close()

	cfg := Config{
		Target:      server.URL,
		Requests:    30,
		Concurrency: 1,
		Timeout:     5 * time.Second,
		Method:      "GET",
	}

	recorder := RunMultipleConcurrent(context.Background(), cfg)
	const (
		wantSuccess = 16
		wantFailed  = 14
		wantTotal   = 30
	)

	if got := recorder.TotalRequests(); got != wantTotal {
		t.Fatalf("recorder.TotalRequests() = %d; want %d", got, wantTotal)
	}
	if got := recorder.Count(); got != wantSuccess {
		t.Errorf("recorder.Count() = %d; want %d (successes)", got, wantSuccess)
	}
	if got := recorder.FailedCount(); got != wantFailed {
		t.Errorf("recorder.FailedCount() = %d; want %d (failures)", got, wantFailed)
	}
	if total := recorder.Count() + recorder.FailedCount(); total != wantTotal {
		t.Errorf("Count()+FailedCount() = %d; want %d", total, wantTotal)
	}
}

func TestAggregate_ConcurrencyPeak(t *testing.T) {
	cases := []struct {
		name        string
		requests    int
		concurrency int
		maxElapsed  time.Duration
	}{
		{"c=5", 20, 5, 600 * time.Millisecond},
		{"c=1", 10, 1, 1200 * time.Millisecond},
		{"c=10", 20, 10, 400 * time.Millisecond},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			const delay = 100 * time.Millisecond
			var current, peak atomic.Int64

			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				cur := current.Add(1)
				for {
					old := peak.Load()
					if cur <= old || peak.CompareAndSwap(old, cur) {
						break
					}
				}
				time.Sleep(delay)
				current.Add(-1)
				w.WriteHeader(http.StatusOK)
			}))
			defer server.Close()

			cfg := Config{
				Target:      server.URL,
				Requests:    tc.requests,
				Concurrency: tc.concurrency,
				Timeout:     5 * time.Second,
				Method:      "GET",
			}

			start := time.Now()
			recorder := RunMultipleConcurrent(context.Background(), cfg)
			elapsed := time.Since(start)

			if recorder.Count() != int64(tc.requests) {
				t.Fatalf("recorder.Count() = %d; want %d", recorder.Count(), tc.requests)
			}

			if tc.concurrency > 1 {
				minPeak := int64(tc.concurrency - 1)
				if got := peak.Load(); got < minPeak {
					t.Errorf("peak concurrency = %d; want >= %d", got, minPeak)
				}
			}

			if elapsed > tc.maxElapsed {
				t.Errorf("elapsed = %v; want < %v", elapsed, tc.maxElapsed)
			}
		})
	}
}

func TestAggregate_AllRequestsFail(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	cfg := Config{
		Target:      server.URL,
		Requests:    10,
		Concurrency: 2,
		Timeout:     5 * time.Second,
		Method:      "GET",
	}

	recorder := RunMultipleConcurrent(context.Background(), cfg)

	if got := recorder.Count(); got != 0 {
		t.Errorf("recorder.Count() = %d; want 0", got)
	}
	if got := recorder.FailedCount(); got != 10 {
		t.Errorf("recorder.FailedCount() = %d; want 10", got)
	}
	if got := recorder.Avg(); got != 0 {
		t.Errorf("recorder.Avg() = %v; want 0 (no successful recordings)", got)
	}
	if got := recorder.Min(); got != 0 {
		t.Errorf("recorder.Min() = %v; want 0 (no successful recordings)", got)
	}
}

func TestAggregate_TimeoutEnforced(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(500 * time.Millisecond)
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	cfg := Config{
		Target:      server.URL,
		Requests:    5,
		Concurrency: 1,
		Timeout:     100 * time.Millisecond,
		Method:      "GET",
	}

	start := time.Now()
	recorder := RunMultipleConcurrent(context.Background(), cfg)
	elapsed := time.Since(start)

	if got := recorder.Count(); got != 0 {
		t.Errorf("recorder.Count() = %d; want 0 (all timed out)", got)
	}
	if got := recorder.FailedCount(); got != 5 {
		t.Errorf("recorder.FailedCount() = %d; want 5 (all timed out)", got)
	}
	if got := recorder.TotalRequests(); got != 5 {
		t.Errorf("recorder.TotalRequests() = %d; want 5", got)
	}

	maxExpected := 800 * time.Millisecond
	if elapsed > maxExpected {
		t.Errorf("elapsed = %v; want < %v (timeout should cut off slow requests)", elapsed, maxExpected)
	}
}

func TestAggregate_DurationMode(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	cfg := Config{
		Target:      server.URL,
		Concurrency: 2,
		Timeout:     5 * time.Second,
		Duration:    2 * time.Second,
		Method:      "GET",
	}

	start := time.Now()
	recorder := RunForDuration(context.Background(), cfg)
	elapsed := time.Since(start)

	if elapsed < 1900*time.Millisecond || elapsed > 2500*time.Millisecond {
		t.Errorf("elapsed = %v; want between 1.9s and 2.5s", elapsed)
	}
	if got := recorder.Count(); got == 0 {
		t.Error("recorder.Count() = 0; want > 0 (requests should have been made)")
	}
}

func TestAggregate_GracefulShutdown(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(50 * time.Millisecond)
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	ctx, cancel := context.WithCancel(context.Background())
	go func() {
		time.Sleep(200 * time.Millisecond)
		cancel()
	}()

	cfg := Config{
		Target:      server.URL,
		Concurrency: 5,
		Timeout:     5 * time.Second,
		Duration:    10 * time.Second,
		Method:      "GET",
	}

	start := time.Now()
	recorder := RunForDuration(ctx, cfg)
	elapsed := time.Since(start)
	if elapsed > 700*time.Millisecond {
		t.Errorf("elapsed = %v; want < 700ms (should stop promptly after cancellation)", elapsed)
	}
	if got := recorder.Count(); got == 0 {
		t.Error("recorder.Count() = 0; want > 0 (partial results should exist)")
	}
	if avg := recorder.Avg(); avg <= 0 {
		t.Errorf("recorder.Avg() = %v; want > 0", avg)
	}
	if min := recorder.Min(); min <= 0 {
		t.Errorf("recorder.Min() = %v; want > 0", min)
	}
}

func TestAggregate_MoreWorkersThanRequests(t *testing.T) {
	var serverCount atomic.Int64
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		serverCount.Add(1)
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	const (
		requests    = 3
		concurrency = 10
	)

	cfg := Config{
		Target:      server.URL,
		Requests:    requests,
		Concurrency: concurrency,
		Timeout:     5 * time.Second,
		Method:      "GET",
	}

	recorder := RunMultipleConcurrent(context.Background(), cfg)

	if got := recorder.TotalRequests(); got != int64(requests) {
		t.Errorf("recorder.TotalRequests() = %d; want %d", got, requests)
	}
	if got := serverCount.Load(); got != int64(requests) {
		t.Errorf("server received %d requests; want %d", got, requests)
	}
}
