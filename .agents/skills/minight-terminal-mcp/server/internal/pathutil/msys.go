package pathutil

import (
	"regexp"
	"runtime"
	"strings"
)

var msysShorthand = regexp.MustCompile(`^/([a-zA-Z])(/|$)`)

// NormalizeMSYSPath converts Git Bash/MSYS drive paths such as /e/foo to E:/foo on Windows.
// On non-Windows platforms the path is returned unchanged.
func NormalizeMSYSPath(path string) (normalized string, changed bool) {
	if runtime.GOOS != "windows" || path == "" {
		return path, false
	}
	if len(path) >= 2 && path[1] == ':' {
		return path, false
	}
	if strings.HasPrefix(path, `\\`) {
		return path, false
	}
	m := msysShorthand.FindStringSubmatch(path)
	if m == nil {
		return path, false
	}
	drive := strings.ToUpper(m[1])
	suffix := strings.TrimPrefix(path, "/"+m[1])
	if suffix == "" {
		suffix = "/"
	}
	return drive + ":" + strings.ReplaceAll(suffix, `\`, `/`), true
}

// HostCWDForExec returns a path suitable for os.Stat and exec.Command Dir on the current OS.
func HostCWDForExec(path string, normalizeWSLPaths bool) string {
	if path == "" {
		return path
	}
	if normalizeWSLPaths {
		if normalized, _ := NormalizeWSLPath(path); normalized != path {
			path = normalized
		}
	}
	if normalized, changed := NormalizeMSYSPath(path); changed {
		return normalized
	}
	return path
}
