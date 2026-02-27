//go:build !windows
// +build !windows

package process

import (
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

func TestKillTerminatesSessionGroup(t *testing.T) {
	reset()
	cmd := Command("sh", "-c", "sleep 60 & wait")
	DetachSession(cmd)
	if err := cmd.Start(); err != nil {
		t.Fatalf("failed to start: %v", err)
	}
	parentPid := cmd.Process.Pid

	time.Sleep(100 * time.Millisecond)

	// with Setsid the pgid equals the parent pid, same as Setpgid
	childPid := findChildInGroup(t, parentPid)

	if err := Kill(cmd.Process); err != nil {
		t.Fatalf("Kill returned error: %v", err)
	}

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
	// use syscall.Getpgid to scan for children in the same group
	// check pids near the parent (children are typically allocated sequentially)
	for pid := parentPid + 1; pid < parentPid+50; pid++ {
		pgid, err := syscall.Getpgid(pid)
		if err != nil {
			continue
		}
		if pgid == parentPid && pid != parentPid {
			return pid
		}
	}
	t.Fatal("could not find child process in group")
	return 0
}
