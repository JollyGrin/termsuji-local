package ui

import "github.com/gdamore/tcell/v2"

// MenuColors defines the Nord-inspired color palette for the menu UI.
var MenuColors = struct {
	Border      tcell.Color // Muted blue-gray for borders
	BorderFocus tcell.Color // Brighter blue for focused borders
	CardBG      tcell.Color // Dark gray background
	Title       tcell.Color // Bright white for title
	TitleAccent tcell.Color // Blue accent for decoration
	Label       tcell.Color // Light gray for labels
	Hint        tcell.Color // Dim gray for hints
	Selected    tcell.Color // Bright blue for selected items
	Unselected  tcell.Color // Dim gray for unselected items
	ButtonBG    tcell.Color // Button background
	ButtonFocus tcell.Color // Focused button
	ButtonText  tcell.Color // Button text
}{
	Border:      tcell.PaletteColor(60),  // Muted blue-gray
	BorderFocus: tcell.PaletteColor(109), // Brighter blue
	CardBG:      tcell.PaletteColor(236), // Dark gray
	Title:       tcell.PaletteColor(255), // Bright white
	TitleAccent: tcell.PaletteColor(109), // Blue accent
	Label:       tcell.PaletteColor(250), // Light gray
	Hint:        tcell.PaletteColor(245), // Dim gray
	Selected:    tcell.PaletteColor(109), // Bright blue
	Unselected:  tcell.PaletteColor(245), // Dim gray
	ButtonBG:    tcell.PaletteColor(60),  // Nord blue
	ButtonFocus: tcell.PaletteColor(109), // Brighter blue
	ButtonText:  tcell.PaletteColor(255), // White
}
