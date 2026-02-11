package ui

import (
	"fmt"
	"strconv"

	"github.com/gdamore/tcell/v2"
)

// KomiInput is a numeric input field for komi value.
type KomiInput struct {
	label    string
	value    float64
	text     string
	focused  bool
	cursor   int
	onChange func(float64)
}

// NewKomiInput creates a new komi input field.
func NewKomiInput(label string, initial float64, onChange func(float64)) *KomiInput {
	return &KomiInput{
		label:    label,
		value:    initial,
		text:     fmt.Sprintf("%.1f", initial),
		cursor:   3, // after "6.5"
		onChange: onChange,
	}
}

// SetFocused sets the focus state.
func (k *KomiInput) SetFocused(focused bool) {
	k.focused = focused
}

// HandleKey processes keyboard input. Returns true if handled.
func (k *KomiInput) HandleKey(event *tcell.EventKey) bool {
	switch event.Key() {
	case tcell.KeyLeft:
		if k.cursor > 0 {
			k.cursor--
		}
		return true
	case tcell.KeyRight:
		if k.cursor < len(k.text) {
			k.cursor++
		}
		return true
	case tcell.KeyBackspace, tcell.KeyBackspace2:
		if k.cursor > 0 {
			k.text = k.text[:k.cursor-1] + k.text[k.cursor:]
			k.cursor--
			k.updateValue()
		}
		return true
	case tcell.KeyDelete:
		if k.cursor < len(k.text) {
			k.text = k.text[:k.cursor] + k.text[k.cursor+1:]
			k.updateValue()
		}
		return true
	case tcell.KeyRune:
		ch := event.Rune()
		// Allow digits, decimal point, and minus sign
		if (ch >= '0' && ch <= '9') || ch == '.' || ch == '-' {
			k.text = k.text[:k.cursor] + string(ch) + k.text[k.cursor:]
			k.cursor++
			k.updateValue()
		}
		return true
	}
	return false
}

func (k *KomiInput) updateValue() {
	if val, err := strconv.ParseFloat(k.text, 64); err == nil {
		k.value = val
		if k.onChange != nil {
			k.onChange(k.value)
		}
	}
}

// Draw renders the komi input component.
// Returns the number of rows used.
func (k *KomiInput) Draw(screen tcell.Screen, x, y, width int) int {
	bgStyle := tcell.StyleDefault.Background(MenuColors.CardBG)
	labelStyle := tcell.StyleDefault.Foreground(MenuColors.Label).Background(MenuColors.CardBG)
	accentStyle := tcell.StyleDefault.Foreground(MenuColors.TitleAccent).Background(MenuColors.CardBG)
	selectedStyle := tcell.StyleDefault.Foreground(MenuColors.Selected).Background(MenuColors.CardBG)
	inputStyle := tcell.StyleDefault.Foreground(MenuColors.Label).Background(tcell.PaletteColor(238))
	cursorStyle := tcell.StyleDefault.Foreground(MenuColors.CardBG).Background(MenuColors.Selected)

	col := x

	// Focus cursor
	if k.focused {
		screen.SetContent(col, y, '▸', nil, selectedStyle)
	} else {
		screen.SetContent(col, y, ' ', nil, bgStyle)
	}
	col += 2

	// Label with diamond prefix: ◈ Komi
	screen.SetContent(col, y, '◈', nil, accentStyle)
	col += 2

	for _, ch := range k.label {
		screen.SetContent(col, y, ch, nil, labelStyle)
		col++
	}
	col += 3 // spacing

	// Input field with brackets: [ 6.5 ]
	screen.SetContent(col, y, '[', nil, labelStyle)
	col++
	screen.SetContent(col, y, ' ', nil, inputStyle)
	col++

	// Text content
	inputStart := col
	for i, ch := range k.text {
		style := inputStyle
		if k.focused && i == k.cursor {
			style = cursorStyle
		}
		screen.SetContent(col, y, ch, nil, style)
		col++
	}

	// Cursor at end
	if k.focused && k.cursor >= len(k.text) {
		screen.SetContent(col, y, ' ', nil, cursorStyle)
		col++
	}

	// Pad to fixed width
	fieldWidth := 6
	for col < inputStart+fieldWidth {
		screen.SetContent(col, y, ' ', nil, inputStyle)
		col++
	}

	screen.SetContent(col, y, ' ', nil, inputStyle)
	col++
	screen.SetContent(col, y, ']', nil, labelStyle)

	return 1
}

// Value returns the current komi value.
func (k *KomiInput) Value() float64 {
	return k.value
}

// SetValue sets the komi value.
func (k *KomiInput) SetValue(v float64) {
	k.value = v
	k.text = fmt.Sprintf("%.1f", v)
	k.cursor = len(k.text)
	if k.onChange != nil {
		k.onChange(k.value)
	}
}
