package output

import (
	"strings"
	"testing"
)

func TestStripANSI(t *testing.T) {
	input := "\x1b[31merror\x1b[0m"
	got := StripANSI(input)
	if got != "error" {
		t.Fatalf("StripANSI() = %q, want error", got)
	}
}

func TestTruncateShortOutputUnchanged(t *testing.T) {
	input := "hello"
	got := TruncateHeadTail(input, 100)
	if got.Text != input || got.Truncated || got.OmittedBytes != 0 {
		t.Fatalf("unexpected result: %+v", got)
	}
}

func TestTruncateHeadTailPreservesEnds(t *testing.T) {
	input := strings.Repeat("a", 50) + strings.Repeat("b", 50)
	got := TruncateHeadTail(input, 40)

	if !got.Truncated {
		t.Fatal("expected truncated output")
	}
	if !strings.HasPrefix(got.Text, strings.Repeat("a", 9)) {
		t.Fatalf("missing head prefix: %q", got.Text[:20])
	}
	if !strings.HasSuffix(got.Text, strings.Repeat("b", 9)) {
		t.Fatalf("missing tail suffix: %q", got.Text[len(got.Text)-20:])
	}
	if got.OmittedBytes != 79 {
		t.Fatalf("OmittedBytes = %d, want 79", got.OmittedBytes)
	}
}

func TestSanitizeStream(t *testing.T) {
	input := "\x1b[32m" + strings.Repeat("x", 4000) + "\x1b[0m\n"
	got := SanitizeStream(input, 100)
	if strings.Contains(got.Text, "\x1b") {
		t.Fatal("expected ANSI stripped")
	}
	if !got.Truncated {
		t.Fatal("expected truncated output")
	}
}
