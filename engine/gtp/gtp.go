package gtp

import (
	"bufio"
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
	"strings"
	"sync"

	"termsuji-local/engine"
	"termsuji-local/types"
)

var debugLog *log.Logger

func init() {
	f, _ := os.Create("/tmp/termsuji-debug.log")
	debugLog = log.New(f, "", log.Ltime|log.Lmicroseconds)
}

// GTPEngine implements the GameEngine interface using GnuGo via GTP protocol.
type GTPEngine struct {
	cmd    *exec.Cmd
	stdin  io.WriteCloser
	stdout *bufio.Reader

	config      engine.GameConfig
	boardState  *types.BoardState
	myTurn      bool
	passCount   int
	gameOver    bool
	playerColor int // Human's color (1=black, 2=white)

	moveCallback func(x, y, color int, boardState *types.BoardState)
	endCallback  func(outcome string)

	mu sync.Mutex
}

// NewGTPEngine creates a new GTP engine with the given configuration.
func NewGTPEngine(cfg engine.GameConfig) *GTPEngine {
	return &GTPEngine{
		config:      cfg,
		playerColor: cfg.PlayerColor,
		boardState:  types.NewBoardState(cfg.BoardSize),
	}
}

// Connect starts the GnuGo subprocess and initializes the game.
func (g *GTPEngine) Connect() error {
	// Start GnuGo process
	args := []string{
		"--mode", "gtp",
		"--level", fmt.Sprintf("%d", g.config.EngineLevel),
		"--quiet",
	}
	g.cmd = exec.Command(g.config.EnginePath, args...)

	var err error
	g.stdin, err = g.cmd.StdinPipe()
	if err != nil {
		return fmt.Errorf("failed to get stdin pipe: %w", err)
	}

	stdout, err := g.cmd.StdoutPipe()
	if err != nil {
		return fmt.Errorf("failed to get stdout pipe: %w", err)
	}
	g.stdout = bufio.NewReader(stdout)

	// Discard stderr to prevent blocking
	g.cmd.Stderr = nil

	if err := g.cmd.Start(); err != nil {
		return fmt.Errorf("failed to start GnuGo: %w", err)
	}

	// Initialize the board
	if _, err := g.sendCommand(fmt.Sprintf("boardsize %d", g.config.BoardSize)); err != nil {
		return fmt.Errorf("failed to set board size: %w", err)
	}

	if _, err := g.sendCommand("clear_board"); err != nil {
		return fmt.Errorf("failed to clear board: %w", err)
	}

	if _, err := g.sendCommand(fmt.Sprintf("komi %.1f", g.config.Komi)); err != nil {
		return fmt.Errorf("failed to set komi: %w", err)
	}

	// Determine who plays first
	// Black always plays first in Go
	if g.playerColor == 1 {
		// Human is black, human's turn first
		g.myTurn = true
	} else {
		// Human is white, engine (black) plays first
		g.myTurn = false
		go g.triggerEngineMove()
	}

	return nil
}

// sendCommand sends a GTP command and returns the response.
func (g *GTPEngine) sendCommand(cmd string) (string, error) {
	debugLog.Printf("sendCommand: sending '%s'", cmd)

	// Send command
	_, err := fmt.Fprintf(g.stdin, "%s\n", cmd)
	if err != nil {
		debugLog.Printf("sendCommand: write error: %v", err)
		return "", fmt.Errorf("failed to send command: %w", err)
	}

	debugLog.Printf("sendCommand: waiting for response...")

	// Read response
	var response strings.Builder
	for {
		line, err := g.stdout.ReadString('\n')
		if err != nil {
			debugLog.Printf("sendCommand: read error: %v", err)
			return "", fmt.Errorf("failed to read response: %w", err)
		}

		line = strings.TrimRight(line, "\r\n")
		debugLog.Printf("sendCommand: read line '%s'", line)

		// Empty line signals end of response
		if line == "" {
			break
		}

		if response.Len() > 0 {
			response.WriteString("\n")
		}
		response.WriteString(line)
	}

	result := response.String()
	debugLog.Printf("sendCommand: complete response '%s'", result)

	// Check for error response (starts with '?')
	if strings.HasPrefix(result, "?") {
		return "", fmt.Errorf("GTP error: %s", strings.TrimPrefix(result, "? "))
	}

	// Success response starts with '='
	return strings.TrimPrefix(strings.TrimPrefix(result, "="), " "), nil
}

// GetBoardState returns the current board state.
func (g *GTPEngine) GetBoardState() *types.BoardState {
	g.mu.Lock()
	defer g.mu.Unlock()
	return g.boardState
}

