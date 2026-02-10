package config

var DefaultConfig Config
var DefaultTheme Theme

func init() {
	DefaultTheme = Theme{
		DrawStoneBackground:      false,
		DrawCursorBackground:     true,
		DrawLastPlayedBackground: true,
		FullWidthLetters:         false,
		UseGridLines:             true,
		Colors: ConfigColors{
			BoardColor:        180,
			BoardColorAlt:     180,
			BlackColor:        232,
			BlackColorAlt:     232,
			WhiteColor:        255,
			WhiteColorAlt:     255,
			LineColor:         94,
			CursorColorFG:     2,
			CursorColorBG:     4,
			LastPlayedColorBG: 2,
		},
		Symbols: ConfigSymbols{
			BlackStone:  '●',
			WhiteStone:  '●',
			BoardSquare: '┼',
			Cursor:      '┼',
			LastPlayed:  '┼',
		},
	}

	DefaultConfig = Config{
		Theme: DefaultTheme,
		GnuGo: GnuGoConfig{
			Path:             "gnugo",
			DefaultBoardSize: 19,
			DefaultKomi:      6.5,
			DefaultLevel:     5,
		},
	}
}
