# GoPerf — Code Review & Path Forward

Read this end to end. Don't skim it. Every section has a reason for being here.

---

## Part 1: Bugs — Things That Are Broken Right Now

These are not opinions or style preferences. These are defects in the code that produce incorrect behavior.

### 1.1 HTTP Status Codes Are Silently Ignored

Open [`internal/httpclient/client.go`, lines 84-89](internal/httpclient/client.go#L84-L89):

```go
_, duration, err := MakeRequest(ctx, rawURL, timeout, method, body)
if err == nil {
    recorder.Record(duration)
} else if !isContextCancellation(err) {
    recorder.RecordFailure()
}
```

`MakeRequest` returns `(statusCode, duration, err)`. That first return value — the status code — is discarded with `_`.

This means: if a server returns HTTP 500 Internal Server Error in 5ms, goperf counts it as a **successful** request with 5ms latency. An HTTP 503 Service Unavailable? Success. A 429 Too Many Requests? Success.

The tool's primary job is to tell us how the server behaves under load. If it counts server errors as successes, the output is meaningless. We could have 80% of requests returning 500 and the summary would say "all succeeded."

The same pattern appears in [`RunForDuration` at line 122-126](internal/httpclient/client.go#L122-L126).

### 1.2 The HTTP Transport Makes Every Measurement Wrong

[`client.go`, line 17](internal/httpclient/client.go#L17):

```go
var client = &http.Client{}
```

This creates an HTTP client with Go's default transport. Go's default `http.Transport` has this setting:

```go
MaxIdleConnsPerHost: 2
```

When you run `goperf run http://target -n 1000 -c 50`, you have 50 goroutines sharing a connection pool that keeps only 2 idle connections per host. The other 48 goroutines can't find an idle connection in the pool, so they establish **new TCP connections** — including DNS lookup, TCP handshake, and TLS negotiation. Under sustained concurrent load, the pool is almost always empty because only 2 connections can be returned to it at a time, so the vast majority of requests pay the full connection setup cost.

That overhead gets included in the latency measurement. We're not measuring how fast the server responds. We're measuring how fast the server responds **plus** how long it takes to establish a new connection from scratch. On HTTPS targets, TLS alone can add 10-50ms per request.

For comparison, here's what `hey` (the tool the README told you to study) does:

```go
tr := &http.Transport{
    TLSClientConfig:     &tls.Config{InsecureSkipVerify: ...},
    MaxIdleConnsPerHost: concurrency,
    DisableCompression:  true,
    DisableKeepAlives:   disableKeepAlives,
}
```

It sets `MaxIdleConnsPerHost` to match concurrency so every worker reuses its connection. It also disables compression (because we want to measure server performance, not the CPU's decompression speed).

This isn't a performance optimization. It's a correctness fix. A load testing tool that adds its own connection overhead to every measurement is producing wrong numbers.

### 1.3 Histogram Records Slow Requests as Failures

[`internal/stats/histogram.go`, lines 21-23](internal/stats/histogram.go#L21-L23):

```go
func NewHistogramRecorder(timeout time.Duration) *HistogramRecorder {
    return &HistogramRecorder{
        histogram: hdrhistogram.New(1, timeout.Nanoseconds(), 3),
```

The second argument to `hdrhistogram.New` is the maximum trackable value. You set it to `timeout.Nanoseconds()`.

Now look at [`Record()`, lines 35-39](internal/stats/histogram.go#L35-L39):

```go
err := h.histogram.RecordValue(ns)
if err != nil {
    h.failed++
    return
}
```

If a request takes even 1 nanosecond longer than the timeout (due to goroutine scheduling, GC, OS jitter), `RecordValue` returns an out-of-range error. That request gets counted as a **failure** and its latency disappears from the statistics.

So: slow requests are removed from our latency data and added to the failure count. The p99 becomes artificially low because the slowest requests are excluded. The failure count becomes artificially high because it includes measurement artifacts, not actual failures.

The upper bound should be at least `2 * timeout` to account for scheduling jitter. Even better: `10 * timeout`, since the memory cost is negligible (HDR Histogram uses logarithmic compression).

### 1.4 Output Goes to Two Different Places

[`cmd/run.go`, line 114](cmd/run.go#L114) (duration mode):

```go
fmt.Fprintf(cmd.OutOrStdout(), "Running for %v against %s with concurrency %d\n", ...)
```

[Line 122-123](cmd/run.go#L122-L123) (request-count mode):

```go
fmt.Println("Parsed URL:", u)
fmt.Printf("Making %d requests to %s with concurrency %d\n", requests, u, concurrency)
```

`fmt.Println` and `fmt.Printf` write to `os.Stdout` directly. `cmd.OutOrStdout()` writes to whatever writer Cobra is configured with — which in tests is a `bytes.Buffer`.

This means: in request-count mode, the "Parsed URL" and "Making N requests" lines bypass the test buffer and go to the real stdout. Your integration tests can't capture or verify this output. It also means if someone ever pipes goperf's output, the duration mode is pipeable but the request-count mode has rogue output going to a different file descriptor.

Also: `"Parsed URL:"` is debug output. It tells the user nothing useful. It shouldn't be in the final output at all.

### 1.5 Workers Spin Instead of Stopping on Cancellation

[`client.go`, lines 80-83](internal/httpclient/client.go#L80-L83) in `RunMultipleConcurrent`:

```go
for range jobs {
    if ctx.Err() != nil {
        continue
    }
```

When the user presses Ctrl+C, the context is cancelled. But workers don't stop — they `continue` draining the job channel, checking the context on every iteration, doing nothing useful. The channel buffer holds up to `concurrency` items, and any remaining buffered jobs get drained through these no-op iterations instead of being abandoned immediately.

This should be `return`, not `continue`. Compare with `RunForDuration` at [line 119](internal/httpclient/client.go#L119-L120) which correctly uses `return`.

---

## Part 2: Design Problems — Things That Work But Are Wrong

These won't cause test failures, but they make the codebase harder to change, harder to test, and harder to reason about.

### 2.1 No Configuration Abstraction

Count the parameters in `RunMultipleConcurrent`:

```go
func RunMultipleConcurrent(ctx context.Context, rawURL string, n, concurrency int,
    timeout time.Duration, method string, body string) *stats.HistogramRecorder
```

Seven parameters. The same seven appear in `RunForDuration`, `runCommandMultipleConcurrent`, and `runCommandDuration`. When you added `method` and `body` in PR #36, you had to modify every function in the chain — 340 additions and 24 deletions across 7 files. With a config struct, the same feature would have touched 2-3 files.

Now think about what happens when you add custom headers. Then config file support. Then TLS settings. Then proxy support. Each feature adds more parameters to every function signature.

The fix is a struct:

```go
type Config struct {
    Target      string
    Requests    int
    Concurrency int
    Timeout     time.Duration
    Duration    time.Duration
    Method      string
    Body        string
    Headers     map[string]string  // future
}
```

Then: `RunMultipleConcurrent(ctx context.Context, cfg Config) *stats.HistogramRecorder`. Adding headers means adding one field to the struct and one line in `MakeRequest`. No signature changes anywhere else.

This is basic software design — when you see the same group of values passed together repeatedly, they belong in a struct. Not because it's "cleaner" in some abstract sense, but because it makes the code cheaper to change. Right now, every new feature has a blast radius of 6+ files. With a config struct, it's usually 2.

### 2.2 Global Mutable State Everywhere

The HTTP client is a package-level global:

```go
var client = &http.Client{}
```

`runCmd` is a package-level global:

```go
var runCmd = &cobra.Command{...}
```

`rootCmd` is a package-level global:

```go
var rootCmd = &cobra.Command{...}
```

This creates three problems.

First, you can't configure the HTTP client per test or per run. Want to test with a custom transport? Want to disable redirects? Want to set a proxy? You'd have to modify the global, which is not safe from concurrent tests.

Second, your integration tests all share the same `runCmd` instance. That's why every test has this:

```go
defer func() {
    _ = runCmd.Flags().Set("requests", "1")
    _ = runCmd.Flags().Set("concurrency", "1")
    _ = runCmd.Flags().Set("timeout", "10s")
    _ = runCmd.Flags().Set("method", "GET")
    _ = runCmd.Flags().Set("body", "")
    _ = runCmd.Flags().Set("duration", "0s")
    runCmd.Flags().Lookup("duration").Changed = false
    runCmd.Flags().Lookup("requests").Changed = false
}()
```

This appears in every integration test. If you forget to reset one flag, the next test inherits stale state. This is fragile. Tests should be independent by construction, not by careful manual cleanup.

Third, none of this is injectable. You use `httptest.NewServer` for testing, which is the standard Go pattern — but it still makes real HTTP calls over the loopback interface. Because the client is hardcoded, you can't swap in a custom `http.Transport` for deterministic testing (e.g., fixed latencies without network jitter), and you can't benchmark our measurement overhead separately from actual HTTP round-trips.

In Go, the standard pattern for this is an interface:

```go
type HTTPDoer interface {
    Do(req *http.Request) (*http.Response, error)
}
```

`*http.Client` already satisfies this. You'd store it in a struct, pass it as a dependency, and in tests, you can swap in a mock.

### 2.3 No Content-Type When Sending a Body

When you do `goperf run http://target -m POST -b '{"key":"value"}'`, the request goes out with no `Content-Type` header. The server has to guess what format the body is in. Most servers won't — they'll reject it or misparse it.

This came up in PR #36's review. Rahul noted it and you tracked it as issue #37. But it's more urgent than a future "custom headers" feature — when a user sends a body, you should default to `Content-Type: application/json` (or let them override it). Without this, the `-b` flag is broken for most real-world APIs.

### 2.4 Validation Lives in the Wrong Place

`cmd/run.go` has six standalone validation functions: `validateTarget`, `validateRequests`, `validateTimeout`, `validateConcurrency`, `validateDuration`, `validateMethod`.

These validate individual fields, but the real validation logic is scattered inside the `RunE` closure at [lines 80-126](cmd/run.go#L80-L126). The mutual exclusion check (`--requests` vs `--duration`) is at [line 109](cmd/run.go#L109). The conditional validation of `requests` only when duration is zero is at [line 118](cmd/run.go#L118).

The validation is interleaved with flag reading, configuration assembly, and execution. There's no single place you can look at to understand "what constitutes a valid configuration?"

If you had a `Config` struct, validation becomes one function:

```go
func (c Config) Validate() error { ... }
```

All rules in one place. Easy to test. Easy to read. Easy to add new rules.

### 2.5 `printHistogramStatistics` Is a Wall of Repetition

[`cmd/run.go`, lines 155-208](cmd/run.go#L155-L208). This is 53 lines that do one thing: print formatted statistics. It's eleven `fmt.Fprintf` calls, each individually error-checked with the same pattern:

```go
if _, err := fmt.Fprintf(out, "..."); err != nil {
    return err
}
```

This pattern is repeated eleven times. There are a few things wrong with this.

The error checking is noise. If `fmt.Fprintf` to stdout fails, your terminal is gone. Checking every line individually doesn't help you recover — there's nothing to recover to. `fmt.Fprintf` to a buffer (in tests) effectively never fails. The error checks make the function 2x longer without adding value.

The formatting is hardcoded. Want JSON output? CSV? You'd have to write a completely separate function. Instead, separate the data from the formatting:

```go
type Result struct {
    Target     string
    Elapsed    time.Duration
    Total      int64
    Succeeded  int64
    Failed     int64
    Min, Max   time.Duration
    Avg        time.Duration
    P50, P90   time.Duration
    P99        time.Duration
    Throughput float64
}
```

Then rendering is one concern (`func (r Result) PrintText(w io.Writer)`) and can have siblings (`PrintJSON`, `PrintCSV`) without touching the data collection.

### 2.6 The Command Description Is Misleading

[`cmd/run.go`, line 73](cmd/run.go#L73):

```go
Short: "Command to give input URL",
```

This is the help text users see when they run `goperf --help`. It tells them nothing about what the command does. "Command to give input URL" sounds like a description of how to invoke it, not what it does.

It should describe the command's purpose:

```go
Short: "Run a load test against an HTTP endpoint",
```

Small thing, but CLI tools are user-facing products. Every user-visible string should be intentional.

### 2.7 The `runCommandMultipleConcurrent` Wrapper Adds Nothing

[`cmd/run.go`, lines 140-153](cmd/run.go#L140-L153):

```go
func runCommandMultipleConcurrent(...) error {
    ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
    defer stop()
    start := time.Now()
    recorder := httpclient.RunMultipleConcurrent(ctx, target, n, concurrency, timeout, method, body)
    elapsed := time.Since(start)
    if err := printHistogramStatistics(out, recorder, target, elapsed); err != nil {
        return err
    }
    return nil
}
```

Compare to [`runCommandDuration` at lines 129-138](cmd/run.go#L129-L138):

```go
func runCommandDuration(...) error {
    ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
    defer stop()
    start := time.Now()
    recorder := httpclient.RunForDuration(ctx, target, concurrency, timeout, duration, method, body)
    elapsed := time.Since(start)
    return printHistogramStatistics(out, recorder, target, elapsed)
}
```

Two things. First, these functions are nearly identical. The only difference is which `httpclient` function they call. That's a hint they should be one function that takes a "runner" as a parameter.

Second, `runCommandMultipleConcurrent` has the verbose pattern:

```go
if err := printHistogramStatistics(...); err != nil {
    return err
}
return nil
```

This is just `return printHistogramStatistics(...)`. `runCommandDuration` already does it correctly on [line 137](cmd/run.go#L137). This inconsistency within the same file tells me the code was written at different times without re-reading what was already there.

### 2.8 The `run` Command Is Registered Twice

[`cmd/root.go`, line 21](cmd/root.go#L21):

```go
func init() {
    rootCmd.AddCommand(runCmd)
}
```

[`cmd/run.go`, line 217](cmd/run.go#L217):

```go
func init() {
    // ...flags...
    rootCmd.AddCommand(runCmd)
}
```

Cobra handles the duplicate silently, so it doesn't crash. But you now have two locations claiming to wire up the same command, and if either changes, the behavior depends on which `init()` runs first. Pick one — `run.go` makes sense since that's where the command is defined — and remove the other.

### 2.9 Valid Methods Are Incomplete

[`cmd/run.go`, lines 57-62](cmd/run.go#L57-L62) only allows GET, POST, PUT, DELETE. PATCH is missing — and PATCH is standard HTTP for partial updates, used heavily in modern REST APIs. HEAD and OPTIONS are also absent without reason. There's no technical barrier to adding them; the validation map just needs more entries.

---

## Part 3: Measurement Correctness — The Core of This Tool

goperf is a measurement tool. If the measurements are wrong, nothing else matters. This section is about understanding what we're actually measuring and whether it's accurate.

### 3.1 You're Running a Closed-Loop Test (And You Should Know What That Means)

In `RunMultipleConcurrent`, each worker does:

```
1. Pick up a job from the channel
2. Send a request
3. Wait for the response
4. Record the result
5. Go back to step 1
```

This is called **closed-loop** testing. The rate at which requests are sent depends on how fast the server responds. If the server slows down, you send fewer requests. If it speeds up, you send more.

The problem: when the server is slow, we're sending fewer requests during the slow period, which means we have fewer samples of the bad behavior. The p99 will undercount how bad things actually are, because the slowest responses suppress the request rate that would have revealed more slow responses.

This is called **coordinated omission**. It's the single most common mistake in load testing tools. Gil Tene (the creator of HDR Histogram, the library you're using) has an entire talk about this.

I'm not asking you to fix this right now. I'm asking you to **understand it** and **document it** in your README — something like:

> goperf uses closed-loop testing. Each worker waits for a response before sending the next request. This means throughput is limited by server response time. For constant-rate testing, consider tools like wrk2 or k6.

A tool that knows its own limitations is more useful than one that pretends to have none.

### 3.2 You Don't Separate Connection Time from Response Time

`MakeRequest` starts timing at [line 23](internal/httpclient/client.go#L23) and stops at [line 36](internal/httpclient/client.go#L36):

```go
start := time.Now()
// ... create request ...
resp, err := client.Do(req)
duration = time.Since(start)
```

This duration includes:
- DNS resolution (if not cached)
- TCP handshake
- TLS negotiation (for HTTPS)
- Sending the request
- Server processing
- Receiving the response headers

But it does **not** include reading/discarding the response body — the body read at [line 58](internal/httpclient/client.go#L58) happens after `duration = time.Since(start)` on line 36. So what we're measuring is essentially TTFB (Time to First Byte), which is a valid metric — but you're not documenting it as such, and users with large response bodies might see different numbers than they expect.

Tools like `hey` provide phase-level breakdown:

```
DNS+dialup:  0.0013 secs, ...
DNS-lookup:  0.0000 secs, ...
req write:   0.0000 secs, ...
resp wait:   0.0122 secs, ...
resp read:   0.0001 secs, ...
```

You don't need to build this right now, but you should understand what that single duration number actually represents and document it.

### 3.3 The Min() Bug You Fixed — What It Was Really Telling You

Eshaan, you wrote:

> On my machine, Min() was returning 0s, so the assertion Min() > 0 failed. After digging into it, we realized this was mostly due to timing precision and hardware speed. Some requests to the local test server were completing so fast that time.Since() rounded them to 0ns.

Your fix was clamping durations to 1ns and tracking min/max manually. That fix works, but it treated the symptom rather than understanding the cause.

The real issue: your test server at [`client_test.go:130-131`](internal/httpclient/client_test.go#L130-L131) responds instantly:

```go
server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
    w.WriteHeader(http.StatusOK)
}))
```

An `httptest.NewServer` running on localhost with no processing can respond in under 1 microsecond. That's realistic for the test — but it's not realistic for any actual HTTP server. No server on the internet responds that fast. Your test was telling you: **"this test scenario doesn't represent real-world usage."**

A better fix is to make the test server behave like a real server:

```go
server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
    time.Sleep(5 * time.Millisecond)  // simulate realistic processing
    w.WriteHeader(http.StatusOK)
}))
```

The clamping at [`histogram.go:32-34`](internal/stats/histogram.go#L32-L34) should stay (it's defensive), but the test should exercise realistic latencies. You noticed this already in `TestRunCommand_RequestCountMode` where you use `time.Sleep(10 * time.Millisecond)` — that pattern should be consistent across all performance-related tests.

### 3.4 No Correctness Test for the Tool's Primary Purpose

Your tests verify "does the output contain these label strings?" They don't verify "are the numbers right?"

You have no test that says: "I set up a server with 50ms latency, ran 100 requests, and verified the average is between 45ms and 70ms." That's the most important test for a measurement tool.

Here's what that test looks like:

```go
func TestLatencyAccuracy(t *testing.T) {
    expectedDelay := 50 * time.Millisecond
    server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        time.Sleep(expectedDelay)
        w.WriteHeader(http.StatusOK)
    }))
    defer server.Close()

    recorder := httpclient.RunMultipleConcurrent(
        context.Background(), server.URL, 20, 4, 5*time.Second, "GET", "")

    avg := recorder.Avg()
    if avg < expectedDelay*8/10 || avg > expectedDelay*2 {
        t.Errorf("average %v outside expected range [%v, %v]",
            avg, expectedDelay*8/10, expectedDelay*2)
    }
}
```

If this test fails, the tool is broken. If it passes, we have evidence that the tool actually measures what it claims to measure. This is more valuable than all the "flag exists" tests combined.

---

## Part 4: What Your Tests Are Actually Testing

You have 42 tests across 5 test files (`run_test.go`, `run_integration_test.go`, `root_test.go`, `client_test.go`, `histogram_test.go`). Here's how they break down.

### Tests that verify Cobra's behavior (not yours):

- `TestRunCmdHasNFlag` — checks if `.Lookup("requests")` returns non-nil
- `TestNFlagDefaultValue` — checks if `.GetInt("requests")` returns 1
- `TestRunCmdHasTimeoutFlag` — same pattern
- `TestTimeoutFlagDefaultValue` — same pattern
- `TestConcurrencyFlagExists` — same pattern
- `TestDurationFlagExists` — same pattern
- `TestDurationFlagDefault` — same pattern
- `TestMethodFlagExists` — same pattern
- `TestMethodFlagDefaultValue` — same pattern
- `TestBodyFlagExists` — same pattern
- `TestBodyFlagDefaultValue` — same pattern
- `TestRootCmdInitialization` — checks rootCmd isn't nil

That's 12 tests (29% of your test suite) that test whether Cobra correctly registered a flag. These tests will never fail unless you delete the flag registration. They don't test your logic. They test the framework's.

You don't write a test to check that `fmt.Println` prints to stdout. You trust the standard library. Cobra's flag registration is equally reliable. These tests give you a count of 42, but 12 of them carry zero weight.

### Tests that test validators in isolation:

- `TestValidateTarget` — 8 cases, good coverage of URL edge cases
- `TestNFlagPositiveValidation` — 4 cases for positive integer check
- `TestValidateTimeout` — 5 cases
- `TestValidateConcurrency` — 4 cases
- `TestValidateDuration` — 4 cases
- `TestValidateMethod` — 7 cases

These are well-structured table-driven tests. They test real logic. But they're testing the simplest logic in the codebase — "is this number positive?" and "is this URL valid?" The complex logic (concurrency behavior, timing accuracy, error aggregation) has minimal testing.

### Tests that verify real behavior:

- `TestRunCommand_RequestCountMode` — end-to-end with request counting
- `TestRunCommand_ConnectionError` — verifies graceful failure handling
- `TestRunCommand_Concurrency` — verifies actual parallel execution (this is your best test)
- `TestRunCommand_DurationMode` — verifies time-based execution
- `TestRunCommand_MethodFlag` — verifies HTTP methods reach the server
- `TestRunCommand_InvalidMethod` — verifies rejection
- `TestRunCommand_MethodWithBody` — verifies body delivery
- `TestRunMultipleConcurrent_UsesConcurrency` — verifies concurrency timing
- `TestRunForDuration_ReturnsHistogram` — verifies duration mode runs
- `TestRunForDuration_RespectsContext` — verifies cancellation
- `TestMakeRequest_Errors` — verifies error handling
- `TestMakeRequest_Methods` — verifies method propagation
- `TestMakeRequestWithBody` / `TestMakeRequestGetNoBody` — body handling
- `TestPrintHistogramStatistics` — output format verification
- Histogram recorder tests (8 tests)

These are the ones that matter. Some are very good — `TestRunCommand_Concurrency` uses atomic counters to verify actual parallelism, which shows real understanding. But none of them test **measurement accuracy**, which is the tool's primary purpose.

### What's missing:

- No test that verifies latency measurements are correct within a tolerance
- No test that an HTTP 500 is counted as a failure (it currently isn't)
- No test for what happens when concurrency > requests
- No test for very high concurrency behavior
- No benchmark (`testing.B`) for measurement overhead

---

## Part 5: Missing Features — What the README Required

Go back and re-read the README. Here's the "Should Have" list, checked against reality:

| Requirement | Status |
|---|---|
| Supports different HTTP methods | Done for GET/POST/PUT/DELETE (PR #36). PATCH, HEAD, OPTIONS missing — see 2.9 |
| Supports custom headers | Not started |
| Supports request bodies | Done (PR #36) |
| Loads configuration from a file | Not started (issue #38 open) |
| Stores test history for comparison | Not needed as a standalone feature — JSON output (Tier 3) covers this |
| Automated build and deployment (CI/CD) | CI done, CD partial (no EC2 deploy) |
| Documented and tested | Partially |

And the "Must Have" MVP:

| Requirement | Status |
|---|---|
| Concurrent HTTP requests for specified duration | Done |
| Reports latency statistics including percentiles | Done (but see Part 3 on accuracy) |
| Reports throughput | Done |
| Reports success/failure counts | Partially (see 1.1 — status codes ignored) |
| **Runs on AWS EC2** | **Not done** |

The EC2 requirement was in the **Must Have MVP by Week 4**. Issues #23 and #24 are open and untouched.

---

## Part 6: Features to Build — Ordered by Impact

Not week-by-week. Ordered by how much value each delivers and what depends on what.

### Tier 1: Fix What's Broken

These come first because everything else builds on correct foundations.

**Fix status code handling.** Record HTTP 4xx/5xx as failures with error classification. The output should show:

```
Requests:   1000 total (920 succeeded, 80 failed)
  Errors:
    HTTP 500:  45
    HTTP 503:  15
    Timeout:   12
    Conn refused: 8
```

This requires a change to the recorder (track error categories, not just a count) and a change to the worker loop (pass the status code through).

**Fix HTTP transport configuration.** Create the `http.Client` with a properly configured transport. At minimum, `MaxIdleConnsPerHost` must match concurrency. Disable compression.

**Fix histogram upper bound.** Use `10 * timeout.Nanoseconds()` as the max trackable value.

**Fix the output stream inconsistency.** Replace `fmt.Println` / `fmt.Printf` on [lines 122-123](cmd/run.go#L122-L123) with `fmt.Fprintf(cmd.OutOrStdout(), ...)`.

**Fix worker cancellation.** Change `continue` to `return` in `RunMultipleConcurrent` [line 82](internal/httpclient/client.go#L82).

**Remove duplicate `rootCmd.AddCommand(runCmd)`** from either `root.go` or `run.go`.

**Add PATCH to valid methods.** Add HEAD and OPTIONS while you're at it.

**Set Content-Type when body is provided.** Default to `application/json` when `-b` is used. Let users override it once custom headers land.

**Fix the command description.** `Short: "Command to give input URL"` → `Short: "Run a load test against an HTTP endpoint"`. This is the first thing users see in `--help` output.

### Tier 2: Design Improvements

These make every subsequent feature cheaper to build.

**Create a `Config` struct.** Replace the 7-parameter function signatures. Every function in the chain should pass `Config` instead of individual values.

**Define an `HTTPDoer` interface.** Inject the HTTP client as a dependency. This makes the transport configurable and the code testable with mocks.

**Create a `Result` struct.** Separate data from formatting. This enables JSON output, CSV output, and file export without touching the measurement code.

**Write a latency accuracy test.** Server with known delay, verify the measurement is within tolerance. This is the single most important test going forward.

**Write Go benchmarks.** `go test -bench .` for `Record()` and `MakeRequest()`. Know the tool's overhead.

### Tier 3: Required Features

**Custom headers** (`-H "Authorization: Bearer token"`). Without this, goperf can't test any authenticated API — which is most real APIs. Store headers in the `Config` struct and pass them through to `http.NewRequestWithContext`.

**Config file support** (YAML or TOML). Load a config file that specifies target, headers, method, body, concurrency, etc. This makes complex test configurations repeatable and shareable. Use a library like `viper` (which integrates with Cobra) or keep it simple with `encoding/json`.

**Progress output during tests.** A user running a 60-second load test sees nothing for 60 seconds. Print periodic stats (every 1-2 seconds) showing current request rate, error rate, and latency. This is straightforward: a ticker goroutine that reads the recorder's current state.

**JSON output mode** (`--output json`). This enables piping goperf into other tools, storing results for comparison, and building dashboards. Trivial once you have the `Result` struct from Tier 2.

**Verbose mode** (`--verbose` / `-v`). Right now if a user gets unexpected results, there's no way to see what happened during the test. A verbose flag should print per-request status codes, latencies, and errors to stderr while the summary still goes to stdout. Useful for debugging and for understanding what's happening during a run.

### Tier 4: CI/CD and Deployment

Your CI is solid (lint, test, govulncheck, build) and your CD already cross-compiles and creates GitHub Releases on tags. That's good work. But the pipeline stops at "binary exists on GitHub." Nobody is picking it up and putting it on EC2.

The expectation was always: merge to main, tag a release, and the latest version is running on EC2 before Friday's demo. No manual steps, no "let me scp this real quick."

What's missing:

**Dockerfile.** Multi-stage build — compile in a Go builder stage, copy the binary into a minimal runtime image. Tools like k6, kubectl, and aws-cli all ship Docker images. Study how they do it.

**Push to a container registry.** Extend `cd.yml` to build and push the image to ghcr.io on tagged releases. GitHub Actions has built-in support for this.

**EC2 deployment.** Issues #23 and #24 have been open since the start — this was a Must Have by Week 4. The AWS account and VPC are ready (see the README). Extend `cd.yml` to deploy the latest container image to an EC2 instance on every tagged release. The goal is zero manual steps between tagging and having the new version running. Credentials go in GitHub Secrets.

Once this works end-to-end, the release process is: tag → CI/CD runs → deployed on EC2. That's the bar.

### Tier 5: Stretch Goals

**Latency distribution output.** Show percentile breakdown (p10, p25, p50, p75, p90, p95, p99) and optionally an ASCII histogram of the distribution. The data is already in the HDR Histogram — this is a rendering exercise.

**Per-interval stats.** Collect per-second metrics and show how latency/throughput/error-rate evolve over time. This reveals degradation patterns that summary statistics hide.

**Real-time progress.** Replace the periodic print with a live-updating terminal display (using terminal control codes or a library like `bubbletea`).

---

## Part 7: Process — How to Work Going Forward

### PR Discipline

Your PRs grew from 62 lines (PR #13) to 574 lines (PR #33). PR #33 had 22 commits and touched 11 files. That's not a reviewable PR. It's a feature branch dump.

A good PR is:
- One logical change (a bug fix, a single feature, a refactor)
- Under 200 lines of implementation (tests can be longer)
- 3-5 commits that tell a clean story
- A description that explains **why**, not just **what**

When you find yourself with a PR that does 4 things, split it. It's more work upfront but it gets reviewed faster, merged faster, and if something breaks, you know exactly which change caused it.

### Commit Hygiene

Learn `git rebase -i`. Before opening a PR, clean up your commits:
- Squash "fix lint" into the commit that introduced the lint error
- Squash "address review feedback" into the original commit
- Remove merge commits from `origin/main`

Each commit should compile and pass tests independently. A reviewer should be able to read commits top-to-bottom and understand the progression of the change.

### Peer Review

You should be each other's first reviewer. Not a rubber stamp — actually read the code, think about edge cases, and leave comments. This is a skill. The way to develop it is practice.

When reviewing, ask:
- Does this handle errors?
- What happens at the boundaries (zero, negative, very large)?
- Is there a simpler way to do this?
- Can I understand what this does without reading the PR description?

### Design Before Code

Before building any Tier 2+ feature, write a short design doc (half a page to a page is fine):
1. What are we building and why?
2. What approaches did we consider?
3. What did `hey` / `wrk` / `k6` do for this? (actually go look)
4. What's our approach and why?
5. What are the trade-offs?

Share it before writing code. This prevents building the wrong thing and it produces artifacts that show your thinking.

---

## Part 8: Go Skills to Develop

These aren't abstract recommendations. Each one directly applies to goperf.

### Interfaces and Dependency Injection

Read the `net/http` package source and notice how `http.Handler`, `http.ResponseWriter`, and `io.Reader` / `io.Writer` are interfaces. Go's standard library is built on small interfaces composed together.

In goperf, the immediate application: define `HTTPDoer` and inject it. This single change makes the HTTP layer testable without hitting the network.

### Benchmarking

Run `go test -bench . -benchmem ./internal/stats/`. If you don't have benchmarks, write one for `Record()`. It will tell you how many nanoseconds and allocations each call costs. This number matters because it's overhead we add to every measurement.

### Profiling

Run the tool against a local server with high concurrency. While it runs, capture a CPU profile with `runtime/pprof` or use `go tool pprof`. Look at where time is spent. Is it in your code? In the mutex? In the HTTP client? In the runtime scheduler? Understanding this is what separates someone who writes code from someone who understands systems.

### Context Propagation

You already use `context.Context` for cancellation, which is good. Understand the full pattern: contexts carry deadlines, cancellation signals, and values. Every I/O operation should accept a context. This is how Go programs handle timeouts and graceful shutdown composably.

### Error Wrapping

You use `fmt.Errorf("...: %w", err)` in some places, which is correct. Be consistent. Every error that crosses a package boundary should be wrapped with context about what operation failed. This makes error messages useful:

```
connection refused: dial tcp 127.0.0.1:8080: connect: connection refused
```

is much more useful than just:

```
dial tcp 127.0.0.1:8080: connect: connection refused
```

---

## Part 9: Understanding Your Domain

You're building a load testing tool. That's a specific domain with decades of research and well-known pitfalls. You should know the domain, not just the Go syntax.

### Study `hey`

The README asked you to study existing tools. `hey` is ~1000 lines of Go. Read it. Understand every design decision. Pay special attention to:
- How it configures the HTTP transport
- How it collects results (channel-based collector pattern)
- How it handles connection warmup
- How it structures the `Work` type
- How it renders output

### Closed-Loop vs. Open-Loop Testing

Part 3 explains why goperf is a closed-loop tool and what that means for our measurements. Go deeper than that. Read about open-loop testing — `wrk2` implements this, where requests are sent at a fixed rate regardless of how fast the server responds. If the server slows down, requests queue up and latency climbs, which is what happens to real users.

You need to be able to explain why goperf's numbers will differ from an open-loop tool testing the same server, and when each model is the right choice.

### Coordinated Omission

HDR Histogram (the library you're using) has a coordinated omission correction feature built in. You're not using it. Read Gil Tene's writing on this — he created the library specifically because this problem is so pervasive in load testing. At minimum, understand the concept well enough to document goperf's limitation in the README.

---

## What I Expect

1. Read this document fully and discuss it with each other.
2. For the bugs in Part 1 — fix them. These are non-negotiable.
3. For the design issues in Part 2 — address them as you build new features. Don't rewrite the world in one PR. Let the Config struct and the interface emerge naturally as you add headers and config file support.
4. For every new feature — write a half-page design doc before coding. Share it.
5. Read `hey`'s source code. Write down what you learned.
6. Keep PRs under 200 lines. Review each other's code.

The goal isn't to check boxes. The goal is to build something you can explain, defend, and be proud of. Right now, if someone asked "are goperf's latency numbers accurate?" — the honest answer is "we don't know, and the code suggests they aren't." That needs to change.
