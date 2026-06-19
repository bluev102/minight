package config

import (
	"os"
	"testing"
	"time"
)

func TestLoadDefaults(t *testing.T) {
	t.Setenv("MAX_TIMEOUT_SECONDS", "")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if cfg.DefaultTimeout != 30*time.Second {
		t.Fatalf("DefaultTimeout = %v, want 30s", cfg.DefaultTimeout)
	}
	if cfg.MaxTimeout != 300*time.Second {
		t.Fatalf("MaxTimeout = %v, want 300s", cfg.MaxTimeout)
	}
	if cfg.OutputLimit != 3000 {
		t.Fatalf("OutputLimit = %d, want 3000", cfg.OutputLimit)
	}
}

func TestLoadMaxTimeoutFromEnv(t *testing.T) {
	t.Setenv("MAX_TIMEOUT_SECONDS", "120")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if cfg.MaxTimeout != 120*time.Second {
		t.Fatalf("MaxTimeout = %v, want 120s", cfg.MaxTimeout)
	}
}

func TestLoadInvalidMaxTimeout(t *testing.T) {
	t.Setenv("MAX_TIMEOUT_SECONDS", "not-a-number")

	_, err := Load()
	if err == nil {
		t.Fatal("Load() expected error for invalid MAX_TIMEOUT_SECONDS")
	}
}

func TestLoadShellPathFromEnv(t *testing.T) {
	t.Setenv("MAX_TIMEOUT_SECONDS", "")
	t.Setenv("SHELL", "/bin/zsh")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if cfg.ShellPath != "/bin/zsh" {
		t.Fatalf("ShellPath = %q, want /bin/zsh", cfg.ShellPath)
	}
}

func TestLoadShellPathFallback(t *testing.T) {
	t.Setenv("MAX_TIMEOUT_SECONDS", "")
	if err := os.Unsetenv("SHELL"); err != nil {
		t.Fatalf("Unsetenv(SHELL): %v", err)
	}

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if cfg.ShellPath != "/bin/sh" {
		t.Fatalf("ShellPath = %q, want /bin/sh", cfg.ShellPath)
	}
}
