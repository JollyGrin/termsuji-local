// termsuji-local is a terminal application to play Go against GnuGo offline.
package main

import (
	"flag"
	"fmt"
	"os/exec"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"

	"termsuji-local/config"
	"termsuji-local/engine"
	"termsuji-local/engine/gtp"
	"termsuji-local/ui"
)

// Command-line flags
var (
	flagBoardSize  = flag.Int("boardsize", 0, "Board size (9, 13, or 19)")
	flagColor      = flag.String("color", "", "Player color (black or white)")
	flagDifficulty = flag.Int("difficulty", 0, "GnuGo difficulty level (1-10)")
	flagKomi       = flag.Float64("komi", -1, "Komi value")
	flagQuickStart = flag.Bool("play", false, "Start game immediately with defaults")
)

var app *tview.Application
var rootPage *tview.Pages
var gameBoard *ui.GoBoardUI
var gameFrame *tview.Flex
var gameHint *tview.TextView
var cfg *config.Config

func main() {
	flag.Parse()

	var err error
	cfg, err = config.InitConfig()
	if err != nil {
		panic(err)
	}

	// Always use the default theme (lines theme) on startup
	cfg.Theme = config.DefaultTheme

	// Check if GnuGo is available
	if err := checkGnuGo(); err != nil {
		fmt.Println("Error: GnuGo not found.")
		fmt.Println("Please install GnuGo:")
		fmt.Println("  macOS:  brew install gnugo")
		fmt.Println("  Ubuntu: sudo apt install gnugo")
		fmt.Println("  Fedora: sudo dnf install gnugo")
		return
	}

	// Check if quick start requested
	quickStart := *flagQuickStart || *flagBoardSize > 0 || *flagColor != "" || *flagDifficulty > 0 || *flagKomi >= 0

	app = tview.NewApplication()
	rootPage = tview.NewPages()
	rootPage.SetBorder(true).SetTitle(" termsuji-local ")

	// Game view setup
	gameFrame = tview.NewFlex().SetDirection(tview.FlexRow)
	gameHint = tview.NewTextView()
	gameHint.SetBorder(true)
	gameBoard = ui.NewGoBoard(app, cfg, gameHint)

	// Initial layout: board on top, hint at bottom
	gameFrame.
		AddItem(gameBoard.Box, 0, 1, true).
		AddItem(gameHint, 7, 0, false)

	// Game board input handling
	gameBoard.Box.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		if event.Key() == tcell.KeyRune && event.Rune() == 'q' {
			if gameBoard.SelectedTile() != nil {
				gameBoard.ResetSelection()
			} else {
				gameBoard.Close()
				rootPage.SwitchToPage("setup")
			}
			return nil
		}
		switch event.Key() {
		case tcell.KeyUp:
			gameBoard.MoveSelection(0, -1)
		case tcell.KeyDown:
			gameBoard.MoveSelection(0, 1)
		case tcell.KeyLeft:
			gameBoard.MoveSelection(-1, 0)
		case tcell.KeyRight:
			gameBoard.MoveSelection(1, 0)
		case tcell.KeyEnter:
			selTile := gameBoard.SelectedTile()
			if selTile == nil {
				return nil
			}
			gameBoard.PlayMove(selTile.X, selTile.Y)
		case tcell.KeyRune:
			switch event.Rune() {
			case 'p':
				gameBoard.Pass()
			}
		}
		return event
	})

	// Game setup screen
	setupUI := ui.NewGameSetup(
		func(gameCfg engine.GameConfig) {
			startGame(gameCfg)
		},
		func() {
			app.Stop()
		},
		func() {
			rootPage.SwitchToPage("colors")
		},
	)

	// Color configuration screen
	colorConfig := ui.NewColorConfig(cfg, func() {
		// Refresh the game board with new colors
		gameBoard.SetConfig(cfg)
		rootPage.SwitchToPage("setup")
	})
	colorConfig.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		if event.Key() == tcell.KeyEsc || (event.Key() == tcell.KeyRune && event.Rune() == 'q') {
			rootPage.SwitchToPage("setup")
			return nil
		}
		if event.Key() == tcell.KeyTab {
			colorConfig.ToggleMode()
			return nil
		}
		return event
	})

	// Add pages - start on setup by default, or gameview if quick start
	rootPage.AddPage("setup", setupUI.Form(), true, !quickStart)
	rootPage.AddPage("gameview", gameFrame, true, quickStart)
	rootPage.AddPage("colors", colorConfig.Flex(), true, false)

	// Quick start if flags provided
	if quickStart {
		gameCfg := buildGameConfigFromFlags()
		startGame(gameCfg)
	}

	if err := app.SetRoot(rootPage, true).Run(); err != nil {
		panic(err)
	}
}

// startGame starts a game with the given configuration.
func startGame(gameCfg engine.GameConfig) {
	// Use configured GnuGo path
	gameCfg.EnginePath = cfg.GnuGo.Path

	// Update game board flex layout
	gameFrame.Clear()
	gameFrame.
		AddItem(gameBoard.Box, 0, 1, true).
		AddItem(gameHint, 7, 0, false)

	// Start the game
	eng := gtp.NewGTPEngine(gameCfg)
	if err := gameBoard.ConnectEngine(eng); err != nil {
		// Show error modal
		modal := tview.NewModal().
			SetText(fmt.Sprintf("Failed to start game:\n%s", err.Error())).
			AddButtons([]string{"OK"}).
			SetDoneFunc(func(buttonIndex int, buttonLabel string) {
				rootPage.HidePage("error")
			})
		rootPage.AddPage("error", modal, true, true)
		return
	}
	rootPage.SwitchToPage("gameview")
}

// buildGameConfigFromFlags creates a GameConfig from command-line flags.
func buildGameConfigFromFlags() engine.GameConfig {
	// Start with defaults
	gameCfg := engine.GameConfig{
		BoardSize:   cfg.GnuGo.DefaultBoardSize,
		Komi:        cfg.GnuGo.DefaultKomi,
		PlayerColor: 1, // Black by default
		EngineLevel: cfg.GnuGo.DefaultLevel,
		EnginePath:  cfg.GnuGo.Path,
	}

	// Override with flags
	if *flagBoardSize == 9 || *flagBoardSize == 13 || *flagBoardSize == 19 {
		gameCfg.BoardSize = *flagBoardSize
	}

	if *flagColor == "black" || *flagColor == "b" {
		gameCfg.PlayerColor = 1
	} else if *flagColor == "white" || *flagColor == "w" {
		gameCfg.PlayerColor = 2
	}

	if *flagDifficulty >= 1 && *flagDifficulty <= 10 {
		gameCfg.EngineLevel = *flagDifficulty
	}

	if *flagKomi >= 0 {
		gameCfg.Komi = *flagKomi
	}

	return gameCfg
}

// checkGnuGo verifies that GnuGo is installed and accessible.
func checkGnuGo() error {
	path := cfg.GnuGo.Path
	if path == "" {
		path = "gnugo"
	}
	_, err := exec.LookPath(path)
	return err
}