// PlayMove plays a move at the given coordinates.
func (g *GTPEngine) PlayMove(x, y int) error {
	debugLog.Printf("PlayMove: starting x=%d y=%d", x, y)
	g.mu.Lock()
	debugLog.Printf("PlayMove: acquired lock")

	if g.gameOver {
		g.mu.Unlock()
		return fmt.Errorf("game is over")
	}

	if !g.myTurn {
		g.mu.Unlock()
		return fmt.Errorf("not your turn")
	}

	vertex := posToGTP(x, y, g.config.BoardSize)
	color := colorToGTP(g.playerColor)

	debugLog.Printf("PlayMove: sending play command")
	_, err := g.sendCommand(fmt.Sprintf("play %s %s", color, vertex))
	if err != nil {
		debugLog.Printf("PlayMove: play command failed: %v", err)
		g.mu.Unlock()
		return fmt.Errorf("illegal move: %w", err)
	}
	debugLog.Printf("PlayMove: play command succeeded")

	// Update board state
	g.boardState.Board[y][x] = g.playerColor
	g.boardState.LastMove.X = x
	g.boardState.LastMove.Y = y
	g.boardState.MoveNumber++
	g.boardState.PlayerToMove = oppositeColor(g.playerColor)
	g.passCount = 0

	// Update captures by refreshing board state from GnuGo
	debugLog.Printf("PlayMove: updating board from GnuGo")
	g.updateBoardFromGnuGo()
	debugLog.Printf("PlayMove: board updated")

	g.myTurn = false
	playerColor := g.playerColor
	// Copy board state before releasing lock
	boardStateCopy := g.copyBoardState()
	debugLog.Printf("PlayMove: releasing lock")
	g.mu.Unlock()
	debugLog.Printf("PlayMove: lock released")

	// Notify callback (outside lock to prevent deadlock)
	debugLog.Printf("PlayMove: calling callback")
	if g.moveCallback != nil {
		g.moveCallback(x, y, playerColor, boardStateCopy)
	}
	debugLog.Printf("PlayMove: callback done")

	// Trigger engine response
	debugLog.Printf("PlayMove: starting engine goroutine")
	go g.triggerEngineMove()

	debugLog.Printf("PlayMove: returning")
	return nil
}

// Pass passes the current turn.
func (g *GTPEngine) Pass() error {
	g.mu.Lock()

	if g.gameOver {
		g.mu.Unlock()
		return fmt.Errorf("game is over")
	}

	if !g.myTurn {
		g.mu.Unlock()
		return fmt.Errorf("not your turn")
	}

	color := colorToGTP(g.playerColor)

	_, err := g.sendCommand(fmt.Sprintf("play %s pass", color))
	if err != nil {
		g.mu.Unlock()
		return fmt.Errorf("failed to pass: %w", err)
	}

	g.boardState.LastMove.X = -1
	g.boardState.LastMove.Y = -1
	g.boardState.MoveNumber++
	g.boardState.PlayerToMove = oppositeColor(g.playerColor)
	g.passCount++
	passCount := g.passCount

	g.myTurn = false
	playerColor := g.playerColor
	boardStateCopy := g.copyBoardState()
	g.mu.Unlock()

	// Notify callback (outside lock to prevent deadlock)
	if g.moveCallback != nil {
		g.moveCallback(-1, -1, playerColor, boardStateCopy)
	}

	// Check for double pass
	if passCount >= 2 {
		g.handleGameEnd()
		return nil
	}

	// Trigger engine response
	go g.triggerEngineMove()

	return nil
}

// triggerEngineMove asks the engine to generate and play a move.
func (g *GTPEngine) triggerEngineMove() {
	g.mu.Lock()

	if g.gameOver {
		g.mu.Unlock()
		return
	}

	engineColor := oppositeColor(g.playerColor)
	response, err := g.sendCommand(fmt.Sprintf("genmove %s", colorToGTP(engineColor)))
	if err != nil {
		g.mu.Unlock()
		return
	}

	response = strings.TrimSpace(strings.ToUpper(response))

	if response == "RESIGN" {
		g.gameOver = true
		g.boardState.Phase = "finished"
		winner := "Black"
		if g.playerColor == 2 {
			winner = "White"
		}
		g.boardState.Outcome = fmt.Sprintf("%s wins by resignation", winner)
		outcome := g.boardState.Outcome
		g.mu.Unlock()

		if g.endCallback != nil {
			g.endCallback(outcome)
		}
		return
	}

	if response == "PASS" {
		g.boardState.LastMove.X = -1
		g.boardState.LastMove.Y = -1
		g.boardState.MoveNumber++
		g.boardState.PlayerToMove = g.playerColor
		g.passCount++
		passCount := g.passCount

		g.myTurn = true
		boardStateCopy := g.copyBoardState()
		g.mu.Unlock()

		// Notify callback (outside lock)
		if g.moveCallback != nil {
			g.moveCallback(-1, -1, engineColor, boardStateCopy)
		}

		// Check for double pass
		if passCount >= 2 {
			g.handleGameEnd()
		}
		return
	}

	// Parse the move
	x, y, err := gtpToPos(response, g.config.BoardSize)
	if err != nil {
		g.mu.Unlock()
		return
	}

	// Update board state
	g.boardState.Board[y][x] = engineColor
	g.boardState.LastMove.X = x
	g.boardState.LastMove.Y = y
	g.boardState.MoveNumber++
	g.boardState.PlayerToMove = g.playerColor
	g.passCount = 0

	// Update captures
	g.updateBoardFromGnuGo()

	g.myTurn = true
	boardStateCopy := g.copyBoardState()
	g.mu.Unlock()

	// Notify callback (outside lock)
	if g.moveCallback != nil {
		g.moveCallback(x, y, engineColor, boardStateCopy)
	}
}

