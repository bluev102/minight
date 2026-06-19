package runner

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/minight/minight-terminal/internal/backend"
	"github.com/minight/minight-terminal/internal/config"
	"github.com/minight/minight-terminal/internal/output"
	"github.com/minight/minight-terminal/internal/pathutil"
	"github.com/minight/minight-terminal/internal/safety"
	"github.com/minight/minight-terminal/internal/session"
)

type Request struct {
	Command        string
	SessionID      string
	Timeout        time.Duration
	CWD            string
	Verbose        bool
	FailOnAnyError bool
	Pipefail       bool
	StripCRLF      *bool
}

type Response struct {
	Stdout              string   `json:"stdout"`
	Stderr              string   `json:"stderr"`
	ReturnCode          int      `json:"return_code"`
	TimedOut            bool     `json:"timed_out"`
	CurrentCWD          string   `json:"current_cwd"`
	Truncated           bool     `json:"truncated"`
	DurationMS          int64    `json:"duration_ms,omitempty"`
	StdoutOmittedBytes  int      `json:"stdout_omitted_bytes,omitempty"`
	StderrOmittedBytes  int      `json:"stderr_omitted_bytes,omitempty"`
	StdoutTotalBytes    int      `json:"stdout_total_bytes,omitempty"`
	StderrTotalBytes    int      `json:"stderr_total_bytes,omitempty"`
	SessionID           string   `json:"session_id,omitempty"`
	EnvChangedCount     int      `json:"env_changed_count,omitempty"`
	HadFailure          bool     `json:"had_failure,omitempty"`
	CWDPersisted        bool     `json:"cwd_persisted,omitempty"`
	EnvironmentWarnings []string `json:"environment_warnings,omitempty"`
	SuggestedTimeout    int      `json:"suggested_timeout,omitempty"`
	Error               string   `json:"error,omitempty"`
}

type Runner struct {
	cfg     config.Config
	session *session.Manager
	backend backend.Backend
}

func New(cfg config.Config, sessions *session.Manager) *Runner {
	return &Runner{
		cfg:     cfg,
		session: sessions,
		backend: backend.New(cfg),
	}
}

