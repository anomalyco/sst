package multiplexer

import (
	"fmt"
	"slices"
	"sort"
	"strings"
	"unicode/utf8"

	"github.com/gdamore/tcell/v2"
	"github.com/gdamore/tcell/v2/views"
)

func (s *Multiplexer) draw() {
	defer s.screen.Show()
	s.screen.Clear()
	softGray := tcell.NewRGBColor(138, 138, 138)
	for _, w := range s.stack.Widgets() {
		s.stack.RemoveWidget(w)
	}
	selected := s.selectedProcess()

	for index, item := range s.processes {
		if index > 0 && !s.processes[index-1].dead && item.dead {
			spacer := views.NewTextBar()
			spacer.SetLeft("──────────────────────", tcell.StyleDefault.Foreground(tcell.ColorGray))
			s.stack.AddWidget(spacer, 0)
		}
		style := tcell.StyleDefault
		if item.dead {
			style = style.Foreground(tcell.ColorGray)
		}
		if index == s.selected {
			style = style.Bold(true)
			if !s.focused {
				style = style.Foreground(tcell.ColorOrange)
			}
		}
		label := item.Title
		textStyle := style
		if item.filter != "" {
			label = item.filter
			textStyle = textStyle.Italic(true)
		}
		title := views.NewTextBar()
		title.SetStyle(style)
		title.SetLeft(item.Icon+" "+label, textStyle)
		s.stack.AddWidget(title, 0)
	}
	s.stack.AddWidget(views.NewSpacer(), 1)

	hotkeys := map[string]string{}
	if s.filtering && s.filterSearching {
		hotkeys["esc"] = "cancel"
		hotkeys["enter"] = "confirm"
	} else if s.filtering {
		hotkeys["j/k/↓/↑"] = "move"
		hotkeys["enter"] = "select"
		hotkeys["/"] = "search"
		hotkeys["esc"] = "clear"
	} else {
		if selected != nil && selected.Killable && !s.focused {
			if !selected.dead {
				hotkeys["x"] = "kill"
				hotkeys["enter"] = "focus"
			}

			if selected.dead {
				hotkeys["enter"] = "start"
			}
		}
		if !s.focused {
			hotkeys["j/k/↓/↑"] = "up/down"
		}
		if s.focused {
			hotkeys["ctrl-z"] = "sidebar"
		}
		if selected != nil && selected.vt.HasSelection() {
			hotkeys["enter"] = "copy"
		}
		hotkeys["ctrl-u/d"] = "scroll"
		if selected != nil && selected.isScrolling() {
			hotkeys["ctrl-g"] = "bottom"
		}
		hotkeys["ctrl-l"] = "clear"
		if selected != nil && !s.focused && selected.Filterable && selected.filterAvailable {
			hotkeys["f"] = "filter"
		}
	}
	// sort hotkeys
	keys := make([]string, 0, len(hotkeys))
	for key := range hotkeys {
		keys = append(keys, key)
	}
	slices.SortFunc(keys, func(i, j string) int {
		ilength := utf8.RuneCountInString(i)
		jlength := utf8.RuneCountInString(j)
		if ilength != jlength {
			return ilength - jlength
		}
		return strings.Compare(i, j)
	})
	for _, key := range keys {
		label := hotkeys[key]
		title := views.NewTextBar()
		title.SetStyle(tcell.StyleDefault.Foreground(softGray))
		title.SetLeft(key, tcell.StyleDefault.Foreground(softGray).Bold(true))
		title.SetRight(label+" ", tcell.StyleDefault)
		s.stack.AddWidget(title, 0)
	}
	s.stack.Draw()

	borderStyle := tcell.StyleDefault.Foreground(tcell.ColorGray).Dim(true)
	for i := PAD_HEIGHT; i < s.height-PAD_HEIGHT; i++ {
		s.screen.SetContent(SIDEBAR_WIDTH, i, '│', nil, borderStyle)
	}

	// render virtual terminal
	if selected != nil && !s.filtering {
		selected.vt.Draw()
		if s.focused {
			y, x, _, _ := selected.vt.Cursor()
			s.screen.ShowCursor(s.mainX()+x, y+s.contentY())
		}
		if !s.focused {
			s.screen.HideCursor()
		}
	}

	if s.filtering {
		if !s.filterSearching {
			s.screen.HideCursor()
		}
		s.drawFilterSelect(selected)
	}
}

