package config

var DefaultConfig Config
var DefaultTheme Theme

func init() {
	// Minimalist Zen theme - warm wood tones with subtle accents
	DefaultTheme = Theme{
		DrawStoneBackground:      false,
		DrawCursorBackground:     true,
		DrawLastPlayedBackground: true,
		FullWidthLetters:         false,
		UseGridLines:             true,
		Colors: ConfigColors{
			BoardColor:        180, // Warm tan/wood
			BoardColorAlt:     180,
			BlackColor:        232, // Pure black stones
			BlackColorAlt:     232,
			WhiteColor:        255, // Pure white stones
			WhiteColorAlt:     255,
			LineColor:         137, // Subtle brown grid lines
			CursorColorFG:     30,  // Teal accent
			CursorColorBG:     30,  // Teal cursor highlight
			LastPlayedColorBG: 65,  // Soft green for last move
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
