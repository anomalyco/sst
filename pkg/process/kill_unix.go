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
	if process.Pid <= 0 {
		return errors.New("invalid process")
	}
	if err := syscall.Kill(-process.Pid, sig); err != nil {
		if !errors.Is(err, syscall.ESRCH) {
			slog.Warn("failed to kill process group", "pid", process.Pid, "signal", sig, "err", err)
		}
		// Group kill failed — either no such group (ESRCH) or other error.
		// Fall back to signaling the individual process.
		if err := process.Signal(sig); err != nil {
			if errors.Is(err, os.ErrProcessDone) {
				slog.Info("process already exited", "pid", process.Pid, "signal", sig)
				return nil
			}
			return err
		}
		slog.Info("sent signal to individual process", "pid", process.Pid, "signal", sig)
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
