# GoPerf: The Comprehensive Guide

Welcome to the definitive documentation for **GoPerf**, a specialized command-line tool designed for load testing HTTP APIs and accurately reporting performance metrics. This guide covers every feature, how to use them, and important architectural concepts behind the tool.

---

## What is GoPerf?

When building robust APIs, you need to answer critical questions: _How fast is it? How many users can it handle concurrently? What happens to latency under load?_

GoPerf helps you answer these questions by sending thousands of concurrent HTTP requests and reporting exactly what your users will experience. It is a powerful, lightweight, single-binary application built in Go, optimized for developers to validate their backend services before shipping to production.

---

## Installation

The easiest way to get started is by downloading the pre-built binaries via our install scripts:

**Linux / macOS:**

```bash
curl -sL https://raw.githubusercontent.com/infraspecdev/goperf/main/install.sh | sh
```

**Windows (PowerShell):**

```powershell
irm https://raw.githubusercontent.com/infraspecdev/goperf/main/install.ps1 | iex
```

Alternatively, you can download the binaries manually from our [GitHub Releases](https://github.com/infraspecdev/goperf/releases) page or build from source using Go 1.25+.

---

## Feature Overview & Usage

GoPerf is invoked via the `run` command.

```bash
goperf run <url> [flags]
```

### 1. Basic Load Testing

To run a simple test with 100 requests (the default is 1 request if not specified):

```bash
goperf run https://httpbin.org/get -n 100
```

This sends 100 sequential requests and reports the outcome.

### 2. Concurrency (`-c, --concurrency`)

To test how your server handles simultaneous connections, increase the concurrency:

```bash
goperf run https://httpbin.org/get -n 1000 -c 50
```

_Usage:_ `goperf` will spin up 50 concurrent workers, distributing the 1000 requests among them as quickly as the server allows.

### 3. Duration-Based Testing (`-d, --duration`)

Instead of a fixed number of requests, you can run the test for a specific duration. This is highly recommended for identifying memory leaks or sustained load degradation.

```bash
goperf run https://httpbin.org/get -c 50 -d 30s
```

_Note:_ When `-d` is provided, the `-n` (requests) flag is ignored.

### 4. HTTP Methods & Payloads (`-m`, `-b`)

By default, GoPerf issues `GET` requests. You can test `POST`, `PUT`, `DELETE`, `PATCH`, `HEAD`, and `OPTIONS` operations and provide a body payload.

```bash
goperf run https://httpbin.org/post -m POST -b '{"title":"foo","body":"bar","userId":1}'
```

### 5. Custom Headers (`-H, --header`) & Security

If your API requires authentication, content-type declarations, or specific user-agents, you can provide multiple headers.

> [!WARNING]
> **Security Anti-Pattern:** Passing secrets (like Bearer tokens or API keys) via the `-H` command-line flag stores them permanently in your `~/.bash_history` and exposes them via `ps aux`. For secure automated testing, define secrets inside a configuration file (see below) instead of the CLI.

```bash
# Not recommended for secure tokens!
goperf run https://httpbin.org/get \
  -H "Authorization: Bearer my-token" \
  -H "Content-Type: application/json"
```

### 6. File-Based Configuration (`-f, --config`)

For complex tests, CI/CD pipelines, or hiding security credentials, defining parameters in a JSON or YAML config file keeps things organized, secure, and reproducible.

```bash
goperf run -f load-test.yaml
```

_Example Configuration:_

```yaml
target: "https://httpbin.org/post"
concurrency: 100
duration: "1m"
method: "POST"
headers:
  - "Authorization: Bearer my-secret-token"
  - "Content-Type: application/json"
body: '{"test":"data"}'
```

### 7. Automation & CI/CD Pipelines (`-o, --output`)

By default, GoPerf provides human-readable text output. For CI/CD automation, use the JSON output.

_Example: Failing a CI pipeline if the p99 latency goes above 1500ms using `jq`._

```bash
P99=$(goperf run https://httpbin.org/get -n 100 -c 10 -o json | jq '.p99_ms')

if (( $(echo "$P99 > 1500" | bc -l) )); then
  echo "Performance test failed! p99 Latency is ${P99}ms (Limit: 1500ms)"
  exit 1
fi
```

### 8. Timeout Limits (`-t, --timeout`)

Prevent hanging requests by enforcing a strict timeout (default is 10s).

```bash
goperf run https://httpbin.org/delay/3 -t 2s
```

### 9. Verbose Mode (`-v, --verbose`)

Enable verbose logging to print the result and latency of every single request. Extremely useful for debugging HTTP errors that occur mid-test.

```bash
goperf run https://httpbin.org/get -n 10 -v
```

---

## Important Architectural Concepts

To interpret the results effectively, it is vital to understand how GoPerf operates under the hood.

### What Latency Does GoPerf Measure? (TTFB)

GoPerf specifically records the **Time To First Byte (TTFB)**.
When a request is initiated, the timer starts. The timer stops the moment the server begins sending the response headers back to the client.

**Crucial Note:** GoPerf **does not include response body download time** in its latency metrics. This mathematically guarantees that network bandwidth limitations on the client machine downloading massive payloads do not skew the backend processing metrics. It measures _how fast your server processed the request_, not how fast your network can download the response body.

### Closed-Loop Testing & Coordinated Omission

GoPerf operates on a **Closed-Loop** load testing model.

**What it means:**
In closed-loop testing, a worker sends a request and _waits_ for the response before it sends its next request. The concurrency limit (`-c 50`) means there will be exactly 50 connections open at maximum.

**The implication (Coordinated Omission):**
Because a worker waits for a response before firing the next request, the tool suffers from **Coordinated Omission**. If your server locks up for 5 seconds during an outage, the tool inherently stops firing new requests for those 5 seconds. This artificially makes your latency percentiles (like `p99`) look _better_ than reality because the tool failed to record the hundreds of requests that _should_ have queued up during that 5-second stall.

**When to use other tools:**
If you want to validate if your server survives **Open-Loop** traffic—where requests arrive at a strict, unyielding constant rate (e.g., exactly 1,000 requests per second) _regardless_ of how slow the server responds—you should look into open-loop load generators like `Vegeta`. Closed-loop testing (GoPerf) is excellent for finding the maximum capacity curve of your application, whereas open-loop testing is better for testing breaking points under unrelenting traffic spikes.

---

## Example Output Explained

```text
$ goperf run https://httpbin.org/get -c 50 -d 30s
Running for 30s against https://httpbin.org/get with concurrency 50
  [2s]  98 reqs | 49.0/s
  [4s]  210 reqs | 52.5/s
  ...

Target:     https://httpbin.org/get
Duration:   30.0s
Requests:   1,523 total (1,520 succeeded, 3 failed)

Status code distribution:
  [200] 1520 responses
  [500] 3 responses

Latency:
  Fastest:  12.00ms   <- The quickest single request
  Slowest:  892.00ms  <- The worst outlier request
  Average:  45.00ms   <- Standard mean latency
  p50:      38.00ms   <- 50% of your users experienced 38ms or better (Median)
  p90:      89.00ms   <- 90% of your users experienced 89ms or better
  p99:      234.00ms  <- 99% of your users experienced 234ms or better

Response time histogram:
  12.000 [50]   |■■■■■■■■■■■■■■■■■■■■■
  100.000 [150] |■■■■■■■■■■■■■■■■■■■■■■■■■■■■■■■■■■■■■■■■
  892.000 [3]   |■

Throughput: 50.7 requests/sec <- How much total work was accomplished
```

> [!TIP]
> **Never trust the Average.** Averages hide disastrous outliers entirely. Always use `p95` and `p99` percentiles as your primary source of truth for optimization. The `p99` metric reveals the painful reality that your 1% worst-case users are experiencing.
