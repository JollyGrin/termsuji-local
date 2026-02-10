// Package ui provides terminal UI components for termsuji-local.
package ui

import (
	"strconv"
	"strings"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"

	"termsuji-local/engine"
)

// GameSetupUI provides a form for configuring a new game.
type GameSetupUI struct {
	form     *tview.Form
	flex     *tview.Flex
	onStart  func(engine.GameConfig)
	onCancel func()
	onColors func()

	boardSize   int
	playerColor int
	level       int
	komi        float64
}

// NewGameSetup creates a new game setup form.
func NewGameSetup(onStart func(engine.GameConfig), onCancel func(), onColors func()) *GameSetupUI {
	setup := &GameSetupUI{
		onStart:     onStart,
		onCancel:    onCancel,
		onColors:    onColors,
		boardSize:   19,
		playerColor: 1,
		level:       5,
		komi:        6.5,
	}

	boardSizes := []string{"9x9", "13x13", "19x19"}
	colors := []string{"Black (play first)", "White (play second)"}
	levels := []string{"1 (easiest)", "2", "3", "4", "5", "6", "7", "8", "9", "10 (hardest)"}

	form := tview.NewForm()

	form.AddDropDown("Board Size", boardSizes, 2, func(option string, index int) {
		switch index {
		case 0:
			setup.boardSize = 9
		case 1:
			setup.boardSize = 13
		case 2:
			setup.boardSize = 19
		}
	})

	form.AddDropDown("Your Color", colors, 0, func(option string, index int) {
		setup.playerColor = index + 1 // 1=black, 2=white
	})

	form.AddDropDown("GnuGo Strength", levels, 4, func(option string, index int) {
		setup.level = index + 1
	})

	form.AddInputField("Komi", "6.5", 8, func(text string, lastChar rune) bool {
		// Allow digits, decimal point, and minus sign
		return (lastChar >= '0' && lastChar <= '9') || lastChar == '.' || lastChar == '-'
	}, func(text string) {
		if val, err := strconv.ParseFloat(strings.TrimSpace(text), 64); err == nil {
			setup.komi = val
		}
	})

	form.AddButton("Start Game", func() {
		cfg := engine.GameConfig{
			BoardSize:   setup.boardSize,
			Komi:        setup.komi,
			PlayerColor: setup.playerColor,
			EngineLevel: setup.level,
			EnginePath:  "gnugo",
		}
		onStart(cfg)
	})

	form.AddButton("Board Color", func() {
		if onColors != nil {
			onColors()
		}
	})

	form.AddButton("Quit", func() {
		onCancel()
	})

	form.SetBorder(true)
	form.SetTitle(" New Game ")
	form.SetTitleAlign(tview.AlignCenter)
	form.SetButtonBackgroundColor(tcell.ColorDarkCyan)
	form.SetButtonTextColor(tcell.ColorWhite)

	// Create help text
	helpText := tview.NewTextView().
		SetText("Tab/Shift+Tab: navigate fields  |  Arrow keys: change dropdown  |  Enter: confirm").
		SetTextAlign(tview.AlignCenter)
	helpText.SetTextColor(tcell.ColorGray)

	// Create flex layout with form and help text
	flex := tview.NewFlex().SetDirection(tview.FlexRow).
		AddItem(form, 0, 1, true).
		AddItem(helpText, 1, 0, false)

	setup.form = form
	setup.flex = flex
	return setup
}

// Form returns the flex container with form and help text.
func (s *GameSetupUI) Form() *tview.Flex {
	return s.flex
}

// SetInputCapture sets the input capture function for the form.
func (s *GameSetupUI) SetInputCapture(capture func(event *tcell.EventKey) *tcell.EventKey) {
	s.form.SetInputCapture(capture)
}
