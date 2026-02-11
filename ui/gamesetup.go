// Package ui provides terminal UI components for termsuji-local.
package ui

import (
	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"

	"termsuji-local/engine"
)

// GameSetupUI provides a styled card UI for configuring a new game.
type GameSetupUI struct {
	box      *tview.Box
	flex     *tview.Flex
	onStart  func(engine.GameConfig)
	onCancel func()
	onColors func()

	// Components
	card        *MenuCard
	boardSelect *RadioSelect
	colorSelect *RadioSelect
	levelSlider *LevelSlider
	komiInput   *KomiInput
	playButton  *MenuButton
	colorButton *MenuButton
	quitButton  *MenuButton

	// Focus management
	focusIndex int
	focusables []focusableComponent

	// Config values
	boardSize   int
	playerColor int
	level       int
	komi        float64
}

// focusableComponent wraps different component types for focus management.
type focusableComponent interface {
	SetFocused(bool)
	HandleKey(*tcell.EventKey) bool
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

	// Create card container
	setup.card = NewMenuCard("T E R M S U J I")

	// Board size radio select
	boardOptions := []RadioOption{
		{Label: "9×9", Description: "Beginner"},
		{Label: "13×13", Description: "Intermediate"},
		{Label: "19×19", Description: "Standard"},
	}
	setup.boardSelect = NewRadioSelect("Board Size", boardOptions, 2, func(idx int) {
		switch idx {
		case 0:
			setup.boardSize = 9
		case 1:
			setup.boardSize = 13
		case 2:
			setup.boardSize = 19
		}
	})

	// Color radio select
	colorOptions := []RadioOption{
		{Label: "Black", Description: "(play first)"},
		{Label: "White", Description: "(play second)"},
	}
	setup.colorSelect = NewRadioSelect("Your Color", colorOptions, 0, func(idx int) {
		setup.playerColor = idx + 1 // 1=black, 2=white
	})

	// Level slider
	setup.levelSlider = NewLevelSlider("Strength", 1, 10, 5, func(level int) {
		setup.level = level
	})

	// Komi input
	setup.komiInput = NewKomiInput("Komi", 6.5, func(komi float64) {
		setup.komi = komi
	})

	// Buttons
	setup.playButton = NewMenuButton("(P)LAY", true, func() {
		cfg := engine.GameConfig{
			BoardSize:   setup.boardSize,
			Komi:        setup.komi,
			PlayerColor: setup.playerColor,
			EngineLevel: setup.level,
			EnginePath:  "gnugo",
		}
		onStart(cfg)
	})

	setup.colorButton = NewMenuButton("COLORS", false, func() {
		if onColors != nil {
			onColors()
		}
	})

	setup.quitButton = NewMenuButton("QUIT", false, func() {
		onCancel()
	})

	// Set up focus chain
	setup.focusables = []focusableComponent{
		setup.boardSelect,
		setup.colorSelect,
		setup.levelSlider,
		setup.komiInput,
		setup.playButton,
		setup.colorButton,
		setup.quitButton,
	}
	setup.focusIndex = 0
	setup.boardSelect.SetFocused(true)

	// Create the main box with custom draw function
	setup.box = tview.NewBox()
	setup.box.SetDrawFunc(setup.draw)
	setup.box.SetInputCapture(setup.handleInput)

	// Create help text
	helpText := tview.NewTextView().
		SetText("↑↓ options · Tab next · p play · ctrl-c quit").
		SetTextAlign(tview.AlignCenter)
	helpText.SetTextColor(MenuColors.Hint)
	helpText.SetBackgroundColor(tcell.ColorDefault)

	// Create inner flex layout with box and help text
	innerFlex := tview.NewFlex().SetDirection(tview.FlexRow).
		AddItem(nil, 0, 1, false).        // Top spacer
		AddItem(setup.box, 20, 0, true).  // Card (fixed height)
		AddItem(nil, 0, 1, false).        // Bottom spacer
		AddItem(helpText, 1, 0, false)

	// Center horizontally
	setup.flex = tview.NewFlex().SetDirection(tview.FlexColumn).
		AddItem(nil, 0, 1, false).        // Left spacer
		AddItem(innerFlex, 48, 0, true).  // Card (fixed width)
		AddItem(nil, 0, 1, false)         // Right spacer

	return setup
}

// draw renders all components onto the screen.
func (s *GameSetupUI) draw(screen tcell.Screen, x, y, width, height int) (int, int, int, int) {
	// Fill background
	bgStyle := tcell.StyleDefault.Background(MenuColors.CardBG)
	for row := y; row < y+height; row++ {
		for col := x; col < x+width; col++ {
			screen.SetContent(col, row, ' ', nil, bgStyle)
		}
	}

	// Draw card border and title
	s.drawCard(screen, x, y, width, height)

	// Content starts after title
	contentX := x + 4
	contentY := y + 4
	contentWidth := width - 8

	// Draw board size selector
	rows := s.boardSelect.Draw(screen, contentX, contentY, contentWidth)
	contentY += rows + 1

	// Draw color selector
	rows = s.colorSelect.Draw(screen, contentX, contentY, contentWidth)
	contentY += rows + 1

	// Draw level slider
	rows = s.levelSlider.Draw(screen, contentX, contentY, contentWidth)
	contentY += rows + 1

	// Draw komi input
	rows = s.komiInput.Draw(screen, contentX, contentY, contentWidth)
	contentY += rows + 2 // spacing before buttons

	// Draw buttons centered
	s.drawButtons(screen, x, contentY, width)

	return x, y, width, height
}

