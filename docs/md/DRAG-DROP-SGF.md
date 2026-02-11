# Drag-and-Drop SGF Loading

## Overview

Allow a user to drag an `.sgf` file from their file manager (Finder, Nautilus, Explorer) onto the terminal window running termsuji, and have it immediately load that game — abandoning any in-progress state.

## How Terminal Drag-and-Drop Actually Works

There is **no dedicated escape sequence or protocol** for file drag-and-drop in terminals. Every major terminal emulator handles it the same way: when a file is dragged onto the window, the terminal **injects the file's absolute path as text input**, as if the user typed or pasted it.

If the application has **bracketed paste mode** enabled (`\e[?2004h`), the terminal wraps the path in paste markers:

```
\e[200~ /path/to/file.sgf \e[201~
```

If bracketed paste is off, the path arrives character-by-character as keyboard events — indistinguishable from typing.

There is no way for the application to know the text came from a file drop vs. a clipboard paste vs. the user typing. The strategy is: detect pasted text, check if it looks like an SGF file path, and act on it.

## Terminal Quoting Behavior (the hard part)

Every terminal quotes the dropped path differently. This is the main source of edge cases.

| Terminal | Quoting of `/path/to/my file ($1).sgf` |
|---|---|
| macOS Terminal.app | `/path/to/my\ file\ (\$1).sgf` |
| iTerm2 | `/path/to/my\ file\ (\$1).sgf` |
| WezTerm (default macOS) | `/path/to/my\ file\ ($1).sgf` (SpacesOnly) |
| WezTerm (Posix mode) | `"/path/to/my file (\$1).sgf"` |
| Alacritty | `'/path/to/my file ($1).sgf'` |
| GNOME Terminal / Konsole | `'/path/to/my file ($1).sgf'` |
| kitty | `/path/to/my file ($1).sgf` (or `file://` URI) |
| Ghostty | `/path/to/my\ file\ ($1).sgf` |
| Windows Terminal | `"C:\Users\...\my file ($1).sgf"` |

WezTerm has a user-configurable `quote_dropped_files` option with five modes: `None`, `SpacesOnly`, `Posix`, `Windows`, `WindowsAlwaysQuoted`.

### Additional behaviors

- Some terminals append a **trailing space** or **trailing newline** after the path.
- **Multiple files** are separated by spaces (macOS) or newlines (some Linux terminals).
- kitty may paste `file:///path/to/file.sgf` instead of a bare path.
- Windows paths need translation in WSL (`C:\Users\...` -> `/mnt/c/Users/...`).

## tcell/tview Support

Our stack (tview on tcell) supports bracketed paste natively.

### tcell level

```go
screen.EnablePaste()

// In event loop:
case *tcell.EventPaste:
    ev.Start()  // paste began
    ev.End()    // paste ended
    // EventKey events between Start/End are the pasted content
```

### tview level

tview doesn't expose a direct `SetPasteHandler` on arbitrary primitives. The paste events flow through tcell's event system. We'd handle this by intercepting `tcell.EventPaste` events at the application level via the application's `SetBeforeDrawFunc` or by wrapping the input capture to accumulate paste content between start/end markers.

Alternatively, tview's `InputField` and `TextArea` widgets handle paste natively, but our game board is a custom-drawn `Box` — we need to handle paste events ourselves in the input capture chain.

## Detection Heuristic

Since drops look identical to pastes, we need a heuristic:

1. Receive pasted text (between bracketed paste markers)
2. Normalize the path (strip quoting, trim whitespace)
3. Check if it ends in `.sgf` (case-insensitive)
4. Check if the file exists on disk via `os.Stat()`
5. If all checks pass, treat it as an SGF file drop

This means a user who pastes a path to an SGF file from their clipboard gets the same behavior — which is actually desirable.

## Path Normalization

A robust normalizer needs to handle all the quoting variants:

```
Input:  "'/Users/me/games/my file.sgf'"       -> /Users/me/games/my file.sgf
Input:  "/Users/me/games/my\ file.sgf"         -> /Users/me/games/my file.sgf
Input:  "file:///Users/me/games/file.sgf"       -> /Users/me/games/file.sgf
Input:  '"/Users/me/games/my file.sgf"'         -> /Users/me/games/my file.sgf
Input:  "/Users/me/games/my file.sgf\n"         -> /Users/me/games/my file.sgf
```

