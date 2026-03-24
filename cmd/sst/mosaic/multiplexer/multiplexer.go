package multiplexer

import (
	"encoding/base64"
	"fmt"
	"log/slog"
	"os"
	"runtime/debug"
	"strings"
	"syscall"
	"time"

	"github.com/gdamore/tcell/v2"
	"github.com/gdamore/tcell/v2/views"
	tcellterm "github.com/sst/sst/v3/cmd/sst/mosaic/multiplexer/tcell-term"
	"github.com/sst/sst/v3/cmd/sst/mosaic/ui"
	"github.com/sst/sst/v3/pkg/process"
)

var PAD_HEIGHT = 1
var PAD_WIDTH = 1
var MAIN_PAD_WIDTH = 3
var CONTENT_PAD_HEIGHT = 0
var SIDEBAR_WIDTH = 24

type Multiplexer struct {
	focused   bool
	width     int
	height    int
	selected  int
	processes []*pane
	screen    tcell.Screen
	root      *views.ViewPort
	main      *views.ViewPort
	stack     *views.BoxLayout

	dragging     bool
	click        *tcell.EventMouse
	scrollTicker *time.Ticker
	scrollDone   chan struct{}
	scrollDir    int
	lastDragX    int

	filtering       bool
	filterOptions   []FilterOption
	filterFiltered  []int
	filterSelected  int
	filterScroll    int
	filterSearching bool
	filterQuery     string
}

type FilterOption struct {
	Label       string
	Description string
	Value       string
}

func New() (*Multiplexer, error) {
	var err error
	result := &Multiplexer{}
	result.processes = []*pane{}
	result.screen, err = tcell.NewScreen()
	if err != nil {
		return nil, err
	}
	result.screen.Init()
	result.screen.EnableMouse()
	result.screen.Show()
	width, height := result.screen.Size()
	result.width = width
	result.height = height
	result.root = views.NewViewPort(result.screen, 0, 0, 0, 0)
	result.main = views.NewViewPort(result.screen, 0, 0, 0, 0)
	result.stack = views.NewBoxLayout(views.Vertical)
	result.stack.SetView(result.root)
	if os.Getenv("TMUX") != "" {
		process.Command("tmux", "set-option", "-p", "set-clipboard", "on").Run()
	}
	return result, nil
}

func (s *Multiplexer) mainRect() (int, int) {
	return s.width - s.mainX(), s.mainHeight()
}

func (s *Multiplexer) mainX() int {
	return SIDEBAR_WIDTH + 1 + MAIN_PAD_WIDTH
}

func (s *Multiplexer) contentY() int {
	return PAD_HEIGHT + CONTENT_PAD_HEIGHT
}

func (s *Multiplexer) mainHeight() int {
	return max(0, s.height-s.contentY()*2)
}

func (s *Multiplexer) sidebarWidth() int {
	return SIDEBAR_WIDTH - PAD_WIDTH - 1
}

func (s *Multiplexer) resize(width int, height int) {
	s.width = width
	s.height = height
	s.root.Resize(PAD_WIDTH, s.contentY(), s.sidebarWidth(), s.mainHeight())
	s.main.Resize(s.mainX(), s.contentY(), width-s.mainX(), s.mainHeight())
	mw, mh := s.main.Size()
	for _, p := range s.processes {
		p.vt.Resize(mw, mh)
	}
}

