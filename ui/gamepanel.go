package ui

import (
	"fmt"

	"github.com/rivo/tview"

	"termsuji-local/engine/gtp"
	"termsuji-local/sgf"
	"termsuji-local/types"
)

// GameInfoPanel displays game information and move history alongside the board.
type GameInfoPanel struct {
	box         *tview.TextView
	boardState  *types.BoardState
	komi        float64
	moveHistory *[]MoveEntry
	boardSize   int
	planTree    *sgf.GameTree // non-nil when in planning mode
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

// SetMoveHistory sets a pointer to the move history slice and the board size for coordinate display.
func (p *GameInfoPanel) SetMoveHistory(history *[]MoveEntry, boardSize int) {
	p.moveHistory = history
	p.boardSize = boardSize
}

// SetPlanningMode enables planning mode display with the given tree.
func (p *GameInfoPanel) SetPlanningMode(tree *sgf.GameTree) {
	p.planTree = tree
}

// ClearPlanningMode disables planning mode display.
func (p *GameInfoPanel) ClearPlanningMode() {
	p.planTree = nil
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

	// Planning mode: show exploration path
	if p.planTree != nil {
		text += "\n[yellow::b]PLAN[-:-:-]\n"
		text += "[dimgray]──────────────────────[-:-:-]\n"

		// Show variation info
		if p.planTree.NumVariations() > 1 {
			text += fmt.Sprintf("[dimgray]var %d/%d[-]\n", p.planTree.VariationIndex()+1, p.planTree.NumVariations())
		}

		path := p.planTree.PathFromRoot()
		if len(path) == 0 {
			text += "[dimgray]  (no moves)[-]\n"
		} else {
			maxVisible := 12
			start := 0
			if len(path) > maxVisible {
				start = len(path) - maxVisible
			}

			// Find current position in the path
			currentIdx := len(p.planTree.PathFromRoot()) - 1

			for i := start; i < len(path); i++ {
				color, x, y := parsePlanMoveForPanel(path[i])
				moveNum := i + 1

				colorStr := "[white]B[-]"
				if color == 2 {
					colorStr = "[dimgray]W[-]"
				}

				coord := "pass"
				if x >= 0 && y >= 0 {
					size := p.boardSize
					if p.boardState != nil && p.boardState.Width() > 0 {
						size = p.boardState.Width()
					}
					if size > 0 {
						coord = gtp.PosToGTPDisplay(x, y, size)
					}
				}

				marker := " "
				if i == currentIdx {
					marker = "[yellow]>[-]"
				}

				text += fmt.Sprintf("%s[dimgray]%3d.[-] %s %s\n", marker, moveNum, colorStr, coord)
			}

			if start > 0 {
				text += fmt.Sprintf("[dimgray]  ··· %d earlier[-]\n", start)
			}
		}
	} else if p.moveHistory != nil && len(*p.moveHistory) > 0 {
		// Normal mode: show move history
		text += "\n[white::b]Moves[-:-:-]\n"
		text += "[dimgray]──────────────────────[-:-:-]\n"

		moves := *p.moveHistory
		// Show last N moves that fit, with scroll
		maxVisible := 12
		start := 0
		if len(moves) > maxVisible {
			start = len(moves) - maxVisible
		}

		for i := start; i < len(moves); i++ {
			m := moves[i]
			moveNum := i + 1

			colorStr := "[white]B[-]"
			if m.Color == 2 {
				colorStr = "[dimgray]W[-]"
			}

			coord := "pass"
			if m.X >= 0 && m.Y >= 0 {
				size := p.boardSize
				if p.boardState != nil && p.boardState.Width() > 0 {
					size = p.boardState.Width()
				}
				if size > 0 {
					coord = gtp.PosToGTPDisplay(m.X, m.Y, size)
				}
			}

			marker := " "
			if i == len(moves)-1 {
				marker = "[white]>[-]"
			}

			text += fmt.Sprintf("%s[dimgray]%3d.[-] %s %s\n", marker, moveNum, colorStr, coord)
		}

		if start > 0 {
			text += fmt.Sprintf("[dimgray]  ··· %d earlier[-]\n", start)
		}
	}

	p.box.SetText(text)
}

// parsePlanMoveForPanel extracts color, x, y from an SGF move string like ";B[pd]".
func parsePlanMoveForPanel(move string) (color, x, y int) {
	if len(move) < 3 {
		return 0, -1, -1
	}
	color = 1
	if move[1] == 'W' {
		color = 2
	}
	start := -1
	end := -1
	for i, ch := range move {
		if ch == '[' {
			start = i + 1
		} else if ch == ']' {
			end = i
			break
		}
	}
	if start == -1 || end == -1 || end <= start {
		return color, -1, -1
	}
	coord := move[start:end]
	if len(coord) != 2 {
		return color, -1, -1
	}
	x = int(coord[0] - 'a')
	y = int(coord[1] - 'a')
	return color, x, y
}

// CreateGameLayout creates the main game layout with board and side panel.
func CreateGameLayout(board *GoBoardUI, hint *tview.TextView) *tview.Flex {
	// Create the info panel
	infoPanel := NewGameInfoPanel()

	// Store panel reference in board for updates
	board.infoPanel = infoPanel
	infoPanel.SetMoveHistory(&board.moveHistory, board.gameConfig.BoardSize)

	// Create horizontal flex: board | info panel
	boardRow := tview.NewFlex().SetDirection(tview.FlexColumn)
	boardRow.AddItem(board.Box, 0, 1, true)         // Board (flexible, takes remaining space)
	boardRow.AddItem(infoPanel.Box(), 26, 0, false) // Info panel (fixed width)

	// Main vertical flex: board area on top, compact status bar at bottom
	mainFlex := tview.NewFlex().SetDirection(tview.FlexRow)
	mainFlex.AddItem(boardRow, 0, 1, true)
	mainFlex.AddItem(hint, 2, 0, false) // Compact: just 2 rows

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

// RebuildNormalLayout restores the normal game layout with board, info panel, and hint.
func RebuildNormalLayout(gameFrame *tview.Flex, board *GoBoardUI, hint *tview.TextView) {
	gameFrame.Clear()

	// Create the info panel
	infoPanel := NewGameInfoPanel()

	// Store panel reference in board for updates
	board.infoPanel = infoPanel
	infoPanel.SetMoveHistory(&board.moveHistory, board.gameConfig.BoardSize)

	// Refresh the info panel with current state
	if board.BoardState != nil {
		infoPanel.SetBoardState(board.BoardState)
	}

	// Create horizontal flex: board | info panel
	boardRow := tview.NewFlex().SetDirection(tview.FlexColumn)
	boardRow.AddItem(board.Box, 0, 1, true)         // Board (flexible, takes remaining space)
	boardRow.AddItem(infoPanel.Box(), 26, 0, false) // Info panel (fixed width)

	// Main vertical flex: board area on top, compact status bar at bottom
	gameFrame.SetDirection(tview.FlexRow)
	gameFrame.AddItem(boardRow, 0, 1, true)
	gameFrame.AddItem(hint, 2, 0, false) // Compact: just 2 rows
}

// BuildFocusLayout builds the focus mode layout with just the centered board.
func BuildFocusLayout(gameFrame *tview.Flex, board *GoBoardUI) {
	gameFrame.Clear()

	// Calculate board dimensions
	boardWidth := 22  // default for 9x9
	boardHeight := 11
	if board.BoardState != nil && board.BoardState.Width() > 0 {
		boardWidth = board.BoardState.Width()*2 + 4  // 2 chars per cell + coordinates
		boardHeight = board.BoardState.Height() + 2 // + coordinates
	}

	// Center board with flex spacers
	gameFrame.SetDirection(tview.FlexRow)
	gameFrame.AddItem(nil, 0, 1, false) // top spacer

	centerRow := tview.NewFlex().SetDirection(tview.FlexColumn)
	centerRow.AddItem(nil, 0, 1, false)                // left spacer
	centerRow.AddItem(board.Box, boardWidth, 0, true)  // board (fixed width)
	centerRow.AddItem(nil, 0, 1, false)                // right spacer

	gameFrame.AddItem(centerRow, boardHeight, 0, true) // center row (fixed height)
	gameFrame.AddItem(nil, 0, 1, false)                // bottom spacer
}
