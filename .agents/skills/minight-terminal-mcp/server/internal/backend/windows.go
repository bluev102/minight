package backend

import (
	"bytes"
	"context"
	"fmt"
	"os/exec"
)

// WindowsBackend executes commands through PowerShell on native Windows.
type WindowsBackend struct {
	ShellPath string
}

func (b *WindowsBackend) Name() string { return "windows" }

func (b *WindowsBackend) Execute(ctx context.Context, opts ExecuteOpts) ExecResult {
	shell := b.ShellPath
	if shell == "" {
		shell = defaultWindowsShell()
	}

	wrapped := wrapWindowsCommand(opts.Command, opts.FailOnAnyError)
	cmdCtx, cancel := context.WithTimeout(ctx, opts.Timeout)
	defer cancel()

	args := []string{"-NoProfile", "-NonInteractive", "-Command", wrapped}
	cmd := exec.CommandContext(cmdCtx, shell, args...)
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
			Stdout:   stdoutBuf.String(),
			Stderr:   stderrBuf.String() + "\ncommand timed out",
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

func defaultWindowsShell() string {
	for _, candidate := range []string{"pwsh", "powershell.exe"} {
		if path, err := exec.LookPath(candidate); err == nil {
			return path
		}
	}
	return "powershell.exe"
}

func wrapWindowsCommand(command string, failOnAnyError bool) string {
	anyFailBlock := `Write-Output ("__MINIGHT_ANY_FAIL=0")`
	if failOnAnyError {
		anyFailBlock = `if ($__minight_rc -ne 0) { Write-Output "__MINIGHT_ANY_FAIL=1" } else { Write-Output "__MINIGHT_ANY_FAIL=0" }`
	}

	return fmt.Sprintf(`$ErrorActionPreference = 'Continue'
%s
$__minight_rc = $LASTEXITCODE
%s
$__minight_bg = @()
Get-CimInstance Win32_Process -Filter "ParentProcessId=$PID" -ErrorAction SilentlyContinue | ForEach-Object { $__minight_bg += $_.ProcessId }
Write-Output '__MINIGHT_TRAILER_BEGIN__'
Write-Output ("__MINIGHT_RC=" + $__minight_rc)
Write-Output ("__MINIGHT_CWD=" + (Get-Location).Path)
Write-Output ("__MINIGHT_BG=" + ($__minight_bg -join ' '))
Write-Output '__MINIGHT_ENV_BEGIN__'
Get-ChildItem Env: | ForEach-Object { Write-Output ("$($_.Name)=$($_.Value)") }
Write-Output '__MINIGHT_ENV_END__'
Write-Output '__MINIGHT_TRAILER_END__'
exit $__minight_rc`, command, anyFailBlock)
}
