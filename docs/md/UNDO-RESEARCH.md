# Undo Feature Research

## Overview

Implement an undo button that retracts the player's last move. Since the bot responds immediately after the player moves, undo must retract **two moves**: the bot's response and the player's move. This returns the board to the state before the player's last turn.

## Feasibility: Simple

GnuGo natively supports the GTP `undo` command, which handles all the hard parts: restoring captured stones, reverting ko state, and rolling back the position. We send `undo` twice (bot's move + player's move) and resync the board. No need to build our own move history or board state stack.

Estimated scope: ~50-80 lines across 3-4 files.

## Current Architecture (relevant parts)

### Move flow today
1. Player presses Enter → `main.go:145` calls `gameBoard.PlayMove(x, y)`
2. UI layer → `ui/goboard.go:232` calls `g.eng.PlayMove(x, y)`
3. Engine → `engine/gtp/gtp.go:164` sends GTP `play` command to GnuGo, updates board state, calls `updateBoardFromGnuGo()` for captures
4. Engine → `engine/gtp/gtp.go:221` fires `go g.triggerEngineMove()` — bot responds immediately
5. UI callback refreshes the board display

### What already exists that helps
- **`updateBoardFromGnuGo()`** (`gtp.go:365`) — queries GnuGo for all stone positions and rebuilds the board. After undo, this will give us the correct restored position for free.
- **`copyBoardState()`** (`gtp.go:440`) — deep-copies board state. Pattern to follow.
- **GTP `undo` command** — standard GTP command, GnuGo supports it. Retracts the last move played.
- **`sendCommand()`** (`gtp.go:108`) — already handles GTP command/response protocol.

### What doesn't exist yet
- No `Undo()` method on the engine interface
- No move history (only `MoveNumber` counter and single `LastMove`)
- No keybinding for undo

## Implementation Plan

### 1. Engine Interface (`engine/engine.go`)

Add to `GameEngine` interface:

```go
// Undo retracts the last full turn (bot's move + player's move).
// Returns an error if there are no moves to undo.
Undo() error
```

### 2. GTP Engine (`engine/gtp/gtp.go`)

Implement `Undo()`:

```go
func (g *GTPEngine) Undo() error {
    g.mu.Lock()
    defer g.mu.Unlock()

    if g.gameOver {
        return fmt.Errorf("game is over")
    }
    if !g.myTurn {
        return fmt.Errorf("not your turn")
    }
    if g.boardState.MoveNumber < 2 {
        return fmt.Errorf("no moves to undo")
    }

    // Undo bot's move
    if _, err := g.sendCommand("undo"); err != nil {
        return fmt.Errorf("undo failed: %w", err)
    }
    // Undo player's move
    if _, err := g.sendCommand("undo"); err != nil {
        return fmt.Errorf("undo failed: %w", err)
    }

    g.boardState.MoveNumber -= 2
    g.boardState.PlayerToMove = g.playerColor

    // Resync board from GnuGo (handles captures, ko, etc.)
    g.updateBoardFromGnuGo()

    // Restore LastMove from GnuGo
    // GnuGo supports `last_move` command returning color + vertex
    resp, err := g.sendCommand("last_move")
    if err != nil {
        // No previous move (we're back to start)
        g.boardState.LastMove.X = -1
        g.boardState.LastMove.Y = -1
    } else {
        // Parse "color vertex" response
        // set LastMove accordingly
    }

    return nil
}
```

Key detail: `updateBoardFromGnuGo()` rebuilds the entire board from GnuGo's authoritative state, so captured stones are automatically restored correctly.

### 3. UI Board (`ui/goboard.go`)

Add `Undo()` method mirroring the `Pass()` pattern:

```go
func (g *GoBoardUI) Undo() {
    if g.finished { return }
    if g.eng == nil { return }
    if !g.eng.IsMyTurn() { return }
    g.eng.Undo()
}
```

The existing `OnMove` callback will handle the UI refresh if we fire it after undo, or we can call `g.BoardState = g.eng.GetBoardState()` and trigger a redraw.

### 4. Keybinding (`main.go`)

Add `'u'` key in the game input handler (around line 146-164):

```go
case 'u':
    gameBoard.Undo()
```

## Edge Cases

| Case | Handling |
|------|----------|
| No moves played yet | `MoveNumber < 2` check, return error silently |
| Bot hasn't responded yet (not player's turn) | `IsMyTurn()` guard in UI layer |
| Game is over | `gameOver` check in engine |
| Player is white (bot moved first) | Works the same — undo still retracts 2 moves |
| First move as black (MoveNumber=2 after bot responds) | Should work — undoes back to empty board |
| Bot passed on its last move | GTP `undo` handles passes too |
| Multiple undos in a row | Each undo retracts 2 more moves; `MoveNumber < 2` prevents going below 0 |
| `last_move` command not supported | Fall back to `LastMove = (-1, -1)` — minor visual-only issue |

## What GnuGo's `undo` Handles for Us

- Restoring captured stones to the board
- Reverting ko state
- Reverting the internal move tree
- Handling pass moves in the undo stack

We don't need to implement any of this ourselves.

## Risk Assessment

**Low risk.** The feature is isolated — it adds a new code path but doesn't modify existing move/play logic. If `undo` fails, we return an error and the board stays as-is. No state corruption possible since we resync from GnuGo after undo.

## Future Considerations

- **SGF integration**: If/when SGF recording is implemented, undo would need to also retract the last moves from the SGF writer's history
- **Move history in engine**: The SGF implementation doc (`docs/md/SGF-IMPLEMENTATION.md:253-259`) already identifies "Option B: Add move history to engine (enables undo)" — this undo implementation avoids needing that by leaning on GnuGo's state
- **Undo limit**: Could add a configurable max undo count, but not needed for MVP
