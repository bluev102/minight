package config

import (
	"fmt"
	"os"
	"runtime"
	"strconv"
	"strings"
	"time"
)

const (
	defaultTimeoutSeconds = 30
	defaultMaxTimeout     = 300
	defaultOutputLimit    = 3000
	defaultShellPath      = "/bin/sh"
)

type Config struct {
	DefaultTimeout     time.Duration
	MaxTimeout         time.Duration
	OutputLimit        int
	ShellPath          string
	Backend            string
	StripCRLF          bool
	NormalizeWSLPaths  bool
	DefaultTimeoutFrom int
}

func Load() (Config, error) {
	cfg := Config{
		DefaultTimeout:    defaultTimeoutSeconds * time.Second,
		MaxTimeout:        defaultMaxTimeout * time.Second,
		OutputLimit:       defaultOutputLimit,
		ShellPath:         defaultShellPath,
		Backend:           "auto",
		StripCRLF:         true,
		NormalizeWSLPaths: true,
	}

	if raw := os.Getenv("DEFAULT_TIMEOUT_SECONDS"); raw != "" {
		seconds, err := strconv.Atoi(raw)
		if err != nil {
			return Config{}, fmt.Errorf("invalid DEFAULT_TIMEOUT_SECONDS: %w", err)
		}
		if seconds <= 0 {
			return Config{}, fmt.Errorf("DEFAULT_TIMEOUT_SECONDS must be positive")
		}
		cfg.DefaultTimeout = time.Duration(seconds) * time.Second
		cfg.DefaultTimeoutFrom = seconds
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

	if raw := os.Getenv("MINIGHT_BACKEND"); raw != "" {
		cfg.Backend = strings.ToLower(strings.TrimSpace(raw))
	}

	if raw := os.Getenv("MINIGHT_SHELL"); raw != "" {
		cfg.ShellPath = raw
	} else if shell := os.Getenv("SHELL"); shell != "" {
		cfg.ShellPath = shell
	}

	if raw := os.Getenv("MINIGHT_STRIP_CRLF"); raw != "" {
		cfg.StripCRLF = parseBool(raw, true)
	}

	if raw := os.Getenv("MINIGHT_NORMALIZE_WSL_PATHS"); raw != "" {
		cfg.NormalizeWSLPaths = parseBool(raw, true)
	}

	if raw := os.Getenv("MINIGHT_OUTPUT_LIMIT"); raw != "" {
		limit, err := strconv.Atoi(raw)
		if err != nil {
			return Config{}, fmt.Errorf("invalid MINIGHT_OUTPUT_LIMIT: %w", err)
		}
		if limit <= 0 {
			return Config{}, fmt.Errorf("MINIGHT_OUTPUT_LIMIT must be positive")
		}
		cfg.OutputLimit = limit
	}

	if cfg.DefaultTimeout > cfg.MaxTimeout {
		cfg.DefaultTimeout = cfg.MaxTimeout
	}

	if runtime.GOOS == "windows" && cfg.Backend == "auto" {
		cfg.Backend = "windows"
	}

	return cfg, nil
}

func parseBool(raw string, defaultVal bool) bool {
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case "1", "true", "yes", "on":
		return true
	case "0", "false", "no", "off":
		return false
	default:
		return defaultVal
	}
}
