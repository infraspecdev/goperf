package cmd

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
)

type fileConfig struct {
	Target      *string  `json:"target" yaml:"target"`
	Requests    *int     `json:"requests" yaml:"requests"`
	Concurrency *int     `json:"concurrency" yaml:"concurrency"`
	Timeout     *string  `json:"timeout" yaml:"timeout"`
	Duration    *string  `json:"duration" yaml:"duration"`
	Method      *string  `json:"method" yaml:"method"`
	Body        *string  `json:"body" yaml:"body"`
	Headers     []string `json:"headers" yaml:"headers"`
}

func loadConfig(path string) (*fileConfig, error) {
	ext := strings.ToLower(filepath.Ext(path))

	switch ext {
	case ".json":
		return loadJSON(path)
	case ".yaml", ".yml":
		return loadYAML(path)
	default:
		return nil, fmt.Errorf("unsupported config file extension %q, supported: .json, .yaml, .yml", ext)
	}
}

func loadJSON(path string) (*fileConfig, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	if len(bytes.TrimSpace(data)) == 0 {
		return &fileConfig{}, nil
	}

	var cfg fileConfig
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("failed to parse JSON config: %w", err)
	}

	return &cfg, nil
}

func loadYAML(path string) (*fileConfig, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	if len(bytes.TrimSpace(data)) == 0 {
		return &fileConfig{}, nil
	}

	var cfg fileConfig
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("failed to parse YAML config: %w", err)
	}

	return &cfg, nil
}

func mergeConfig(file *fileConfig, cli RunConfig, changed map[string]bool) (RunConfig, error) {
	if file == nil {
		return cli, nil
	}

	merged := cli

	if file.Target != nil && !changed["target"] {
		merged.Target = *file.Target
	}

	if file.Requests != nil && !changed["requests"] {
		merged.Requests = *file.Requests
	}

	if file.Concurrency != nil && !changed["concurrency"] {
		merged.Concurrency = *file.Concurrency
	}

	if file.Timeout != nil && !changed["timeout"] {
		d, err := time.ParseDuration(*file.Timeout)
		if err != nil {
			return merged, fmt.Errorf("invalid timeout format in config file: %w", err)
		}
		merged.Timeout = d
	}

	if file.Duration != nil && !changed["duration"] {
		d, err := time.ParseDuration(*file.Duration)
		if err != nil {
			return merged, fmt.Errorf("invalid duration format in config file: %w", err)
		}
		merged.Duration = d
	}

	if file.Method != nil && !changed["method"] {
		merged.Method = strings.ToUpper(*file.Method)
	}

	if file.Body != nil && !changed["body"] {
		merged.Body = *file.Body
	}

	if len(file.Headers) > 0 && !changed["header"] {
		merged.Headers = make([]string, len(file.Headers))
		copy(merged.Headers, file.Headers)
	}

	return merged, nil
}
