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
	bgStyle := tcell.StyleDefault.Background(MenuColors.CardBG)
	buttonBG := MenuColors.ButtonBG
	textColor := MenuColors.ButtonText

	if b.focused {
		buttonBG = MenuColors.ButtonFocus
	}

	buttonStyle := tcell.StyleDefault.Foreground(textColor).Background(buttonBG)

	// Prepare label with optional play arrow
	label := b.label
	if b.primary {
		label = "▶ " + label
	}

	// Button width with padding
	padding := 2
	width := len([]rune(label)) + padding*2

	if b.focused {
		// Focused: double border ╔══╗║╚══╝
		// Top border
		screen.SetContent(x, y, '╔', nil, buttonStyle)
		for i := 1; i < width-1; i++ {
			screen.SetContent(x+i, y, '═', nil, buttonStyle)
		}
		screen.SetContent(x+width-1, y, '╗', nil, buttonStyle)

		// Middle with text
		screen.SetContent(x, y+1, '║', nil, buttonStyle)
		col := x + 1
		// Left padding
		for i := 0; i < padding-1; i++ {
			screen.SetContent(col, y+1, ' ', nil, buttonStyle)
			col++
		}
		// Label
		for _, ch := range label {
			screen.SetContent(col, y+1, ch, nil, buttonStyle)
			col++
		}
		// Right padding
		for col < x+width-1 {
			screen.SetContent(col, y+1, ' ', nil, buttonStyle)
			col++
		}
		screen.SetContent(x+width-1, y+1, '║', nil, buttonStyle)

		// Bottom border
		screen.SetContent(x, y+2, '╚', nil, buttonStyle)
		for i := 1; i < width-1; i++ {
			screen.SetContent(x+i, y+2, '═', nil, buttonStyle)
		}
		screen.SetContent(x+width-1, y+2, '╝', nil, buttonStyle)
	} else {
		// Normal: single border ┌──┐│└──┘
		borderStyle := tcell.StyleDefault.Foreground(MenuColors.Border).Background(MenuColors.CardBG)
		innerStyle := tcell.StyleDefault.Foreground(MenuColors.Label).Background(MenuColors.CardBG)

		// Top border
		screen.SetContent(x, y, '┌', nil, borderStyle)
		for i := 1; i < width-1; i++ {
			screen.SetContent(x+i, y, '─', nil, borderStyle)
		}
		screen.SetContent(x+width-1, y, '┐', nil, borderStyle)

		// Middle with text
		screen.SetContent(x, y+1, '│', nil, borderStyle)
		col := x + 1
		// Left padding
		for i := 0; i < padding-1; i++ {
			screen.SetContent(col, y+1, ' ', nil, bgStyle)
			col++
		}
		// Label
		for _, ch := range label {
			screen.SetContent(col, y+1, ch, nil, innerStyle)
			col++
		}
		// Right padding
		for col < x+width-1 {
			screen.SetContent(col, y+1, ' ', nil, bgStyle)
			col++
		}
		screen.SetContent(x+width-1, y+1, '│', nil, borderStyle)

		// Bottom border
		screen.SetContent(x, y+2, '└', nil, borderStyle)
		for i := 1; i < width-1; i++ {
			screen.SetContent(x+i, y+2, '─', nil, borderStyle)
		}
		screen.SetContent(x+width-1, y+2, '┘', nil, borderStyle)
	}

	return width
}

// Width returns the button width.
func (b *MenuButton) Width() int {
	label := b.label
	if b.primary {
		label = "▶ " + label
	}
	return len([]rune(label)) + 4 // 2 padding on each side
}
