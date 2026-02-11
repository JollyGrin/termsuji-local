# Save Config Defaults

## Overview

The menu screen hardcodes initial values (19x19, black, level 5, komi 6.5) every time the app launches. Game settings like board size, komi, and level are never persisted — only theme/colors are saved to `~/.config/termsuji-local/config.json`.

This feature adds:
1. A **save hotkey** (`s`) on the menu to persist current selections as defaults
2. A **reset hotkey** (`r`) that appears only when menu values deviate from saved config
3. **Startup initialization** from saved config instead of hardcoded values

---

## Config Changes

### New Field: `DefaultPlayerColor`

`GnuGoConfig` in `config/config.go` already has `DefaultBoardSize`, `DefaultKomi`, and `DefaultLevel`, but player color is missing — it's hardcoded to black (1) everywhere.

```go
// config/config.go
type GnuGoConfig struct {
    Path               string  `json:"gnugo_path"`
    DefaultBoardSize   int     `json:"default_board_size"`
    DefaultKomi        float64 `json:"default_komi"`
    DefaultLevel       int     `json:"default_level"`
    DefaultPlayerColor int     `json:"default_player_color"` // NEW: 1=black, 2=white
}
```

### Default Value

Update `DefaultConfig` in `config/defaultconfigs.go`:

```go
GnuGo: GnuGoConfig{
    Path:               "gnugo",
    DefaultBoardSize:   19,
    DefaultKomi:        6.5,
    DefaultLevel:       5,
    DefaultPlayerColor: 1, // NEW: black
},
```

### Backward Compatibility

Old config files won't have `default_player_color`. When Go unmarshals JSON into the struct, missing int fields default to `0`, which is neither black (1) nor white (2).

Add a `NormalizeDefaults()` method to fix this after loading:

```go
// config/config.go
func (c *Config) NormalizeDefaults() {
    if c.GnuGo.DefaultPlayerColor < 1 || c.GnuGo.DefaultPlayerColor > 2 {
        c.GnuGo.DefaultPlayerColor = 1 // default to black
    }
}
```

Call `NormalizeDefaults()` in `InitConfig()` after `readCfgFile()`, before `Validate()`.

---

## Menu Integration

### Accept Config in Constructor

`NewGameSetup()` currently takes 3 callbacks and hardcodes initial values. Change the signature to accept `*config.Config`:

```go
// ui/gamesetup.go

// Before:
func NewGameSetup(onStart func(engine.GameConfig), onCancel func(), onColors func()) *GameSetupUI {

// After:
func NewGameSetup(cfg *config.Config, onStart func(engine.GameConfig), onCancel func(), onColors func()) *GameSetupUI {
```

### Initialize From Config

Replace hardcoded values with config reads:

```go
setup := &GameSetupUI{
    cfg:         cfg,                            // store reference for save/reset
    onStart:     onStart,
    onCancel:    onCancel,
    onColors:    onColors,
    boardSize:   cfg.GnuGo.DefaultBoardSize,     // was: 19
    playerColor: cfg.GnuGo.DefaultPlayerColor,   // was: 1
    level:       cfg.GnuGo.DefaultLevel,          // was: 5
    komi:        cfg.GnuGo.DefaultKomi,            // was: 6.5
}
```

Add `cfg` field to the struct:

```go
type GameSetupUI struct {
    // ... existing fields ...
    cfg *config.Config
}
```

### Board Size Index Helper

The `RadioSelect` for board size takes an index (0/1/2), not a raw size. Add a helper:

```go
func boardSizeToIndex(size int) int {
    switch size {
    case 9:
        return 0
    case 13:
        return 1
    default:
        return 2 // 19x19
    }
}
```

Use it when creating the board select:

```go
setup.boardSelect = NewRadioSelect("Board Size", boardOptions, boardSizeToIndex(cfg.GnuGo.DefaultBoardSize), ...)
```

Similarly for color select (index is `DefaultPlayerColor - 1`), level slider, and komi input — pass the config value as the initial value to each component constructor.

### Wiring in main.go

Update the `NewGameSetup` call in `main.go` to pass config:

```go
// main.go line ~170
setupUI := ui.NewGameSetup(cfg,
    func(gameCfg engine.GameConfig) {
        startGame(gameCfg)
    },
    func() {
        app.Stop()
    },
    func() {
        rootPage.SwitchToPage("colors")
    },
)
```

---

## Hotkeys

### Save (`s`)

Pressing `s` persists the current menu values into the config struct and writes to disk.

```go
// In handleInput(), inside the tcell.KeyRune case:
if event.Rune() == 's' && s.focusIndex != 3 {
    s.saveDefaults()
    return nil
}
```

The `focusIndex != 3` guard prevents the hotkey from firing when the komi text input is focused, matching the existing `p` hotkey pattern.

```go
func (s *GameSetupUI) saveDefaults() {
    s.cfg.GnuGo.DefaultBoardSize = s.boardSize
    s.cfg.GnuGo.DefaultPlayerColor = s.playerColor
    s.cfg.GnuGo.DefaultLevel = s.level
    s.cfg.GnuGo.DefaultKomi = s.komi
    s.cfg.Save()
}
```