Steps:
1. `strings.TrimSpace()` — remove trailing newlines/spaces
2. Strip `file://` prefix
3. Strip surrounding single or double quotes
4. Unescape `\ ` -> ` `, `\(` -> `(`, `\)` -> `)`, `\$` -> `$`, `\\` -> `\`

## Current Architecture (relevant parts)

### Existing SGF load flow

Today, loading a saved game goes through:

1. User opens History Browser, selects a game, presses `o`
2. `main.go:loadGame()` receives `sgf.GameInfo`
3. Constructs `engine.GameConfig` with `LoadSGFPath` set
4. Creates new GTP engine, connects it
5. Replays moves from SGF into the UI's move history
6. Opens SGF file for continued recording

Key files: `main.go` (loadGame ~line 292), `sgf/reader.go`, `engine/gtp/gtp.go`

### Existing input handling

All keyboard events are captured via `SetInputCapture` on the game board Box in `main.go` (~line 122). This is where we'd add paste detection.

### Page navigation

The app uses `tview.Pages` (`rootPage`) to switch between screens: `"setup"`, `"game"`, `"color"`, `"history"`. A drag-drop should work from any page — probably the most natural behavior is to switch to `"game"` and start the loaded game immediately.

## Implementation Plan

### 1. Enable bracketed paste in tcell

tcell's `Screen` has `EnablePaste()`. Call it after the screen is initialized. This makes tcell emit `*tcell.EventPaste` events with `Start()`/`End()` markers when pasted text arrives.

Location: `main.go`, after `app.Run()` setup or via tview's `SetBeforeDrawFunc` to get screen access.

### 2. Add paste event interception

Add an `InputCapture` at the **application level** (not just the game board) that:
- Detects `tcell.EventPaste` start → sets a "collecting paste" flag
- Accumulates `tcell.EventKey` rune events into a buffer while collecting
- Detects `tcell.EventPaste` end → processes the buffered text
- Returns `nil` for all events consumed during paste collection (so they don't reach widgets)

This needs to be at the app level so it works regardless of which page is focused (setup, game, history, color config).

### 3. Add path normalization function

New function, probably in a `util` package or directly in `main.go`:

```go
func cleanDroppedPath(raw string) string { ... }
```

Handles all the quoting variants documented above.

### 4. Add SGF detection and loading

When paste ends:
1. Call `cleanDroppedPath()` on the accumulated text
2. Check `strings.HasSuffix(lower, ".sgf")`
3. Check `os.Stat()` for existence
4. If valid, call a new `loadSGFFromPath(path string)` function

### 5. Add `loadSGFFromPath()` function

This is similar to the existing `loadGame()` but takes a raw file path instead of a `sgf.GameInfo` struct:

1. Call `sgf.ParseHeader(path)` to get game metadata (board size, komi, players, move count)
2. Tear down any existing engine connection
3. Construct `engine.GameConfig` with `LoadSGFPath`
4. Infer player color from SGF metadata (or default to black)
5. Infer engine level (default to config default, since external SGFs won't have this info)
6. Create and connect new engine
7. Replay moves into UI history
8. Switch to `"game"` page
9. Optionally open the SGF for continued recording (or start a new recording that includes the loaded position)

### 6. Handle "drop everything" semantics

The user wants the drop to abandon the current game. This means:
- If an engine is running, close it (`eng.Close()`)
- Clear the UI state (move history, planning mode, recorder)
- Don't prompt "are you sure?" — the drag is the intent

### 7. Handle external SGFs gracefully

SGFs from the wild differ from termsuji's own recordings:
- May lack player name fields (`PB[]`, `PW[]`) — default to "Black"/"White"
- May have unusual komi or board sizes — trust the SGF
- May have setup stones (`AB[]`/`AW[]`) — already handled by `sgf.ReplayToEnd()`
- May have variations/branches — current reader follows the main line only (first variation), which is fine
- May have incomplete games (no result) — fine, game continues from last position

## Edge Cases

| Case | Handling |
|------|----------|
| File doesn't end in `.sgf` | Ignore — pass events through to normal input handling |
| File doesn't exist on disk | Ignore silently |
| Malformed SGF | Show error in hint bar, stay on current screen |
| SGF with unsupported board size | GnuGo supports 2-19; show error for anything outside that |
| Multiple files dropped | Take only the first path (split on space or newline, take first `.sgf` match) |
| Drop during bot's turn | Still load — we're tearing down the engine anyway |
| Drop during planning mode | Exit planning mode, load the new game |
| Paste that isn't a file path | Falls through detection heuristic, events pass to normal handlers |
| Very long paste (not a path) | Cap buffer at ~4096 chars; if exceeded, flush and treat as normal input |
| User pastes SGF path from clipboard | Same behavior as drop — this is fine and useful |
| Bracketed paste not supported by terminal | Path arrives as rapid keystrokes; won't be detected as a paste. Could add a `/load <path>` command as fallback. |

## Files to Modify

| File | Change |
|------|--------|
| `main.go` | Enable paste, add app-level input capture, add `loadSGFFromPath()` |
| `ui/goboard.go` | Possibly add a `Reset()` or `Disconnect()` method to cleanly tear down state |
| `sgf/reader.go` | No changes needed — `ParseHeader()` and `ReplayToEnd()` already handle external SGFs |
| `engine/gtp/gtp.go` | No changes needed — `Close()` already exists |

Estimated scope: ~100-150 lines, mostly in `main.go`.

## Risk Assessment

**Low risk.** The feature is additive — it adds a new input path (paste detection) but doesn't modify any existing move/play/engine logic. If detection fails or the SGF is bad, we show an error and the current game continues undisturbed.

The main risk is **false positives**: pasted text that happens to end in `.sgf` and matches a real file. This is extremely unlikely in normal usage, and the `os.Stat()` check makes it near-impossible.

## Future Considerations

- **`/load <path>` command**: For terminals that don't support bracketed paste, a typed command would be a useful fallback.
- **Drop onto history browser**: Could add the dropped SGF to the history list instead of immediately loading it.
- **Preview before loading**: Show a modal with game metadata before committing. But the user asked for "immediately load", so skip this for now.
- **Drag-and-drop on Windows/WSL**: Would need path translation (`C:\...` -> `/mnt/c/...`). Low priority unless there's demand.
