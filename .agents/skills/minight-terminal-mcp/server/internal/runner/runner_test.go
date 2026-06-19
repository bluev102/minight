package runner

import (
	"context"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/minight/minight-terminal/internal/config"
	"github.com/minight/minight-terminal/internal/session"
)

func testRunner(t *testing.T) *Runner {
	t.Helper()
	t.Setenv("MINIGHT_BACKEND", "posix")
	cfg, err := config.Load()
	if err != nil {
		t.Fatalf("config.Load(): %v", err)
	}
	if cfg.ShellPath == "" || !strings.Contains(cfg.ShellPath, "sh") {
		if sh := os.Getenv("SHELL"); sh != "" {
			cfg.ShellPath = sh
		} else {
			cfg.ShellPath = "/bin/bash"
		}
	}
	return New(cfg, session.NewManager())
}

func TestRunPWD(t *testing.T) {
	r := testRunner(t)
	home, _ := os.UserHomeDir()

	resp := r.Run(context.Background(), Request{
		Command:   "pwd",
		SessionID: "test-pwd",
		Timeout:   5 * time.Second,
		CWD:       home,
	})
	if resp.ReturnCode != 0 {
		t.Fatalf("ReturnCode = %d, stderr=%q error=%q", resp.ReturnCode, resp.Stderr, resp.Error)
	}
	if strings.TrimSpace(resp.Stdout) != home {
		t.Fatalf("stdout = %q, want %q", resp.Stdout, home)
	}
}

func TestRunUpdatesCWD(t *testing.T) {
	r := testRunner(t)
	tmp := t.TempDir()

	first := r.Run(context.Background(), Request{
		Command:   "cd " + tmp + " && pwd",
		SessionID: "test-cwd",
		Timeout:   5 * time.Second,
	})
	if first.ReturnCode != 0 {
		t.Fatalf("first ReturnCode = %d", first.ReturnCode)
	}
	if !first.CWDPersisted {
		t.Fatal("expected cwd_persisted=true")
	}

	second := r.Run(context.Background(), Request{
		Command:   "pwd",
		SessionID: "test-cwd",
		Timeout:   5 * time.Second,
	})
	if strings.TrimSpace(second.Stdout) != tmp {
		t.Fatalf("second stdout = %q, want %q", second.Stdout, tmp)
	}
	if second.CurrentCWD != tmp {
		t.Fatalf("CurrentCWD = %q, want %q", second.CurrentCWD, tmp)
	}
}

