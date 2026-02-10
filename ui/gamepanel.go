package ui

import (
	"fmt"

	"github.com/rivo/tview"

	"termsuji-local/types"
)

// GameInfoPanel displays game information alongside the board.
type GameInfoPanel struct {
	box        *tview.TextView
	boardState *types.BoardState
	komi       float64
}

// NewGameInfoPanel creates a new game info panel.
func NewGameInfoPanel() *GameInfoPanel {
	panel := &GameInfoPanel{
		box:  tview.NewTextView(),
		komi: 6.5,
	}

	panel.box.SetDynamicColors(true)
	panel.box.SetBorder(false)
	panel.box.SetTextAlign(tview.AlignLeft)

	return panel
}

// Box returns the underlying tview component.
func (p *GameInfoPanel) Box() *tview.TextView {
	return p.box
}

// SetBoardState updates the panel with current board state.
func (p *GameInfoPanel) SetBoardState(state *types.BoardState) {
	p.boardState = state
	p.refresh()
}

// SetKomi sets the komi value for display.
func (p *GameInfoPanel) SetKomi(komi float64) {
	p.komi = komi
	p.refresh()
}

// refresh updates the panel text.
func (p *GameInfoPanel) refresh() {
	if p.boardState == nil {
		p.box.SetText("")
		return
	}

	var text string

	// Game Info section
	text += "[white::b]Game Info[-:-:-]\n"
	text += "[dimgray]──────────────────────[-:-:-]\n"

	// Komi
	text += fmt.Sprintf("[white]Komi:[-:-:-] %.1f\n", p.komi)

	// Move count
	text += fmt.Sprintf("[white]Move:[-:-:-] %d\n", p.boardState.MoveNumber)

	p.box.SetText(text)
}

// CreateGameLayout creates the main game layout with board and side panel.
func CreateGameLayout(board *GoBoardUI, hint *tview.TextView) *tview.Flex {
	// Create the info panel
	infoPanel := NewGameInfoPanel()

	// Store panel reference in board for updates
	board.infoPanel = infoPanel

	// Create horizontal flex: board | info panel
	boardRow := tview.NewFlex().SetDirection(tview.FlexColumn)
	boardRow.AddItem(board.Box, 0, 1, true)         // Board (flexible, takes remaining space)
	boardRow.AddItem(infoPanel.Box(), 26, 0, false) // Info panel (fixed width)

	// Main vertical flex: board area on top, status at bottom
	mainFlex := tview.NewFlex().SetDirection(tview.FlexRow)
	mainFlex.AddItem(boardRow, 0, 1, true)
	mainFlex.AddItem(hint, 7, 0, false)

	return mainFlex
}

// CreateCenteredForm creates a centered form container for the setup screen.
func CreateCenteredForm(form *tview.Flex, maxWidth int) *tview.Flex {
	centered := tview.NewFlex().SetDirection(tview.FlexColumn)
	centered.AddItem(nil, 0, 1, false)        // Left spacer
	centered.AddItem(form, maxWidth, 0, true) // Form with max width
	centered.AddItem(nil, 0, 1, false)        // Right spacer

	return centered
}
