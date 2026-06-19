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
	cfg, err := config.Load()
	if err != nil {
		t.Fatalf("config.Load(): %v", err)
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
}

func TestRunTimeoutDoesNotUpdateSession(t *testing.T) {
	r := testRunner(t)
	home, _ := os.UserHomeDir()
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
	_ = home
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

func TestParseTrailerBlock(t *testing.T) {
	block := trailerBegin + "\n__MINIGHT_RC=3\n/tmp\nFOO=bar\x00BAZ=qux\x00"
	data, err := parseTrailerBlock(block)
	if err != nil {
		t.Fatalf("parseTrailerBlock() error = %v", err)
	}
	if data.ExitCode != 3 || data.CWD != "/tmp" {
		t.Fatalf("unexpected trailer: %+v", data)
	}
	if data.Env["FOO"] != "bar" || data.Env["BAZ"] != "qux" {
		t.Fatalf("unexpected env: %+v", data.Env)
	}
}