### Reset (`r`)

Pressing `r` resets the menu to saved config values. Only active when menu has deviated.

```go
if event.Rune() == 'r' && s.focusIndex != 3 && s.hasDeviated() {
    s.resetToDefaults()
    return nil
}
```

```go
func (s *GameSetupUI) resetToDefaults() {
    s.boardSize = s.cfg.GnuGo.DefaultBoardSize
    s.playerColor = s.cfg.GnuGo.DefaultPlayerColor
    s.level = s.cfg.GnuGo.DefaultLevel
    s.komi = s.cfg.GnuGo.DefaultKomi

    // Update UI components to reflect reset values
    s.boardSelect.SetSelected(boardSizeToIndex(s.boardSize))
    s.colorSelect.SetSelected(s.playerColor - 1)
    s.levelSlider.SetValue(s.level)
    s.komiInput.SetValue(s.komi)
}
```

**Note**: `SetSelected()`, `SetValue()` methods may need to be added to `RadioSelect`, `LevelSlider`, and `KomiInput` if they don't already exist. These are simple setter methods that update internal state and trigger redraw.

---

## Deviation Detection

### `hasDeviated()`

Compares the live menu fields against the saved config:

```go
func (s *GameSetupUI) hasDeviated() bool {
    return s.boardSize != s.cfg.GnuGo.DefaultBoardSize ||
        s.playerColor != s.cfg.GnuGo.DefaultPlayerColor ||
        s.level != s.cfg.GnuGo.DefaultLevel ||
        s.komi != s.cfg.GnuGo.DefaultKomi
}
```

### State Transitions

```
App Launch       → menu == config → no deviation
User changes     → menu != config → deviated
User saves (s)   → config = menu  → no deviation
User resets (r)  → menu = config  → no deviation
```

---

## Dynamic Help Bar

### Current State

The help text is a static `tview.TextView` created once in `NewGameSetup()`:

```go
helpText := tview.NewTextView().
    SetText("↑↓ options · Tab next · p play · ctrl-c quit").
    SetTextAlign(tview.AlignCenter)
```

### New Behavior

Store `helpText` as a struct field so it can be updated:

```go
type GameSetupUI struct {
    // ... existing fields ...
    helpText *tview.TextView
}
```

Add `updateHelpText()`:

```go
func (s *GameSetupUI) updateHelpText() {
    base := "↑↓ · Tab · p play · s save"
    if s.hasDeviated() {
        base += " · r reset"
    }
    s.helpText.SetText(base)
}
```

Call `updateHelpText()` with `defer` at the top of `handleInput()` so it runs after every key press:

```go
func (s *GameSetupUI) handleInput(event *tcell.EventKey) *tcell.EventKey {
    defer s.updateHelpText()
    // ... existing handler logic ...
}
```

This ensures the hint bar always reflects the current state after any input.

---

## Files Modified

| File | Changes |
|------|---------|
| `config/config.go` | Add `DefaultPlayerColor` field to `GnuGoConfig`, add `NormalizeDefaults()` method, call it in `InitConfig()` |
| `config/defaultconfigs.go` | Add `DefaultPlayerColor: 1` to `DefaultConfig` |
| `ui/gamesetup.go` | Accept `*config.Config`, init from config, add `saveDefaults()`, `resetToDefaults()`, `hasDeviated()`, `updateHelpText()`, `boardSizeToIndex()`, `s`/`r` hotkeys, store `helpText` as field |
| `main.go` | Pass `cfg` to `NewGameSetup()` |

---

## Config JSON Format

### Before (current)

```json
{
  "theme": { ... },
  "gnugo": {
    "gnugo_path": "gnugo",
    "default_board_size": 19,
    "default_komi": 6.5,
    "default_level": 5
  }
}
```

### After

```json
{
  "theme": { ... },
  "gnugo": {
    "gnugo_path": "gnugo",
    "default_board_size": 19,
    "default_komi": 6.5,
    "default_level": 5,
    "default_player_color": 1
  }
}
```

### Backward Compatibility

- Old config files missing `default_player_color` → JSON unmarshal yields `0` → `NormalizeDefaults()` corrects to `1` (black)
- Old config files with all other fields present continue to work unchanged
- The `Config.Save()` method already writes the full struct, so saving from the menu will add the new field automatically

---

## Implementation Notes

- `Config.Save()` already exists and writes the full struct to `~/.config/termsuji-local/config.json` — no new file I/O code needed
- `buildGameConfigFromFlags()` in `main.go` already reads `cfg.GnuGo.DefaultBoardSize`, `cfg.GnuGo.DefaultKomi`, and `cfg.GnuGo.DefaultLevel` for CLI quick-start — adding `DefaultPlayerColor` there too keeps the CLI path consistent
- The `playButton` callback currently hardcodes `EnginePath: "gnugo"` — this should also read from `cfg.GnuGo.Path` for consistency (existing bug, not part of this feature but worth noting)
