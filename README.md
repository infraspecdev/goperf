<br>

<img
  src="https://github.com/user-attachments/assets/917f03ef-29fa-47bf-96fb-a60789cdad4e"
  width="500"
/>
<br><br>
[![GitHub release](https://img.shields.io/github/v/release/infraspecdev/goperf?style=flat-square)](https://github.com/infraspecdev/goperf/releases)
[![CI status](https://img.shields.io/github/actions/workflow/status/infraspecdev/goperf/ci.yml?branch=main&style=flat-square)](https://github.com/infraspecdev/goperf/actions/workflows/ci.yml)
[![Go Report Card](https://goreportcard.com/badge/github.com/infraspecdev/goperf?style=flat-square)](https://goreportcard.com/report/github.com/infraspecdev/goperf)

`goperf` is a lightweight, high-performance HTTP load testing and benchmarking tool written in Go. It focuses on closed-loop load testing to help developers quickly validate API performance, measure concurrency limits, and analyze real-world latency metrics like p90 and p99.

## Features

- **Concurrency & Duration Testing:** Test by a strict number of requests or over a sustained duration.
- **Detailed Metrics:** Accurate reporting of TTFB (Time To First Byte) latencies, including min, max, average, p50, p90, and p99 percentiles.
- **Live Progress:** Real-time updates every 2 seconds showing request count, throughput (req/s), and error count.
- **Response Time Histogram:** Visual distribution of response times to quickly spot latency patterns.
- **Error Categorization:** Automatic breakdown of network-level errors (timeouts, connection refused, DNS failures, etc.) alongside HTTP status code distribution.
- **CI/CD Ready:** Native JSON output support for easy integration into automated pipelines.
- **Configurable:** Support for complex requests via YAML/JSON configuration files.


## Installation

**Linux / macOS**
```bash
curl -sL https://raw.githubusercontent.com/infraspecdev/goperf/main/install.sh | sh
```
**Windows (PowerShell)**
```powershell
irm https://raw.githubusercontent.com/infraspecdev/goperf/main/install.ps1 | iex
```
**Build from Source (Requires Go 1.26.1 or newer)**
```bash
git clone https://github.com/infraspecdev/goperf.git
cd goperf
make build
./bin/goperf --help
```

## Usage

`goperf` runs the provided number of requests at the provided concurrency level and prints latency stats.

```text
Usage: goperf run <url> [options...]

Options:
  -n          Number of requests to execute. Default is 1.
  -c          Number of concurrent workers. Default is 1.
  -d          Duration to run the test. When duration is reached, the application
              stops and exits. If duration is specified, -n is ignored.
              Examples: -d 10s, -d 1m.
  -o          Output format. "text" or "json". Default is text.

  -m          HTTP method, one of GET, POST, PUT, DELETE, PATCH, OPTIONS, HEAD.
              Default is GET.
  -H          Custom HTTP header. You can specify as many as needed by repeating the flag.
              For example: -H "Accept: text/html" -H "Content-Type: application/json".
  -b          HTTP request body content.
  -D          Path to a file containing the request body. Use this for large
              payloads instead of -b.
  -t          Timeout for each request. Default is 10s.

  -f          Path to configuration file (JSON/YAML).
  -v          Enable verbose output. Prints every request's latency or error.
```

## Examples

Make 100 requests sequentially:
```bash
goperf run https://httpbin.org/get -n 100
```

Make 1000 requests with 50 concurrent workers:
```bash
goperf run https://httpbin.org/get -n 1000 -c 50
```

Run load test for 30 seconds:
```bash
goperf run https://httpbin.org/get -c 50 -d 30s
```

Make POST request with custom body:
```bash
goperf run https://httpbin.org/post \
    -m POST \
    -b '{"title":"foo","body":"bar"}'
```

Make POST request with a body from a file:
```bash
goperf run https://httpbin.org/post \
    -m POST \
    -D payload.json
```

Example `payload.json`:
```json
{
  "title": "foo",
  "body": "bar",
  "userId": 1
}
```

Add custom headers:
```bash
goperf run https://httpbin.org/get \
    -H "Accept: application/json" \
    -H "Authorization: Bearer token"
```

Run test using a configuration file:
```bash
goperf run -f load-test.yaml
```

Example `load-test.yaml`:
```yaml
target: "https://httpbin.org/post"
concurrency: 100
duration: "1m"
method: "POST"
headers:
  - "Authorization: Bearer your-token-here"
  - "Content-Type: application/json"
body: '{"test":"data"}'
```

Prevent hanging requests by enforcing a strict per-request timeout:
```bash
goperf run https://httpbin.org/delay/3 -t 2s
```
 Use Verbose Mode for Debugging, print the result and latency of every individual request:
```bash
goperf run https://httpbin.org/get -n 10 -v
```


Output stats as JSON for CI/CD automation:
```bash
goperf run https://httpbin.org/get -n 500 -c 20 -o json
```

## How it Works

- **We measure TTFB (Time To First Byte):** Our timer stops the exact millisecond your server starts sending response headers. We don't include the time it takes to download the actual response body.Because we want to measure how fast your server processes data, not how fast your local internet connection is.

- **Closed-Loop Testing:** `goperf` sends a request, waits for the response, and then sends the next one. If you use `-c 50`, you will have exactly 50 connections open at all times.

- **The "Coordinated Omission" :** Because our workers wait for responses, if your server completely locks up for 5 seconds, `goperf` will patiently wait and stop sending new requests. This means your p99 latencies might look slightly better than reality during an outage. If you need to test traffic that hits your server at a constant, unforgiving rate (open-loop testing), we highly recommend checking out [Vegeta](https://github.com/tsenart/vegeta?tab=readme-ov-file).

## Example Output Explained

```text
$ goperf run https://httpbin.org/get -c 50 -d 7s -t 1s
Running for 7s against https://httpbin.org/get with concurrency 50
  [2s]  170 reqs | 85.0/s | 45 errors       <- Live progress every 2s: total requests, rate, and error count
  [4s]  565 reqs | 141.2/s | 46 errors
  [6s]  933 reqs | 155.5/s | 46 errors

Target:     https://httpbin.org/get           <- The URL that was tested
Duration:   7.001s                            <- Total wall-clock time of the test
Requests:   1132 total (1086 succeeded, 46 failed)  <- Summary of all requests sent

Status code distribution:                     <- Breakdown of HTTP status codes received
  [200] 1086 responses

Error distribution:                           <- Breakdown of network-level errors (timeouts, connection refused, etc.)
  [46] context deadline exceeded              <- 46 requests exceeded the 1s timeout

Latency:
  Fastest:  208.76ms                          <- The quickest single request
  Slowest:  954.37ms                          <- The worst outlier request
  Average:  271.02ms                          <- Arithmetic mean of all successful requests
  p50:      212.21ms                          <- 50% of requests were faster than this (Median)
  p90:      431.23ms                          <- 90% of requests were faster than this
  p99:      795.87ms                          <- 99% of requests were faster than this
Response time histogram:                      <- Visual distribution of response times
  208.760 [826]  |■■■■■■■■■■■■■■■■■■■■■■■■■■■■■■■■■■■■■■■■
  283.320 [52]   |■■
  357.881 [131]  |■■■■■■
  432.442 [26]   |■
  507.003 [11]   |
  581.563 [16]   |
  656.124 [7]    |
  730.685 [6]    |
  805.246 [5]    |
  879.807 [6]    |

Throughput: 161.7 requests/sec                <- Overall requests completed per second
```

## License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) for details.
