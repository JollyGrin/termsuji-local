// Package ui specifies custom controls for tview to assist in playing Go in the terminal.
package ui

import (
	"fmt"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"

	"termsuji-local/config"
	"termsuji-local/engine"
	"termsuji-local/sgf"
	"termsuji-local/types"
)

// MoveEntry records a single move for the history panel.
type MoveEntry struct {
	X, Y  int // -1,-1 for pass
	Color int // 1=black, 2=white
}

type GoBoardUI struct {
	Box          *tview.Box
	BoardState   *types.BoardState
	hint         *tview.TextView
	cfg          *config.Config
	finished     bool
	selX         int
	selY         int
	lastTurnPass bool
	app          *tview.Application
	eng          engine.GameEngine
	styles       []tcell.Color
	infoPanel    *GameInfoPanel
	focusMode    bool
	recorder     *sgf.GameRecord
	gameConfig   engine.GameConfig
	moveHistory  []MoveEntry

	// Planning mode state
	planningMode   bool
	planTree       *sgf.GameTree
	planBoard      [][]int           // local board for planning (board[y][x])
	planColor      int               // next color to play (alternates)
	planLastMove   [2]int            // last move in planning for highlight (-1,-1 if none)
	prePlanBoard   *types.BoardState // snapshot to restore when exiting
	prePlanHistory []MoveEntry       // snapshot of move history
}

// ToggleFocusMode toggles focus mode and returns the new state.
func (g *GoBoardUI) ToggleFocusMode() bool {
	g.focusMode = !g.focusMode
	g.refreshHint()
	return g.focusMode
}

// SetFocusMode sets focus mode to the given state.
func (g *GoBoardUI) SetFocusMode(enabled bool) {
	g.focusMode = enabled
	g.refreshHint()
}

// IsFocusMode returns true if focus mode is enabled.
func (g *GoBoardUI) IsFocusMode() bool {
	return g.focusMode
}

func (g *GoBoardUI) SelectedTile() *types.BoardPos {
	if g.selX == -1 && g.selY == -1 {
		return nil
	}
	return &types.BoardPos{X: g.selX, Y: g.selY}
}

func (g *GoBoardUI) MoveSelection(h, v int) {
	if !g.planningMode && g.BoardState.Finished() {
		g.ResetSelection()
		return
	}
	prevTile := g.SelectedTile()
	if prevTile == nil {
		if g.planningMode {
			g.selX = g.planLastMove[0]
			g.selY = g.planLastMove[1]
		} else {
			g.selX = g.BoardState.LastMove.X
			g.selY = g.BoardState.LastMove.Y
		}
		if g.SelectedTile() == nil {
			// No previous move made, use board center
			g.selX = int(g.BoardState.Width() / 2)
			g.selY = int(g.BoardState.Height() / 2)
		}
		return
	}
	if g.selX+h < 0 || g.selX+h >= g.BoardState.Width() {
		return
	}
	if g.selY+v < 0 || g.selY+v >= g.BoardState.Width() {
		return
	}
	g.selX += h
	g.selY += v
}

func (g *GoBoardUI) ResetSelection() {
	g.selX = -1
	g.selY = -1
}

