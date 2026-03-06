package process

import (
	"os"
	"testing"
	"time"
)

func TestKillNil(t *testing.T) {
	if err := Kill(nil); err != nil {
		t.Fatalf("Kill(nil) returned error: %v", err)
	}
}

func TestKillTerminatesProcess(t *testing.T) {
	reset()
	cmd := Command("sleep", "60")
	if err := cmd.Start(); err != nil {
		t.Fatalf("failed to start: %v", err)
	}
	pid := cmd.Process.Pid

	if err := Kill(cmd.Process); err != nil {
		t.Fatalf("Kill returned error: %v", err)
	}

	// process should no longer exist
	if err := signalProcess(pid); err == nil {
		t.Fatal("process still alive after Kill")
	}
}

func TestCleanupKillsAllTracked(t *testing.T) {
	reset()
	var pids []int
	for i := 0; i < 3; i++ {
		cmd := Command("sleep", "60")
		if err := cmd.Start(); err != nil {
			t.Fatalf("failed to start process %d: %v", i, err)
		}
		pids = append(pids, cmd.Process.Pid)
	}

	if err := Cleanup(); err != nil {
		t.Fatalf("Cleanup returned error: %v", err)
	}

	for _, pid := range pids {
		if err := signalProcess(pid); err == nil {
			t.Errorf("process %d still alive after Cleanup", pid)
		}
	}
}

func TestKillAlreadyExited(t *testing.T) {
	reset()
	cmd := Command("true")
	if err := cmd.Start(); err != nil {
		t.Fatalf("failed to start: %v", err)
	}
	cmd.Wait()
	// should not error on already-exited process
	if err := Kill(cmd.Process); err != nil {
		t.Fatalf("Kill on exited process returned error: %v", err)
	}
}

func TestCommandTracksProcess(t *testing.T) {
	reset()
	cmd := Command("sleep", "60")
	if err := cmd.Start(); err != nil {
		t.Fatalf("failed to start: %v", err)
	}
	defer Kill(cmd.Process)

	lock.Lock()
	count := len(cmds)
	lock.Unlock()

	if count != 1 {
		t.Fatalf("expected 1 tracked command, got %d", count)
	}
}

func TestKillUntracksProcess(t *testing.T) {
	reset()
	cmd := Command("sleep", "60")
	if err := cmd.Start(); err != nil {
		t.Fatalf("failed to start: %v", err)
	}

	Kill(cmd.Process)

	lock.Lock()
	count := len(cmds)
	lock.Unlock()

	if count != 0 {
		t.Fatalf("expected 0 tracked commands after Kill, got %d", count)
	}
}

func signalProcess(pid int) error {
	p, err := os.FindProcess(pid)
	if err != nil {
		return err
	}
	return p.Signal(os.Interrupt)
}

func init() {
	killWait = 1 * time.Second
}
