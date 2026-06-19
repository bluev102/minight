package runner

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/minight/minight-terminal/internal/config"
	"github.com/minight/minight-terminal/internal/output"
	"github.com/minight/minight-terminal/internal/safety"
	"github.com/minight/minight-terminal/internal/session"
)

const (
	trailerBegin = "__MINIGHT_TRAILER_BEGIN__"
	trailerEnd   = "__MINIGHT_TRAILER_END__"
)

type Request struct {
	Command   string
	SessionID string
	Timeout   time.Duration
	CWD       string
	Verbose   bool
}

type Response struct {
	Stdout             string `json:"stdout"`
	Stderr             string `json:"stderr"`
	ReturnCode         int    `json:"return_code"`
	TimedOut           bool   `json:"timed_out"`
	CurrentCWD         string `json:"current_cwd"`
	Truncated          bool   `json:"truncated"`
	DurationMS         int64  `json:"duration_ms,omitempty"`
	StdoutOmittedBytes int    `json:"stdout_omitted_bytes,omitempty"`
	StderrOmittedBytes int    `json:"stderr_omitted_bytes,omitempty"`
	SessionID          string `json:"session_id,omitempty"`
	EnvChangedCount    int    `json:"env_changed_count,omitempty"`
	Error              string `json:"error,omitempty"`
}

type Runner struct {
	cfg     config.Config
	session *session.Manager
}

func New(cfg config.Config, sessions *session.Manager) *Runner {
	return &Runner{cfg: cfg, session: sessions}
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

	wrapped := wrapCommand(req.Command)
	cmdCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	cmd := exec.Command(r.cfg.ShellPath, "-lc", wrapped)
	cmd.Dir = cwd
	cmd.Env = buildEnv(state.Env)
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}

	var stdoutBuf, stderrBuf bytes.Buffer
	cmd.Stdout = &stdoutBuf
	cmd.Stderr = &stderrBuf

	if err := cmd.Start(); err != nil {
		return Response{
			ReturnCode: 1,
			CurrentCWD: state.CWD,
			Error:      fmt.Sprintf("failed to start command: %v", err),
		}
	}

	waitDone := make(chan error, 1)
	go func() {
		waitDone <- cmd.Wait()
	}()

	var runErr error
	select {
	case <-cmdCtx.Done():
		if cmd.Process != nil {
			_ = syscall.Kill(-cmd.Process.Pid, syscall.SIGKILL)
		}
		runErr = cmdCtx.Err()
		<-waitDone
	case runErr = <-waitDone:
	}

	duration := time.Since(start)
	timedOut := cmdCtx.Err() == context.DeadlineExceeded

	stdoutRaw := stdoutBuf.String()
	stderrRaw := stderrBuf.String()

	if timedOut {
		stdoutOut := output.SanitizeStream(stdoutRaw, r.cfg.OutputLimit)
		stderrOut := output.SanitizeStream(stderrRaw+"\ncommand timed out", r.cfg.OutputLimit)
		resp := Response{
			Stdout:     stdoutOut.Text,
			Stderr:     stderrOut.Text,
			ReturnCode: 124,
			TimedOut:   true,
			CurrentCWD: state.CWD,
			Truncated:  stdoutOut.Truncated || stderrOut.Truncated,
		}
		return r.withVerbose(resp, req.Verbose, sessionID, duration, stdoutOut, stderrOut, 0)
	}

	userStdout, userStderr, trailer, parseErr := splitTrailer(stdoutRaw, stderrRaw)
	if parseErr != nil {
		stdoutOut := output.SanitizeStream(userStdout, r.cfg.OutputLimit)
		stderrOut := output.SanitizeStream(userStderr+"\n"+parseErr.Error(), r.cfg.OutputLimit)
		resp := Response{
			Stdout:     stdoutOut.Text,
			Stderr:     stderrOut.Text,
			ReturnCode: 1,
			CurrentCWD: state.CWD,
			Truncated:  stdoutOut.Truncated || stderrOut.Truncated,
			Error:      parseErr.Error(),
		}
		return r.withVerbose(resp, req.Verbose, sessionID, duration, stdoutOut, stderrOut, 0)
	}

	beforeEnv := state.Env
	updated := r.session.Update(sessionID, trailer.CWD, trailer.Env)
	envChanged := envDiffCount(beforeEnv, updated.Env)

	stdoutOut := output.SanitizeStream(userStdout, r.cfg.OutputLimit)
	stderrOut := output.SanitizeStream(userStderr, r.cfg.OutputLimit)

	returnCode := trailer.ExitCode
	if runErr != nil {
		if exitErr, ok := runErr.(*exec.ExitError); ok {
			returnCode = exitErr.ExitCode()
		} else if returnCode == 0 {
			returnCode = 1
		}
	}

	resp := Response{
		Stdout:     stdoutOut.Text,
		Stderr:     stderrOut.Text,
		ReturnCode: returnCode,
		TimedOut:   false,
		CurrentCWD: updated.CWD,
		Truncated:  stdoutOut.Truncated || stderrOut.Truncated,
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
	resp.SessionID = sessionID
	resp.EnvChangedCount = envChanged
	return resp
}

