package ui

import (
	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

// MenuCard is a styled card container with rounded borders and title.
type MenuCard struct {
	*tview.Box
	title   string
	focused bool
}

// NewMenuCard creates a new menu card with the given title.
func NewMenuCard(title string) *MenuCard {
	card := &MenuCard{
		Box:   tview.NewBox(),
		title: title,
	}
	return card
}

// Draw renders the menu card with rounded borders.
func (c *MenuCard) Draw(screen tcell.Screen) {
	c.Box.DrawForSubclass(screen, c)

	x, y, width, height := c.GetInnerRect()
	if width < 10 || height < 5 {
		return
	}

	borderColor := MenuColors.Border
	if c.focused {
		borderColor = MenuColors.BorderFocus
	}
	borderStyle := tcell.StyleDefault.Foreground(borderColor).Background(MenuColors.CardBG)
	bgStyle := tcell.StyleDefault.Background(MenuColors.CardBG)

	// Fill background
	for row := y; row < y+height; row++ {
		for col := x; col < x+width; col++ {
			screen.SetContent(col, row, ' ', nil, bgStyle)
		}
	}

	// Draw rounded corners and borders
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

	// Draw title centered with decoration
	if c.title != "" {
		titleStyle := tcell.StyleDefault.Foreground(MenuColors.Title).Background(MenuColors.CardBG).Bold(true)
		accentStyle := tcell.StyleDefault.Foreground(MenuColors.TitleAccent).Background(MenuColors.CardBG)

		// Title with hexagon decoration: ⬡ T E R M S U J I
		fullTitle := "⬡  " + c.title
		titleLen := len([]rune(fullTitle))
		titleX := x + (width-titleLen)/2

		// Draw on row y+2 (after top border and a blank line)
		titleY := y + 2

		// Draw the accent character
		screen.SetContent(titleX, titleY, '⬡', nil, accentStyle)

		// Draw spaces
		screen.SetContent(titleX+1, titleY, ' ', nil, bgStyle)
		screen.SetContent(titleX+2, titleY, ' ', nil, bgStyle)

		// Draw title text
		for i, ch := range c.title {
			screen.SetContent(titleX+3+i, titleY, ch, nil, titleStyle)
		}

		// Draw divider after title: ├───┤
		divY := y + 4
		screen.SetContent(x, divY, '├', nil, borderStyle)
		for col := x + 1; col < x+width-1; col++ {
			screen.SetContent(col, divY, '─', nil, borderStyle)
		}
		screen.SetContent(x+width-1, divY, '┤', nil, borderStyle)
	}
}

// DrawDivider draws a horizontal divider at the given y position.
func (c *MenuCard) DrawDivider(screen tcell.Screen, divY int) {
	x, _, width, _ := c.GetInnerRect()
	borderColor := MenuColors.Border
	if c.focused {
		borderColor = MenuColors.BorderFocus
	}
	borderStyle := tcell.StyleDefault.Foreground(borderColor).Background(MenuColors.CardBG)

	screen.SetContent(x, divY, '├', nil, borderStyle)
	for col := x + 1; col < x+width-1; col++ {
		screen.SetContent(col, divY, '─', nil, borderStyle)
	}
	screen.SetContent(x+width-1, divY, '┤', nil, borderStyle)
}

// SetFocused sets the focus state of the card.
func (c *MenuCard) SetFocused(focused bool) {
	c.focused = focused
}