// drawCard renders the card border and title.
func (s *GameSetupUI) drawCard(screen tcell.Screen, x, y, width, height int) {
	borderColor := MenuColors.Border
	borderStyle := tcell.StyleDefault.Foreground(borderColor).Background(MenuColors.CardBG)

	// Top border: ╭───╮
	screen.SetContent(x, y, '╭', nil, borderStyle)
	for col := x + 1; col < x+width-1; col++ {
		screen.SetContent(col, y, '─', nil, borderStyle)
	}
	screen.SetContent(x+width-1, y, '╮', nil, borderStyle)

	// Side borders
	for row := y + 1; row < y+height-1; row++ {
		screen.SetContent(x, row, '│', nil, borderStyle)
		screen.SetContent(x+width-1, row, '│', nil, borderStyle)
	}

	// Bottom border: ╰───╯
	screen.SetContent(x, y+height-1, '╰', nil, borderStyle)
	for col := x + 1; col < x+width-1; col++ {
		screen.SetContent(col, y+height-1, '─', nil, borderStyle)
	}
	screen.SetContent(x+width-1, y+height-1, '╯', nil, borderStyle)

	// Title centered with decoration: ⬡ T E R M S U J I
	titleStyle := tcell.StyleDefault.Foreground(MenuColors.Title).Background(MenuColors.CardBG).Bold(true)
	accentStyle := tcell.StyleDefault.Foreground(MenuColors.TitleAccent).Background(MenuColors.CardBG)

	title := "T E R M S U J I"
	fullTitle := "⬡  " + title
	titleLen := len([]rune(fullTitle))
	titleX := x + (width-titleLen)/2
	titleY := y + 2

	// Draw decoration
	screen.SetContent(titleX, titleY, '⬡', nil, accentStyle)
	titleX += 3

	// Draw title
	for _, ch := range title {
		screen.SetContent(titleX, titleY, ch, nil, titleStyle)
		titleX++
	}
}

// drawButtons draws the action buttons centered.
func (s *GameSetupUI) drawButtons(screen tcell.Screen, x, y, width int) {
	// Calculate total button width
	playW := s.playButton.Width()
	colorW := s.colorButton.Width()
	quitW := s.quitButton.Width()
	spacing := 2
	totalW := playW + colorW + quitW + spacing*2

	// Center buttons
	buttonX := x + (width-totalW)/2
	buttonY := y

	// Draw buttons
	buttonX += s.playButton.Draw(screen, buttonX, buttonY)
	buttonX += spacing
	buttonX += s.colorButton.Draw(screen, buttonX, buttonY)
	buttonX += spacing
	s.quitButton.Draw(screen, buttonX, buttonY)
}

// handleInput processes keyboard input for focus management and delegation.
func (s *GameSetupUI) handleInput(event *tcell.EventKey) *tcell.EventKey {
	// Let current focused component try to handle the key first
	if s.focusIndex >= 0 && s.focusIndex < len(s.focusables) {
		if s.focusables[s.focusIndex].HandleKey(event) {
			return nil
		}
	}

	// Handle focus navigation
	switch event.Key() {
	case tcell.KeyTab:
		s.cycleFocus(1)
		return nil
	case tcell.KeyBacktab:
		s.cycleFocus(-1)
		return nil
	case tcell.KeyDown:
		// Move to next component if current doesn't handle down
		if s.focusIndex < 4 { // Not in buttons
			s.cycleFocus(1)
			return nil
		}
	case tcell.KeyUp:
		// Move to previous component if current doesn't handle up
		if s.focusIndex > 0 && s.focusIndex <= 4 {
			s.cycleFocus(-1)
			return nil
		}
	case tcell.KeyLeft:
		// Handle left arrow in button row
		if s.focusIndex > 4 {
			s.cycleFocus(-1)
			return nil
		}
	case tcell.KeyRight:
		// Handle right arrow in button row
		if s.focusIndex >= 4 && s.focusIndex < 6 {
			s.cycleFocus(1)
			return nil
		}
	case tcell.KeyEscape:
		s.onCancel()
		return nil
	case tcell.KeyRune:
		// Hotkey 'p' to play (unless in komi input)
		if event.Rune() == 'p' && s.focusIndex != 3 {
			s.playButton.HandleKey(tcell.NewEventKey(tcell.KeyEnter, 0, tcell.ModNone))
			return nil
		}
	}

	return event
}

// cycleFocus moves focus to the next/previous component.
func (s *GameSetupUI) cycleFocus(delta int) {
	// Unfocus current
	if s.focusIndex >= 0 && s.focusIndex < len(s.focusables) {
		s.focusables[s.focusIndex].SetFocused(false)
	}

	// Move to next
	s.focusIndex = (s.focusIndex + delta + len(s.focusables)) % len(s.focusables)

	// Focus new
	if s.focusIndex >= 0 && s.focusIndex < len(s.focusables) {
		s.focusables[s.focusIndex].SetFocused(true)
	}
}

// Form returns the flex container with form and help text.
func (s *GameSetupUI) Form() *tview.Flex {
	return s.flex
}

// SetInputCapture sets the input capture function for the form.
func (s *GameSetupUI) SetInputCapture(capture func(event *tcell.EventKey) *tcell.EventKey) {
	originalCapture := s.box.GetInputCapture()
	s.box.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		// Let the original handler process first
		if result := originalCapture(event); result == nil {
			return nil
		}
		// Then call the user's capture
		return capture(event)
	})
}