// updateBoardFromGnuGo refreshes the board state by parsing GnuGo's showboard output.
func (g *GTPEngine) updateBoardFromGnuGo() {
	// Use list_stones to get accurate positions
	blackStones, _ := g.sendCommand("list_stones black")
	whiteStones, _ := g.sendCommand("list_stones white")

	// Clear the board
	for y := 0; y < g.config.BoardSize; y++ {
		for x := 0; x < g.config.BoardSize; x++ {
			g.boardState.Board[y][x] = 0
		}
	}

	// Parse black stones
	for _, vertex := range strings.Fields(blackStones) {
		x, y, err := gtpToPos(vertex, g.config.BoardSize)
		if err == nil && x >= 0 && y >= 0 {
			g.boardState.Board[y][x] = 1
		}
	}

	// Parse white stones
	for _, vertex := range strings.Fields(whiteStones) {
		x, y, err := gtpToPos(vertex, g.config.BoardSize)
		if err == nil && x >= 0 && y >= 0 {
			g.boardState.Board[y][x] = 2
		}
	}
}

// handleGameEnd calculates the final score and ends the game.
func (g *GTPEngine) handleGameEnd() {
	g.mu.Lock()

	g.gameOver = true
	g.boardState.Phase = "finished"

	// Get final score from GnuGo
	score, err := g.sendCommand("final_score")
	if err != nil {
		g.boardState.Outcome = "Game ended"
	} else {
		g.boardState.Outcome = score
	}

	outcome := g.boardState.Outcome
	g.mu.Unlock()

	// Notify callback (outside lock)
	if g.endCallback != nil {
		g.endCallback(outcome)
	}
}

// IsMyTurn returns true if it's the human player's turn.
func (g *GTPEngine) IsMyTurn() bool {
	debugLog.Printf("IsMyTurn: trying to acquire lock")
	g.mu.Lock()
	debugLog.Printf("IsMyTurn: lock acquired")
	defer g.mu.Unlock()
	result := g.myTurn && !g.gameOver
	debugLog.Printf("IsMyTurn: returning %v", result)
	return result
}

// GetPlayerColor returns the human player's color (1=black, 2=white).
func (g *GTPEngine) GetPlayerColor() int {
	return g.playerColor
}

// OnMove registers a callback for when a move is played.
func (g *GTPEngine) OnMove(callback func(x, y, color int, boardState *types.BoardState)) {
	g.moveCallback = callback
}

// copyBoardState creates a deep copy of the current board state.
// Must be called while holding the lock.
func (g *GTPEngine) copyBoardState() *types.BoardState {
	size := g.config.BoardSize
	boardCopy := make([][]int, size)
	for i := range boardCopy {
		boardCopy[i] = make([]int, size)
		copy(boardCopy[i], g.boardState.Board[i])
	}
	return &types.BoardState{
		MoveNumber:   g.boardState.MoveNumber,
		PlayerToMove: g.boardState.PlayerToMove,
		Phase:        g.boardState.Phase,
		Board:        boardCopy,
		Outcome:      g.boardState.Outcome,
		LastMove:     g.boardState.LastMove,
	}
}

// OnGameEnd registers a callback for when the game ends.
func (g *GTPEngine) OnGameEnd(callback func(outcome string)) {
	g.endCallback = callback
}

// Close shuts down the GnuGo subprocess.
func (g *GTPEngine) Close() {
	if g.stdin != nil {
		g.sendCommand("quit")
		g.stdin.Close()
	}
	if g.cmd != nil && g.cmd.Process != nil {
		g.cmd.Wait()
	}
}