func TestRunCWDAfterCdAndCommand(t *testing.T) {
	r := testRunner(t)
	tmp := t.TempDir()
	marker := t.TempDir()
	if err := os.WriteFile(tmp+"/pyproject.toml", []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	_ = marker

	first := r.Run(context.Background(), Request{
		Command:   "cd " + tmp + " && ls pyproject.toml",
		SessionID: "repro-cwd",
		Timeout:   5 * time.Second,
	})
	if first.ReturnCode != 0 {
		t.Fatalf("first ReturnCode = %d stderr=%q error=%q", first.ReturnCode, first.Stderr, first.Error)
	}
	if first.CurrentCWD != tmp {
		t.Fatalf("first CurrentCWD = %q, want %q", first.CurrentCWD, tmp)
	}

	second := r.Run(context.Background(), Request{
		Command:   "pwd",
		SessionID: "repro-cwd",
		Timeout:   5 * time.Second,
	})
	if strings.TrimSpace(second.Stdout) != tmp {
		t.Fatalf("second stdout = %q, want %q", second.Stdout, tmp)
	}
}

func TestRunPersistsEnv(t *testing.T) {
	r := testRunner(t)
	sessionID := "test-env"

	first := r.Run(context.Background(), Request{
		Command:   "export MINIGHT_TEST=ok",
		SessionID: sessionID,
		Timeout:   5 * time.Second,
	})
	if first.ReturnCode != 0 {
		t.Fatalf("export ReturnCode = %d", first.ReturnCode)
	}

	second := r.Run(context.Background(), Request{
		Command:   "printenv MINIGHT_TEST",
		SessionID: sessionID,
		Timeout:   5 * time.Second,
	})
	if strings.TrimSpace(second.Stdout) != "ok" {
		t.Fatalf("printenv stdout = %q, want ok", second.Stdout)
	}
}

func TestRunNonZeroStillUpdatesSession(t *testing.T) {
	r := testRunner(t)
	tmp := t.TempDir()
	sessionID := "test-nonzero"

	resp := r.Run(context.Background(), Request{
		Command:   "cd " + tmp + " && sh -c 'exit 7'",
		SessionID: sessionID,
		Timeout:   5 * time.Second,
	})
	if resp.ReturnCode != 7 {
		t.Fatalf("ReturnCode = %d, want 7", resp.ReturnCode)
	}
	if resp.CurrentCWD != tmp {
		t.Fatalf("CurrentCWD = %q, want %q", resp.CurrentCWD, tmp)
	}
	if !resp.HadFailure {
		t.Fatal("expected had_failure=true")
	}
}

func TestRunSemicolonFailureMetadata(t *testing.T) {
	r := testRunner(t)
	if !strings.Contains(r.cfg.ShellPath, "bash") {
		t.Skip("requires bash for fail_on_any_error ERR trap")
	}

	resp := r.Run(context.Background(), Request{
		Command:        "false; echo rc=$?",
		SessionID:      "semicolon-fail",
		Timeout:        5 * time.Second,
		FailOnAnyError: true,
	})
	if resp.ReturnCode != 0 {
		t.Fatalf("return_code = %d, want 0 (final echo succeeds)", resp.ReturnCode)
	}
	if !resp.HadFailure {
		t.Fatal("expected had_failure=true for false; echo")
	}
	if !strings.Contains(resp.Stdout, "rc=1") {
		t.Fatalf("stdout = %q", resp.Stdout)
	}
}

func TestRunPipefail(t *testing.T) {
	r := testRunner(t)
	if !strings.Contains(r.cfg.ShellPath, "bash") {
		t.Skip("requires bash for pipefail")
	}

	resp := r.Run(context.Background(), Request{
		Command:  "false | true",
		SessionID: "pipefail",
		Timeout:  5 * time.Second,
		Pipefail: true,
	})
	if resp.ReturnCode == 0 {
		t.Fatal("expected non-zero return_code with pipefail when false is piped")
	}
}

func TestRunTimeoutDoesNotUpdateSession(t *testing.T) {
	r := testRunner(t)
	sessionID := "test-timeout"

	setup := r.Run(context.Background(), Request{
		Command:   "cd " + t.TempDir(),
		SessionID: sessionID,
		Timeout:   5 * time.Second,
	})
	if setup.ReturnCode != 0 {
		t.Fatalf("setup ReturnCode = %d", setup.ReturnCode)
	}
	beforeCWD := setup.CurrentCWD

	resp := r.Run(context.Background(), Request{
		Command:   "sleep 2",
		SessionID: sessionID,
		Timeout:   200 * time.Millisecond,
	})
	if !resp.TimedOut {
		t.Fatal("expected timed_out=true")
	}
	if resp.CurrentCWD != beforeCWD {
		t.Fatalf("CurrentCWD = %q, want unchanged %q", resp.CurrentCWD, beforeCWD)
	}
	if resp.SuggestedTimeout == 0 {
		t.Fatal("expected suggested_timeout in verbose-less timeout near limit")
	}
}

func TestRunInvalidCWD(t *testing.T) {
	r := testRunner(t)
	resp := r.Run(context.Background(), Request{
		Command:   "pwd",
		SessionID: "test-invalid-cwd",
		CWD:       "/path/that/does/not/exist-minight",
		Timeout:   5 * time.Second,
	})
	if resp.Error == "" {
		t.Fatal("expected error for invalid cwd")
	}
}

func TestRunTracksBackgroundPIDs(t *testing.T) {
	r := testRunner(t)
	resp := r.Run(context.Background(), Request{
		Command:   "(sleep 60 >/dev/null 2>&1 &)",
		SessionID: "bg-track",
		Timeout:   5 * time.Second,
	})
	if resp.ReturnCode != 0 {
		t.Fatalf("ReturnCode = %d stderr=%q error=%q", resp.ReturnCode, resp.Stderr, resp.Error)
	}

	state := r.session.Get("bg-track")
	if len(state.BackgroundPIDs) == 0 {
		t.Fatal("expected background PIDs to be tracked")
	}
}
