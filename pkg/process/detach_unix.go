//go:build !windows
// +build !windows

package process

import (
	"os/exec"
	"syscall"
)

func Detach(cmd *exec.Cmd) {
	if cmd.SysProcAttr == nil {
		cmd.SysProcAttr = &syscall.SysProcAttr{}
	}
	// Ensure the child is the leader of its own process group without
	// clobbering any other existing SysProcAttr fields set by the caller.
	cmd.SysProcAttr.Setpgid = true
}

func DetachSession(cmd *exec.Cmd) {
	if cmd.SysProcAttr == nil {
		cmd.SysProcAttr = &syscall.SysProcAttr{}
	}
	cmd.SysProcAttr.Setsid = true
}
