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
	TotalBytes   int
	ShownBytes   int
}

type SanitizeOpts struct {
	Limit     int
	StripCRLF bool
}

func StripANSI(s string) string {
	return ansiPattern.ReplaceAllString(s, "")
}

func StripCRLF(s string) string {
	s = strings.ReplaceAll(s, "\r\n", "\n")
	s = strings.ReplaceAll(s, "\r", "\n")
	return s
}

func TruncateHeadTail(s string, limit int) TruncatedOutput {
	total := len(s)
	if limit <= 0 || total <= limit {
		return TruncatedOutput{Text: s, TotalBytes: total, ShownBytes: total}
	}

	marker := "\n...[truncated]...\n"
	markerLen := len(marker)
	if limit <= markerLen+2 {
		text := s[:limit]
		return TruncatedOutput{
			Text:         text,
			Truncated:    true,
			OmittedBytes: total - len(text),
			TotalBytes:   total,
			ShownBytes:   len(text),
		}
	}

	remaining := limit - markerLen
	head := remaining / 2
	tail := remaining - head
	text := s[:head] + marker + s[total-tail:]

	return TruncatedOutput{
		Text:         text,
		Truncated:    true,
		OmittedBytes: total - head - tail,
		TotalBytes:   total,
		ShownBytes:   head + tail + markerLen,
	}
}

func SanitizeStream(s string, opts SanitizeOpts) TruncatedOutput {
	clean := StripANSI(s)
	if opts.StripCRLF {
		clean = StripCRLF(clean)
	}
	clean = strings.TrimRight(clean, "\n")
	return TruncateHeadTail(clean, opts.Limit)
}
