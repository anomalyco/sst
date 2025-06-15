package monoplexer

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"os/exec"

	"github.com/sst/sst/v3/pkg/process"
)

type Monoplexer struct {
	processes map[string]*Process
	lines     chan Line
}

type Line struct {
	process string
	line    string
}

type Process struct {
	name  string
	title string
	cmd   *exec.Cmd
	dir   string
}

func (p *Process) IsDifferent(title string, command []string, directory string) bool {
	if len(command) != len(p.cmd.Args) {
		return true
	}
	for i := range command {
		if command[i] != p.cmd.Args[i] {
			return true
		}
	}
	if title != p.title {
		return true
	}
	if directory != p.dir {
		return true
	}
	return false
}

func New() *Monoplexer {
	return &Monoplexer{
		processes: map[string]*Process{},
		lines:     make(chan Line),
	}
}

func (m *Monoplexer) AddProcess(name string, command []string, directory string, title string) {
	exists, ok := m.processes[name]
	if ok {
		if !exists.IsDifferent(title, command, directory) {
			return
		}
		m.lines <- Line{
			line:    "dev config changed, restarting...",
			process: name,
		}
		process.Kill(exists.cmd.Process)
		delete(m.processes, name)
	}

	r, pipeWriter := io.Pipe()

	var logWriter io.Writer = pipeWriter
	if directory != "" {
		logDir := filepath.Join(directory, ".sst", "log")
		os.MkdirAll(logDir, 0o755)
		file, err := os.Create(filepath.Join(logDir, name+".log"))
		if err == nil {
			logWriter = io.MultiWriter(pipeWriter, file)
		}
	}

	cmd := process.Command(command[0], command[1:]...)
	cmd.SysProcAttr = getProcAttr()
	cmd.Stdout = logWriter
	cmd.Stderr = logWriter
	if directory != "" {
		cmd.Dir = directory
	}
	go func() {
		// read r line by line
		scanner := bufio.NewScanner(r)
		for scanner.Scan() {
			m.lines <- Line{
				line:    scanner.Text(),
				process: name,
			}
		}
	}()
	cmd.Start()
	m.processes[name] = &Process{
		name:  name,
		title: title,
		cmd:   cmd,
		dir:   directory,
	}
}
func (m *Monoplexer) Start(ctx context.Context) error {
	for {
		select {
		case line := <-m.lines:
			match, ok := m.processes[line.process]
			if !ok {
				continue
			}
			fmt.Println("["+match.title+"]", line.line)
		case <-ctx.Done():
			return nil
		}
	}
}
