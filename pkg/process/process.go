package process

import (
	"context"
	"errors"
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
	managed  = map[*os.Process]*Managed{}
	killWait = 5 * time.Second
)

// Managed owns the wait lifecycle for a started process.
type Managed struct {
	cmd  *exec.Cmd
	done chan struct{}
	stop sync.Mutex
}

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

// Start starts cmd and immediately arranges for its process to be reaped.
func Start(cmd *exec.Cmd) (*Managed, error) {
	result := &Managed{
		cmd:  cmd,
		done: make(chan struct{}),
	}
	if cmd.Cancel != nil {
		cmd.Cancel = func() error {
			go result.Stop()
			return nil
		}
	}

	lock.Lock()
	if err := cmd.Start(); err != nil {
		untrackCommandLocked(cmd)
		lock.Unlock()
		return nil, err
	}
	managed[cmd.Process] = result
	lock.Unlock()

	go result.wait()
	return result, nil
}

// Done is closed after the process has been reaped.
func (m *Managed) Done() <-chan struct{} {
	return m.done
}

// Stop terminates the process through its existing reaper.
func (m *Managed) Stop() error {
	if m == nil {
		return nil
	}

	m.stop.Lock()
	defer m.stop.Unlock()
	select {
	case <-m.done:
		return nil
	default:
	}

	slog.Info("killing process", "pid", m.cmd.Process.Pid)
	return escalatingKill(m.cmd.Process, m.done)
}

func (m *Managed) wait() {
	// The caller owns any pipe readers, so this is the sole wait for the command.
	_ = m.cmd.Wait()
	lock.Lock()
	if owner, ok := managed[m.cmd.Process]; ok && owner == m {
		delete(managed, m.cmd.Process)
	}
	untrackCommandLocked(m.cmd)
	close(m.done)
	lock.Unlock()
}

func reset() {
	lock.Lock()
	defer lock.Unlock()
	cmds = []*exec.Cmd{}
	managed = map[*os.Process]*Managed{}
}

func track(cmd *exec.Cmd) {
	lock.Lock()
	defer lock.Unlock()
	cmds = append(cmds, cmd)
}

func Cleanup() error {
	type trackedProcess struct {
		process *os.Process
		managed *Managed
	}

	lock.Lock()
	processes := make([]trackedProcess, 0, len(cmds))
	for _, cmd := range cmds {
		if cmd.Process == nil {
			continue
		}
		owner := managed[cmd.Process]
		if owner != nil {
			processes = append(processes, trackedProcess{
				process: cmd.Process,
				managed: owner,
			})
			continue
		}
		if cmd.ProcessState != nil {
			continue
		}
		processes = append(processes, trackedProcess{
			process: cmd.Process,
			managed: nil,
		})
	}
	lock.Unlock()

	var wg sync.WaitGroup
	errorsCh := make(chan error, len(processes))

	for _, item := range processes {
		wg.Add(1)
		go func(item trackedProcess) {
			defer wg.Done()
			var err error
			if item.managed != nil {
				err = item.managed.Stop()
			} else {
				err = Kill(item.process)
			}
			if err != nil {
				errorsCh <- err
			}
		}(item)
	}

	done := make(chan struct{})
	go func() {
		wg.Wait()
		close(done)
		close(errorsCh)
	}()

	select {
	case <-done:
		for err := range errorsCh {
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

	lock.Lock()
	owner := managed[process]
	lock.Unlock()
	if owner != nil {
		return owner.Stop()
	}

	slog.Info("killing process", "pid", process.Pid)
	done := make(chan struct{})
	go func() {
		_, _ = process.Wait()
		close(done)
	}()

	killErr := escalatingKill(process, done)
	untrackProcess(process)
	return killErr
}

func escalatingKill(process *os.Process, done <-chan struct{}) error {
	if err := sendTermSignal(process); err != nil {
		if errors.Is(err, os.ErrProcessDone) {
			select {
			case <-done:
				return nil
			case <-time.After(killWait):
				return os.ErrProcessDone
			}
		}
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
		if errors.Is(err, os.ErrProcessDone) {
			select {
			case <-done:
				return nil
			case <-time.After(killWait):
				return os.ErrProcessDone
			}
		}
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

func untrackProcess(process *os.Process) {
	lock.Lock()
	defer lock.Unlock()
	for i := len(cmds) - 1; i >= 0; i-- {
		if cmds[i].Process == process {
			cmds[i] = cmds[len(cmds)-1]
			cmds = cmds[:len(cmds)-1]
			return
		}
	}
	slog.Info("process not found in tracked list", "pid", process.Pid)
}

func untrackCommandLocked(cmd *exec.Cmd) {
	for i := len(cmds) - 1; i >= 0; i-- {
		if cmds[i] == cmd {
			cmds[i] = cmds[len(cmds)-1]
			cmds = cmds[:len(cmds)-1]
			return
		}
	}
}
