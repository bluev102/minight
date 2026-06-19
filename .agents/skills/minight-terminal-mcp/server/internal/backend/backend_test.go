package backend

import (
	"strings"
	"testing"
)

func TestParseTrailerBlockPOSIX(t *testing.T) {
	block := trailerBegin + "\n__MINIGHT_RC=3\n__MINIGHT_CWD=/tmp\n__MINIGHT_ANY_FAIL=1\n__MINIGHT_BG=100 101\n" +
		envBegin + "\nFOO=bar\x00BAZ=qux\x00\n" + envEnd + "\n" + trailerEnd
	data, err := parseTrailerBlock(block)
	if err != nil {
		t.Fatalf("parseTrailerBlock() error = %v", err)
	}
	if data.ExitCode != 3 || data.CWD != "/tmp" {
		t.Fatalf("unexpected trailer: %+v", data)
	}
	if !data.HadFailure {
		t.Fatal("expected had_failure")
	}
	if len(data.BackgroundPIDs) != 2 {
		t.Fatalf("BackgroundPIDs = %v", data.BackgroundPIDs)
	}
	if data.Env["FOO"] != "bar" || data.Env["BAZ"] != "qux" {
		t.Fatalf("unexpected env: %+v", data.Env)
	}
}

func TestSplitTrailerFromStdout(t *testing.T) {
	stdout := "hello\n" + trailerBegin + "\n__MINIGHT_RC=0\n__MINIGHT_CWD=/tmp\n__MINIGHT_ANY_FAIL=0\n__MINIGHT_BG=\n" +
		envBegin + "\n" + envEnd + "\n" + trailerEnd
	userStdout, userStderr, data, err := splitTrailer(stdout, "warn")
	if err != nil {
		t.Fatalf("splitTrailer() error = %v", err)
	}
	if strings.TrimSpace(userStdout) != "hello" {
		t.Fatalf("userStdout = %q", userStdout)
	}
	if userStderr != "warn" {
		t.Fatalf("userStderr = %q", userStderr)
	}
	if data.CWD != "/tmp" {
		t.Fatalf("cwd = %q", data.CWD)
	}
}

func TestParseLineEnv(t *testing.T) {
	env := parseLineEnv("FOO=bar\nBAZ=qux\n")
	if env["FOO"] != "bar" || env["BAZ"] != "qux" {
		t.Fatalf("env = %+v", env)
	}
}

func TestWrapPOSIXCommandIncludesExplicitCWD(t *testing.T) {
	wrapped := wrapPOSIXCommand("cd /tmp && ls", "/bin/bash", true, true)
	for _, needle := range []string{"__MINIGHT_CWD=", envBegin, envEnd, "trap"} {
		if !strings.Contains(wrapped, needle) {
			t.Fatalf("wrapped command missing %q: %s", needle, wrapped)
		}
	}
}
