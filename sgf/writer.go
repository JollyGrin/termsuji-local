// Package sgf implements SGF FF[4] writing and reading for Go game records.
package sgf

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// GameRecord tracks a game in progress and writes it as SGF.
type GameRecord struct {
	FilePath    string
	BoardSize   int
	Komi        float64
	PlayerBlack string
	PlayerWhite string
	Date        string
	Result      string
	moves       []string // ";B[pd]", ";W[dp]", ...
	setupBlack  []string // AB coords for mid-game toggle
	setupWhite  []string // AW coords
	file        *os.File
}

// NewGameRecord creates a new SGF file in dir and writes the initial header.
// playerColor is 1=black, 2=white (the human player's color).
func NewGameRecord(dir string, boardSize int, komi float64, playerColor, engineLevel int) (*GameRecord, error) {
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, fmt.Errorf("create history dir: %w", err)
	}

	now := time.Now()
	filename := fmt.Sprintf("%s_%dx%d.sgf", now.Format("2006-01-02_150405"), boardSize, boardSize)
	path := filepath.Join(dir, filename)

	f, err := os.Create(path)
	if err != nil {
		return nil, fmt.Errorf("create sgf file: %w", err)
	}

	human := "Player"
	engine := fmt.Sprintf("GnuGo Level %d", engineLevel)

	var pb, pw string
	if playerColor == 1 {
		pb, pw = human, engine
	} else {
		pb, pw = engine, human
	}

	rec := &GameRecord{
		FilePath:    path,
		BoardSize:   boardSize,
		Komi:        komi,
		PlayerBlack: pb,
		PlayerWhite: pw,
		Date:        now.Format("2006-01-02"),
		Result:      "?",
		file:        f,
	}

	if err := rec.flush(); err != nil {
		f.Close()
		return nil, err
	}

	return rec, nil
}

// sgfCoord converts 0-indexed board coordinates to SGF letter pair.
// (0,0) -> "aa", (3,4) -> "de", (18,18) -> "ss".
func sgfCoord(x, y int) string {
	return string(rune('a'+x)) + string(rune('a'+y))
}

// AddMove appends a move to the record. Pass is indicated by x==-1 && y==-1.
func (r *GameRecord) AddMove(x, y, color int) error {
	colorChar := "B"
	if color == 2 {
		colorChar = "W"
	}

	var node string
	if x == -1 && y == -1 {
		node = fmt.Sprintf(";%s[]", colorChar)
	} else {
		node = fmt.Sprintf(";%s[%s]", colorChar, sgfCoord(x, y))
	}

	r.moves = append(r.moves, node)
	return r.flush()
}

// AddSetupPosition scans a board and records AB[]/AW[] setup properties.
// board is indexed as board[y][x] where 1=black, 2=white.
func (r *GameRecord) AddSetupPosition(board [][]int) error {
	r.setupBlack = nil
	r.setupWhite = nil
	for y := range board {
		for x := range board[y] {
			switch board[y][x] {
			case 1:
				r.setupBlack = append(r.setupBlack, sgfCoord(x, y))
			case 2:
				r.setupWhite = append(r.setupWhite, sgfCoord(x, y))
			}
		}
	}
	return r.flush()
}

// UndoMoves removes the last n moves from the record.
func (r *GameRecord) UndoMoves(n int) error {
	if n > len(r.moves) {
		n = len(r.moves)
	}
	r.moves = r.moves[:len(r.moves)-n]
	return r.flush()
}

// SetResult parses a game outcome string and sets the SGF RE property.
// Accepts GnuGo output like "White wins by 5.5 points" or "Black wins by resign"
// as well as already-formatted SGF like "W+5.5", "B+R".
func (r *GameRecord) SetResult(outcome string) error {
	r.Result = parseResult(outcome)
	return r.flush()
}

// Close performs a final flush and closes the file handle.
func (r *GameRecord) Close() {
	if r.file == nil {
		return
	}
	r.flush()
	r.file.Close()
	r.file = nil
}

