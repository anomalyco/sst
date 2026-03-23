package multiplexer

import (
	"os/exec"

	"github.com/gdamore/tcell/v2"
	tcellterm "github.com/sst/sst/v3/cmd/sst/mosaic/multiplexer/tcell-term"
	"github.com/sst/sst/v3/pkg/process"
)

type PaneConfig struct {
	Key             string
	Args            []string
	Icon            string
	Title           string
	Cwd             string
	Killable        bool
	Autostart       bool
	Env             []string
	Filterable      bool
	FilterTitle     string
	FilterSubtitle  string
	ListOptions     func() []FilterOption
	OnFilterChanged func(string)
}

type pane struct {
	PaneConfig
	filterAvailable bool
	vt              *tcellterm.VT
	dead            bool
	cmd             *exec.Cmd
	filter          string
}

type EventProcess struct {
	tcell.EventTime
	PaneConfig
}

func (s *Multiplexer) AddProcess(cfg PaneConfig) {
	s.screen.PostEventWait(&EventProcess{
		PaneConfig: cfg,
	})
}

func (p *pane) start() error {
	p.cmd = process.Command(p.Args[0], p.Args[1:]...)
	p.cmd.Env = p.Env
	if p.Cwd != "" {
		p.cmd.Dir = p.Cwd
	}
	p.vt.Clear()
	err := p.vt.Start(p.cmd)
	if err != nil {
		return err
	}
	p.dead = false
	return nil
}

func (p *pane) Kill() {
	p.vt.Close()
}

func (s *pane) scrollUp(offset int) {
	s.vt.ScrollUp(offset)
}

func (s *pane) scrollDown(offset int) {
	s.vt.ScrollDown(offset)
}

func (s *pane) scrollReset() {
	s.vt.ScrollReset()
}

func (s *pane) isScrolling() bool {
	return s.vt.IsScrolling()
}

func (s *pane) scrollable() bool {
	return s.vt.Scrollable()
}

func (s *pane) Clear() {
	s.vt.Clear()
	s.vt.ScrollReset()
	s.vt.ClearScrollback()
}
