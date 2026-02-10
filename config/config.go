package config

import (
	"encoding/json"
	"fmt"
	"io/fs"
	"os"

	"github.com/adrg/xdg"
)

var (
	cfgFile = "termsuji-local/config.json"
)

type InvalidConfig struct {
	err string
}

func (e *InvalidConfig) Error() string {
	return fmt.Sprintf("Config error: %s", e.err)
}

type ConfigColors struct {
	BoardColor        int `json:"board"`
	BoardColorAlt     int `json:"board_alt"`
	BlackColor        int `json:"black"`
	BlackColorAlt     int `json:"black_alt"`
	WhiteColor        int `json:"white"`
	WhiteColorAlt     int `json:"white_alt"`
	LineColor         int `json:"line"`
	CursorColorFG     int `json:"cursor_fg"`
	CursorColorBG     int `json:"cursor_bg"`
	LastPlayedColorBG int `json:"last_played_bg"`
}

type ConfigSymbols struct {
	BlackStone  rune `json:"black"`
	WhiteStone  rune `json:"white"`
	BoardSquare rune `json:"board"`
	Cursor      rune `json:"cursor"`
	LastPlayed  rune `json:"last_played"`
}

type Theme struct {
	DrawStoneBackground      bool          `json:"draw_stone_bg"`
	DrawCursorBackground     bool          `json:"draw_cursor_bg"`
	DrawLastPlayedBackground bool          `json:"draw_last_played_bg"`
	FullWidthLetters         bool          `json:"fullwidth_letters"`
	UseGridLines             bool          `json:"use_grid_lines"`
	Colors                   ConfigColors  `json:"colors"`
	Symbols                  ConfigSymbols `json:"symbols"`
}

// GnuGoConfig holds GnuGo-specific settings.
type GnuGoConfig struct {
	Path             string  `json:"gnugo_path"`
	DefaultBoardSize int     `json:"default_board_size"`
	DefaultKomi      float64 `json:"default_komi"`
	DefaultLevel     int     `json:"default_level"`
}

type Config struct {
	Theme  Theme       `json:"theme"`
	GnuGo  GnuGoConfig `json:"gnugo"`
}

func InitConfig() (*Config, error) {
	config := DefaultConfig
	absPath, err := xdg.SearchConfigFile(cfgFile)
	if err == nil {
		readCfgFile(absPath, &config)
	}
	if err = config.Validate(); err != nil {
		return nil, err
	}
	return &config, nil
}

func (c *Config) Validate() error {
	for _, r := range []rune{c.Theme.Symbols.BlackStone, c.Theme.Symbols.WhiteStone, c.Theme.Symbols.BoardSquare} {
		if r < 32 || (r >= 127 && r <= 159) {
			return &InvalidConfig{"Unicode characters 1-31 and 127-159 are not allowed"}
		}
	}
	return nil
}

func (c *Config) Save() {
	absPath, err := xdg.ConfigFile(cfgFile)
	if err != nil {
		panic(err)
	}
	saveCfgFile(absPath, c, 0664)
}

func saveCfgFile(filePath string, a interface{}, perm fs.FileMode) {
	jsonData, err := json.MarshalIndent(a, "", "  ")
	if err != nil {
		panic(err)
	}
	err = os.WriteFile(filePath, jsonData, perm)
	if err != nil {
		panic(err)
	}
}

func readCfgFile(filePath string, a interface{}) {
	configReader, err := os.ReadFile(filePath)
	if err == nil {
		err = json.Unmarshal(configReader, &a)
		if err != nil {
			panic(err)
		}
	}
}