// flush rewrites the complete SGF file from scratch.
func (r *GameRecord) flush() error {
	if r.file == nil {
		return fmt.Errorf("file already closed")
	}

	var b strings.Builder

	// Root node
	b.WriteString("(;GM[1]FF[4]CA[UTF-8]")
	b.WriteString(fmt.Sprintf("AP[termsuji-local:1.0]"))
	b.WriteString(fmt.Sprintf("SZ[%d]", r.BoardSize))
	b.WriteString(fmt.Sprintf("KM[%.1f]", r.Komi))
	b.WriteString(fmt.Sprintf("PB[%s]", r.PlayerBlack))
	b.WriteString(fmt.Sprintf("PW[%s]", r.PlayerWhite))
	b.WriteString(fmt.Sprintf("DT[%s]", r.Date))
	b.WriteString(fmt.Sprintf("RE[%s]", r.Result))
	b.WriteString("\n")

	// Setup node (AB/AW for mid-game toggle-on)
	if len(r.setupBlack) > 0 || len(r.setupWhite) > 0 {
		b.WriteString(";")
		if len(r.setupBlack) > 0 {
			b.WriteString("AB")
			for _, c := range r.setupBlack {
				b.WriteString(fmt.Sprintf("[%s]", c))
			}
		}
		if len(r.setupWhite) > 0 {
			b.WriteString("AW")
			for _, c := range r.setupWhite {
				b.WriteString(fmt.Sprintf("[%s]", c))
			}
		}
		b.WriteString("\n")
	}

	// Move nodes
	for _, m := range r.moves {
		b.WriteString(m)
	}

	b.WriteString(")\n")

	// Rewrite file from start
	if _, err := r.file.Seek(0, 0); err != nil {
		return err
	}
	if err := r.file.Truncate(0); err != nil {
		return err
	}
	content := b.String()
	if _, err := r.file.WriteString(content); err != nil {
		return err
	}
	return r.file.Sync()
}

// parseResult converts various outcome formats to SGF RE[] value.
func parseResult(outcome string) string {
	o := strings.TrimSpace(outcome)

	// Already in SGF format
	if isValidSGFResult(o) {
		return o
	}

	low := strings.ToLower(o)

	// "White wins by 5.5 points" / "Black wins by 5.5 points"
	// "White wins by resign" / "Black wins by resignation"
	var winner string
	switch {
	case strings.HasPrefix(low, "white wins"):
		winner = "W"
	case strings.HasPrefix(low, "black wins"):
		winner = "B"
	default:
		return "?"
	}

	byIdx := strings.Index(low, " by ")
	if byIdx == -1 {
		return winner + "+?"
	}
	rest := strings.TrimSpace(low[byIdx+4:])

	if strings.HasPrefix(rest, "resign") {
		return winner + "+R"
	}
	if strings.HasPrefix(rest, "time") {
		return winner + "+T"
	}
	if strings.HasPrefix(rest, "forfeit") {
		return winner + "+F"
	}

	// Try to extract numeric score: "5.5 points" or "5.5"
	parts := strings.Fields(rest)
	if len(parts) > 0 {
		score := parts[0]
		// Validate it looks like a number
		valid := true
		dotSeen := false
		for _, ch := range score {
			if ch == '.' {
				if dotSeen {
					valid = false
					break
				}
				dotSeen = true
			} else if ch < '0' || ch > '9' {
				valid = false
				break
			}
		}
		if valid && len(score) > 0 {
			return winner + "+" + score
		}
	}

	return winner + "+?"
}

// isValidSGFResult checks if a string is already a valid SGF result.
func isValidSGFResult(s string) bool {
	if s == "?" || s == "Jigo" || s == "Void" || s == "0" {
		return true
	}
	if len(s) < 3 {
		return false
	}
	if (s[0] != 'B' && s[0] != 'W') || s[1] != '+' {
		return false
	}
	rest := s[2:]
	if rest == "R" || rest == "T" || rest == "F" || rest == "?" {
		return true
	}
	// Check for numeric score
	dotSeen := false
	for _, ch := range rest {
		if ch == '.' {
			if dotSeen {
				return false
			}
			dotSeen = true
		} else if ch < '0' || ch > '9' {
			return false
		}
	}
	return len(rest) > 0
}
