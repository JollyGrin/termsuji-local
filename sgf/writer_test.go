package sgf

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestSgfCoord(t *testing.T) {
	tests := []struct {
		x, y int
		want string
	}{
		{0, 0, "aa"},
		{3, 4, "de"},
		{18, 18, "ss"},
		{15, 3, "pd"},  // common star point
		{3, 15, "dp"},  // common star point
	}
	for _, tt := range tests {
		got := sgfCoord(tt.x, tt.y)
		if got != tt.want {
			t.Errorf("sgfCoord(%d, %d) = %q, want %q", tt.x, tt.y, got, tt.want)
		}
	}
}

func TestParseResult(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		// Already SGF format
		{"W+5.5", "W+5.5"},
		{"B+R", "B+R"},
		{"B+3.5", "B+3.5"},
		{"?", "?"},
		{"Jigo", "Jigo"},

		// GnuGo output
		{"White wins by 5.5 points", "W+5.5"},
		{"Black wins by 3.5 points", "B+3.5"},
		{"White wins by resign", "W+R"},
		{"Black wins by resignation", "B+R"},
		{"White wins by time", "W+T"},
		{"Black wins by forfeit", "B+F"},

		// Edge cases
		{"White wins by 0.5 points", "W+0.5"},
		{"Black wins", "B+?"},
		{"something else", "?"},
		{"", "?"},
	}
	for _, tt := range tests {
		got := parseResult(tt.input)
		if got != tt.want {
			t.Errorf("parseResult(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

func TestNewGameRecord(t *testing.T) {
	dir := t.TempDir()
	rec, err := NewGameRecord(dir, 19, 6.5, 1, 5)
	if err != nil {
		t.Fatalf("NewGameRecord: %v", err)
	}
	defer rec.Close()

	// File should exist
	if _, err := os.Stat(rec.FilePath); os.IsNotExist(err) {
		t.Fatal("SGF file not created")
	}

	// Read and verify content
	content, err := os.ReadFile(rec.FilePath)
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}
	s := string(content)

	// Check required properties
	for _, prop := range []string{"GM[1]", "FF[4]", "SZ[19]", "KM[6.5]", "PB[Player]", "PW[GnuGo Level 5]"} {
		if !strings.Contains(s, prop) {
			t.Errorf("SGF missing property %s in:\n%s", prop, s)
		}
	}

	// Verify it's valid SGF structure
	if !strings.HasPrefix(s, "(;") {
		t.Error("SGF should start with '(;'")
	}
	if !strings.Contains(s, ")") {
		t.Error("SGF should contain closing ')'")
	}
}

func TestNewGameRecordWhitePlayer(t *testing.T) {
	dir := t.TempDir()
	rec, err := NewGameRecord(dir, 9, 7.5, 2, 3)
	if err != nil {
		t.Fatalf("NewGameRecord: %v", err)
	}
	defer rec.Close()

	content, _ := os.ReadFile(rec.FilePath)
	s := string(content)

	if !strings.Contains(s, "PB[GnuGo Level 3]") {
		t.Error("When human plays white, black should be engine")
	}
	if !strings.Contains(s, "PW[Player]") {
		t.Error("When human plays white, white should be Player")
	}
	if !strings.Contains(s, "SZ[9]") {
		t.Error("Board size should be 9")
	}
	if !strings.Contains(s, "KM[7.5]") {
		t.Error("Komi should be 7.5")
	}
}

func TestAddMove(t *testing.T) {
	dir := t.TempDir()
	rec, err := NewGameRecord(dir, 19, 6.5, 1, 5)
	if err != nil {
		t.Fatalf("NewGameRecord: %v", err)
	}
	defer rec.Close()

	// Play some moves
	rec.AddMove(15, 3, 1)  // B[pd]
	rec.AddMove(3, 15, 2)  // W[dp]
	rec.AddMove(15, 15, 1) // B[pp]

	content, _ := os.ReadFile(rec.FilePath)
	s := string(content)

	for _, move := range []string{";B[pd]", ";W[dp]", ";B[pp]"} {
		if !strings.Contains(s, move) {
			t.Errorf("SGF missing move %s in:\n%s", move, s)
		}
	}
}

func TestAddMovePass(t *testing.T) {
	dir := t.TempDir()
	rec, err := NewGameRecord(dir, 9, 6.5, 1, 5)
	if err != nil {
		t.Fatalf("NewGameRecord: %v", err)
	}
	defer rec.Close()

	rec.AddMove(4, 4, 1) // B[ee]
	rec.AddMove(-1, -1, 2) // W[] pass
	rec.AddMove(-1, -1, 1) // B[] pass

	content, _ := os.ReadFile(rec.FilePath)
	s := string(content)

	if !strings.Contains(s, ";B[ee]") {
		t.Error("Missing first move")
	}
	if !strings.Contains(s, ";W[]") {
		t.Error("Missing white pass")
	}
	// Count passes - should have both B[] and W[]
	if strings.Count(s, ";W[]") != 1 {
		t.Error("Should have exactly one white pass")
	}
	if strings.Count(s, ";B[]") != 1 {
		t.Error("Should have exactly one black pass")
	}
}

func TestSetResult(t *testing.T) {
	dir := t.TempDir()
	rec, err := NewGameRecord(dir, 19, 6.5, 1, 5)
	if err != nil {
		t.Fatalf("NewGameRecord: %v", err)
	}
	defer rec.Close()

	rec.AddMove(15, 3, 1)
	rec.SetResult("White wins by 5.5 points")

	content, _ := os.ReadFile(rec.FilePath)
	s := string(content)

	if !strings.Contains(s, "RE[W+5.5]") {
		t.Errorf("Expected RE[W+5.5] in:\n%s", s)
	}
}

func TestAddSetupPosition(t *testing.T) {
	dir := t.TempDir()
	rec, err := NewGameRecord(dir, 9, 6.5, 1, 5)
	if err != nil {
		t.Fatalf("NewGameRecord: %v", err)
	}
	defer rec.Close()

	// Create a board with some stones
	board := make([][]int, 9)
	for i := range board {
		board[i] = make([]int, 9)
	}
	board[2][3] = 1 // black at (3,2) = "dc"
	board[4][5] = 2 // white at (5,4) = "fe"

	rec.AddSetupPosition(board)

	content, _ := os.ReadFile(rec.FilePath)
	s := string(content)

	if !strings.Contains(s, "AB[dc]") {
		t.Errorf("Missing AB[dc] setup in:\n%s", s)
	}
	if !strings.Contains(s, "AW[fe]") {
		t.Errorf("Missing AW[fe] setup in:\n%s", s)
	}
}

func TestFullGameRoundtrip(t *testing.T) {
	dir := t.TempDir()
	rec, err := NewGameRecord(dir, 9, 6.5, 1, 5)
	if err != nil {
		t.Fatalf("NewGameRecord: %v", err)
	}

	// Play a short game
	moves := [][3]int{
		{4, 4, 1},   // B center
		{2, 2, 2},   // W
		{6, 6, 1},   // B
		{2, 6, 2},   // W
		{6, 2, 1},   // B
		{-1, -1, 2}, // W pass
		{-1, -1, 1}, // B pass
	}

	for _, m := range moves {
		if err := rec.AddMove(m[0], m[1], m[2]); err != nil {
			t.Fatalf("AddMove(%d,%d,%d): %v", m[0], m[1], m[2], err)
		}
	}

	rec.SetResult("Black wins by 12.5 points")
	rec.Close()

	// Read back and verify
	content, err := os.ReadFile(rec.FilePath)
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}
	s := string(content)

	// Verify structure
	if !strings.HasPrefix(s, "(;GM[1]") {
		t.Error("Should start with SGF header")
	}
	if !strings.HasSuffix(strings.TrimSpace(s), ")") {
		t.Error("Should end with closing paren")
	}

	// Verify all moves present
	expected := []string{";B[ee]", ";W[cc]", ";B[gg]", ";W[cg]", ";B[gc]", ";W[]", ";B[]"}
	for _, m := range expected {
		if !strings.Contains(s, m) {
			t.Errorf("Missing move %s in:\n%s", m, s)
		}
	}

	// Verify result
	if !strings.Contains(s, "RE[B+12.5]") {
		t.Errorf("Missing result RE[B+12.5] in:\n%s", s)
	}
}

func TestFilenameFormat(t *testing.T) {
	dir := t.TempDir()
	rec, err := NewGameRecord(dir, 13, 6.5, 1, 5)
	if err != nil {
		t.Fatalf("NewGameRecord: %v", err)
	}
	defer rec.Close()

	base := filepath.Base(rec.FilePath)
	if !strings.HasSuffix(base, "_13x13.sgf") {
		t.Errorf("Filename should end with _13x13.sgf, got %s", base)
	}
	if !strings.HasPrefix(base, "20") {
		t.Errorf("Filename should start with year, got %s", base)
	}
}

func TestCloseIdempotent(t *testing.T) {
	dir := t.TempDir()
	rec, err := NewGameRecord(dir, 9, 6.5, 1, 5)
	if err != nil {
		t.Fatalf("NewGameRecord: %v", err)
	}

	rec.Close()
	rec.Close() // Should not panic
}

func TestCrashSafety(t *testing.T) {
	dir := t.TempDir()
	rec, err := NewGameRecord(dir, 9, 6.5, 1, 5)
	if err != nil {
		t.Fatalf("NewGameRecord: %v", err)
	}

	// Add moves without closing
	rec.AddMove(4, 4, 1)
	rec.AddMove(2, 2, 2)

	// Simulate crash: read file directly (it should be valid SGF after each flush)
	content, err := os.ReadFile(rec.FilePath)
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}
	s := string(content)

	if !strings.HasPrefix(s, "(;") {
		t.Error("File should be valid SGF even without Close()")
	}
	if !strings.Contains(s, ")") {
		t.Error("File should have closing paren even without Close()")
	}
	if !strings.Contains(s, ";B[ee]") {
		t.Error("File should contain moves even without Close()")
	}

	rec.Close()
}
