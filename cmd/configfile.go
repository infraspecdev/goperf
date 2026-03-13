package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

type FileConfig struct {
	Target      *string  `json:"target" yaml:"target"`
	Requests    *int     `json:"requests" yaml:"requests"`
	Concurrency *int     `json:"concurrency" yaml:"concurrency"`
	Timeout     *string  `json:"timeout" yaml:"timeout"`
	Duration    *string  `json:"duration" yaml:"duration"`
	Method      *string  `json:"method" yaml:"method"`
	Body        *string  `json:"body" yaml:"body"`
	Headers     []string `json:"headers" yaml:"headers"`
}

func LoadConfig(path string) (*FileConfig, error) {
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

func loadJSON(path string) (*FileConfig, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	if len(strings.TrimSpace(string(data))) == 0 {
		return &FileConfig{}, nil
	}

	var cfg FileConfig
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("failed to parse JSON config: %w", err)
	}

	return &cfg, nil
}

func loadYAML(path string) (*FileConfig, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	if len(strings.TrimSpace(string(data))) == 0 {
		return &FileConfig{}, nil
	}

	var cfg FileConfig
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("failed to parse YAML config: %w", err)
	}

	return &cfg, nil
}
