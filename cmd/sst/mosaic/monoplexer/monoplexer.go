package monoplexer

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"os"
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
	name    string
	title   string
	command []string
	cmd     *exec.Cmd
	dir     string
	env     []string
	started bool
}

func (p *Process) IsDifferent(title string, command []string, directory string) bool {
	if len(command) != len(p.command) {
		return true
	}
	for i := range command {
		if command[i] != p.command[i] {
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
		lines:     make(chan Line, 32),
	}
}

func (p *Process) start(lines chan Line) {
	r, w := io.Pipe()
	cmd := process.Command(p.command[0], p.command[1:]...)
	cmd.SysProcAttr = getProcAttr()
	cmd.Stdout = w
	cmd.Stderr = w
	if p.dir != "" {
		cmd.Dir = p.dir
	}
	if len(p.env) > 0 {
		cmd.Env = append(os.Environ(), p.env...)
	}
	go func() {
		// read r line by line
		scanner := bufio.NewScanner(r)
		for scanner.Scan() {
			lines <- Line{
				line:    scanner.Text(),
				process: p.name,
			}
		}
	}()
	cmd.Start()
	p.cmd = cmd
	p.started = true
}

func (m *Monoplexer) AddProcess(name string, command []string, directory string, title string, autostart bool, env ...string) {
	exists, ok := m.processes[name]
	if ok {
		if !exists.IsDifferent(title, command, directory) {
			if !exists.started && autostart {
				exists.start(m.lines)
			}
			return
		}
		if exists.started {
			m.lines <- Line{
				line:    "dev config changed, restarting...",
				process: name,
			}
			process.Kill(exists.cmd.Process)
		}
		delete(m.processes, name)
	}

	proc := &Process{
		name:    name,
		title:   title,
		command: command,
		dir:     directory,
		env:     env,
	}
	m.processes[name] = proc
	if autostart {
		proc.start(m.lines)
		return
	}
	m.lines <- Line{
		line:    "auto-start disabled",
		process: name,
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
