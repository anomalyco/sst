//go:build !windows
// +build !windows

package process

import (
	"fmt"
	"os"
	"syscall"
	"testing"
	"time"
)

func TestKillTerminatesProcessGroup(t *testing.T) {
	reset()
	// spawn a shell that starts a child; both share the process group
	cmd := Command("sh", "-c", "sleep 60 & wait")
	Detach(cmd)
	if err := cmd.Start(); err != nil {
		t.Fatalf("failed to start: %v", err)
	}
	parentPid := cmd.Process.Pid

	// give child time to spawn
	time.Sleep(100 * time.Millisecond)

	// find a child in the same process group
	childPid := findChildInGroup(t, parentPid)

	if err := Kill(cmd.Process); err != nil {
		t.Fatalf("Kill returned error: %v", err)
	}

	// both parent and child should be dead
	time.Sleep(100 * time.Millisecond)
	if err := syscall.Kill(parentPid, 0); err == nil {
		t.Error("parent still alive after Kill")
	}
	if childPid != 0 {
		if err := syscall.Kill(childPid, 0); err == nil {
			t.Error("child still alive after group Kill")
		}
	}
}

func TestSendSignalInvalidPid(t *testing.T) {
	p := &os.Process{}
	// Pid 0 should be rejected
	if err := sendSignal(p, syscall.SIGTERM); err == nil {
		t.Error("expected error for pid 0")
	}
}

func TestSendSignalAlreadyExited(t *testing.T) {
	cmd := Command("true")
	Detach(cmd)
	if err := cmd.Start(); err != nil {
		t.Fatalf("failed to start: %v", err)
	}
	cmd.Wait()

	// should return nil (ESRCH is swallowed)
	if err := sendSignal(cmd.Process, syscall.SIGTERM); err != nil {
		t.Fatalf("expected nil for exited process, got: %v", err)
	}
}

// findChildInGroup looks for a process whose pgid matches the parent pid.
func findChildInGroup(t *testing.T, parentPid int) int {
	t.Helper()
	// read /proc or use ps to find children
	// fallback: just check the group is killable
	entries, err := os.ReadDir("/proc")
	if err != nil {
		// not on linux (e.g. macOS), skip child check
		return 0
	}
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		var pid int
		if _, err := fmt.Sscanf(e.Name(), "%d", &pid); err != nil {
			continue
		}
		if pid == parentPid {
			continue
		}
		pgid, err := syscall.Getpgid(pid)
		if err != nil {
			continue
		}
		if pgid == parentPid {
			return pid
		}
	}
	return 0
}
