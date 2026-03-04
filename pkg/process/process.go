package process

import (
	"context"
	"log/slog"
	"os"
	"os/exec"
	"sync"
	"syscall"
	"time"
)

var (
	lock     sync.Mutex
	cmds     = []*exec.Cmd{}
	killWait = 5 * time.Second
)

func Command(name string, args ...string) *exec.Cmd {
	cmd := exec.Command(name, args...)
	track(cmd)
	return cmd
}

func CommandContext(ctx context.Context, name string, args ...string) *exec.Cmd {
	cmd := exec.CommandContext(ctx, name, args...)
	cmd.Cancel = func() error {
		return Kill(cmd.Process)
	}
	track(cmd)
	return cmd
}

func reset() {
	lock.Lock()
	defer lock.Unlock()
	cmds = []*exec.Cmd{}
}

func track(cmd *exec.Cmd) {
	lock.Lock()
	defer lock.Unlock()
	cmds = append(cmds, cmd)
}

func Cleanup() error {
	lock.Lock()
	processes := make([]*exec.Cmd, len(cmds))
	copy(processes, cmds)
	lock.Unlock()

	var wg sync.WaitGroup
	errors := make(chan error, len(processes))

	for _, cmd := range processes {
		if cmd.Process == nil {
			continue
		}
		if cmd.ProcessState != nil {
			continue
		}

		wg.Add(1)
		go func(p *os.Process) {
			defer wg.Done()
			if err := Kill(p); err != nil {
				errors <- err
			}
		}(cmd.Process)
	}

	// Wait for all processes to be killed
	done := make(chan struct{})
	go func() {
		wg.Wait()
		close(done)
		close(errors)
	}()

	select {
	case <-done:
		// Check for any errors
		for err := range errors {
			if err != nil {
				return err
			}
		}
		return nil
	case <-time.After(killWait * 2):
		return syscall.ETIMEDOUT
	}
}

func Kill(process *os.Process) error {
	if process == nil {
		return nil
	}
	slog.Info("killing process", "pid", process.Pid)

	done := make(chan struct{})
	go func() {
		_, _ = process.Wait()
		close(done)
	}()

	killErr := escalatingKill(process, done)
	untrack(process.Pid)
	return killErr
}

func escalatingKill(process *os.Process, done <-chan struct{}) error {
	if err := sendTermSignal(process); err != nil {
		slog.Error("term signal failed, escalating", "pid", process.Pid, "err", err)
		return forceKill(process, done)
	}
	select {
	case <-done:
		slog.Info("process killed with term", "pid", process.Pid)
		return nil
	case <-time.After(killWait):
		slog.Info("term timeout, escalating", "pid", process.Pid)
		return forceKill(process, done)
	}
}

func forceKill(process *os.Process, done <-chan struct{}) error {
	if err := sendKillSignal(process); err != nil {
		slog.Error("kill signal failed", "pid", process.Pid, "err", err)
		return err
	}
	select {
	case <-done:
		slog.Info("process killed with kill", "pid", process.Pid)
		return nil
	case <-time.After(killWait):
		slog.Info("timed out waiting for kill", "pid", process.Pid)
		return syscall.ETIMEDOUT
	}
}

func untrack(pid int) {
	lock.Lock()
	defer lock.Unlock()
	for i := len(cmds) - 1; i >= 0; i-- {
		if cmds[i].Process != nil && cmds[i].Process.Pid == pid {
			cmds[i] = cmds[len(cmds)-1]
			cmds = cmds[:len(cmds)-1]
			return
		}
	}
	slog.Info("process not found in tracked list", "pid", pid)
}