func NewGoBoard(app *tview.Application, c *config.Config, hint *tview.TextView) *GoBoardUI {
	goBoard := &GoBoardUI{
		Box:        tview.NewBox(),
		BoardState: &types.BoardState{},
		hint:       hint,
		app:        app,
		selX:       -1,
		selY:       -1,
	}
	goBoard.SetConfig(c)
	goBoard.Box.SetDrawFunc(func(screen tcell.Screen, x int, y int, width int, height int) (int, int, int, int) {
		if goBoard.BoardState == nil || goBoard.BoardState.Width() == 0 {
			return x, y, 1, 1
		}
		// 2 characters per cell for square appearance
		boardW, boardH := goBoard.BoardState.Width()*2, goBoard.BoardState.Height()

		// Choose board data and last-move indicator based on planning mode
		boardData := goBoard.BoardState.Board
		lastMoveX, lastMoveY := goBoard.BoardState.LastMove.X, goBoard.BoardState.LastMove.Y
		if goBoard.planningMode && goBoard.planBoard != nil {
			boardData = goBoard.planBoard
			lastMoveX, lastMoveY = goBoard.planLastMove[0], goBoard.planLastMove[1]
		}

		for boardY := 0; boardY < goBoard.BoardState.Height(); boardY++ {
			for boardX := 0; boardX < goBoard.BoardState.Width(); boardX++ {
				stone := boardData[boardY][boardX]
				i := stone
				if !goBoard.cfg.Theme.DrawStoneBackground {
					i = 0
				}
				var fgColor tcell.Color
				// Get color and inverted color
				iInv := 0
				if i == 1 {
					iInv = 2
				} else if i == 2 {
					iInv = 1
				}
				if (boardX%2 + boardY%2) == 1 {
					i += 3
					iInv += 3
				}
				var drawRune rune
				if goBoard.cfg.Theme.UseGridLines && stone == 0 {
					// Use grid lines for empty intersections
					boardSize := goBoard.BoardState.Width()
					hoshi := isHoshiPoint(boardX, boardY, boardSize)
					drawRune = getGridRune(boardX, boardY, goBoard.BoardState.Width(), goBoard.BoardState.Height(), hoshi)
				} else {
					drawRune = goBoard.cfg.Theme.Symbols.BoardSquare
				}

				if stone > 0 {
					switch stone {
					case 1:
						drawRune = goBoard.cfg.Theme.Symbols.BlackStone
					case 2:
						drawRune = goBoard.cfg.Theme.Symbols.WhiteStone
					}
					if goBoard.cfg.Theme.DrawStoneBackground {
						// Cursor color is inverted stone color, or cursor color when not on a stone.
						fgColor = goBoard.styles[iInv]
					} else {
						// There's a stone but no background drawing, adjust the fg color instead to selected stone
						fgColor = goBoard.styles[stone]
					}
				} else {
					// No stone, use line color for grid
					fgColor = goBoard.styles[9]
				}
				if boardX == goBoard.selX && boardY == goBoard.selY {
					if goBoard.cfg.Theme.DrawCursorBackground {
						i = 8
					} else if !goBoard.cfg.Theme.UseGridLines {
						drawRune = goBoard.cfg.Theme.Symbols.Cursor
					}
					// For grid lines theme, keep the grid character but cursor background will highlight
				} else if boardX == lastMoveX && boardY == lastMoveY {
					if goBoard.cfg.Theme.DrawLastPlayedBackground {
						i = 7
					} else if !goBoard.cfg.Theme.UseGridLines {
						drawRune = goBoard.cfg.Theme.Symbols.LastPlayed
					}
				}

				if goBoard.cfg.Theme.UseGridLines && stone == 0 {
					// Check if there's a stone to the right (no line should connect to it)
					hasStoneRight := false
					if boardX < goBoard.BoardState.Width()-1 {
						hasStoneRight = boardData[boardY][boardX+1] > 0
					}
					// Empty intersection with grid lines - draw grid character + connectors
					drawGridCell(screen, tcell.StyleDefault.Background(goBoard.styles[i]).Foreground(fgColor), drawRune, boardX, boardY, x+4, y, goBoard.BoardState.Width(), hasStoneRight)
				} else {
					// Stone or non-grid theme - use stone cell drawing
					drawStoneCell(screen, tcell.StyleDefault.Background(goBoard.styles[i]).Foreground(fgColor), drawRune, boardX, boardY, x+4, y)
				}
			}
		}
		drawCoordinates(screen, x, y, goBoard)
		// Add offset for coordinate display
		return x, y, boardW + 4, boardH + 2
	})
	return goBoard
}

