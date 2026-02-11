package ui

import (
	"fmt"
	"os"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"

	"termsuji-local/config"
	"termsuji-local/sgf"
)

// HistoryBrowserUI provides a screen for browsing saved SGF game history.
type HistoryBrowserUI struct {
	flex     *tview.Flex
	gameList *tview.List
	preview  *tview.Box
	hint     *tview.TextView
	games    []sgf.GameInfo
	boards   map[int][][]int // cached final positions
	selected int
	onDone   func()
}

// NewHistoryBrowser creates a new history browser screen.
func NewHistoryBrowser(onDone func()) *HistoryBrowserUI {
	hb := &HistoryBrowserUI{
		onDone: onDone,
		boards: make(map[int][][]int),
	}

	// Game list (left panel)
	hb.gameList = tview.NewList()
	hb.gameList.SetBorder(true)
	hb.gameList.SetTitle(" Game History ")
	hb.gameList.ShowSecondaryText(false)
	hb.gameList.SetHighlightFullLine(true)
	hb.gameList.SetMainTextStyle(tcell.StyleDefault.Foreground(MenuColors.Label))
	hb.gameList.SetSelectedStyle(tcell.StyleDefault.
		Foreground(MenuColors.ButtonText).
		Background(MenuColors.ButtonFocus))

	// Preview box (right panel)
	hb.preview = tview.NewBox()
	hb.preview.SetBorder(true)
	hb.preview.SetTitle(" Preview ")
	hb.preview.SetDrawFunc(hb.drawPreview)

	// Hint bar
	hb.hint = tview.NewTextView()
	hb.hint.SetDynamicColors(true)
	hb.hint.SetBorder(false)
	hb.hint.SetText("  [dimgray]d[-] delete  [dimgray]q[-] back")

	// Handle list selection changes
	hb.gameList.SetChangedFunc(func(index int, mainText, secondaryText string, shortcut rune) {
		hb.selected = index
	})

	// Input handling
	hb.gameList.SetInputCapture(hb.handleInput)

	// Layout: list left, preview right, hint bottom
	topRow := tview.NewFlex().SetDirection(tview.FlexColumn).
		AddItem(hb.gameList, 38, 0, true).
		AddItem(hb.preview, 0, 1, false)

	hb.flex = tview.NewFlex().SetDirection(tview.FlexRow).
		AddItem(topRow, 0, 1, true).
		AddItem(hb.hint, 1, 0, false)

	hb.loadGames()
	return hb
}

// Flex returns the flex container for this UI.
func (hb *HistoryBrowserUI) Flex() *tview.Flex {
	return hb.flex
}

// Refresh reloads the game list from disk.
func (hb *HistoryBrowserUI) Refresh() {
	hb.boards = make(map[int][][]int)
	hb.loadGames()
}

// loadGames scans the history directory for SGF files.
func (hb *HistoryBrowserUI) loadGames() {
	hb.gameList.Clear()
	hb.games = nil
	hb.selected = 0

	games, err := sgf.ListGames(config.HistoryDir())
	if err != nil || len(games) == 0 {
		hb.gameList.AddItem("[dimgray]No games found[-]", "", 0, nil)
		return
	}

	hb.games = games
	for _, g := range games {
		result := g.Result
		if result == "" || result == "?" {
			result = "..."
		}
		label := fmt.Sprintf("%s  %dx%d  %s", g.Date, g.BoardSize, g.BoardSize, result)
		hb.gameList.AddItem(label, "", 0, nil)
	}
}

// handleInput processes keyboard input for the history browser.
func (hb *HistoryBrowserUI) handleInput(event *tcell.EventKey) *tcell.EventKey {
	switch event.Key() {
	case tcell.KeyEscape:
		if hb.onDone != nil {
			hb.onDone()
		}
		return nil
	case tcell.KeyRune:
		switch event.Rune() {
		case 'q':
			if hb.onDone != nil {
				hb.onDone()
			}
			return nil
		case 'd':
			hb.deleteSelected()
			return nil
		}
	}
	return event
}

// deleteSelected removes the currently selected game file.
func (hb *HistoryBrowserUI) deleteSelected() {
	if hb.selected < 0 || hb.selected >= len(hb.games) {
		return
	}

	game := hb.games[hb.selected]
	os.Remove(game.FilePath)

	// Clear board cache and reload
	hb.boards = make(map[int][][]int)
	hb.loadGames()
}

// drawPreview renders a mini board preview and game metadata.
func (hb *HistoryBrowserUI) drawPreview(screen tcell.Screen, x, y, width, height int) (int, int, int, int) {
	if hb.selected < 0 || hb.selected >= len(hb.games) {
		return x, y, width, height
	}

	game := hb.games[hb.selected]

	// Lazy-load and cache the board position
	board, ok := hb.boards[hb.selected]
	if !ok {
		b, _, err := sgf.ReplayToEnd(game.FilePath)
		if err == nil {
			board = b
			hb.boards[hb.selected] = board
		}
	}

	// Draw mini board
	if board != nil {
		size := len(board)
		startX := x + 2
		startY := y + 1

		// Check we have room
		if width >= size+4 && height >= size+6 {
			emptyStyle := tcell.StyleDefault.Foreground(tcell.PaletteColor(240))
			blackStyle := tcell.StyleDefault.Foreground(tcell.PaletteColor(255)).Bold(true)
			whiteStyle := tcell.StyleDefault.Foreground(tcell.PaletteColor(250))

			for by := 0; by < size; by++ {
				for bx := 0; bx < size; bx++ {
					ch := '·'
					style := emptyStyle
					switch board[by][bx] {
					case 1:
						ch = '●'
						style = blackStyle
					case 2:
						ch = '○'
						style = whiteStyle
					}
					screen.SetContent(startX+bx, startY+by, ch, nil, style)
				}
			}

			// Metadata below the board
			infoY := startY + size + 1
			infoStyle := tcell.StyleDefault.Foreground(tcell.PaletteColor(250))
			dimStyle := tcell.StyleDefault.Foreground(tcell.PaletteColor(245))

			drawText(screen, startX, infoY, fmt.Sprintf("%dx%d", game.BoardSize, game.BoardSize), infoStyle)
			drawText(screen, startX+6, infoY, fmt.Sprintf("| %d moves", game.MoveCount), dimStyle)

			infoY++
			drawText(screen, startX, infoY, fmt.Sprintf("B: %s", game.PlayerBlack), dimStyle)
			infoY++
			drawText(screen, startX, infoY, fmt.Sprintf("W: %s", game.PlayerWhite), dimStyle)

			infoY++
			result := game.Result
			if result == "" || result == "?" {
				result = "Unfinished"
			}
			resultStyle := tcell.StyleDefault.Foreground(tcell.PaletteColor(109))
			drawText(screen, startX, infoY, fmt.Sprintf("Result: %s", result), resultStyle)
		}
	}

	return x, y, width, height
}

// drawText writes a string to the screen at the given position.
func drawText(screen tcell.Screen, x, y int, text string, style tcell.Style) {
	for i, ch := range text {
		screen.SetContent(x+i, y, ch, nil, style)
	}
}
