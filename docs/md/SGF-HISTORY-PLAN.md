# SGF Game History -- Implementation Plan

> Based on research in [SGF-IMPLEMENTATION.md](./SGF-IMPLEMENTATION.md), which covers the SGF FF[4] format specification, coordinate systems, and initial architectural analysis of integration points.

## Context

termsuji-local currently has no game persistence. Every game is lost when you quit. This plan implements automatic SGF recording, a toggle to control it, and a history browser screen -- built incrementally so each phase is independently useful.

---

## Phase 1: SGF Writer Module

Self-contained package, no UI changes. Foundation for everything else.

**Create `sgf/writer.go`**

```go
type GameRecord struct {
    FilePath    string
    BoardSize   int
    Komi        float64
    PlayerBlack string  // "Player" or "GnuGo Level N"
    PlayerWhite string
    Date        string
    Result      string
    moves       []string   // ";B[pd]", ";W[dp]", ...
    setupBlack  []string   // AB coords for mid-game toggle
    setupWhite  []string   // AW coords
    file        *os.File
}
```

Functions:
- `NewGameRecord(dir, boardSize, komi, playerColor, engineLevel) (*GameRecord, error)` -- creates history dir via `os.MkdirAll`, opens file named `2026-02-11_150405_19x19.sgf`, writes initial SGF
- `AddMove(x, y, color int) error` -- converts coords via `sgfCoord(x,y)` = `string('a'+x) + string('a'+y)`, pass when x==-1/y==-1 writes `B[]`/`W[]`, appends to moves slice, calls `flush()`
- `AddSetupPosition(board [][]int)` -- scans board for stones, builds AB[]/AW[] setup node (for mid-game toggle-on)
- `SetResult(outcome string) error` -- parses engine outcome ("W+5.5", "Black wins by resignation") to SGF RE format ("W+5.5", "B+R"), calls `flush()`
- `Close()` -- final flush, close file handle

`flush()` rewrites the complete file on every call (header + setup + moves + closing `)`). Files are <5KB even for 300-move games, so this is fast, always-valid SGF, and crash-safe after each `file.Sync()`.

**Create `sgf/writer_test.go`** -- test coord conversion, move recording, pass handling, setup positions, result parsing, full game roundtrip.

---

## Phase 2: Wire Recording into Game Loop + Config Toggle

**Modify `config/config.go`** -- add to Config struct:
```go
EnableRecording bool `json:"enable_recording"`
```
Add `HistoryDir()` helper:
```go
func HistoryDir() string {
    return filepath.Join(xdg.ConfigHome, "termsuji-local", "history")
}
```

**Modify `config/defaultconfigs.go`** -- set `EnableRecording: true` in DefaultConfig.

**Modify `ui/goboard.go`**:
- Add field: `recorder *sgf.GameRecord`
- Add field: `gameConfig engine.GameConfig` (stored for mid-game toggle-on)
- In `ConnectEngine()`, after existing `OnMove` callback body, add:
  ```go
  if g.recorder != nil {
      g.recorder.AddMove(x, y, color)
  }
  ```
- Same pattern in `OnGameEnd` callback -- call `g.recorder.SetResult(outcome)`
- In `Close()` -- call `g.recorder.Close()` if non-nil, then nil it out
- Add setter: `SetRecorder(rec *sgf.GameRecord)` and `SetGameConfig(gc engine.GameConfig)`

**Modify `main.go` `startGame()`** -- after `gameBoard.ConnectEngine(eng)` succeeds:
```go
gameBoard.SetGameConfig(gameCfg)
if cfg.EnableRecording {
    rec, err := sgf.NewGameRecord(config.HistoryDir(), gameCfg.BoardSize, gameCfg.Komi, gameCfg.PlayerColor, gameCfg.EngineLevel)
    if err == nil {
        gameBoard.SetRecorder(rec)
    }
}
```

**Modify `refreshHint()` in `ui/goboard.go`** -- when not in focus mode and recorder is non-nil, prepend `[red]REC[-] ` to the status text. When recording is off, show nothing extra.

After this phase: games auto-save to `~/.config/termsuji-local/history/` as SGF files with a red REC indicator in the status bar.

---

## Phase 3: Mid-Game Recording Toggle

**Modify `main.go`** -- add `'r'` keybinding in the game input handler:
```go
case 'r':
    gameBoard.ToggleRecording(cfg)
```

**Modify `ui/goboard.go`** -- add `ToggleRecording(cfg *config.Config)`:
- If `g.recorder != nil`: close recorder, set nil (stop recording)
- If `g.recorder == nil`: create new GameRecord using stored `g.gameConfig`. If `g.BoardState.MoveNumber > 0`, call `rec.AddSetupPosition(g.BoardState.Board)` to snapshot current position via AB[]/AW[] properties. Set recorder.
- Call `refreshHint()` to update indicator

**Modify `refreshHint()`** -- add `r` to controls hint text:
```
[dimgray]r[-] rec
```

After this phase: press `r` to toggle recording on/off. Toggling on mid-game captures the current board position via standard SGF setup properties, then records all future moves normally.

---

## Phase 4: SGF Reader Module

Lightweight parser for the history browser. No external dependencies.

**Create `sgf/reader.go`**

