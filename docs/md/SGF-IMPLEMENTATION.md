# SGF Game History Implementation

## Overview

This document outlines the implementation of automatic game saving using the SGF (Smart Game Format) standard. Every game will be recorded and persisted to disk, surviving application exits.

## SGF Format Specification

SGF FF[4] is a text-only, tree-based format for storing board game records. It's the universal standard for Go game records.

### Basic Structure

```sgf
(;GM[1]FF[4]SZ[19]KM[6.5]PB[Human]PW[GnuGo]
;B[pd]
;W[dp]
;B[pq]
;W[dd]
;B[fq])
```

- Wrapped in parentheses `( ... )`
- Nodes separated by semicolons `;`
- Properties are `KEY[value]` pairs
- Root node contains game metadata
- Subsequent nodes contain moves

### Required Root Properties

| Property | Description | Example |
|----------|-------------|---------|
| `GM[1]` | Game type (1 = Go) | Always `GM[1]` |
| `FF[4]` | File format version | Always `FF[4]` |
| `SZ[n]` | Board size | `SZ[9]`, `SZ[13]`, `SZ[19]` |
| `KM[n]` | Komi | `KM[6.5]` |

### Game Info Properties

| Property | Description | Example |
|----------|-------------|---------|
| `PB[name]` | Black player name | `PB[Human]` |
| `PW[name]` | White player name | `PW[GnuGo Level 5]` |
| `DT[date]` | Date played | `DT[2024-01-15]` |
| `RE[result]` | Game result | `RE[B+3.5]`, `RE[W+R]` |
| `AP[app]` | Application | `AP[termsuji-local:1.0]` |
| `CA[charset]` | Character set | `CA[UTF-8]` |

### Move Notation

- **Black move**: `B[xy]` where x=column, y=row (a-s)
- **White move**: `W[xy]`
- **Pass**: `B[]` or `W[]` (empty brackets)
- **Resign**: Indicated in `RE` property as `+R`

### Coordinate System

SGF uses letters a-s for 19x19 boards:
- Origin: top-left corner
- `aa` = top-left (0,0)
- `ss` = bottom-right (18,18)
- Letter 'i' is NOT skipped (unlike GTP protocol)

### Result Notation

| Format | Meaning |
|--------|---------|
| `RE[B+3.5]` | Black wins by 3.5 points |
| `RE[W+R]` | White wins by resignation |
| `RE[B+T]` | Black wins on time |
| `RE[W+F]` | White wins by forfeit |
| `RE[Jigo]` | Draw |
| `RE[?]` | Unfinished/unknown |

---

## Implementation Design

### Storage Location

```
~/.config/termsuji-local/history/
├── 2024-01-15_143022_9x9.sgf
├── 2024-01-15_151030_19x19.sgf
└── ...
```

**Filename format**: `{date}_{time}_{size}x{size}.sgf`

Using `~/.config` (via XDG) for consistency with existing config storage.

### Incremental Saving Strategy

**Key insight**: Games are saved continuously, not just at exit.

```
Game Start          → Create SGF with root node
Every Move          → Append move to SGF file
Game End            → Update RE[] property
Unexpected Exit     → File already contains all moves up to that point
```

**Hook points in code**:
1. `ConnectEngine()` - Create new SGF file
2. `OnMove()` callback - Append move node
3. `OnGameEnd()` callback - Finalize with result
4. `app.SetQuitFunc()` - Graceful cleanup (optional)

### Data Flow

```
GTPEngine.PlayMove()
    ↓
OnMove callback fires
    ↓
SGF writer appends: ";B[pd]" or ";W[dp]"
    ↓
File is flushed to disk
    ↓
Game continues...
```

This approach ensures:
- Every move is persisted immediately
- Crash-safe: file always contains latest state
- No data loss on unexpected exit

### SGF Writer Module

New file: `sgf/writer.go`

```go
type SGFWriter struct {
    file     *os.File
    gameInfo GameInfo
    moveNum  int
}

type GameInfo struct {
    BoardSize   int
    Komi        float64
    BlackPlayer string
    WhitePlayer string
    StartTime   time.Time
}

func NewSGFWriter(info GameInfo) (*SGFWriter, error)
func (w *SGFWriter) WriteMove(x, y, color int) error
func (w *SGFWriter) WritePass(color int) error
func (w *SGFWriter) SetResult(result string) error
func (w *SGFWriter) Close() error
```

### Coordinate Conversion

termsuji-local uses 0-indexed (x, y) coordinates.
SGF uses letter-based coordinates (a-s).

```go
func toSGFCoord(x, y int) string {
    return string(rune('a'+x)) + string(rune('a'+y))
}
// (0, 0) → "aa"
// (3, 4) → "de"
// (18, 18) → "ss"
```

---

## UX Considerations

### Automatic Behavior (No User Interaction Required)

- Games auto-save silently in background
- No save dialogs or prompts
- No "save game" button needed

### Potential Future UX Additions

1. **Game History Browser** (future feature)
   - List past games with date, result, size
   - Preview board position
   - Load/replay games

2. **Status Indicator** (minimal)
   - Small indicator showing "Recording..."
   - Or simply note in game info panel

3. **Export/Share** (future)
   - Copy SGF to clipboard
   - Reveal in file manager

### Current Scope (MVP)

For this implementation:
- Automatic silent recording
- No UI changes required
- History folder created on first game

---

## Integration Points

### Existing Code Touchpoints

1. **`engine/gtp/gtp.go`**
   - After `Connect()`: Initialize SGFWriter
   - In `moveCallback`: Call `WriteMove()`
   - In `OnGameEnd()`: Call `SetResult()`

2. **`ui/goboard.go`**
   - `ConnectEngine()`: Start recording
   - `Close()`: Ensure file is closed

3. **`config/config.go`**
   - Add history directory initialization

4. **New: `sgf/writer.go`**
   - Self-contained SGF writing module

### File Structure After Implementation

```
termsuji-local/
├── sgf/
│   └── writer.go      # New: SGF generation
├── engine/
│   └── gtp/
│       └── gtp.go     # Modified: integrate SGFWriter
├── ui/
│   └── goboard.go     # Modified: pass writer reference
└── config/
    └── config.go      # Modified: history dir init
```

---

## Technical Notes

### Why Not JSON?

- SGF is the universal Go format
- Compatible with all Go software (Sabaki, KGS, OGS, etc.)
- Human-readable and editable
- Smaller than JSON for game data
- Standard for archiving and sharing games

### Error Handling

- If SGF write fails, log error but don't interrupt game
- Create history directory on first use
- Handle disk full gracefully

### Future: Move History in Memory

Currently the engine doesn't store move history in memory. For SGF:
- Option A: SGFWriter tracks moves (simpler, current approach)
- Option B: Add move history to engine (enables undo, needed later)

Current implementation uses Option A for minimal changes.

---

## References

- [SGF FF[4] Specification](https://www.red-bean.com/sgf/)
- [SGF Format Overview](https://homepages.cwi.nl/~aeb/go/misc/sgf.html)
- [Wikipedia: Smart Game Format](https://en.wikipedia.org/wiki/Smart_Game_Format)