func wrapCommand(command string) string {
	return fmt.Sprintf(
		`%s
__minight_rc=$?
printf '%%s\n' %q
printf '__MINIGHT_RC=%%s\n' "$__minight_rc"
pwd
env -0
printf '%%s\n' %q
exit $__minight_rc`,
		command,
		trailerBegin,
		trailerEnd,
	)
}

type trailerData struct {
	ExitCode int
	CWD      string
	Env      map[string]string
}

func splitTrailer(stdout, stderr string) (userStdout, userStderr string, trailer trailerData, err error) {
	combined := stdout
	if idx := strings.LastIndex(combined, trailerBegin); idx >= 0 {
		userStdout = combined[:idx]
		block := combined[idx:]
		if endIdx := strings.Index(block, trailerEnd); endIdx >= 0 {
			block = block[:endIdx]
		}
		trailer, err := parseTrailerBlock(block)
		return userStdout, stderr, trailer, err
	}

	if idx := strings.LastIndex(stderr, trailerBegin); idx >= 0 {
		userStderr = stderr[:idx]
		block := stderr[idx:]
		if endIdx := strings.Index(block, trailerEnd); endIdx >= 0 {
			block = block[:endIdx]
		}
		trailer, err := parseTrailerBlock(block)
		return stdout, userStderr, trailer, err
	}

	return stdout, stderr, trailerData{}, fmt.Errorf("missing command trailer")
}

func parseTrailerBlock(block string) (trailerData, error) {
	lines := strings.Split(block, "\n")
	if len(lines) < 3 {
		return trailerData{}, fmt.Errorf("invalid command trailer")
	}

	exitCode := 0
	cwd := ""
	envStart := 0

	for i, line := range lines {
		if strings.HasPrefix(line, "__MINIGHT_RC=") {
			rc, err := strconv.Atoi(strings.TrimPrefix(line, "__MINIGHT_RC="))
			if err != nil {
				return trailerData{}, fmt.Errorf("invalid trailer exit code")
			}
			exitCode = rc
			envStart = i + 1
			break
		}
	}

	if envStart >= len(lines) {
		return trailerData{}, fmt.Errorf("invalid trailer cwd")
	}

	cwd = strings.TrimSpace(lines[envStart])
	if cwd == "" {
		return trailerData{}, fmt.Errorf("invalid trailer cwd")
	}

	envBlob := strings.Join(lines[envStart+1:], "\n")
	envBlob = strings.TrimSuffix(envBlob, trailerEnd)
	envBlob = strings.TrimSpace(envBlob)

	env, err := parseEnvZero(envBlob)
	if err != nil {
		return trailerData{}, err
	}

	return trailerData{ExitCode: exitCode, CWD: cwd, Env: env}, nil
}

func parseEnvZero(blob string) (map[string]string, error) {
	env := make(map[string]string)
	if blob == "" {
		return env, nil
	}
	parts := strings.Split(blob, "\x00")
	for _, part := range parts {
		if part == "" {
			continue
		}
		key, value, ok := strings.Cut(part, "=")
		if !ok {
			return nil, fmt.Errorf("invalid env entry")
		}
		env[key] = value
	}
	return env, nil
}

func buildEnv(sessionEnv map[string]string) []string {
	base := make(map[string]string)
	for _, entry := range os.Environ() {
		key, value, ok := strings.Cut(entry, "=")
		if ok {
			base[key] = value
		}
	}
	for key, value := range sessionEnv {
		base[key] = value
	}
	out := make([]string, 0, len(base))
	for key, value := range base {
		out = append(out, key+"="+value)
	}
	return out
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
