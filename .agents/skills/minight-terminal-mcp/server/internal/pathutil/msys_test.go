package pathutil

import (
	"runtime"
	"testing"
)

func TestNormalizeMSYSPath(t *testing.T) {
	tests := []struct {
		in      string
		want    string
		changed bool
	}{
		{"/e/work2026/project", "E:/work2026/project", true},
		{"/c/Users/HienLV5", "C:/Users/HienLV5", true},
		{"/e", "E:/", true},
		{"E:/work2026/project", "E:/work2026/project", false},
		{"/tmp/project", "/tmp/project", false},
	}
	for _, tc := range tests {
		got, changed := NormalizeMSYSPath(tc.in)
		if runtime.GOOS != "windows" {
			if changed || got != tc.in {
				t.Fatalf("NormalizeMSYSPath(%q) outside windows = (%q, %v), want unchanged", tc.in, got, changed)
			}
			continue
		}
		if got != tc.want || changed != tc.changed {
			t.Fatalf("NormalizeMSYSPath(%q) = (%q, %v), want (%q, %v)", tc.in, got, changed, tc.want, tc.changed)
		}
	}
}

func TestHostCWDForExecMSYS(t *testing.T) {
	if runtime.GOOS != "windows" {
		got := HostCWDForExec("/e/project", true)
		if got != "/e/project" {
			t.Fatalf("HostCWDForExec() = %q, want unchanged outside windows", got)
		}
		return
	}
	got := HostCWDForExec("/e/project", true)
	if got != "E:/project" {
		t.Fatalf("HostCWDForExec() = %q, want E:/project", got)
	}
}
