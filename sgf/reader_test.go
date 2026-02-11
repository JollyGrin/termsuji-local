package sgf

import (
	"os"
	"path/filepath"
	"testing"
)

const testSGF = `(;GM[1]FF[4]CA[UTF-8]AP[termsuji-local:1.0]SZ[9]KM[6.5]PB[Player]PW[GnuGo Level 5]DT[2026-01-15]RE[B+3.5]
;B[ee];W[cc];B[gg];W[cg];B[gc])`

func writeTempSGF(t *testing.T, dir, name, content string) string {
	t.Helper()
	path := filepath.Join(dir, name)
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatalf("write temp sgf: %v", err)
	}
	return path
}

func TestParseHeader(t *testing.T) {
	dir := t.TempDir()
	path := writeTempSGF(t, dir, "test.sgf", testSGF)

	info, err := ParseHeader(path)
	if err != nil {
		t.Fatalf("ParseHeader: %v", err)
	}

	if info.BoardSize != 9 {
		t.Errorf("BoardSize = %d, want 9", info.BoardSize)
	}
	if info.Komi != 6.5 {
		t.Errorf("Komi = %f, want 6.5", info.Komi)
	}
	if info.PlayerBlack != "Player" {
		t.Errorf("PlayerBlack = %q, want %q", info.PlayerBlack, "Player")
	}
	if info.PlayerWhite != "GnuGo Level 5" {
		t.Errorf("PlayerWhite = %q, want %q", info.PlayerWhite, "GnuGo Level 5")
	}
	if info.Date != "2026-01-15" {
		t.Errorf("Date = %q, want %q", info.Date, "2026-01-15")
	}
	if info.Result != "B+3.5" {
		t.Errorf("Result = %q, want %q", info.Result, "B+3.5")
	}
	if info.MoveCount != 5 {
		t.Errorf("MoveCount = %d, want 5", info.MoveCount)
	}
}

func TestParseHeaderMissingFile(t *testing.T) {
	_, err := ParseHeader("/nonexistent/file.sgf")
	if err == nil {
		t.Error("Expected error for missing file")
	}
}

func TestReplayToEnd(t *testing.T) {
	dir := t.TempDir()
	path := writeTempSGF(t, dir, "test.sgf", testSGF)

	board, moveCount, err := ReplayToEnd(path)
	if err != nil {
		t.Fatalf("ReplayToEnd: %v", err)
	}

	if moveCount != 5 {
		t.Errorf("moveCount = %d, want 5", moveCount)
	}

	// Verify stones are placed correctly
	// B[ee] = (4,4), W[cc] = (2,2), B[gg] = (6,6), W[cg] = (2,6), B[gc] = (6,2)
	checks := []struct {
		x, y, color int
	}{
		{4, 4, 1}, // B[ee]
		{2, 2, 2}, // W[cc]
		{6, 6, 1}, // B[gg]
		{2, 6, 2}, // W[cg]
		{6, 2, 1}, // B[gc]
	}
	for _, c := range checks {
		if board[c.y][c.x] != c.color {
			t.Errorf("board[%d][%d] = %d, want %d", c.y, c.x, board[c.y][c.x], c.color)
		}
	}
}

func TestReplayWithCaptures(t *testing.T) {
	// Set up a capture scenario on 9x9:
	// Black surrounds a white stone at (1,0) on the top edge
	// W at (1,0), B at (0,0), (2,0), (1,1) -> white captured
	sgf := `(;GM[1]FF[4]SZ[9]KM[6.5]PB[B]PW[W]DT[2026-01-01]RE[?]
;B[aa];W[ba];B[ca];W[ee];B[bb])`
	// Move sequence:
	// B[aa] = (0,0) black
	// W[ba] = (1,0) white
	// B[ca] = (2,0) black
	// W[ee] = (4,4) white (random move)
	// B[bb] = (1,1) black -- this captures white at (1,0)

	dir := t.TempDir()
	path := writeTempSGF(t, dir, "capture.sgf", sgf)

	board, moveCount, err := ReplayToEnd(path)
	if err != nil {
		t.Fatalf("ReplayToEnd: %v", err)
	}

	if moveCount != 5 {
		t.Errorf("moveCount = %d, want 5", moveCount)
	}

	// White at (1,0) should be captured (removed)
	if board[0][1] != 0 {
		t.Errorf("board[0][1] = %d, want 0 (captured)", board[0][1])
	}

	// Surrounding blacks should still be there
	if board[0][0] != 1 {
		t.Errorf("board[0][0] = %d, want 1 (black)", board[0][0])
	}
	if board[0][2] != 1 {
		t.Errorf("board[0][2] = %d, want 1 (black)", board[0][2])
	}
	if board[1][1] != 1 {
		t.Errorf("board[1][1] = %d, want 1 (black)", board[1][1])
	}

	// White's other stone should still be there
	if board[4][4] != 2 {
		t.Errorf("board[4][4] = %d, want 2 (white)", board[4][4])
	}
}

