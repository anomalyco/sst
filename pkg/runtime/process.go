package runtime

import (
	"context"
	"io"
	"os"
	"os/exec"
	"sync"
	"time"

	"github.com/sst/sst/v3/pkg/process"
)

const workerLogDrainIdle = time.Second

type workerLogs struct {
	*os.File
	activity chan struct{}
	done     chan struct{}
	drained  sync.Once
	closed   sync.Once
	closeErr error
}

func newWorkerLogs(file *os.File) *workerLogs {
	return &workerLogs{
		File:     file,
		activity: make(chan struct{}, 1),
		done:     make(chan struct{}),
	}
}

func (l *workerLogs) Read(p []byte) (int, error) {
	n, err := l.File.Read(p)
	if n > 0 {
		select {
		case l.activity <- struct{}{}:
		default:
		}
	}
	if err != nil {
		l.markDrained()
	}
	return n, err
}

func (l *workerLogs) Close() error {
	l.closed.Do(func() {
		l.closeErr = l.File.Close()
		l.markDrained()
	})
	return l.closeErr
}

func (l *workerLogs) drain() {
	timer := time.NewTimer(workerLogDrainIdle)
	defer timer.Stop()
	for {
		select {
		case <-l.done:
			_ = l.Close()
			return
		case <-l.activity:
			if !timer.Stop() {
				select {
				case <-timer.C:
				default:
				}
			}
			timer.Reset(workerLogDrainIdle)
		case <-timer.C:
			_ = l.Close()
			return
		}
	}
}

func (l *workerLogs) markDrained() {
	l.drained.Do(func() {
		close(l.done)
	})
}

type processWorker struct {
	logs    *workerLogs
	managed *process.Managed
}

// StartWorker starts cmd and returns a worker whose process is always reaped.
func StartWorker(ctx context.Context, cmd *exec.Cmd) (Worker, error) {
	reader, writer, err := os.Pipe()
	if err != nil {
		return nil, err
	}
	logs := newWorkerLogs(reader)
	cmd.Stdout = writer
	cmd.Stderr = writer

	managed, err := process.Start(cmd)
	_ = writer.Close()
	if err != nil {
		_ = logs.Close()
		return nil, err
	}

	w := &processWorker{
		logs:    logs,
		managed: managed,
	}

	go func() {
		select {
		case <-ctx.Done():
			_ = managed.Stop()
			<-managed.Done()
		case <-managed.Done():
		}
		logs.drain()
	}()

	return w, nil
}

func (w *processWorker) Stop() {
	_ = w.managed.Stop()
}

func (w *processWorker) Logs() io.ReadCloser {
	return w.logs
}
