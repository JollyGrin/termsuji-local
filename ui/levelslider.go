package ui

import (
	"fmt"

	"github.com/gdamore/tcell/v2"
)

// LevelSlider is a horizontal slider component for selecting a level.
type LevelSlider struct {
	label    string
	min      int
	max      int
	value    int
	focused  bool
	onChange func(int)
}

// NewLevelSlider creates a new level slider.
func NewLevelSlider(label string, min, max, initial int, onChange func(int)) *LevelSlider {
	return &LevelSlider{
		label:    label,
		min:      min,
		max:      max,
		value:    initial,
		onChange: onChange,
	}
}

// SetFocused sets the focus state.
func (s *LevelSlider) SetFocused(focused bool) {
	s.focused = focused
}

// HandleKey processes keyboard input. Returns true if handled.
func (s *LevelSlider) HandleKey(event *tcell.EventKey) bool {
	switch event.Key() {
	case tcell.KeyLeft:
		if s.value > s.min {
			s.value--
			if s.onChange != nil {
				s.onChange(s.value)
			}
		}
		return true
	case tcell.KeyRight:
		if s.value < s.max {
			s.value++
			if s.onChange != nil {
				s.onChange(s.value)
			}
		}
		return true
	}
	return false
}

// Draw renders the slider component.
// Returns the number of rows used.
func (s *LevelSlider) Draw(screen tcell.Screen, x, y, width int) int {
	bgStyle := tcell.StyleDefault.Background(MenuColors.CardBG)
	labelStyle := tcell.StyleDefault.Foreground(MenuColors.Label).Background(MenuColors.CardBG)
	accentStyle := tcell.StyleDefault.Foreground(MenuColors.TitleAccent).Background(MenuColors.CardBG)
	selectedStyle := tcell.StyleDefault.Foreground(MenuColors.Selected).Background(MenuColors.CardBG)
	unselectedStyle := tcell.StyleDefault.Foreground(MenuColors.Unselected).Background(MenuColors.CardBG)

	col := x

	// Focus cursor
	if s.focused {
		screen.SetContent(col, y, '▸', nil, selectedStyle)
	} else {
		screen.SetContent(col, y, ' ', nil, bgStyle)
	}
	col += 2

	// Label with diamond prefix: ◈ Strength
	screen.SetContent(col, y, '◈', nil, accentStyle)
	col += 2

	for _, ch := range s.label {
		screen.SetContent(col, y, ch, nil, labelStyle)
		col++
	}
	col += 3 // spacing

	// Left arrow
	arrowStyle := unselectedStyle
	if s.focused {
		arrowStyle = selectedStyle
	}
	screen.SetContent(col, y, '◀', nil, arrowStyle)
	col += 2

	// Progress bar
	barWidth := s.max - s.min + 1
	filled := s.value - s.min + 1

	for i := 0; i < barWidth; i++ {
		char := '░'
		style := unselectedStyle
		if i < filled {
			char = '█'
			style = selectedStyle
		}
		screen.SetContent(col, y, char, nil, style)
		col++
	}
	col++

	// Value display
	valueStr := fmt.Sprintf("%d", s.value)
	for _, ch := range valueStr {
		screen.SetContent(col, y, ch, nil, labelStyle)
		col++
	}
	col++

	// Right arrow
	screen.SetContent(col, y, '▶', nil, arrowStyle)

	return 1
}

// Value returns the current slider value.
func (s *LevelSlider) Value() int {
	return s.value
}

// SetValue sets the slider value.
func (s *LevelSlider) SetValue(v int) {
	if v >= s.min && v <= s.max {
		s.value = v
		if s.onChange != nil {
			s.onChange(s.value)
		}
	}
}