// ConnectEngine connects the board to a game engine.
func (g *GoBoardUI) ConnectEngine(e engine.GameEngine) error {
	g.finished = false
	g.eng = e
	g.moveHistory = nil

	if err := e.Connect(); err != nil {
		return err
	}

	e.OnMove(func(x, y, color int, boardState *types.BoardState) {
		g.lastTurnPass = (x == -1 && y == -1)
		g.BoardState = boardState
		g.moveHistory = append(g.moveHistory, MoveEntry{X: x, Y: y, Color: color})
		if g.recorder != nil {
			g.recorder.AddMove(x, y, color)
		}
		g.refreshHint()
		// Spawn goroutine to avoid deadlock when called from main thread
		go func() {
			g.app.QueueUpdateDraw(func() {})
		}()
	})

	e.OnGameEnd(func(outcome string) {
		g.finished = true
		g.BoardState = e.GetBoardState()
		if g.recorder != nil {
			g.recorder.SetResult(outcome)
		}
		g.ResetSelection()
		g.refreshHint()
		go func() {
			g.app.QueueUpdateDraw(func() {})
		}()
	})

	g.BoardState = e.GetBoardState()
	g.refreshHint()
	return nil
}

// PlayMove plays a move at the given coordinates.
func (g *GoBoardUI) PlayMove(x, y int) {
	if g.planningMode {
		g.PlanPlayMove(x, y)
		return
	}
	if g.finished {
		return
	}
	if g.eng == nil {
		return
	}
	if !g.eng.IsMyTurn() {
		return
	}
	if err := g.eng.PlayMove(x, y); err != nil {
		// Could show error for illegal move
		return
	}
}

// Pass passes the current turn.
func (g *GoBoardUI) Pass() {
	if g.planningMode {
		g.planPass()
		return
	}
	if g.finished {
		return
	}
	if g.eng == nil {
		return
	}
	if !g.eng.IsMyTurn() {
		return
	}
	g.eng.Pass()
}

// Close disconnects the engine and finalizes any active recording.
func (g *GoBoardUI) Close() {
	if g.recorder != nil {
		g.recorder.Close()
		g.recorder = nil
	}
	if g.eng == nil {
		return
	}
	g.eng.Close()
}

// SetRecorder sets the active SGF recorder.
func (g *GoBoardUI) SetRecorder(rec *sgf.GameRecord) {
	g.recorder = rec
}

// SetGameConfig stores the game configuration for mid-game recording toggle.
func (g *GoBoardUI) SetGameConfig(gc engine.GameConfig) {
	g.gameConfig = gc
}

// SetMoveHistory populates the move history from loaded game data.
func (g *GoBoardUI) SetMoveHistory(moves [][3]int) {
	g.moveHistory = nil
	for _, m := range moves {
		g.moveHistory = append(g.moveHistory, MoveEntry{X: m[1], Y: m[2], Color: m[0]})
	}
}

// UndoMove undoes the last player+engine move pair so it's the player's turn again.
func (g *GoBoardUI) UndoMove() {
	if g.finished || g.eng == nil {
		return
	}
	if !g.eng.IsMyTurn() {
		return
	}
	// Need at least 2 moves to undo (engine response + player move)
	if len(g.moveHistory) < 2 {
		return
	}

	// Undo engine's last response
	if err := g.eng.Undo(); err != nil {
		return
	}
	// Undo player's last move
	if err := g.eng.Undo(); err != nil {
		return
	}

	// Truncate move history
	g.moveHistory = g.moveHistory[:len(g.moveHistory)-2]

	// Truncate SGF recorder
	if g.recorder != nil {
		g.recorder.UndoMoves(2)
	}

	// Resync board state from engine
	g.BoardState = g.eng.GetBoardState()
	g.lastTurnPass = false

	// Restore last move indicator from history
	if len(g.moveHistory) > 0 {
		last := g.moveHistory[len(g.moveHistory)-1]
		g.BoardState.LastMove.X = last.X
		g.BoardState.LastMove.Y = last.Y
	}

	g.refreshHint()
	go func() {
		g.app.QueueUpdateDraw(func() {})
	}()
}

