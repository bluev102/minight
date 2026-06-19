package backend

import (
	"context"
	"time"
)

// ExecuteOpts configures a single command execution.
type ExecuteOpts struct {
	Command        string
	CWD            string
	Env            map[string]string
	Timeout        time.Duration
	FailOnAnyError bool
	Pipefail       bool
}

// ExecResult is the raw outcome of command execution before output sanitization.
type ExecResult struct {
	Stdout         string
	Stderr         string
	ExitCode       int
	CWD            string
	Env            map[string]string
	HadFailure     bool
	BackgroundPIDs []int
	TimedOut       bool
	StartErr       error
}

// Backend executes shell commands on a host platform.
type Backend interface {
	Name() string
	Execute(ctx context.Context, opts ExecuteOpts) ExecResult
}
