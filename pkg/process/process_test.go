package process

import (
	"context"
	"os"
	"sync"
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

func TestManagedNaturalExitIsReaped(t *testing.T) {
	reset()
	cmd := Command("true")
	managed, err := Start(cmd)
	if err != nil {
		t.Fatalf("failed to start: %v", err)
	}

	waitForDone(t, managed.Done())
	if err := cmd.Process.Signal(os.Interrupt); err == nil {
		t.Fatal("process still signalable after it was reaped")
	}
	assertTrackedCount(t, 0)
}

func TestManagedStopIsConcurrentAndIdempotent(t *testing.T) {
	reset()
	cmd := Command("sleep", "60")
	managed, err := Start(cmd)
	if err != nil {
		t.Fatalf("failed to start: %v", err)
	}

	var wg sync.WaitGroup
	errs := make(chan error, 10)
	for range 10 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			errs <- managed.Stop()
		}()
	}
	wg.Wait()
	close(errs)
	for err := range errs {
		if err != nil {
			t.Fatalf("Stop returned error: %v", err)
		}
	}
	if err := managed.Stop(); err != nil {
		t.Fatalf("repeated Stop returned error: %v", err)
	}
	waitForDone(t, managed.Done())
	assertTrackedCount(t, 0)
}

func TestManagedStopRacesNaturalExit(t *testing.T) {
	for range 20 {
		reset()
		cmd := Command("true")
		managed, err := Start(cmd)
		if err != nil {
			t.Fatalf("failed to start: %v", err)
		}

		var wg sync.WaitGroup
		errs := make(chan error, 4)
		for range 4 {
			wg.Add(1)
			go func() {
				defer wg.Done()
				errs <- managed.Stop()
			}()
		}
		wg.Wait()
		close(errs)
		for err := range errs {
			if err != nil {
				t.Fatalf("Stop returned error: %v", err)
			}
		}
		waitForDone(t, managed.Done())
	}
}

func TestCleanupStopsManagedProcess(t *testing.T) {
	reset()
	cmd := Command("sleep", "60")
	managed, err := Start(cmd)
	if err != nil {
		t.Fatalf("failed to start: %v", err)
	}

	if err := Cleanup(); err != nil {
		t.Fatalf("Cleanup returned error: %v", err)
	}
	waitForDone(t, managed.Done())
	assertTrackedCount(t, 0)
}
func TestManagedCommandContextCancellation(t *testing.T) {
	reset()
	ctx, cancel := context.WithCancel(context.Background())
	cmd := CommandContext(ctx, "sleep", "60")
	managed, err := Start(cmd)
	if err != nil {
		t.Fatalf("failed to start: %v", err)
	}

	cancel()
	select {
	case <-managed.Done():
	case <-time.After(500 * time.Millisecond):
		t.Fatal("managed command cancellation blocked on its own Wait")
	}
	assertTrackedCount(t, 0)
}

func waitForDone(t *testing.T, done <-chan struct{}) {
	t.Helper()
	select {
	case <-done:
	case <-time.After(5 * time.Second):
		t.Fatal("timed out waiting for process to be reaped")
	}
}

func assertTrackedCount(t *testing.T, expected int) {
	t.Helper()
	lock.Lock()
	defer lock.Unlock()
	if len(cmds) != expected {
		t.Fatalf("expected %d tracked commands, got %d", expected, len(cmds))
	}
	if len(managed) != expected {
		t.Fatalf("expected %d managed commands, got %d", expected, len(managed))
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