// IsPlanningMode returns true if planning mode is active.
func (g *GoBoardUI) IsPlanningMode() bool {
	return g.planningMode
}

// TogglePlanningMode enters or exits planning mode.
// When entering: snapshots board state and move history, creates a new game tree.
// When exiting: restores the pre-plan state, discards the tree.
func (g *GoBoardUI) TogglePlanningMode() {
	if g.planningMode {
		// Exit planning mode - restore pre-plan state
		g.BoardState = g.prePlanBoard
		g.moveHistory = g.prePlanHistory
		g.planningMode = false
		g.planTree = nil
		g.planBoard = nil
		g.prePlanBoard = nil
		g.prePlanHistory = nil
	} else {
		if g.finished || g.BoardState == nil {
			return
		}
		// Enter planning mode - snapshot current state
		g.prePlanBoard = g.copyBoardState()
		g.prePlanHistory = make([]MoveEntry, len(g.moveHistory))
		copy(g.prePlanHistory, g.moveHistory)

		// Initialize plan board from current board
		size := g.BoardState.Width()
		g.planBoard = sgf.MakeBoard(size)
		for y := 0; y < size; y++ {
			copy(g.planBoard[y], g.BoardState.Board[y])
		}

		// Set next color to play
		if g.eng != nil {
			if g.eng.IsMyTurn() {
				g.planColor = g.eng.GetPlayerColor()
			} else {
				g.planColor = oppositeColor(g.eng.GetPlayerColor())
			}
		} else {
			g.planColor = g.BoardState.PlayerToMove
		}

		g.planLastMove = [2]int{g.BoardState.LastMove.X, g.BoardState.LastMove.Y}
		g.planTree = sgf.NewGameTree()
		g.planningMode = true
	}
	g.refreshHint()
	go func() {
		g.app.QueueUpdateDraw(func() {})
	}()
}

// PlanPlayMove places a stone locally during planning mode.
func (g *GoBoardUI) PlanPlayMove(x, y int) {
	if !g.planningMode || g.planBoard == nil {
		return
	}
	size := g.BoardState.Width()
	if x < 0 || x >= size || y < 0 || y >= size {
		return
	}
	// Must be empty intersection
	if g.planBoard[y][x] != 0 {
		return
	}

	// Place stone and handle captures
	g.planBoard[y][x] = g.planColor
	sgf.RemoveCaptures(g.planBoard, size, x, y, g.planColor)

	// Check for suicide: if the placed stone's group has no liberties after captures
	if !sgf.HasLiberty(g.planBoard, size, x, y, g.planColor) {
		g.planBoard[y][x] = 0 // undo the placement
		return
	}

	// Build SGF move string
	colorChar := "B"
	if g.planColor == 2 {
		colorChar = "W"
	}
	coord := string(rune('a'+x)) + string(rune('a'+y))
	move := fmt.Sprintf(";%s[%s]", colorChar, coord)

	g.planTree.AddMove(move)
	g.planLastMove = [2]int{x, y}
	g.planColor = oppositeColor(g.planColor)
	g.refreshHint()
	go func() {
		g.app.QueueUpdateDraw(func() {})
	}()
}

// planPass adds a pass node in planning mode.
func (g *GoBoardUI) planPass() {
	if !g.planningMode {
		return
	}
	colorChar := "B"
	if g.planColor == 2 {
		colorChar = "W"
	}
	move := fmt.Sprintf(";%s[]", colorChar)
	g.planTree.AddMove(move)
	g.planLastMove = [2]int{-1, -1}
	g.planColor = oppositeColor(g.planColor)
	g.refreshHint()
	go func() {
		g.app.QueueUpdateDraw(func() {})
	}()
}

