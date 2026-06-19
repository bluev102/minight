package pathutil

import (
	"os"
	"regexp"
	"strings"
)

var wslShorthand = regexp.MustCompile(`^/([a-zA-Z])(/|$)`)

// InWSL reports whether the process appears to be running inside WSL.
func InWSL() bool {
	if os.Getenv("WSL_DISTRO_NAME") != "" {
		return true
	}
	data, err := os.ReadFile("/proc/version")
	if err != nil {
		return false
	}
	return strings.Contains(strings.ToLower(string(data)), "microsoft")
}

// NormalizeWSLPath converts shorthand drive paths such as /e/foo to /mnt/e/foo.
func NormalizeWSLPath(path string) (normalized string, changed bool) {
	if path == "" || !InWSL() {
		return path, false
	}
	m := wslShorthand.FindStringSubmatch(path)
	if m == nil {
		return path, false
	}
	drive := strings.ToLower(m[1])
	suffix := strings.TrimPrefix(path, "/"+m[1])
	return "/mnt/" + drive + suffix, true
}

// IsWSLDrvfsMount reports whether path is on a WSL drvfs mount (/mnt/<drive>/...).
func IsWSLDrvfsMount(path string) bool {
	if !InWSL() {
		return false
	}
	if !strings.HasPrefix(path, "/mnt/") {
		return false
	}
	rest := strings.TrimPrefix(path, "/mnt/")
	if len(rest) < 2 || rest[1] != '/' {
		return false
	}
	drive := rest[0]
	return drive >= 'a' && drive <= 'z' || drive >= 'A' && drive <= 'Z'
}