func (s *Multiplexer) Start() {
	defer func() {
		s.stopAutoScroll()
		s.screen.Fini()
	}()

	s.resize(s.screen.Size())

	for {
		unknown := s.screen.PollEvent()
		if unknown == nil {
			continue
		}
		shouldBreak := false
		func() {
			defer func() {
				if r := recover(); r != nil {
					slog.Error("mutliplexer panic", "err", r, "stack", string(debug.Stack()))
				}
			}()

			selected := s.selectedProcess()

			switch evt := unknown.(type) {

			case *tcell.EventInterrupt:
				if s.scrollDir != 0 && s.dragging && selected != nil {
					if s.scrollDir < 0 {
						s.scrollUp(1)
						selected.vt.SelectEnd(s.lastDragX, 0)
					} else {
						s.scrollDown(1)
						selected.vt.SelectEnd(s.lastDragX, max(0, s.mainHeight()-1))
					}
				}
				return

			case *EventExit:
				shouldBreak = true
				return

			case *EventCheckFilter:
				for _, p := range s.processes {
					if p.Key != evt.PaneKey || !p.Filterable {
						continue
					}
					p.filterAvailable = len(evt.Names) > 0
					if p.filter == "" {
						continue
					}
					found := false
					for _, name := range evt.Names {
						if name == p.filter {
							found = true
							break
						}
					}
					if !found {
						s.clearPaneFilter(p)
					}
				}
				s.draw()
				return

			case *EventProcess:
				for _, p := range s.processes {
					if p.Key == evt.Key {
						if p.dead && evt.Autostart {
							p.start()
							s.sort()
							s.draw()
						}
						return
					}
				}
				proc := &pane{PaneConfig: evt.PaneConfig}
				term := tcellterm.New()
				term.SetSurface(s.main)
				term.Attach(func(ev tcell.Event) {
					s.screen.PostEvent(ev)
				})
				proc.vt = term
				if evt.Autostart {
					proc.start()
				}
				if !evt.Autostart {
					proc.vt.Start(process.Command("echo", ui.TEXT_DIM.Render(evt.Key+" has auto-start disabled, press enter to start.")))
					proc.dead = true
				}
				s.processes = append(s.processes, proc)
				s.sort()
				s.draw()
				break

			case *tcell.EventMouse:
				if evt.Buttons()&tcell.WheelUp != 0 {
					s.scrollUp(3)
					return
				}
				if evt.Buttons()&tcell.WheelDown != 0 {
					s.scrollDown(3)
					return
				}
				if evt.Buttons() == tcell.ButtonNone {
					s.stopAutoScroll()
					if s.dragging && selected != nil {
						s.copy()
					}
					s.dragging = false
					return
				}
				if evt.Buttons()&tcell.ButtonPrimary != 0 {
					x, y := evt.Position()
					contentY := y - s.contentY()
					maxContentY := max(0, s.mainHeight()-1)
					if x < s.mainX() && s.dragging && selected != nil {
						if y < s.contentY() {
							s.scrollUp(1)
							s.startAutoScroll(-1)
							selected.vt.SelectEnd(0, 0)
							s.draw()
						} else if y >= s.height-s.contentY() {
							s.scrollDown(1)
							s.startAutoScroll(1)
							selected.vt.SelectEnd(0, maxContentY)
							s.draw()
						} else if contentY >= 0 && contentY <= maxContentY {
							s.stopAutoScroll()
							selected.vt.SelectEnd(0, contentY)
							s.draw()
						}
						return
					}
					if x < SIDEBAR_WIDTH && !s.dragging {
						if contentY < 0 || contentY > maxContentY {
							return
						}
						y = contentY
						alive := 0
						for _, p := range s.processes {
							if !p.dead {
								alive++
							}
						}
						if alive != len(s.processes) {
							if y == alive {
								return
							}
							if y > alive {
								y--
							}
						}
						if y >= len(s.processes) {
							return
						}
						s.selected = y
						s.blur()
						return
					}
					if x >= s.mainX() {
						if selected == nil {
							return
						}
						if !s.dragging && (contentY < 0 || contentY > maxContentY) {
							return
						}
						if !s.dragging && s.click != nil && time.Since(s.click.When()) < time.Millisecond*500 {
							oldX, oldY := s.click.Position()
							if oldX == x && oldY == y {
								selected.vt.SelectStart(0, contentY)
								selected.vt.SelectEnd(s.width-1, contentY)
								s.dragging = true
								s.draw()
								return
							}
						}
						s.click = evt
						offsetX := x - s.mainX()
						s.lastDragX = offsetX
						if s.dragging {
							if y < s.contentY() {
								s.scrollUp(1)
								s.startAutoScroll(-1)
							} else if y >= s.height-s.contentY() {
								s.scrollDown(1)
								s.startAutoScroll(1)
							} else {
								s.stopAutoScroll()
							}
							selected.vt.SelectEnd(offsetX, min(max(contentY, 0), maxContentY))
						}
						if !s.dragging {
							s.dragging = true
							selected.vt.SelectStart(offsetX, contentY)
						}
						s.draw()
						return
					}
				}
				break

			case *tcell.EventResize:
				slog.Info("resize")
				s.resize(evt.Size())
				s.draw()
				s.screen.Sync()
				return

			case *tcellterm.EventRedraw:
				if s.filtering {
					return
				}
				if selected != nil && selected.vt == evt.VT() {
					selected.vt.Draw()
					s.screen.Show()
				}
				return

			case *tcellterm.EventClosed:
				for index, proc := range s.processes {
					if proc.vt == evt.VT() {
						if !proc.dead {
							proc.vt.Start(process.Command("echo", "\n"+ui.TEXT_DIM.Render("[process exited]")))
							proc.dead = true
							s.sort()
							if index == s.selected {
								s.blur()
							}
						}
					}
				}
				s.draw()
				return

			case *tcell.EventKey:
				if s.filtering && evt.Key() != tcell.KeyCtrlC {
					s.handleFilterKey(evt)
					return
				}
				switch evt.Key() {
				case 256:
					switch evt.Rune() {
					case 'j':
						if !s.focused {
							s.move(1)
							return
						}
					case 'k':
						if !s.focused {
							s.move(-1)
							return
						}
					case 'x':
						if selected.Killable && !selected.dead && !s.focused {
							selected.Kill()
						}
					case 'f':
						if !s.focused && selected != nil && selected.Filterable && selected.filterAvailable && selected.ListOptions != nil {
							options := selected.ListOptions()
							if len(options) == 0 {
								return
							}
							s.filterOptions = options
							s.filterFiltered = make([]int, len(options))
							for i := range options {
								s.filterFiltered[i] = i
							}
							s.filterSelected = 0
							for i, opt := range s.filterOptions {
								if opt.Value == selected.filter {
									s.filterSelected = i
									break
								}
							}
							s.filterScroll = 0
							s.filterSearching = false
							s.filterQuery = ""
							s.filtering = true
							s.filterEnsureVisible()
							s.draw()
							return
						}
					}
				case tcell.KeyUp:
					if !s.focused {
						s.move(-1)
						return
					}
				case tcell.KeyDown:
					if !s.focused {
						s.move(1)
						return
					}
				case tcell.KeyCtrlU:
					if selected != nil {
						s.scrollUp(s.mainHeight()/2 + 1)
						return
					}
				case tcell.KeyCtrlD:
					if selected != nil {
						s.scrollDown(s.mainHeight()/2 + 1)
						return
					}
				case tcell.KeyEnter:
					if selected != nil && selected.vt.HasSelection() {
						s.copy()
						selected.vt.ClearSelection()
						s.draw()
						return
					}
					if !s.focused {
						if selected.Killable {
							if selected.dead {
								selected.start()
								s.sort()
								s.draw()
								return
							}
							s.focus()
						}
						return
					}
				case tcell.KeyCtrlC:
					if !s.focused {
						s.move(-99999)
						pid := os.Getpid()
						process, _ := os.FindProcess(pid)
						process.Signal(syscall.SIGINT)
						return
					}
				case tcell.KeyEscape:
					if !s.focused && selected != nil && selected.filter != "" {
						s.clearPaneFilter(selected)
						return
					}
				case tcell.KeyCtrlZ:
					if s.focused {
						s.blur()
						return
					}
				case tcell.KeyCtrlG:
					if selected != nil && selected.isScrolling() {
						selected.scrollReset()
						s.draw()
						s.screen.Sync()
						return
					}
				case tcell.KeyCtrlL:
					if selected != nil {
						selected.Clear()
						s.draw()
						return
					}
				}

				if selected != nil && s.focused && !selected.isScrolling() {
					selected.vt.HandleEvent(evt)
					s.draw()
				}
			}
		}()
		if shouldBreak {
			return
		}
	}
}

