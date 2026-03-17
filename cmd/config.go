package cmd

import (
	"fmt"
	"net/url"
	"sort"
	"strings"
	"time"

	"github.com/infraspecdev/goperf/internal/httpclient"
)

var supportedMethodsStr string

func init() {
	methods := make([]string, 0, len(validMethods))
	for m := range validMethods {
		methods = append(methods, m)
	}
	sort.Strings(methods)
	supportedMethodsStr = strings.Join(methods, ", ")
}

type RunConfig struct {
	Target       string
	ParsedTarget *url.URL
	Requests     int
	Concurrency  int
	Timeout      time.Duration
	Duration     time.Duration
	Method       string
	Body         string
	Headers      []string
	Verbose      bool
}

var validMethods = map[string]bool{
	"GET":     true,
	"POST":    true,
	"PUT":     true,
	"DELETE":  true,
	"PATCH":   true,
	"OPTIONS": true,
	"HEAD":    true,
}

func (c *RunConfig) Validate() error {
	if c.Target == "" {
		return fmt.Errorf("missing target URL: must be provided via CLI argument or config file")
	}

	u, err := url.ParseRequestURI(c.Target)
	if err != nil {
		return fmt.Errorf("invalid target URL provided: parse error: %w", err)
	}
	if u.Scheme == "" || u.Host == "" {
		return fmt.Errorf("invalid target URL provided: missing scheme (e.g., http:// or https://) or host")
	}
	c.ParsedTarget = u

	if c.Concurrency <= 0 {
		return fmt.Errorf("concurrency must be positive, got %d", c.Concurrency)
	}
	if c.Timeout <= 0 {
		return fmt.Errorf("timeout must be positive, got %v", c.Timeout)
	}
	if c.Duration < 0 {
		return fmt.Errorf("duration must not be negative, got %v", c.Duration)
	}
	if !validMethods[c.Method] {
		return fmt.Errorf("invalid HTTP method %q, supported methods: %s", c.Method, supportedMethodsStr)
	}
	for _, h := range c.Headers {
		parts := strings.SplitN(h, ":", 2)
		if len(parts) != 2 || strings.TrimSpace(parts[0]) == "" || strings.Contains(strings.TrimSpace(parts[0]), " ") {
			return fmt.Errorf("invalid header format %q, expected 'Key: Value' without spaces in the key", h)
		}
	}

	if c.Requests > 1 && c.Duration > 0 {
		return fmt.Errorf("cannot use both --requests (-n) and --duration (-d) at the same time")
	}

	if c.Duration == 0 && c.Requests <= 0 {
		return fmt.Errorf("number of requests must be positive, got %d", c.Requests)
	}

	return nil
}

func (c *RunConfig) ToHTTPConfig() httpclient.Config {
	return httpclient.Config{
		Target:      c.Target,
		Requests:    c.Requests,
		Concurrency: c.Concurrency,
		Timeout:     c.Timeout,
		Duration:    c.Duration,
		Method:      c.Method,
		Body:        c.Body,
		Headers:     c.Headers,
		Verbose:     c.Verbose,
	}
}