// PlanBack navigates one move back in the planning tree.
func (g *GoBoardUI) PlanBack() {
	if !g.planningMode || g.planTree == nil {
		return
	}
	if !g.planTree.Back() {
		return
	}
	g.rebuildPlanBoard()
	g.refreshHint()
	go func() {
		g.app.QueueUpdateDraw(func() {})
	}()
}

// PlanForward navigates one move forward (follows first variation).
func (g *GoBoardUI) PlanForward() {
	if !g.planningMode || g.planTree == nil {
		return
	}
	if !g.planTree.Forward(0) {
		return
	}
	g.rebuildPlanBoard()
	g.refreshHint()
	go func() {
		g.app.QueueUpdateDraw(func() {})
	}()
}

// PlanNextVariation switches to the next sibling variation.
func (g *GoBoardUI) PlanNextVariation() {
	if !g.planningMode || g.planTree == nil {
		return
	}
	if !g.planTree.NextVariation() {
		return
	}
	g.rebuildPlanBoard()
	g.refreshHint()
	go func() {
		g.app.QueueUpdateDraw(func() {})
	}()
}

// PlanPrevVariation switches to the previous sibling variation.
func (g *GoBoardUI) PlanPrevVariation() {
	if !g.planningMode || g.planTree == nil {
		return
	}
	if !g.planTree.PrevVariation() {
		return
	}
	g.rebuildPlanBoard()
	g.refreshHint()
	go func() {
		g.app.QueueUpdateDraw(func() {})
	}()
}

// ResumeFromPlan takes the planning path and replays it on the engine, then exits planning mode.
func (g *GoBoardUI) ResumeFromPlan() {
	if !g.planningMode || g.planTree == nil || g.eng == nil {
		return
	}

	planPath := g.planTree.PathFromRoot()
	if len(planPath) == 0 {
		// Nothing explored, just exit
		g.TogglePlanningMode()
		return
	}

	// Build combined move sequence: pre-plan history + planning path
	var allMoves [][3]int
	for _, m := range g.prePlanHistory {
		allMoves = append(allMoves, [3]int{m.Color, m.X, m.Y})
	}
	for _, moveStr := range planPath {
		color, x, y := parsePlanMove(moveStr)
		allMoves = append(allMoves, [3]int{color, x, y})
	}

	// Reset engine and replay all moves
	if err := g.eng.ResetAndReplay(allMoves); err != nil {
		// Failed to resume, just exit planning
		g.TogglePlanningMode()
		return
	}

	// Update move history
	g.moveHistory = nil
	for _, m := range allMoves {
		g.moveHistory = append(g.moveHistory, MoveEntry{X: m[1], Y: m[2], Color: m[0]})
	}

	// Update SGF recorder
	if g.recorder != nil {
		g.recorder.UndoMoves(len(g.prePlanHistory))
		for _, m := range allMoves {
			g.recorder.AddMove(m[1], m[2], m[0])
		}
	}

	// Sync board state from engine
	g.BoardState = g.eng.GetBoardState()

	// Exit planning mode without restoring snapshot
	g.planningMode = false
	g.planTree = nil
	g.planBoard = nil
	g.prePlanBoard = nil
	g.prePlanHistory = nil

	g.refreshHint()
	go func() {
		g.app.QueueUpdateDraw(func() {})
	}()
}

