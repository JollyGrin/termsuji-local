// Package ui provides terminal UI components for termsuji-local.
package ui

import (
	"fmt"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"

	"termsuji-local/config"
)

// ColorConfigUI provides a color configuration screen with live preview.
type ColorConfigUI struct {
	flex       *tview.Flex
	colorList  *tview.List
	preview    *tview.Box
	cfg        *config.Config
	onDone     func()

	// Current selection
	selectedBoardColor int
	selectedLineColor  int
	editingLine        bool // true = editing line color, false = editing board color
}

// Common board colors to choose from (warm wood-like tones)
var boardColors = []struct {
	code int
	name string
}{
	{230, "Light Cream"},
	{229, "Pale Yellow"},
	{228, "Light Gold"},
	{222, "Gold"},
	{220, "Bright Yellow"},
	{214, "Orange Gold"},
	{208, "Dark Orange"},
	{180, "Tan"},
	{179, "Light Brown"},
	{172, "Brown"},
	{136, "Dark Brown"},
	{94, "Saddle Brown"},
	{252, "Light Gray"},
	{250, "Gray"},
	{248, "Medium Gray"},
	{244, "Dark Gray"},
	{188, "Light Beige"},
	{181, "Dusty Rose"},
	{223, "Peach"},
	{216, "Salmon"},
}

// Line colors (darker tones that contrast with board)
var lineColors = []struct {
	code int
	name string
}{
	{94, "Saddle Brown"},
	{130, "Dark Orange"},
	{136, "Dark Brown"},
	{88, "Dark Red"},
	{52, "Dark Maroon"},
	{22, "Dark Green"},
	{23, "Teal"},
	{24, "Dark Cyan"},
	{17, "Navy Blue"},
	{54, "Purple"},
	{232, "Black"},
	{236, "Dark Gray"},
	{240, "Gray"},
	{244, "Medium Gray"},
	{16, "True Black"},
}

// NewColorConfig creates a new color configuration screen.
func NewColorConfig(cfg *config.Config, onDone func()) *ColorConfigUI {
	cc := &ColorConfigUI{
		cfg:                cfg,
		onDone:             onDone,
		selectedBoardColor: cfg.Theme.Colors.BoardColor,
		selectedLineColor:  cfg.Theme.Colors.LineColor,
		editingLine:        false,
	}

	// Create the color list
	cc.colorList = tview.NewList()
	cc.colorList.SetBorder(true)
	cc.colorList.ShowSecondaryText(false)

	// Populate with board colors initially
	cc.populateColorList()

	// Handle selection change (preview)
	cc.colorList.SetChangedFunc(func(index int, mainText, secondaryText string, shortcut rune) {
		if cc.editingLine {
			if index >= 0 && index < len(lineColors) {
				cc.selectedLineColor = lineColors[index].code
				cc.updatePreview()
			}
		} else {
			if index >= 0 && index < len(boardColors) {
				cc.selectedBoardColor = boardColors[index].code
				cc.updatePreview()
			}
		}
	})

	// Handle selection confirm (apply)
	cc.colorList.SetSelectedFunc(func(index int, mainText, secondaryText string, shortcut rune) {
		if cc.editingLine {
			if index >= 0 && index < len(lineColors) {
				cc.cfg.Theme.Colors.LineColor = cc.selectedLineColor
				cc.cfg.Save()
				// Switch back to board color selection
				cc.editingLine = false
				cc.populateColorList()
			}
		} else {
			if index >= 0 && index < len(boardColors) {
				cc.cfg.Theme.Colors.BoardColor = cc.selectedBoardColor
				cc.cfg.Theme.Colors.BoardColorAlt = cc.selectedBoardColor
				cc.cfg.Save()
				onDone()
			}
		}
	})

	// Create preview box
	cc.preview = tview.NewBox()
	cc.preview.SetBorder(true)
	cc.preview.SetTitle(" Board Preview ")
	cc.preview.SetDrawFunc(cc.drawPreview)

	// Layout: list on left, preview on right
	cc.flex = tview.NewFlex().
		AddItem(cc.colorList, 30, 0, true).
		AddItem(cc.preview, 0, 1, false)

	return cc
}

