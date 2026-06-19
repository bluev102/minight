package backend

import (
	"os"
	"runtime"
	"strings"

	"github.com/minight/minight-terminal/internal/config"
)

// New selects the execution backend from configuration.
func New(cfg config.Config) Backend {
	switch strings.ToLower(cfg.Backend) {
	case "windows":
		return &WindowsBackend{ShellPath: cfg.ShellPath}
	case "posix":
		return &POSIXBackend{ShellPath: cfg.ShellPath}
	default:
		if runtime.GOOS == "windows" && cfg.Backend != "posix" {
			shell := cfg.ShellPath
			if shell == "" {
				shell = defaultWindowsShell()
			}
			return &WindowsBackend{ShellPath: shell}
		}
		shell := cfg.ShellPath
		if shell == "" {
			shell = "/bin/sh"
		}
		if sh := os.Getenv("SHELL"); sh != "" && cfg.ShellPath == "" {
			shell = sh
		}
		return &POSIXBackend{ShellPath: shell}
	}
}