func (s *Multiplexer) handleFilterKey(evt *tcell.EventKey) {
	if s.filterSearching {
		s.handleFilterSearchKey(evt)
		return
	}
	switch evt.Key() {
	case tcell.KeyEscape:
		selected := s.selectedProcess()
		if selected != nil && selected.filter != "" {
			s.clearPaneFilter(selected)
		}
		s.filtering = false
		s.draw()
	case tcell.KeyEnter:
		if len(s.filterFiltered) == 0 {
			return
		}
		value := s.filterOptions[s.filterFiltered[s.filterSelected]].Value
		selected := s.selectedProcess()
		if selected != nil && selected.filter == value {
			value = ""
		}
		s.applyFilter(value)
		s.filtering = false
		s.draw()
	case tcell.KeyUp:
		if s.filterSelected > 0 {
			s.filterSelected--
			s.filterEnsureVisible()
			s.draw()
		}
	case tcell.KeyDown:
		if s.filterSelected < len(s.filterFiltered)-1 {
			s.filterSelected++
			s.filterEnsureVisible()
			s.draw()
		}
	case tcell.KeyRune:
		switch evt.Rune() {
		case 'j':
			if s.filterSelected < len(s.filterFiltered)-1 {
				s.filterSelected++
				s.filterEnsureVisible()
				s.draw()
			}
		case 'k':
			if s.filterSelected > 0 {
				s.filterSelected--
				s.filterEnsureVisible()
				s.draw()
			}
		case '/':
			s.filterSearching = true
			s.filterQuery = ""
			s.draw()
		}
	}
}

func (s *Multiplexer) handleFilterSearchKey(evt *tcell.EventKey) {
	switch evt.Key() {
	case tcell.KeyEscape:
		s.filterSearching = false
		s.filterQuery = ""
		s.refilterOptions()
		s.draw()
	case tcell.KeyEnter:
		if len(s.filterFiltered) == 1 {
			value := s.filterOptions[s.filterFiltered[0]].Value
			selected := s.selectedProcess()
			if selected != nil && selected.filter == value {
				value = ""
			}
			s.applyFilter(value)
			s.filtering = false
			s.filterSearching = false
			s.draw()
			return
		}
		s.filterSearching = false
		s.draw()
	case tcell.KeyBackspace, tcell.KeyBackspace2:
		if len(s.filterQuery) > 0 {
			s.filterQuery = s.filterQuery[:len(s.filterQuery)-1]
			s.refilterOptions()
			s.draw()
		}
	case tcell.KeyRune:
		s.filterQuery += string(evt.Rune())
		s.refilterOptions()
		s.draw()
	}
}