func TestReplayWithSetupPosition(t *testing.T) {
	// SGF with AB/AW setup positions followed by moves
	sgf := `(;GM[1]FF[4]SZ[9]KM[6.5]PB[B]PW[W]DT[2026-01-01]RE[?]
;AB[dd][ff]AW[ee]
;B[cc];W[gg])`

	dir := t.TempDir()
	path := writeTempSGF(t, dir, "setup.sgf", sgf)

	board, moveCount, err := ReplayToEnd(path)
	if err != nil {
		t.Fatalf("ReplayToEnd: %v", err)
	}

	if moveCount != 2 {
		t.Errorf("moveCount = %d, want 2", moveCount)
	}

	// Setup stones
	if board[3][3] != 1 {
		t.Errorf("board[3][3] = %d, want 1 (AB[dd])", board[3][3])
	}
	if board[5][5] != 1 {
		t.Errorf("board[5][5] = %d, want 1 (AB[ff])", board[5][5])
	}
	if board[4][4] != 2 {
		t.Errorf("board[4][4] = %d, want 2 (AW[ee])", board[4][4])
	}

	// Played moves
	if board[2][2] != 1 {
		t.Errorf("board[2][2] = %d, want 1 (B[cc])", board[2][2])
	}
	if board[6][6] != 2 {
		t.Errorf("board[6][6] = %d, want 2 (W[gg])", board[6][6])
	}
}

func TestReplayWithPasses(t *testing.T) {
	sgf := `(;GM[1]FF[4]SZ[9]KM[6.5]PB[B]PW[W]DT[2026-01-01]RE[B+5.0]
;B[ee];W[];B[])`

	dir := t.TempDir()
	path := writeTempSGF(t, dir, "pass.sgf", sgf)

	board, moveCount, err := ReplayToEnd(path)
	if err != nil {
		t.Fatalf("ReplayToEnd: %v", err)
	}

	if moveCount != 3 {
		t.Errorf("moveCount = %d, want 3", moveCount)
	}

	// Only one stone placed
	if board[4][4] != 1 {
		t.Errorf("board[4][4] = %d, want 1", board[4][4])
	}
}

func TestReplayGroupCapture(t *testing.T) {
	// Test capturing a group of 2 white stones
	// White at (0,0) and (1,0), surrounded by black
	sgf := `(;GM[1]FF[4]SZ[9]KM[6.5]PB[B]PW[W]DT[2026-01-01]RE[?]
;B[ca];W[aa];B[ab];W[ba];B[bb];W[ee];B[cb])`
	// B[ca]=(2,0) W[aa]=(0,0) B[ab]=(0,1) W[ba]=(1,0) B[bb]=(1,1) W[ee]=(4,4) B[cb]=(2,1)
	// After B[cb]: white group at (0,0)+(1,0) has no liberties -> captured

	// Wait - let me verify: white (0,0) neighbors: up=OOB, left=OOB, right=(1,0)=white, down=(0,1)=black
	// white (1,0) neighbors: up=OOB, left=(0,0)=white, right=(2,0)=black, down=(1,1)=black
	// After B[cb]=(2,1): this doesn't affect the top group directly...
	// Let me reconsider. The group (0,0)+(1,0):
	// (0,0): up=OOB, left=OOB, down=(0,1)=B, right=(1,0)=W
	// (1,0): up=OOB, left=(0,0)=W, down=(1,1)=B, right=(2,0)=B
	// All liberties blocked! So yes, after B[bb], the group is captured.

	// Actually after B[bb] at (1,1), let's check:
	// white(0,0) adj: up=OOB, left=OOB, right=(1,0)=W, down=(0,1)=B -> no liberty from (0,0) directly
	// white(1,0) adj: up=OOB, left=(0,0)=W, right=(2,0)=B, down=(1,1)=B -> no liberty
	// So the capture happens at B[bb], not B[cb].

	// Let me rewrite with a cleaner scenario
	sgf = `(;GM[1]FF[4]SZ[9]KM[6.5]PB[B]PW[W]DT[2026-01-01]RE[?]
;B[ca];W[aa];B[ab];W[ba];B[bb];W[ee])`
	// After B[bb]=(1,1): white group at (0,0)+(1,0) should be captured
	// because B[bb] is the last played move, removeCaptures checks neighbors of (1,1)
	// neighbor (1,0) is white -> check liberties -> none -> remove group

	dir := t.TempDir()
	path := writeTempSGF(t, dir, "group.sgf", sgf)

	board, _, err := ReplayToEnd(path)
	if err != nil {
		t.Fatalf("ReplayToEnd: %v", err)
	}

	// White group should be captured
	if board[0][0] != 0 {
		t.Errorf("board[0][0] = %d, want 0 (captured)", board[0][0])
	}
	if board[0][1] != 0 {
		t.Errorf("board[0][1] = %d, want 0 (captured)", board[0][1])
	}

	// Black stones remain
	if board[0][2] != 1 {
		t.Errorf("board[0][2] = %d, want 1 (black)", board[0][2])
	}
	if board[1][0] != 1 {
		t.Errorf("board[1][0] = %d, want 1 (black)", board[1][0])
	}
	if board[1][1] != 1 {
		t.Errorf("board[1][1] = %d, want 1 (black)", board[1][1])
	}
}

