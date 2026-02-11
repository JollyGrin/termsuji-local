package ui

import (
	"github.com/gdamore/tcell/v2"
)

// RadioOption represents a single radio button option.
type RadioOption struct {
	Label       string
	Description string
}

// RadioSelect is a radio button group component.
type RadioSelect struct {
	label    string
	options  []RadioOption
	selected int
	focused  bool
	onChange func(int)
}

// NewRadioSelect creates a new radio select component.
func NewRadioSelect(label string, options []RadioOption, initial int, onChange func(int)) *RadioSelect {
	return &RadioSelect{
		label:    label,
		options:  options,
		selected: initial,
		onChange: onChange,
	}
}

// SetFocused sets the focus state.
func (r *RadioSelect) SetFocused(focused bool) {
	r.focused = focused
}

// HandleKey processes keyboard input. Returns true if handled.
func (r *RadioSelect) HandleKey(event *tcell.EventKey) bool {
	switch event.Key() {
	case tcell.KeyUp:
		if r.selected > 0 {
			r.selected--
			if r.onChange != nil {
				r.onChange(r.selected)
			}
		}
		return true
	case tcell.KeyDown:
		if r.selected < len(r.options)-1 {
			r.selected++
			if r.onChange != nil {
				r.onChange(r.selected)
			}
		}
		return true
	}
	return false
}

// Draw renders the radio select component.
// Returns the number of rows used.
func (r *RadioSelect) Draw(screen tcell.Screen, x, y, width int) int {
	bgStyle := tcell.StyleDefault.Background(MenuColors.CardBG)
	labelStyle := tcell.StyleDefault.Foreground(MenuColors.Label).Background(MenuColors.CardBG)
	accentStyle := tcell.StyleDefault.Foreground(MenuColors.TitleAccent).Background(MenuColors.CardBG)
	selectedStyle := tcell.StyleDefault.Foreground(MenuColors.Selected).Background(MenuColors.CardBG)
	unselectedStyle := tcell.StyleDefault.Foreground(MenuColors.Unselected).Background(MenuColors.CardBG)
	hintStyle := tcell.StyleDefault.Foreground(MenuColors.Hint).Background(MenuColors.CardBG)

	row := y

	// Draw label with diamond prefix: ◈ Board Size
	col := x
	screen.SetContent(col, row, '◈', nil, accentStyle)
	col += 2

	for _, ch := range r.label {
		screen.SetContent(col, row, ch, nil, labelStyle)
		col++
	}
	row++

	// Draw options
	for i, opt := range r.options {
		col = x + 2 // Indent options

		// Focus cursor
		if r.focused && i == r.selected {
			screen.SetContent(col, row, '▸', nil, selectedStyle)
		} else {
			screen.SetContent(col, row, ' ', nil, bgStyle)
		}
		col += 2

		// Radio button
		style := unselectedStyle
		bullet := '○'
		if i == r.selected {
			bullet = '●'
			style = selectedStyle
		}
		screen.SetContent(col, row, bullet, nil, style)
		col += 2

		// Option label
		for _, ch := range opt.Label {
			screen.SetContent(col, row, ch, nil, style)
			col++
		}

		// Description (dimmed)
		if opt.Description != "" {
			col++ // space
			for _, ch := range opt.Description {
				screen.SetContent(col, row, ch, nil, hintStyle)
				col++
			}
		}

		row++
	}

	return row - y
}

// Selected returns the currently selected index.
func (r *RadioSelect) Selected() int {
	return r.selected
}

// SetSelected sets the selected index.
func (r *RadioSelect) SetSelected(index int) {
	if index >= 0 && index < len(r.options) {
		r.selected = index
		if r.onChange != nil {
			r.onChange(r.selected)
		}
	}
}