func (s *Multiplexer) drawFilterSelect(selected *pane) {
	startX := s.mainX()
	startY := s.contentY()
	endY := s.height - s.contentY()
	mainW := s.mainWidth()
	softGray := tcell.NewRGBColor(138, 138, 138)
	dimStyle := tcell.StyleDefault.Foreground(tcell.ColorGray)
	grayStyle := tcell.StyleDefault.Foreground(softGray)
	tealStyle := tcell.StyleDefault.Foreground(tcell.ColorTeal).Bold(true)
	tealDimStyle := tcell.StyleDefault.Foreground(tcell.ColorTeal).Dim(true)
	normalStyle := tcell.StyleDefault.Foreground(tcell.ColorWhite)

	// clear main area
	for y := startY; y < endY; y++ {
		for x := startX; x < s.width; x++ {
			s.screen.SetContent(x, y, ' ', nil, tcell.StyleDefault)
		}
	}

	y := startY
	s.drawLine(startX, y, selected.FilterTitle, tealStyle, mainW)
	y++ // blank
	y++

	if s.filterSearching {
		x := s.drawLine(startX, y, "Search: ", grayStyle, mainW)
		x = s.drawLine(x, y, s.filterQuery, normalStyle, mainW-(x-startX))
		s.screen.ShowCursor(x, y)
	} else if s.filterQuery != "" {
		x := s.drawLine(startX, y, "Search: ", grayStyle, mainW)
		s.drawLine(x, y, s.filterQuery, normalStyle.Italic(true), mainW-(x-startX))
	} else {
		s.drawLine(startX, y, selected.FilterSubtitle, grayStyle, mainW)
	}
	y++ // blank
	y++

	total := len(s.filterFiltered)

	if total > 0 && s.filterScroll > 0 {
		s.drawLine(startX, y, fmt.Sprintf("↑ %d more", s.filterScroll), dimStyle, mainW)
	}
	y++

	if total == 0 {
		s.drawLine(startX, y, "No results found.", dimStyle, mainW)
		return
	}

	visible := s.filterVisibleRows()
	end := min(s.filterScroll+visible, total)

	for fi := s.filterScroll; fi < end && y < endY-1; fi++ {
		opt := s.filterOptions[s.filterFiltered[fi]]
		style := normalStyle
		descStyle := dimStyle
		prefix := "  "
		if fi == s.filterSelected {
			style = tealStyle
			descStyle = tealDimStyle
			prefix = "> "
		}
		x := s.drawLine(startX, y, prefix+opt.Label, style, mainW)
		if opt.Description != "" {
			s.drawLine(x, y, "  "+opt.Description, descStyle, mainW-(x-startX))
		}
		y++
	}

	if end < total && y < endY {
		s.drawLine(startX, y, fmt.Sprintf("↓ %d more", total-end), dimStyle, mainW)
	}
}

func (s *Multiplexer) drawLine(startX, y int, text string, style tcell.Style, maxW int) int {
	x := startX
	for _, ch := range text {
		if x-startX >= maxW {
			break
		}
		s.screen.SetContent(x, y, ch, nil, style)
		x++
	}
	return x
}

func (s *Multiplexer) move(offset int) {
	index := s.selected + offset
	if index < 0 {
		index = 0
	}
	if index >= len(s.processes) {
		index = len(s.processes) - 1
	}
	s.selected = index
	s.draw()
}

func (s *Multiplexer) focus() {
	s.focused = true
	s.draw()
}

func (s *Multiplexer) blur() {
	s.focused = false
	selected := s.selectedProcess()
	if selected != nil {
		selected.scrollReset()
	}
	s.screen.HideCursor()
	s.draw()
}

func (s *Multiplexer) sort() {
	if len(s.processes) == 0 {
		return
	}
	key := s.selectedProcess().Key
	sort.Slice(s.processes, func(i, j int) bool {
		if !s.processes[i].Killable && s.processes[j].Killable {
			return true
		}
		if s.processes[i].Killable && !s.processes[j].Killable {
			return false
		}
		if !s.processes[i].dead && s.processes[j].dead {
			return true
		}
		if s.processes[i].dead && !s.processes[j].dead {
			return false
		}
		return len(s.processes[i].Title) < len(s.processes[j].Title)
	})
	for i, p := range s.processes {
		if p.Key == key {
			s.selected = i
			return
		}
	}
}

func (s *Multiplexer) selectedProcess() *pane {
	if s.selected >= len(s.processes) {
		return nil
	}
	return s.processes[s.selected]
}
