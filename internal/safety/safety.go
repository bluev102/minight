package safety

import (
	"fmt"
	"regexp"
	"strings"
)

const maxCommandLength = 100_000

var dangerousPatterns = []*regexp.Regexp{
	regexp.MustCompile(`(?i)\brm\s+(-[^\s]*\s+)*-[^\s]*r[^\s]*\s+/`),
	regexp.MustCompile(`(?i)\brm\s+(-[^\s]*\s+)*-[^\s]*r[^\s]*\s+/\s*$`),
	regexp.MustCompile(`(?i):\(\)\{\s*:\|:&\s*\};:`),
	regexp.MustCompile(`(?i)\bmkfs(\.[a-z0-9]+)?\b`),
	regexp.MustCompile(`(?i)\bdd\s+if=/dev/(zero|random|urandom)\s+of=/dev/[a-z0-9]+`),
	regexp.MustCompile(`(?i)\b(shutdown|reboot|poweroff|halt)\b`),
}

func Check(command string) error {
	trimmed := strings.TrimSpace(command)
	if trimmed == "" {
		return fmt.Errorf("command must not be empty")
	}
	if len(command) > maxCommandLength {
		return fmt.Errorf("command exceeds maximum length")
	}

	lower := strings.ToLower(trimmed)
	for _, pattern := range dangerousPatterns {
		if pattern.MatchString(lower) {
			return fmt.Errorf("command rejected by safety guardrail")
		}
	}
	return nil
}