func (s *Multiplexer) refilterOptions() {
	q := strings.ToLower(s.filterQuery)
	s.filterFiltered = s.filterFiltered[:0]
	for i, opt := range s.filterOptions {
		if q == "" || strings.Contains(strings.ToLower(opt.Label), q) || strings.Contains(strings.ToLower(opt.Description), q) {
			s.filterFiltered = append(s.filterFiltered, i)
		}
	}
	if s.filterSelected >= len(s.filterFiltered) {
		s.filterSelected = max(0, len(s.filterFiltered)-1)
	}
	s.filterEnsureVisible()
}

func (s *Multiplexer) filterVisibleRows() int {
	// header (y=0) + blank + subtitle (y=2) + blank + ↑/blank (y=4) = list starts at y=5
	// reserve 1 row at bottom for ↓ indicator + 6 rows padding
	rows := s.mainHeight() - 5 - 1 - 6
	if rows < 1 {
		rows = 1
	}
	return rows
}

func (s *Multiplexer) filterEnsureVisible() {
	visible := s.filterVisibleRows()
	if s.filterSelected < s.filterScroll {
		s.filterScroll = s.filterSelected
	}
	if s.filterSelected >= s.filterScroll+visible {
		s.filterScroll = s.filterSelected - visible + 1
	}
	if s.filterScroll < 0 {
		s.filterScroll = 0
	}
}

func (s *Multiplexer) clearPaneFilter(p *pane) {
	if p.filter == "" {
		return
	}
	p.filter = ""
	if p.OnFilterChanged != nil {
		p.OnFilterChanged("")
	}
	s.draw()
}

func (s *Multiplexer) applyFilter(value string) {
	selected := s.selectedProcess()
	if selected == nil {
		return
	}
	if selected.filter == value {
		return
	}
	selected.filter = value
	if selected.OnFilterChanged != nil {
		selected.OnFilterChanged(value)
	}
}

type EventExit struct {
	when time.Time
}

func (e *EventExit) When() time.Time {
	return e.when
}

func (s *Multiplexer) Exit() {
	s.screen.PostEvent(&EventExit{})
}

type EventCheckFilter struct {
	tcell.EventTime
	PaneKey string
	Names   []string
}

func (s *Multiplexer) CheckFilter(paneKey string, names []string) {
	s.screen.PostEvent(&EventCheckFilter{PaneKey: paneKey, Names: names})
}

func (s *Multiplexer) stopAutoScroll() {
	if s.scrollTicker != nil {
		s.scrollTicker.Stop()
		close(s.scrollDone)
		s.scrollTicker = nil
		s.scrollDone = nil
		s.scrollDir = 0
	}
}

func (s *Multiplexer) startAutoScroll(dir int) {
	if s.scrollDir == dir && s.scrollTicker != nil {
		return
	}
	s.stopAutoScroll()
	s.scrollDir = dir
	s.scrollTicker = time.NewTicker(50 * time.Millisecond)
	s.scrollDone = make(chan struct{})
	go func() {
		for {
			select {
			case <-s.scrollDone:
				return
			case <-s.scrollTicker.C:
				s.screen.PostEvent(tcell.NewEventInterrupt(nil))
			}
		}
	}()
}

func (s *Multiplexer) scrollDown(n int) {
	selected := s.selectedProcess()
	if selected == nil {
		return
	}
	selected.scrollDown(n)
	s.draw()
	s.screen.Sync()
}

func (s *Multiplexer) scrollUp(n int) {
	selected := s.selectedProcess()
	if selected == nil {
		return
	}
	selected.scrollUp(n)
	s.draw()
}

func (s *Multiplexer) copy() {
	selected := s.selectedProcess()
	if selected == nil {
		return
	}
	data := selected.vt.Copy()
	if data == "" {
		return
	}
	// check if mac terminal
	if os.Getenv("TERM_PROGRAM") == "Apple_Terminal" {
		// use pbcopy
		cmd := process.Command("pbcopy")
		cmd.Stdin = strings.NewReader(data)
		err := cmd.Run()
		if err != nil {
			fmt.Fprintf(os.Stderr, "failed to copy to clipboard: %v\n", err)
		}
		return
	}
	encoded := base64.StdEncoding.EncodeToString([]byte(data))
	fmt.Fprintf(os.Stdout, "\x1b]52;c;%s\x07", encoded)
}
