package config

import (
	"fmt"
	"os"
	"strconv"
	"time"
)

const (
	defaultTimeoutSeconds = 30
	defaultMaxTimeout     = 300
	defaultOutputLimit    = 3000
	defaultShellPath      = "/bin/sh"
)

type Config struct {
	DefaultTimeout time.Duration
	MaxTimeout     time.Duration
	OutputLimit    int
	ShellPath      string
}

func Load() (Config, error) {
	cfg := Config{
		DefaultTimeout: defaultTimeoutSeconds * time.Second,
		MaxTimeout:     defaultMaxTimeout * time.Second,
		OutputLimit:    defaultOutputLimit,
		ShellPath:      defaultShellPath,
	}

	if raw := os.Getenv("MAX_TIMEOUT_SECONDS"); raw != "" {
		seconds, err := strconv.Atoi(raw)
		if err != nil {
			return Config{}, fmt.Errorf("invalid MAX_TIMEOUT_SECONDS: %w", err)
		}
		if seconds <= 0 {
			return Config{}, fmt.Errorf("MAX_TIMEOUT_SECONDS must be positive")
		}
		cfg.MaxTimeout = time.Duration(seconds) * time.Second
	}

	if shell := os.Getenv("SHELL"); shell != "" {
		cfg.ShellPath = shell
	}

	return cfg, nil
}