```go
type GameInfo struct {
    FilePath    string
    FileName    string
    BoardSize   int
    Komi        float64
    PlayerBlack string
    PlayerWhite string
    Date        string
    Result      string
    MoveCount   int
}

func ParseHeader(filePath string) (*GameInfo, error)     // fast: reads root node only
func ReplayToEnd(filePath string) ([][]int, int, error)  // returns final board position
```

`ParseHeader` reads the file, extracts `KEY[value]` pairs from the root node (SZ, KM, PB, PW, DT, RE), counts move nodes.

`ReplayToEnd` parses AB/AW setup, then applies each B[]/W[] move with basic capture logic (~40 lines: place stone, check adjacent opponent groups for zero liberties, remove captured groups). Returns the final `[][]int` board.

**Create `sgf/reader_test.go`** -- test header parsing, position replay with captures, setup position handling.

---

## Phase 5: History Browser Screen

New page modeled on `ui/colorconfig.go` (tview.List + preview Box pattern).

**Create `ui/historybrowser.go`**

Layout:
```
List (left, ~35 chars)         | Preview (right, flexible)
                               |
> 2026-02-11  19x19  B+5.5    |   . . . . . . . . .
  2026-02-10  19x19  W+R      |   . . . ● . . . . .
  2026-02-09   9x9   B+12     |   . . ○ . ● . . . .
  ...                          |   . . . . . . . . .
                               |   19x19 | 142 moves
                               |   B: Player | W: GnuGo L5
                               |   Result: B+5.5
-------------------------------------------------------
  d delete  s save-as  q back
```

Key type:
```go
type HistoryBrowserUI struct {
    flex     *tview.Flex
    gameList *tview.List
    preview  *tview.Box
    hint     *tview.TextView
    games    []sgf.GameInfo
    boards   map[int][][]int  // cached final positions
    selected int
}
```

Functions:
- `NewHistoryBrowser(onDone func()) *HistoryBrowserUI` -- creates layout, calls `loadGames()`
- `loadGames()` -- scans `config.HistoryDir()` for `*.sgf`, sorts newest-first by filename (timestamp-based names sort naturally), calls `sgf.ParseHeader()` for each, populates tview.List
- Preview Box has a custom `DrawFunc` that renders a mini board (1 char per cell: `.` empty, `●` black, `○` white) using the final position from `sgf.ReplayToEnd()` (lazy-loaded and cached). Below the board, show metadata.
- `'d'` key: delete selected SGF file (`os.Remove`), refresh list
- `'s'` key: open a tview.InputField modal for save-as path, copy file to destination

**Modify `ui/gamesetup.go`**:
- Add `onHistory func()` callback parameter to `NewGameSetup()`
- Add `historyButton *MenuButton` -- "HISTORY" button between COLORS and QUIT
- Add to focusables chain

**Modify `main.go`**:
- Create `HistoryBrowserUI`, register as `rootPage.AddPage("history", ...)`
- Pass `onHistory` callback to `NewGameSetup` that switches to history page
- Wire `'q'`/Esc on history page to return to setup

---

## Files Summary

| Phase | File | Action |
|-------|------|--------|
| 1 | `sgf/writer.go` | Create |
| 1 | `sgf/writer_test.go` | Create |
| 2 | `config/config.go` | Modify -- add `EnableRecording`, `HistoryDir()` |
| 2 | `config/defaultconfigs.go` | Modify -- default `EnableRecording: true` |
| 2 | `ui/goboard.go` | Modify -- add `recorder` field, wire callbacks, REC indicator |
| 2 | `main.go` | Modify -- create recorder in `startGame()` |
| 3 | `ui/goboard.go` | Modify -- add `ToggleRecording()`, `'r'` in controls hint |
| 3 | `main.go` | Modify -- add `'r'` keybinding |
| 4 | `sgf/reader.go` | Create |
| 4 | `sgf/reader_test.go` | Create |
| 5 | `ui/historybrowser.go` | Create |
| 5 | `ui/gamesetup.go` | Modify -- add HISTORY button |
| 5 | `main.go` | Modify -- register history page |

## Verification

After each phase:
- **Phase 1**: `go test ./sgf/...` -- unit tests pass
- **Phase 2**: Play a game, check `~/.config/termsuji-local/history/` for `.sgf` file, open in any SGF viewer (Sabaki, etc.), verify moves match. Confirm REC indicator shows in status bar.
- **Phase 3**: Start game with recording off (`enable_recording: false` in config), press `r` mid-game, verify SGF created with AB[]/AW[] setup + subsequent moves. Press `r` again to stop, verify file is properly closed.
- **Phase 4**: `go test ./sgf/...` -- reader tests pass including capture logic
- **Phase 5**: Launch app, click HISTORY on setup screen, browse games, verify thumbnails render, delete a game, save-as to custom path.

## Key Design Decisions

- **Full file rewrite per move** instead of append-only: keeps SGF always-valid with closing `)`, negligible cost at <5KB
- **Mid-game toggle uses AB[]/AW[]**: standard SGF setup properties to snapshot current board position, compatible with all viewers
- **No external SGF library**: the subset we use (linear games, no branches) is trivial to write/parse in ~300 lines total
- **Capture logic in reader**: ~40 lines of flood-fill liberty checking, needed for accurate board thumbnails
- **1 char per cell thumbnails**: `.` for empty, `●`/`○` for stones -- fits 19x19 comfortably in the preview pane