// rebuildPlanBoard replays the planning tree path on the pre-plan board snapshot.
func (g *GoBoardUI) rebuildPlanBoard() {
	size := g.BoardState.Width()
	// Start from pre-plan board snapshot
	g.planBoard = sgf.MakeBoard(size)
	for y := 0; y < size; y++ {
		copy(g.planBoard[y], g.prePlanBoard.Board[y])
	}

	// Determine starting color from pre-plan state
	startColor := g.prePlanBoard.PlayerToMove

	// Replay path from root
	path := g.planTree.PathFromRoot()
	g.planLastMove = [2]int{g.prePlanBoard.LastMove.X, g.prePlanBoard.LastMove.Y}
	currentColor := startColor

	for _, moveStr := range path {
		color, x, y := parsePlanMove(moveStr)
		if color != 0 {
			currentColor = oppositeColor(color)
		}
		if x >= 0 && y >= 0 && x < size && y < size {
			g.planBoard[y][x] = color
			sgf.RemoveCaptures(g.planBoard, size, x, y, color)
			g.planLastMove = [2]int{x, y}
		} else {
			// pass
			g.planLastMove = [2]int{-1, -1}
		}
	}

	if len(path) > 0 {
		lastColor, _, _ := parsePlanMove(path[len(path)-1])
		g.planColor = oppositeColor(lastColor)
	} else {
		g.planColor = currentColor
	}
}

// copyBoardState creates a deep copy of the current board state.
func (g *GoBoardUI) copyBoardState() *types.BoardState {
	if g.BoardState == nil {
		return nil
	}
	size := g.BoardState.Width()
	boardCopy := make([][]int, size)
	for i := range boardCopy {
		boardCopy[i] = make([]int, size)
		copy(boardCopy[i], g.BoardState.Board[i])
	}
	return &types.BoardState{
		MoveNumber:   g.BoardState.MoveNumber,
		PlayerToMove: g.BoardState.PlayerToMove,
		Phase:        g.BoardState.Phase,
		Board:        boardCopy,
		Outcome:      g.BoardState.Outcome,
		LastMove:     g.BoardState.LastMove,
	}
}

