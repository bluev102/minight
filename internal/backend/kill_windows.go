//go:build windows

package backend

import "os"

func killBackgroundPID(pid int) {
	proc, err := os.FindProcess(pid)
	if err != nil {
		return
	}
	_ = proc.Kill()
}
