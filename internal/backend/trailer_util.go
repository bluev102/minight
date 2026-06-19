package backend

import "strings"

func stripPartialTrailer(s string) string {
	if idx := strings.Index(s, trailerBegin); idx >= 0 {
		return s[:idx]
	}
	return s
}
