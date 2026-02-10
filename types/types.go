// Package types contains shared data structures for termsuji-local.
package types

import "encoding/json"

// BoardState represents the complete state of a Go board.
// Board is indexed as Board[y][x] where 0=empty, 1=black, 2=white.
type BoardState struct {
	MoveNumber   int     `json:"move_number"`
	PlayerToMove int     `json:"player_to_move"` // 1=black, 2=white
	Phase        string  `json:"phase"`          // "playing", "finished"
	Board        [][]int `json:"board"`
	Outcome      string  `json:"outcome"`
	LastMove     struct {
		X int `json:"x"`
		Y int `json:"y"`
	} `json:"last_move"`
}

// Finished returns true if the game is over.
func (b *BoardState) Finished() bool {
	return b.Phase == "finished"
}

// Height returns the board height.
func (b *BoardState) Height() int {
	return len(b.Board)
}

// Width returns the board width.
func (b *BoardState) Width() int {
	if b.Height() == 0 {
		return 0
	}
	return len(b.Board[0])
}

// BoardPos represents a position on the board.
type BoardPos struct {
	X int
	Y int
}

// UnmarshalJSON allows BoardPos to be unmarshaled from a JSON array [x, y].
func (p *BoardPos) UnmarshalJSON(data []byte) error {
	var v []float64
	err := json.Unmarshal(data, &v)
	if err != nil {
		return err
	}
	p.X = int(v[0])
	p.Y = int(v[1])
	return nil
}

// NewBoardState creates a new empty board of the given size.
func NewBoardState(size int) *BoardState {
	board := make([][]int, size)
	for i := range board {
		board[i] = make([]int, size)
	}
	return &BoardState{
		MoveNumber:   0,
		PlayerToMove: 1, // Black plays first
		Phase:        "playing",
		Board:        board,
		LastMove: struct {
			X int `json:"x"`
			Y int `json:"y"`
		}{X: -1, Y: -1},
	}
}
