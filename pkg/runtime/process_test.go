package runtime

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/sst/sst/v3/pkg/process"
)

func TestProcessWorkerNaturalExitDrainsLogs(t *testing.T) {
	worker, err := StartWorker(context.Background(), processWorkerTestCommand("output"))
	if err != nil {
		t.Fatalf("failed to start worker: %v", err)
	}

	output, err := io.ReadAll(worker.Logs())
	if err != nil {
		t.Fatalf("failed to read logs: %v", err)
	}
	for _, expected := range []string{"stdout line", "stderr line"} {
		if !strings.Contains(string(output), expected) {
			t.Errorf("missing %q in logs %q", expected, output)
		}
	}

	worker.Stop()
	worker.Stop()
}

func TestProcessWorkerStopsOnContextCancel(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	worker, err := StartWorker(ctx, processWorkerTestCommand("block"))
	if err != nil {
		t.Fatalf("failed to start worker: %v", err)
	}

	cancel()
	managed := worker.(*processWorker).managed
	select {
	case <-managed.Done():
	case <-time.After(5 * time.Second):
		t.Fatal("worker was not reaped after context cancellation")
	}
	worker.Stop()
}

func TestProcessWorkerStopsWithoutLogConsumer(t *testing.T) {
	worker, err := StartWorker(context.Background(), processWorkerTestCommand("flood"))
	if err != nil {
		t.Fatalf("failed to start worker: %v", err)
	}

	stopped := make(chan struct{})
	go func() {
		worker.Stop()
		close(stopped)
	}()
	select {
	case <-stopped:
	case <-time.After(5 * time.Second):
		t.Fatal("Stop blocked without a log consumer")
	}
	_ = worker.Logs().Close()
}
func TestProcessWorkerClosesLogsHeldByDescendant(t *testing.T) {
	worker, err := StartWorker(context.Background(), processWorkerTestCommand("inherit"))
	if err != nil {
		t.Fatalf("failed to start worker: %v", err)
	}

	logs := worker.Logs()
	defer logs.Close()
	reader := bufio.NewReader(logs)
	line, err := reader.ReadString('\n')
	if err != nil {
		t.Fatalf("failed to read descendant pid: %v", err)
	}
	var pid int
	if _, err := fmt.Sscanf(line, "%d", &pid); err != nil {
		t.Fatalf("failed to parse descendant pid from %q: %v", line, err)
	}
	descendant, err := os.FindProcess(pid)
	if err != nil {
		t.Fatalf("failed to find descendant: %v", err)
	}
	defer descendant.Kill()

	drained := make(chan error, 1)
	go func() {
		_, err := io.Copy(io.Discard, reader)
		drained <- err
	}()
	select {
	case <-drained:
	case <-time.After(3 * time.Second):
		t.Fatal("worker logs remained open after the worker exited")
	}
}

func TestProcessWorkerConcurrentStop(t *testing.T) {
	worker, err := StartWorker(context.Background(), processWorkerTestCommand("block"))
	if err != nil {
		t.Fatalf("failed to start worker: %v", err)
	}

	var wg sync.WaitGroup
	for range 8 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			worker.Stop()
		}()
	}

	done := make(chan struct{})
	go func() {
		wg.Wait()
		close(done)
	}()
	select {
	case <-done:
	case <-time.After(5 * time.Second):
		t.Fatal("concurrent Stop calls blocked")
	}
}

func TestProcessWorkerHelper(t *testing.T) {
	if os.Getenv("SST_PROCESS_WORKER_DESCENDANT") == "1" {
		time.Sleep(time.Minute)
		return
	}

	switch os.Getenv("SST_PROCESS_WORKER_HELPER") {
	case "output":
		fmt.Fprintln(os.Stdout, "stdout line")
		fmt.Fprintln(os.Stderr, "stderr line")
	case "flood":
		line := strings.Repeat("x", 256<<10)
		fmt.Fprintln(os.Stdout, line)
		fmt.Fprintln(os.Stderr, line)
	case "block":
		time.Sleep(time.Minute)
	case "inherit":
		cmd := exec.Command(os.Args[0], "-test.run=^TestProcessWorkerHelper$")
		cmd.Env = append(os.Environ(), "SST_PROCESS_WORKER_DESCENDANT=1")
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		if err := cmd.Start(); err != nil {
			t.Fatalf("failed to start descendant: %v", err)
		}
		fmt.Fprintln(os.Stdout, cmd.Process.Pid)
	}
}

func processWorkerTestCommand(mode string) *exec.Cmd {
	cmd := process.Command(os.Args[0], "-test.run=^TestProcessWorkerHelper$")
	cmd.Env = append(os.Environ(), "SST_PROCESS_WORKER_HELPER="+mode)
	return cmd
}
