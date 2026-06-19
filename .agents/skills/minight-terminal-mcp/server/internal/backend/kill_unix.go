//go:build !windows

package backend

import "syscall"

func killBackgroundPID(pid int) {
	_ = syscall.Kill(pid, syscall.SIGKILL)
}
