//go:build !windows

package codexrelay

import (
	"os/exec"
	"syscall"
)

// alive reports whether pid is still running.
func alive(pid int) bool {
	err := syscall.Kill(pid, 0)
	return err == nil || err == syscall.EPERM
}

// detach keeps the relay out of jianmu's process group, so it outlives
// jianmu.
func detach(cmd *exec.Cmd) {
	if cmd.SysProcAttr == nil {
		cmd.SysProcAttr = &syscall.SysProcAttr{}
	}
	cmd.SysProcAttr.Setsid = true
}
