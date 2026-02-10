// Package ui specifies custom controls for tview to assist in playing Go in the terminal.
package ui

import (
	"fmt"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"

	"termsuji-local/config"
	"termsuji-local/engine"
	"termsuji-local/types"
)

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
	if g.BoardState.Finished() {
		g.ResetSelection()
		return
	}
	prevTile := g.SelectedTile()
	if prevTile == nil {
		g.selX = g.BoardState.LastMove.X
		g.selY = g.BoardState.LastMove.Y
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

		for boardY := 0; boardY < goBoard.BoardState.Height(); boardY++ {
			for boardX := 0; boardX < goBoard.BoardState.Width(); boardX++ {
				stone := goBoard.BoardState.Board[boardY][boardX]
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
				} else if boardX == goBoard.BoardState.LastMove.X && boardY == goBoard.BoardState.LastMove.Y {
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
						hasStoneRight = goBoard.BoardState.Board[boardY][boardX+1] > 0
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

	if err := e.Connect(); err != nil {
		return err
	}

	e.OnMove(func(x, y, color int, boardState *types.BoardState) {
		g.lastTurnPass = (x == -1 && y == -1)
		g.BoardState = boardState
		g.refreshHint()
		// Spawn goroutine to avoid deadlock when called from main thread
		go func() {
			g.app.QueueUpdateDraw(func() {})
		}()
	})

	e.OnGameEnd(func(outcome string) {
		g.finished = true
		g.BoardState = e.GetBoardState()
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

// Close disconnects the engine.
func (g *GoBoardUI) Close() {
	if g.eng == nil {
		return
	}
	g.eng.Close()
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
		g.infoPanel.SetBoardState(g.BoardState)
	}

	// Focus mode shows minimal hint
	if g.focusMode {
		g.hint.SetText("  f to toggle")
		return
	}

	var statusLine, turnLine, controlsLine string

	if g.finished {
		// Game over state
		statusLine = "───────── Game Complete ─────────\n\n"
		turnLine = fmt.Sprintf("  Result: %s\n", g.BoardState.Outcome)
		controlsLine = "\n  q · return to menu"
	} else {
		// Active game state
		if g.lastTurnPass {
			statusLine = "  ○ Opponent passed\n\n"
		}

		if g.eng != nil && g.eng.IsMyTurn() {
			stone := "●"
			color := "Black"
			if g.eng.GetPlayerColor() == 2 {
				stone = "○"
				color = "White"
			}
			turnLine = fmt.Sprintf("  %s Your move (%s)\n", stone, color)
		} else {
			turnLine = "  ◌ Thinking...\n"
		}

		controlsLine = `
  hjkl/↑↓←→ move   ⏎ play
         p pass   f focus   q quit`
	}

	g.hint.SetText(fmt.Sprintf("%s%s%s", statusLine, turnLine, controlsLine))
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

	style := tcell.StyleDefault
	highlight := tcell.StyleDefault.Background(ui.styles[8])
	lpHighlight := tcell.StyleDefault.Background(ui.styles[7])

	for ix := 0; ix < w; ix++ {
		_style := style
		if ix == ui.selX {
			_style = highlight
		} else if ix == ui.BoardState.LastMove.X {
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
		} else if iyInv == ui.BoardState.LastMove.Y {
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
