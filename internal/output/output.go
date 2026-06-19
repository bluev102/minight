package output

import (
	"regexp"
	"strings"
)

var ansiPattern = regexp.MustCompile(`\x1b\[[0-9;]*[a-zA-Z]|\x1b\][^\x07]*(\x07|\x1b\\)|\x1b[PX^_][^\x1b]*\x1b\\|\x1b.|\x9b[0-9;]*[a-zA-Z]`)

type TruncatedOutput struct {
	Text         string
	Truncated    bool
	OmittedBytes int
}

func StripANSI(s string) string {
	return ansiPattern.ReplaceAllString(s, "")
}

func TruncateHeadTail(s string, limit int) TruncatedOutput {
	if limit <= 0 || len(s) <= limit {
		return TruncatedOutput{Text: s}
	}

	marker := "\n...[truncated]...\n"
	markerLen := len(marker)
	if limit <= markerLen+2 {
		text := s[:limit]
		return TruncatedOutput{
			Text:         text,
			Truncated:    true,
			OmittedBytes: len(s) - len(text),
		}
	}

	remaining := limit - markerLen
	head := remaining / 2
	tail := remaining - head
	text := s[:head] + marker + s[len(s)-tail:]

	return TruncatedOutput{
		Text:         text,
		Truncated:    true,
		OmittedBytes: len(s) - head - tail,
	}
}

func SanitizeStream(s string, limit int) TruncatedOutput {
	clean := strings.TrimRight(StripANSI(s), "\r\n")
	return TruncateHeadTail(clean, limit)
}