func TestListGames(t *testing.T) {
	dir := t.TempDir()

	// Create a few SGF files with timestamp-like names
	writeTempSGF(t, dir, "2026-01-10_100000_9x9.sgf", `(;GM[1]FF[4]SZ[9]KM[6.5]PB[P]PW[E]DT[2026-01-10]RE[?])`)
	writeTempSGF(t, dir, "2026-01-11_100000_19x19.sgf", `(;GM[1]FF[4]SZ[19]KM[6.5]PB[P]PW[E]DT[2026-01-11]RE[B+5.0])`)
	writeTempSGF(t, dir, "2026-01-12_100000_13x13.sgf", `(;GM[1]FF[4]SZ[13]KM[7.5]PB[P]PW[E]DT[2026-01-12]RE[W+R])`)

	// Also create a non-sgf file to ensure it's skipped
	os.WriteFile(filepath.Join(dir, "notes.txt"), []byte("not an sgf"), 0644)

	games, err := ListGames(dir)
	if err != nil {
		t.Fatalf("ListGames: %v", err)
	}

	if len(games) != 3 {
		t.Fatalf("len(games) = %d, want 3", len(games))
	}

	// Should be newest-first
	if games[0].Date != "2026-01-12" {
		t.Errorf("games[0].Date = %q, want 2026-01-12", games[0].Date)
	}
	if games[1].Date != "2026-01-11" {
		t.Errorf("games[1].Date = %q, want 2026-01-11", games[1].Date)
	}
	if games[2].Date != "2026-01-10" {
		t.Errorf("games[2].Date = %q, want 2026-01-10", games[2].Date)
	}

	if games[0].BoardSize != 13 {
		t.Errorf("games[0].BoardSize = %d, want 13", games[0].BoardSize)
	}
}

func TestListGamesEmptyDir(t *testing.T) {
	dir := t.TempDir()
	games, err := ListGames(dir)
	if err != nil {
		t.Fatalf("ListGames: %v", err)
	}
	if len(games) != 0 {
		t.Errorf("len(games) = %d, want 0", len(games))
	}
}

func TestListGamesNonexistentDir(t *testing.T) {
	games, err := ListGames("/nonexistent/dir")
	if err != nil {
		t.Fatalf("ListGames should not error for nonexistent dir: %v", err)
	}
	if games != nil {
		t.Errorf("games should be nil for nonexistent dir")
	}
}

func TestWriterThenReader(t *testing.T) {
	dir := t.TempDir()

	// Write a game using the writer
	rec, err := NewGameRecord(dir, 9, 6.5, 1, 5)
	if err != nil {
		t.Fatalf("NewGameRecord: %v", err)
	}

	rec.AddMove(4, 4, 1) // B[ee]
	rec.AddMove(2, 2, 2) // W[cc]
	rec.AddMove(6, 6, 1) // B[gg]
	rec.SetResult("Black wins by 12.5 points")
	rec.Close()

	// Read it back with the reader
	info, err := ParseHeader(rec.FilePath)
	if err != nil {
		t.Fatalf("ParseHeader: %v", err)
	}

	if info.BoardSize != 9 {
		t.Errorf("BoardSize = %d, want 9", info.BoardSize)
	}
	if info.Result != "B+12.5" {
		t.Errorf("Result = %q, want B+12.5", info.Result)
	}
	if info.MoveCount != 3 {
		t.Errorf("MoveCount = %d, want 3", info.MoveCount)
	}

	// Replay and verify positions
	board, moveCount, err := ReplayToEnd(rec.FilePath)
	if err != nil {
		t.Fatalf("ReplayToEnd: %v", err)
	}

	if moveCount != 3 {
		t.Errorf("moveCount = %d, want 3", moveCount)
	}
	if board[4][4] != 1 {
		t.Errorf("board[4][4] = %d, want 1", board[4][4])
	}
	if board[2][2] != 2 {
		t.Errorf("board[2][2] = %d, want 2", board[2][2])
	}
	if board[6][6] != 1 {
		t.Errorf("board[6][6] = %d, want 1", board[6][6])
	}
}
