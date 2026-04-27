package monoplexer

import (
	"strings"
	"testing"
	"time"

	"github.com/sst/sst/v3/pkg/process"
)

func TestAddProcessAutostartFalse(t *testing.T) {
	m := New()
	m.AddProcess("web", []string{"sh", "-c", "sleep 5"}, "", "Web", false)

	proc := m.processes["web"]
	if proc == nil {
		t.Fatal("expected process to be registered")
	}
	if proc.started {
		t.Fatal("expected process to remain stopped")
	}

	select {
	case line := <-m.lines:
		if line.process != "web" {
			t.Fatalf("expected message for web, got %q", line.process)
		}
		if !strings.Contains(line.line, "auto-start disabled") {
			t.Fatalf("expected disabled message, got %q", line.line)
		}
	case <-time.After(time.Second):
		t.Fatal("expected disabled message")
	}
}

func TestAddProcessAutostartTrue(t *testing.T) {
	m := New()
	m.AddProcess("web", []string{"sh", "-c", "sleep 5"}, "", "Web", true)

	proc := m.processes["web"]
	if proc == nil {
		t.Fatal("expected process to be registered")
	}
	if !proc.started {
		t.Fatal("expected process to start")
	}
	if proc.cmd == nil || proc.cmd.Process == nil {
		t.Fatal("expected child process to be running")
	}

	defer process.Kill(proc.cmd.Process)
}

func TestAddProcessStartsWhenAutostartEnabledLater(t *testing.T) {
	m := New()
	m.AddProcess("web", []string{"sh", "-c", "sleep 5"}, "", "Web", false)
	<-m.lines

	m.AddProcess("web", []string{"sh", "-c", "sleep 5"}, "", "Web", true)

	proc := m.processes["web"]
	if proc == nil {
		t.Fatal("expected process to be registered")
	}
	if !proc.started {
		t.Fatal("expected process to start when autostart becomes enabled")
	}
	if proc.cmd == nil || proc.cmd.Process == nil {
		t.Fatal("expected child process to be running")
	}

	defer process.Kill(proc.cmd.Process)
}

func TestAddProcessConfigChangeKeepsProcessStopped(t *testing.T) {
	m := New()
	m.AddProcess("web", []string{"sh", "-c", "sleep 5"}, "", "Web", false)
	<-m.lines

	m.AddProcess("web", []string{"sh", "-c", "sleep 10"}, "", "Web", false)

	proc := m.processes["web"]
	if proc == nil {
		t.Fatal("expected process to be registered")
	}
	if proc.started {
		t.Fatal("expected process to remain stopped after config change")
	}

	select {
	case line := <-m.lines:
		if !strings.Contains(line.line, "auto-start disabled") {
			t.Fatalf("expected disabled message, got %q", line.line)
		}
	case <-time.After(time.Second):
		t.Fatal("expected disabled message after config change")
	}
}
