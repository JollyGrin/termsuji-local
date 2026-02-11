package ui

import (
	"github.com/gdamore/tcell/v2"
)

// MenuButton is a styled button component.
type MenuButton struct {
	label    string
	primary  bool
	focused  bool
	onSelect func()
}

// NewMenuButton creates a new menu button.
func NewMenuButton(label string, primary bool, onSelect func()) *MenuButton {
	return &MenuButton{
		label:    label,
		primary:  primary,
		onSelect: onSelect,
	}
}

// SetFocused sets the focus state.
func (b *MenuButton) SetFocused(focused bool) {
	b.focused = focused
}

// HandleKey processes keyboard input. Returns true if handled.
func (b *MenuButton) HandleKey(event *tcell.EventKey) bool {
	switch event.Key() {
	case tcell.KeyEnter:
		if b.onSelect != nil {
			b.onSelect()
		}
		return true
	}
	return false
}

// Draw renders the button component at the given position.
// Returns the width used.
func (b *MenuButton) Draw(screen tcell.Screen, x, y int) int {
	label := b.label
	if b.primary {
		label = "▶ " + label
	}

	padding := 1
	width := len([]rune(label)) + padding*2

	if b.focused {
		// Filled background, bright text
		style := tcell.StyleDefault.
			Foreground(MenuColors.ButtonText).
			Background(MenuColors.ButtonFocus)
		// Draw filled pill
		for i := 0; i < width; i++ {
			screen.SetContent(x+i, y, ' ', nil, style)
		}
		// Draw label centered
		col := x + padding
		for _, ch := range label {
			screen.SetContent(col, y, ch, nil, style)
			col++
		}
	} else {
		// Dim text with brackets, no fill
		dimStyle := tcell.StyleDefault.
			Foreground(MenuColors.Hint).
			Background(MenuColors.CardBG)
		bracketStyle := tcell.StyleDefault.
			Foreground(MenuColors.Border).
			Background(MenuColors.CardBG)

		screen.SetContent(x, y, '[', nil, bracketStyle)
		col := x + 1
		for _, ch := range label {
			screen.SetContent(col, y, ch, nil, dimStyle)
			col++
		}
		screen.SetContent(col, y, ']', nil, bracketStyle)
	}

	return width
}

// Width returns the button width.
func (b *MenuButton) Width() int {
	label := b.label
	if b.primary {
		label = "▶ " + label
	}
	return len([]rune(label)) + 2 // 1 padding on each side (or brackets)
}