// populateColorList fills the list with appropriate colors based on editing mode.
func (cc *ColorConfigUI) populateColorList() {
	cc.colorList.Clear()

	if cc.editingLine {
		cc.colorList.SetTitle(" Select Line Color (Tab: switch to board) ")
		for i, c := range lineColors {
			cc.colorList.AddItem(fmt.Sprintf("[#%06x]████[-] %s (%d)",
				tcell.PaletteColor(c.code).Hex(), c.name, c.code),
				"", rune('a'+i), nil)
		}
		// Set current selection
		for i, c := range lineColors {
			if c.code == cc.selectedLineColor {
				cc.colorList.SetCurrentItem(i)
				break
			}
		}
	} else {
		cc.colorList.SetTitle(" Select Board Color (Tab: switch to line) ")
		for i, c := range boardColors {
			cc.colorList.AddItem(fmt.Sprintf("[#%06x]████[-] %s (%d)",
				tcell.PaletteColor(c.code).Hex(), c.name, c.code),
				"", rune('a'+i), nil)
		}
		// Set current selection
		for i, c := range boardColors {
			if c.code == cc.selectedBoardColor {
				cc.colorList.SetCurrentItem(i)
				break
			}
		}
	}
}

func (cc *ColorConfigUI) updatePreview() {
	// Trigger redraw
	if cc.preview != nil {
		go func() {
			// Force redraw by invalidating
		}()
	}
}

func (cc *ColorConfigUI) drawPreview(screen tcell.Screen, x, y, width, height int) (int, int, int, int) {
	// Draw a mini Go board preview with the selected colors
	boardColor := tcell.PaletteColor(cc.selectedBoardColor)
	blackColor := tcell.PaletteColor(cc.cfg.Theme.Colors.BlackColor)
	whiteColor := tcell.PaletteColor(cc.cfg.Theme.Colors.WhiteColor)
	lineColor := tcell.PaletteColor(cc.selectedLineColor)

	boardStyle := tcell.StyleDefault.Background(boardColor).Foreground(lineColor)
	blackStyle := tcell.StyleDefault.Background(boardColor).Foreground(blackColor)
	whiteStyle := tcell.StyleDefault.Background(boardColor).Foreground(whiteColor)

	// Draw a 7x7 preview board
	startX := x + 2
	startY := y + 1
	size := 7

	if width < 20 || height < 10 {
		return x, y, width, height
	}

	// Sample stone positions for preview
	stones := map[[2]int]int{
		{2, 2}: 1, // black
		{2, 3}: 1,
		{3, 2}: 2, // white
		{3, 3}: 2,
		{4, 4}: 1,
		{3, 4}: 2,
	}

	for row := 0; row < size; row++ {
		for col := 0; col < size; col++ {
			screenX := startX + col*2
			screenY := startY + row

			// Determine intersection character
			var char rune
			isTop := row == 0
			isBottom := row == size-1
			isLeft := col == 0
			isRight := col == size-1

			switch {
			case isTop && isLeft:
				char = '┌'
			case isTop && isRight:
				char = '┐'
			case isBottom && isLeft:
				char = '└'
			case isBottom && isRight:
				char = '┘'
			case isTop:
				char = '┬'
			case isBottom:
				char = '┴'
			case isLeft:
				char = '├'
			case isRight:
				char = '┤'
			default:
				char = '┼'
			}

			// Check for stones
			style := boardStyle
			if stoneColor, ok := stones[[2]int{col, row}]; ok {
				char = '●'
				if stoneColor == 1 {
					style = blackStyle
				} else {
					style = whiteStyle
				}
			}

			screen.SetContent(screenX, screenY, char, nil, style)

			// Draw connector (unless at right edge or stone to right)
			if col < size-1 {
				connector := '─'
				_, hasStoneRight := stones[[2]int{col + 1, row}]
				_, hasStone := stones[[2]int{col, row}]
				if hasStoneRight || hasStone {
					connector = ' '
				}
				screen.SetContent(screenX+1, screenY, connector, nil, boardStyle)
			}
		}
	}

	// Draw color info
	infoStyle := tcell.StyleDefault
	var info string
	if cc.editingLine {
		info = fmt.Sprintf("Line: %d  Board: %d", cc.selectedLineColor, cc.selectedBoardColor)
	} else {
		info = fmt.Sprintf("Board: %d  Line: %d", cc.selectedBoardColor, cc.selectedLineColor)
	}
	for i, ch := range info {
		if startX+i < x+width-1 {
			screen.SetContent(startX+i, startY+size+1, ch, nil, infoStyle)
		}
	}

	return x, y, width, height
}

// Flex returns the flex container for this UI.
func (cc *ColorConfigUI) Flex() *tview.Flex {
	return cc.flex
}

// SetInputCapture sets the input capture for the color list.
func (cc *ColorConfigUI) SetInputCapture(capture func(event *tcell.EventKey) *tcell.EventKey) {
	cc.colorList.SetInputCapture(capture)
}

// ToggleMode switches between board color and line color editing.
func (cc *ColorConfigUI) ToggleMode() {
	cc.editingLine = !cc.editingLine
	cc.populateColorList()
}