// parsePlanMove extracts color, x, y from an SGF move string like ";B[pd]" or ";W[]".
func parsePlanMove(move string) (color, x, y int) {
	if len(move) < 3 {
		return 0, -1, -1
	}
	color = 1
	if move[1] == 'W' {
		color = 2
	}
	// Find coordinates in brackets
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
		// Pass
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

// oppositeColor returns the opposite color (1->2, 2->1).
func oppositeColor(color int) int {
	if color == 1 {
		return 2
	}
	return 1
}

// ToggleRecording toggles SGF recording on or off.
// When toggling on mid-game, captures the current board position via AB[]/AW[].
func (g *GoBoardUI) ToggleRecording(cfg *config.Config) {
	if g.recorder != nil {
		// Stop recording
		g.recorder.Close()
		g.recorder = nil
	} else {
		// Start recording
		gc := g.gameConfig
		rec, err := sgf.NewGameRecord(config.HistoryDir(), gc.BoardSize, gc.Komi, gc.PlayerColor, gc.EngineLevel)
		if err != nil {
			g.refreshHint()
			return
		}
		// If game is in progress, snapshot current position
		if g.BoardState != nil && g.BoardState.MoveNumber > 0 {
			rec.AddSetupPosition(g.BoardState.Board)
		}
		g.recorder = rec
	}
	g.refreshHint()
}

func (g *GoBoardUI) SetConfig(c *config.Config) {
	g.styles = []tcell.Color{
		tcell.PaletteColor(c.Theme.Colors.BoardColor),        // 0
		tcell.PaletteColor(c.Theme.Colors.BlackColor),        // 1
		tcell.PaletteColor(c.Theme.Colors.WhiteColor),        // 2
		tcell.PaletteColor(c.Theme.Colors.BoardColorAlt),     // 3
		tcell.PaletteColor(c.Theme.Colors.BlackColorAlt),     // 4
		tcell.PaletteColor(c.Theme.Colors.WhiteColorAlt),     // 5
		tcell.PaletteColor(c.Theme.Colors.CursorColorFG),     // 6
		tcell.PaletteColor(c.Theme.Colors.LastPlayedColorBG), // 7
		tcell.PaletteColor(c.Theme.Colors.CursorColorBG),     // 8
		tcell.PaletteColor(c.Theme.Colors.LineColor),         // 9
	}
	g.cfg = c
}

// SetKomi sets the komi value on the info panel.
func (g *GoBoardUI) SetKomi(komi float64) {
	if g.infoPanel != nil {
		g.infoPanel.SetKomi(komi)
	}
}

func (g *GoBoardUI) refreshHint() {
	// Update info panel if available
	if g.infoPanel != nil {
		if g.planningMode && g.planTree != nil {
			g.infoPanel.SetPlanningMode(g.planTree)
		} else {
			g.infoPanel.ClearPlanningMode()
		}
		g.infoPanel.SetBoardState(g.BoardState)
	}

	// Focus mode: hide hint pane, bottom title drawn on border via rootPage.SetDrawFunc
	if g.focusMode {
		g.hint.SetText("")
		return
	}

	// Get terminal width for responsive layout
	_, _, width, _ := g.hint.GetInnerRect()
	if width < 40 {
		width = 80 // fallback
	}

	var status, controls string

	if g.planningMode {
		// Planning mode state
		stone := "●"
		colorName := "Black"
		if g.planColor == 2 {
			stone = "○"
			colorName = "White"
		}
		varInfo := ""
		if g.planTree != nil && g.planTree.NumVariations() > 1 {
			varInfo = fmt.Sprintf("  [dimgray]var %d/%d[-]", g.planTree.VariationIndex()+1, g.planTree.NumVariations())
		}
		status = fmt.Sprintf("[yellow]PLAN[-] %s %s%s", stone, colorName, varInfo)
		controls = "[dimgray]⏎[-] play  [dimgray]p[-] pass  [dimgray][ ][-] nav  [dimgray]{ }[-] branch  [dimgray]a[-] exit  [dimgray]A[-] resume"
	} else if g.finished {
		// Game over state
		status = fmt.Sprintf("[::b]Game Complete[::-]  %s", g.BoardState.Outcome)
		controls = "[dimgray]q[-] quit"
	} else {
		// Active game state
		if g.eng != nil && g.eng.IsMyTurn() {
			stone := "●"
			color := "Black"
			if g.eng.GetPlayerColor() == 2 {
				stone = "○"
				color = "White"
			}
			if g.lastTurnPass {
				status = fmt.Sprintf("%s Your move (%s)  [dimgray]· opponent passed[-]", stone, color)
			} else {
				status = fmt.Sprintf("%s Your move (%s)", stone, color)
			}
		} else {
			status = "[dimgray]◌[-] Thinking..."
		}
		controls = "[dimgray]hjkl[-] move  [dimgray]⏎[-] play  [dimgray]p[-] pass  [dimgray]u[-] undo  [dimgray]r[-] rec  [dimgray]a[-] plan  [dimgray]f[-] focus  [dimgray]q[-] quit"
	}

	// Prepend REC indicator when recording
	rec := ""
	if g.recorder != nil {
		rec = "[red]REC[-] "
	}

	// Build the horizontal bar: status left, controls right
	// Calculate spacing to push controls to the right
	statusLen := len(tview.TranslateANSI(rec + status))
	controlsLen := len(tview.TranslateANSI(controls))
	padding := width - statusLen - controlsLen - 4 // 4 for margins
	if padding < 2 {
		padding = 2
	}

	spacer := ""
	for i := 0; i < padding; i++ {
		spacer += " "
	}

	g.hint.SetText(fmt.Sprintf("  %s%s%s%s", rec, status, spacer, controls))
}

// IsFinished returns true if the game is over.
func (g *GoBoardUI) IsFinished() bool {
	return g.finished
}

// drawStoneCell draws a stone cell (2 characters wide)
func drawStoneCell(s tcell.Screen, c tcell.Style, r rune, x, y, l, t int) {
	// Stone at position 0
	s.SetContent(l+x*2, t+y, r, nil, c)
	// Position 1: space (stone covers the area, no line)
	s.SetContent(l+x*2+1, t+y, ' ', nil, c)
}

// drawGridCell draws a cell using box-drawing characters for grid lines
func drawGridCell(s tcell.Screen, c tcell.Style, r rune, x, y, l, t, boardWidth int, hasStoneRight bool) {
	// 2-char cell: [intersection][right-line]
	s.SetContent(l+x*2, t+y, r, nil, c)

	// Right connector: space if at right edge or if there's a stone to the right
	rightConn := '─'
	if x == boardWidth-1 || hasStoneRight {
		rightConn = ' '
	}
	s.SetContent(l+x*2+1, t+y, rightConn, nil, c)
}

// getGridRune returns the appropriate box-drawing character for a grid position
func getGridRune(x, y, width, height int, isHoshi bool) rune {
	if isHoshi {
		return '◦' // Subtle star point marker
	}

	isTop := y == 0
	isBottom := y == height-1
	isLeft := x == 0
	isRight := x == width-1

	switch {
	case isTop && isLeft:
		return '┌'
	case isTop && isRight:
		return '┐'
	case isBottom && isLeft:
		return '└'
	case isBottom && isRight:
		return '┘'
	case isTop:
		return '┬'
	case isBottom:
		return '┴'
	case isLeft:
		return '├'
	case isRight:
		return '┤'
	default:
		return '┼'
	}
}

// isHoshiPoint checks if a position is a hoshi (star point) on the board
func isHoshiPoint(x, y, boardSize int) bool {
	var hoshiPositions [][2]int

	switch boardSize {
	case 9:
		hoshiPositions = [][2]int{
			{2, 2}, {2, 6},
			{4, 4},
			{6, 2}, {6, 6},
		}
	case 13:
		hoshiPositions = [][2]int{
			{3, 3}, {3, 9},
			{6, 6},
			{9, 3}, {9, 9},
		}
	case 19:
		hoshiPositions = [][2]int{
			{3, 3}, {3, 9}, {3, 15},
			{9, 3}, {9, 9}, {9, 15},
			{15, 3}, {15, 9}, {15, 15},
		}
	default:
		return false
	}

	for _, pos := range hoshiPositions {
		if x == pos[0] && y == pos[1] {
			return true
		}
	}
	return false
}

func drawCoordinates(s tcell.Screen, x, y int, ui *GoBoardUI) {
	hCoord := int('A')
	w, h := ui.BoardState.Width(), ui.BoardState.Height()
	if ui.cfg.Theme.FullWidthLetters {
		hCoord = int('Ａ')
	}

	lmX, lmY := ui.BoardState.LastMove.X, ui.BoardState.LastMove.Y
	if ui.planningMode {
		lmX, lmY = ui.planLastMove[0], ui.planLastMove[1]
	}

	style := tcell.StyleDefault
	highlight := tcell.StyleDefault.Background(ui.styles[8])
	lpHighlight := tcell.StyleDefault.Background(ui.styles[7])

	for ix := 0; ix < w; ix++ {
		_style := style
		if ix == ui.selX {
			_style = highlight
		} else if ix == lmX {
			_style = lpHighlight
		}
		// 2-char cells
		s.SetContent(x+4+(ix*2), y+h+1, rune(hCoord+ix), nil, _style)
		s.SetContent(x+4+(ix*2)+1, y+h+1, ' ', nil, _style)
	}

	for iy := 0; iy < h; iy++ {
		iyInv := h - iy - 1 // Board coordinates starts top left, Go board starts bottom left
		_style := style
		if iyInv == ui.selY {
			_style = highlight
		} else if iyInv == lmY {
			_style = lpHighlight
		}
		displayNum := iy + 1
		tensRune := ' '
		if displayNum >= 10 {
			tensRune = rune('0' + int((displayNum-(displayNum%10))/10))
		}
		s.SetContent(1, y+h-iy-1, tensRune, nil, _style)
		s.SetContent(2, y+h-iy-1, rune('0'+(displayNum%10)), nil, _style)
	}
	s.Show()
}
