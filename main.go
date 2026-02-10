// termsuji-local is a terminal application to play Go against GnuGo offline.
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"runtime"
	"strings"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"

	"termsuji-local/config"
	"termsuji-local/engine"
	"termsuji-local/engine/gtp"
	"termsuji-local/ui"
)

// Version is set at build time via ldflags
var Version = "dev"

// Command-line flags
var (
	flagBoardSize  = flag.Int("boardsize", 0, "Board size (9, 13, or 19)")
	flagColor      = flag.String("color", "", "Player color (black or white)")
	flagDifficulty = flag.Int("difficulty", 0, "GnuGo difficulty level (1-10)")
	flagKomi       = flag.Float64("komi", -1, "Komi value")
	flagQuickStart = flag.Bool("play", false, "Start game immediately with defaults")
	flagFocus      = flag.Bool("focus", false, "Start in focus mode (fullscreen board)")
	flagVersion    = flag.Bool("version", false, "Print version and exit")
	flagUpdate     = flag.Bool("update", false, "Update to the latest version")
)

var app *tview.Application
var rootPage *tview.Pages
var gameBoard *ui.GoBoardUI
var gameFrame *tview.Flex
var gameHint *tview.TextView
var cfg *config.Config

func main() {
	flag.Parse()

	// Handle --version
	if *flagVersion {
		latest, err := getLatestVersion()
		if err != nil {
			fmt.Printf("termsuji-local %s\n", Version)
		} else if latest != Version && Version != "dev" {
			fmt.Printf("termsuji-local %s (update available: %s)\n", Version, latest)
			fmt.Println("Run 'termsuji-local --update' to update")
		} else {
			fmt.Printf("termsuji-local %s (latest)\n", Version)
		}
		return
	}

	// Handle --update
	if *flagUpdate {
		if err := selfUpdate(); err != nil {
			fmt.Printf("Update failed: %s\n", err)
			os.Exit(1)
		}
		return
	}

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
		fmt.Println("  macOS:  brew install gnu-go")
		fmt.Println("  Ubuntu: sudo apt install gnugo")
		fmt.Println("  Fedora: sudo dnf install gnugo")
		return
	}

	// Check if quick start requested
	quickStart := *flagQuickStart || *flagBoardSize > 0 || *flagColor != "" || *flagDifficulty > 0 || *flagKomi >= 0 || *flagFocus

	app = tview.NewApplication()
	rootPage = tview.NewPages()
	rootPage.SetBorder(true).SetTitle(" ⬡ termsuji ")

	// Game view setup
	gameHint = tview.NewTextView()
	gameHint.SetBorder(true)
	gameHint.SetBorderPadding(0, 0, 1, 1)
	gameHint.SetTitle(" Status ")
	gameHint.SetTitleAlign(tview.AlignLeft)
	gameBoard = ui.NewGoBoard(app, cfg, gameHint)

	// Create game layout with centered board and side panel
	gameFrame = ui.CreateGameLayout(gameBoard, gameHint)

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
			case 'h':
				gameBoard.MoveSelection(-1, 0)
			case 'j':
				gameBoard.MoveSelection(0, 1)
			case 'k':
				gameBoard.MoveSelection(0, -1)
			case 'l':
				gameBoard.MoveSelection(1, 0)
			case 'p':
				gameBoard.Pass()
			case 'f':
				if gameBoard.ToggleFocusMode() {
					ui.BuildFocusLayout(gameFrame, gameBoard, gameHint)
				} else {
					ui.RebuildNormalLayout(gameFrame, gameBoard, gameHint)
				}
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
		// Enter focus mode if requested
		if *flagFocus {
			gameBoard.SetFocusMode(true)
			ui.BuildFocusLayout(gameFrame, gameBoard, gameHint)
		}
	}

	if err := app.SetRoot(rootPage, true).Run(); err != nil {
		panic(err)
	}
}

// startGame starts a game with the given configuration.
func startGame(gameCfg engine.GameConfig) {
	// Use configured GnuGo path
	gameCfg.EnginePath = cfg.GnuGo.Path

	// Set komi on info panel
	gameBoard.SetKomi(gameCfg.Komi)

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

// getLatestVersion fetches the latest release version from GitHub.
func getLatestVersion() (string, error) {
	resp, err := http.Get("https://api.github.com/repos/JollyGrin/termsuji-local/releases/latest")
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	var release struct {
		TagName string `json:"tag_name"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&release); err != nil {
		return "", err
	}
	return release.TagName, nil
}

// selfUpdate downloads and installs the latest version.
func selfUpdate() error {
	fmt.Println("Checking for updates...")

	latest, err := getLatestVersion()
	if err != nil {
		return fmt.Errorf("failed to check for updates: %w", err)
	}

	if latest == Version {
		fmt.Printf("Already at latest version (%s)\n", Version)
		return nil
	}

	fmt.Printf("Updating from %s to %s...\n", Version, latest)

	// Determine OS and arch
	goos := runtime.GOOS
	goarch := runtime.GOARCH

	ext := ""
	if goos == "windows" {
		ext = ".exe"
	}

	// Download URL
	filename := fmt.Sprintf("termsuji-local_%s_%s%s", goos, goarch, ext)
	url := fmt.Sprintf("https://github.com/JollyGrin/termsuji-local/releases/download/%s/%s", latest, filename)

	// Download to temp file
	resp, err := http.Get(url)
	if err != nil {
		return fmt.Errorf("failed to download: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return fmt.Errorf("download failed: HTTP %d", resp.StatusCode)
	}

	// Get current executable path
	execPath, err := os.Executable()
	if err != nil {
		return fmt.Errorf("failed to get executable path: %w", err)
	}

	// Resolve symlinks
	execPath, err = resolveSymlinks(execPath)
	if err != nil {
		return fmt.Errorf("failed to resolve path: %w", err)
	}

	// Write to temp file
	tmpFile, err := os.CreateTemp("", "termsuji-local-update-*")
	if err != nil {
		return fmt.Errorf("failed to create temp file: %w", err)
	}
	tmpPath := tmpFile.Name()

	_, err = io.Copy(tmpFile, resp.Body)
	tmpFile.Close()
	if err != nil {
		os.Remove(tmpPath)
		return fmt.Errorf("failed to write update: %w", err)
	}

	// Make executable
	if err := os.Chmod(tmpPath, 0755); err != nil {
		os.Remove(tmpPath)
		return fmt.Errorf("failed to set permissions: %w", err)
	}

	// Replace old binary
	if err := os.Rename(tmpPath, execPath); err != nil {
		os.Remove(tmpPath)
		return fmt.Errorf("failed to replace binary: %w", err)
	}

	fmt.Printf("Updated to %s\n", latest)
	return nil
}

// resolveSymlinks resolves the final path of the executable.
func resolveSymlinks(path string) (string, error) {
	for {
		info, err := os.Lstat(path)
		if err != nil {
			return path, err
		}
		if info.Mode()&os.ModeSymlink == 0 {
			return path, nil
		}
		link, err := os.Readlink(path)
		if err != nil {
			return path, err
		}
		if !strings.HasPrefix(link, "/") {
			// Relative symlink
			path = path[:strings.LastIndex(path, "/")+1] + link
		} else {
			path = link
		}
	}
}
