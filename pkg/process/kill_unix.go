//go:build !windows
// +build !windows

package process

import (
	"errors"
	"log/slog"
	"os"
	"syscall"
)

func sendSignal(process *os.Process, sig syscall.Signal) error {
	if process == nil || process.Pid <= 0 {
		return errors.New("invalid process")
	}
	if err := syscall.Kill(-process.Pid, sig); err != nil {
		if errors.Is(err, syscall.ESRCH) {
			slog.Info("process already exited, skipping signal", "pid", process.Pid, "signal", sig)
			return nil
		}
		slog.Warn("failed to kill process group, trying individual process", "pid", process.Pid, "signal", sig, "err", err)
		if err := process.Signal(sig); err != nil {
			return err
		}
	} else {
		slog.Info("sent signal to process group", "pid", process.Pid, "signal", sig)
	}

	return nil
}

func sendTermSignal(process *os.Process) error {
	return sendSignal(process, syscall.SIGTERM)
}

func sendKillSignal(process *os.Process) error {
	return sendSignal(process, syscall.SIGKILL)
}