func (r *Runner) Run(ctx context.Context, req Request) Response {
	start := time.Now()
	sessionID := req.SessionID
	state := r.session.Get(sessionID)

	if err := safety.Check(req.Command); err != nil {
		return Response{
			ReturnCode: 1,
			CurrentCWD: state.CWD,
			Error:      err.Error(),
		}
	}

	cwd := state.CWD
	if req.CWD != "" {
		cwd = req.CWD
	}
	if r.cfg.NormalizeWSLPaths {
		if normalized, _ := pathutil.NormalizeWSLPath(cwd); normalized != cwd {
			cwd = normalized
		}
	}
	if !dirExists(cwd) {
		return Response{
			ReturnCode: 1,
			CurrentCWD: state.CWD,
			Error:      fmt.Sprintf("invalid cwd: %s", cwd),
		}
	}

	timeout := req.Timeout
	if timeout <= 0 {
		timeout = r.cfg.DefaultTimeout
	}
	if timeout > r.cfg.MaxTimeout {
		timeout = r.cfg.MaxTimeout
	}

	stripCRLF := r.cfg.StripCRLF
	if req.StripCRLF != nil {
		stripCRLF = *req.StripCRLF
	}
	sanitizeOpts := output.SanitizeOpts{Limit: r.cfg.OutputLimit, StripCRLF: stripCRLF}

	warnings := environmentWarnings(cwd)

	result := r.backend.Execute(ctx, backend.ExecuteOpts{
		Command:        req.Command,
		CWD:            cwd,
		Env:            state.Env,
		Timeout:        timeout,
		FailOnAnyError: req.FailOnAnyError,
		Pipefail:       req.Pipefail,
	})

	duration := time.Since(start)

	if result.StartErr != nil && result.TimedOut {
		stdoutOut := output.SanitizeStream(result.Stdout, sanitizeOpts)
		stderrOut := output.SanitizeStream(result.Stderr, sanitizeOpts)
		resp := Response{
			Stdout:              stdoutOut.Text,
			Stderr:              stderrOut.Text,
			ReturnCode:          124,
			TimedOut:            true,
			CurrentCWD:          state.CWD,
			Truncated:           stdoutOut.Truncated || stderrOut.Truncated,
			EnvironmentWarnings: warnings,
			SuggestedTimeout:    suggestedTimeoutSeconds(timeout, duration),
		}
		return r.withVerbose(resp, req.Verbose, sessionID, duration, stdoutOut, stderrOut, 0)
	}

	if result.StartErr != nil {
		stdoutOut := output.SanitizeStream(result.Stdout, sanitizeOpts)
		stderrOut := output.SanitizeStream(result.Stderr, sanitizeOpts)
		resp := Response{
			Stdout:              stdoutOut.Text,
			Stderr:              stderrOut.Text,
			ReturnCode:          result.ExitCode,
			CurrentCWD:          state.CWD,
			Truncated:           stdoutOut.Truncated || stderrOut.Truncated,
			Error:               result.StartErr.Error(),
			EnvironmentWarnings: warnings,
		}
		return r.withVerbose(resp, req.Verbose, sessionID, duration, stdoutOut, stderrOut, 0)
	}

	if result.TimedOut {
		stdoutOut := output.SanitizeStream(result.Stdout, sanitizeOpts)
		stderrOut := output.SanitizeStream(result.Stderr, sanitizeOpts)
		resp := Response{
			Stdout:              stdoutOut.Text,
			Stderr:              stderrOut.Text,
			ReturnCode:          124,
			TimedOut:            true,
			CurrentCWD:          state.CWD,
			Truncated:           stdoutOut.Truncated || stderrOut.Truncated,
			EnvironmentWarnings: warnings,
			SuggestedTimeout:    suggestedTimeoutSeconds(timeout, duration),
		}
		return r.withVerbose(resp, req.Verbose, sessionID, duration, stdoutOut, stderrOut, 0)
	}

	beforeEnv := state.Env
	updated := r.session.Update(sessionID, result.CWD, result.Env, session.UpdateMeta{
		LastCommand:    req.Command,
		BackgroundPIDs: result.BackgroundPIDs,
		ReturnCode:     result.ExitCode,
		HadFailure:     result.HadFailure,
	})
	envChanged := envDiffCount(beforeEnv, updated.Env)

	stdoutOut := output.SanitizeStream(result.Stdout, sanitizeOpts)
	stderrOut := output.SanitizeStream(result.Stderr, sanitizeOpts)

	resp := Response{
		Stdout:              stdoutOut.Text,
		Stderr:              stderrOut.Text,
		ReturnCode:          result.ExitCode,
		TimedOut:            false,
		CurrentCWD:          updated.CWD,
		Truncated:           stdoutOut.Truncated || stderrOut.Truncated,
		HadFailure:          result.HadFailure,
		CWDPersisted:        result.CWD != "" && updated.CWD == result.CWD,
		EnvironmentWarnings: warnings,
	}
	if result.CWD != "" && updated.CWD != result.CWD {
		resp.CWDPersisted = false
	}

	return r.withVerbose(resp, req.Verbose, sessionID, duration, stdoutOut, stderrOut, envChanged)
}

func (r *Runner) withVerbose(resp Response, verbose bool, sessionID string, duration time.Duration, stdoutOut, stderrOut output.TruncatedOutput, envChanged int) Response {
	if !verbose {
		return resp
	}
	resp.DurationMS = duration.Milliseconds()
	resp.StdoutOmittedBytes = stdoutOut.OmittedBytes
	resp.StderrOmittedBytes = stderrOut.OmittedBytes
	resp.StdoutTotalBytes = stdoutOut.TotalBytes
	resp.StderrTotalBytes = stderrOut.TotalBytes
	resp.SessionID = sessionID
	resp.EnvChangedCount = envChanged
	return resp
}

func environmentWarnings(cwd string) []string {
	var warnings []string
	if pathutil.IsWSLDrvfsMount(cwd) {
		warnings = append(warnings, "cwd is on a WSL drvfs mount (/mnt/<drive>/); git and filesystem scans may be much slower than native Windows shells")
	}
	if pathutil.InWSL() {
		warnings = append(warnings, "running under WSL; use /mnt/<drive>/ paths or enable MINIGHT_NORMALIZE_WSL_PATHS for /e/ shorthand")
	}
	return warnings
}

func suggestedTimeoutSeconds(current time.Duration, elapsed time.Duration) int {
	secs := int((current + time.Second - 1) / time.Second)
	if secs < 1 {
		secs = 1
	}
	if elapsed >= current-time.Millisecond*100 {
		return min(secs*2, 300)
	}
	return secs
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func dirExists(path string) bool {
	info, err := os.Stat(path)
	return err == nil && info.IsDir()
}

func envDiffCount(before, after map[string]string) int {
	seen := make(map[string]struct{})
	count := 0
	for key, value := range after {
		seen[key] = struct{}{}
		if before[key] != value {
			count++
		}
	}
	for key := range before {
		if _, ok := seen[key]; !ok {
			count++
		}
	}
	return count
}
