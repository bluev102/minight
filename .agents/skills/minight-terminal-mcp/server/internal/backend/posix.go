package backend

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"strings"
)

const (
	trailerBegin = "__MINIGHT_TRAILER_BEGIN__"
	trailerEnd   = "__MINIGHT_TRAILER_END__"
	envBegin     = "__MINIGHT_ENV_BEGIN__"
	envEnd       = "__MINIGHT_ENV_END__"
)

type trailerData struct {
	ExitCode       int
	CWD            string
	Env            map[string]string
	HadFailure     bool
	BackgroundPIDs []int
}

// POSIXBackend executes commands through a POSIX shell.
type POSIXBackend struct {
	ShellPath string
}

func (b *POSIXBackend) Name() string { return "posix" }

func (b *POSIXBackend) Execute(ctx context.Context, opts ExecuteOpts) ExecResult {
	shell := b.ShellPath
	if shell == "" {
		shell = "/bin/sh"
	}

	wrapped := wrapPOSIXCommand(opts.Command, shell, opts.FailOnAnyError, opts.Pipefail)
	cmdCtx, cancel := context.WithTimeout(ctx, opts.Timeout)
	defer cancel()

	cmd := exec.CommandContext(cmdCtx, shell, "-lc", wrapped)
	cmd.Dir = opts.CWD
	cmd.Env = buildEnv(opts.Env)
	cmd.SysProcAttr = newProcAttr()

	var stdoutBuf, stderrBuf bytes.Buffer
	cmd.Stdout = &stdoutBuf
	cmd.Stderr = &stderrBuf

	if err := cmd.Start(); err != nil {
		return ExecResult{StartErr: fmt.Errorf("failed to start command: %w", err)}
	}

	waitDone := make(chan error, 1)
	go func() { waitDone <- cmd.Wait() }()

	var runErr error
	select {
	case <-cmdCtx.Done():
		if cmd.Process != nil {
			killProcessGroup(cmd.Process.Pid)
		}
		runErr = cmdCtx.Err()
		<-waitDone
	case runErr = <-waitDone:
	}

	timedOut := cmdCtx.Err() == context.DeadlineExceeded
	if timedOut {
		return ExecResult{
			Stdout:   stripPartialTrailer(stdoutBuf.String()),
			Stderr:   stripPartialTrailer(stderrBuf.String()) + "\ncommand timed out",
			ExitCode: 124,
			TimedOut: true,
		}
	}

	userStdout, userStderr, trailer, parseErr := splitTrailer(stdoutBuf.String(), stderrBuf.String())
	if parseErr != nil {
		exitCode := 1
		if runErr != nil {
			if exitErr, ok := runErr.(*exec.ExitError); ok {
				exitCode = exitErr.ExitCode()
			}
		}
		return ExecResult{
			Stdout:   userStdout,
			Stderr:   userStderr + "\n" + parseErr.Error(),
			ExitCode: exitCode,
			StartErr: parseErr,
		}
	}

	exitCode := trailer.ExitCode
	if runErr != nil {
		if exitErr, ok := runErr.(*exec.ExitError); ok {
			exitCode = exitErr.ExitCode()
		} else if exitCode == 0 {
			exitCode = 1
		}
	}

	return ExecResult{
		Stdout:         userStdout,
		Stderr:         userStderr,
		ExitCode:       exitCode,
		CWD:            trailer.CWD,
		Env:            trailer.Env,
		HadFailure:     trailer.HadFailure,
		BackgroundPIDs: trailer.BackgroundPIDs,
	}
}

func wrapPOSIXCommand(command, shell string, failOnAnyError, pipefail bool) string {
	prefix := ""
	trackFailures := failOnAnyError || pipefail
	if strings.Contains(shell, "bash") {
		if trackFailures {
			prefix += "__minight_any_fail=0\n" +
				"trap '__minight_any_fail=1' ERR\n" +
				"set +e\n"
		}
		if pipefail {
			prefix += "set -o pipefail\n"
		}
	}

	suffix := ""
	if trackFailures && strings.Contains(shell, "bash") {
		suffix = "trap - ERR\n"
	}

	anyFailLine := "__minight_any_fail=0\n[ $__minight_rc -ne 0 ] && __minight_any_fail=1\n"
	if trackFailures && strings.Contains(shell, "bash") {
		anyFailLine = ""
	}

	return fmt.Sprintf(
		`%s%s
__minight_rc=$?
__minight_bg_pids=$(pgrep -P $$ 2>/dev/null | paste -sd' ' - || true)
%s%s
printf '%%s\n' %q
printf '__MINIGHT_RC=%%s\n' "$__minight_rc"
printf '__MINIGHT_CWD=%%s\n' "$(pwd)"
printf '__MINIGHT_ANY_FAIL=%%s\n' "$__minight_any_fail"
printf '__MINIGHT_BG=%%s\n' "$__minight_bg_pids"
printf '%%s\n' %q
env -0
printf '%%s\n' %q
printf '%%s\n' %q
exit $__minight_rc`,
		prefix,
		command,
		suffix,
		anyFailLine+"\n",
		trailerBegin,
		envBegin,
		envEnd,
		trailerEnd,
	)
}

