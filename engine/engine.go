// Package engine defines the interface for game engines.
package engine

import "termsuji-local/types"

// GameEngine defines the interface for playing Go against an engine.
type GameEngine interface {
	// Connect starts the engine and initializes the game.
	Connect() error

	// GetBoardState returns the current board state.
	GetBoardState() *types.BoardState

	// PlayMove plays a move at the given coordinates.
	// Returns an error if the move is illegal.
	PlayMove(x, y int) error

	// Pass passes the current turn.
	Pass() error

	// IsMyTurn returns true if it's the human player's turn.
	IsMyTurn() bool

	// GetPlayerColor returns the human player's color (1=black, 2=white).
	GetPlayerColor() int

	// OnMove registers a callback for when a move is played (by either player).
	// x, y are -1, -1 for a pass. boardState is passed directly to avoid lock contention.
	OnMove(func(x, y, color int, boardState *types.BoardState))

	// Undo undoes the last move (one ply). Call twice to undo a player+engine move pair.
	Undo() error

	// OnGameEnd registers a callback for when the game ends.
	OnGameEnd(func(outcome string))

	// Close shuts down the engine.
	Close()
}

// GameConfig holds configuration for starting a new game.
type GameConfig struct {
	BoardSize     int     // 9, 13, or 19
	Komi          float64 // Typically 6.5 or 7.5
	PlayerColor   int     // 1=black, 2=white
	EngineLevel   int     // GnuGo level 1-10
	EnginePath    string  // Path to GnuGo binary
	LoadSGFPath   string  // Path to SGF file for GnuGo's loadsgf command
	LoadMoveCount int     // Number of moves in the loaded SGF (for turn determination)
}

// DefaultConfig returns a reasonable default configuration.
func DefaultConfig() GameConfig {
	return GameConfig{
		BoardSize:   19,
		Komi:        6.5,
		PlayerColor: 1, // Human plays black
		EngineLevel: 5,
		EnginePath:  "gnugo",
	}
}
