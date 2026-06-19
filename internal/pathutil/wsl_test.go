package pathutil

import "testing"

func TestNormalizeWSLPath(t *testing.T) {
	if InWSL() {
		got, changed := NormalizeWSLPath("/e/work2026/project")
		if !changed || got != "/mnt/e/work2026/project" {
			t.Fatalf("NormalizeWSLPath() = (%q, %v)", got, changed)
		}
		return
	}

	got, changed := NormalizeWSLPath("/e/work2026/project")
	if changed || got != "/e/work2026/project" {
		t.Fatalf("NormalizeWSLPath() outside WSL = (%q, %v)", got, changed)
	}
}

func TestIsWSLDrvfsMount(t *testing.T) {
	if !InWSL() {
		if IsWSLDrvfsMount("/mnt/e/project") {
			t.Fatal("expected false outside WSL")
		}
		return
	}
	if !IsWSLDrvfsMount("/mnt/e/project") {
		t.Fatal("expected drvfs mount detection under WSL")
	}
	if IsWSLDrvfsMount("/tmp/project") {
		t.Fatal("expected /tmp not to be drvfs")
	}
}