func splitTrailer(stdout, stderr string) (userStdout, userStderr string, trailer trailerData, err error) {
	if idx := strings.LastIndex(stdout, trailerBegin); idx >= 0 {
		userStdout = stdout[:idx]
		block := stdout[idx:]
		if endIdx := strings.Index(block, trailerEnd); endIdx >= 0 {
			block = block[:endIdx+len(trailerEnd)]
		}
		trailer, err = parseTrailerBlock(block)
		return userStdout, stderr, trailer, err
	}

	if idx := strings.LastIndex(stderr, trailerBegin); idx >= 0 {
		userStderr = stderr[:idx]
		block := stderr[idx:]
		if endIdx := strings.Index(block, trailerEnd); endIdx >= 0 {
			block = block[:endIdx+len(trailerEnd)]
		}
		trailer, err = parseTrailerBlock(block)
		return stdout, userStderr, trailer, err
	}

	return stdout, stderr, trailerData{}, fmt.Errorf("missing command trailer")
}

func parseTrailerBlock(block string) (trailerData, error) {
	data := trailerData{Env: make(map[string]string)}

	lines := strings.Split(block, "\n")
	for _, line := range lines {
		switch {
		case strings.HasPrefix(line, "__MINIGHT_RC="):
			rc, err := strconv.Atoi(strings.TrimPrefix(line, "__MINIGHT_RC="))
			if err != nil {
				return trailerData{}, fmt.Errorf("invalid trailer exit code")
			}
			data.ExitCode = rc
		case strings.HasPrefix(line, "__MINIGHT_CWD="):
			data.CWD = strings.TrimPrefix(line, "__MINIGHT_CWD=")
		case strings.HasPrefix(line, "__MINIGHT_ANY_FAIL="):
			data.HadFailure = strings.TrimPrefix(line, "__MINIGHT_ANY_FAIL=") == "1"
		case strings.HasPrefix(line, "__MINIGHT_BG="):
			raw := strings.TrimSpace(strings.TrimPrefix(line, "__MINIGHT_BG="))
			if raw != "" {
				for _, part := range strings.Fields(raw) {
					pid, err := strconv.Atoi(part)
					if err == nil && pid > 0 {
						data.BackgroundPIDs = append(data.BackgroundPIDs, pid)
					}
				}
			}
		}
	}

	begin := strings.Index(block, envBegin)
	end := strings.Index(block, envEnd)
	if begin >= 0 && end > begin {
		envBlob := block[begin+len(envBegin) : end]
		envBlob = strings.TrimLeft(envBlob, "\n")
		if strings.Contains(envBlob, "\x00") {
			env, err := parseEnvZero(envBlob)
			if err != nil {
				return trailerData{}, err
			}
			data.Env = env
		} else {
			data.Env = parseLineEnv(envBlob)
		}
	} else if data.CWD == "" {
		return trailerData{}, fmt.Errorf("invalid command trailer")
	}

	if data.CWD == "" {
		return trailerData{}, fmt.Errorf("invalid trailer cwd")
	}

	if data.ExitCode != 0 {
		data.HadFailure = true
	}

	return data, nil
}

func parseLineEnv(blob string) map[string]string {
	env := make(map[string]string)
	for _, line := range strings.Split(blob, "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		key, value, ok := strings.Cut(line, "=")
		if ok {
			env[key] = value
		}
	}
	return env
}

func parseEnvZero(blob string) (map[string]string, error) {
	env := make(map[string]string)
	if blob == "" {
		return env, nil
	}
	parts := strings.Split(blob, "\x00")
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		key, value, ok := strings.Cut(part, "=")
		if !ok {
			continue
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

func KillBackgroundPIDs(pids []int) {
	for _, pid := range pids {
		killBackgroundPID(pid)
	}
}
