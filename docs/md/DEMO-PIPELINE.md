# Visual Demo Recording Pipeline

> Declarative "tape" scripts that produce `.cast` recordings and MP4 videos via tmux + asciinema. This lets you verify UI behavior visually after each feature.

## Pipeline

```
.tape script  →  tmux + asciinema  →  .cast  →  clean-cast.sh  →  cast-to-mp4.sh  →  .mp4
```

VHS doesn't natively output `.cast` files (only GIF/MP4/WebM). Instead we use **tmux + asciinema** as the recording engine with a custom DSL library inspired by VHS tape syntax. tmux sends real keystrokes that tview/tcell reads natively, and asciinema captures the terminal output perfectly.

## Directory Layout

```
demos/
├── lib/
│   └── tape.sh                  # DSL library (tmux + asciinema helpers)
├── tapes/
│   ├── 01-basic-game.tape       # 9x9 game, a few moves, quit
│   ├── 02-focus-mode.tape       # Normal → focus toggle
│   └── 03-planning-mode.tape    # Plan mode: place, navigate, branch
├── casts/                       # Generated .cast files (gitignored)
├── output/                      # Generated .mp4 files (gitignored)
└── run.sh                       # Orchestrator
```

---

## 1. DSL Library (`demos/lib/tape.sh`)

Sourced by every `.tape` script. Wraps tmux session management, asciinema recording, and keystroke delivery into a small, composable API.

### Functions

| Function | Purpose |
|----------|---------|
| `tape_init <name> [cols] [rows]` | Build binary, create tmux session, start asciinema rec |
| `tape_run <command>` | Launch command in the session |
| `tape_key <key> [key...]` | Send keystrokes via `tmux send-keys` (maps Enter, Up, Down, etc.) |
| `tape_type <text> [delay]` | Type text char-by-char with realistic delay |
| `tape_sleep <seconds>` | Pause (visible in recording) |
| `tape_wait_for <pattern> [timeout]` | Poll `tmux capture-pane` until pattern appears |
| `tape_finish` | Exit app/asciinema, kill tmux, run clean-cast + cast-to-mp4 |

### Implementation Details

```bash
#!/usr/bin/env bash
# demos/lib/tape.sh — DSL library for termsuji tape recordings
set -euo pipefail

REPO_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
TAPE_NAME=""
TAPE_SESSION=""
CAST_FILE=""
MP4_FILE=""

# ── tape_init ────────────────────────────────────────────────
# Build binary (if stale), create tmux session, start asciinema.
#   tape_init <name> [cols] [rows]
tape_init() {
  local name="${1:?tape_init requires a name}"
  local cols="${2:-100}"
  local rows="${3:-30}"
  TAPE_NAME="$name"
  TAPE_SESSION="termsuji-tape-${name}-$$"
  CAST_FILE="${REPO_ROOT}/demos/casts/${name}.cast"
  MP4_FILE="${REPO_ROOT}/demos/output/${name}.mp4"

  mkdir -p "${REPO_ROOT}/demos/casts" "${REPO_ROOT}/demos/output"

  # Check dependencies
  for cmd in tmux asciinema; do
    if ! command -v "$cmd" &>/dev/null; then
      echo "ERROR: $cmd not found. Install it first." >&2
      exit 1
    fi
  done

  # Auto-build if binary is stale
  local binary="${REPO_ROOT}/termsuji-local"
  local newest_go
  newest_go="$(find "$REPO_ROOT" -name '*.go' -newer "$binary" 2>/dev/null | head -1)"
  if [[ ! -x "$binary" || -n "$newest_go" ]]; then
    echo "Building termsuji-local..."
    (cd "$REPO_ROOT" && go build)
  fi

  # Clean up any stale sessions with this name prefix
  tmux list-sessions -F '#{session_name}' 2>/dev/null \
    | grep "^termsuji-tape-${name}-" \
    | xargs -I{} tmux kill-session -t {} 2>/dev/null || true

  # Create tmux session
  tmux new-session -d -s "$TAPE_SESSION" -x "$cols" -y "$rows"

  # Trap for cleanup on unexpected exit
  trap '_tape_cleanup' EXIT INT TERM

  # Start asciinema recording inside the tmux session
  tmux send-keys -t "$TAPE_SESSION" \
    "asciinema rec --cols ${cols} --rows ${rows} --overwrite '${CAST_FILE}'" Enter
  sleep 1  # Let asciinema start
}

# ── tape_run ─────────────────────────────────────────────────
# Launch a command inside the recording session.
#   tape_run "./termsuji-local"
tape_run() {
  local command="${1:?tape_run requires a command}"
  tmux send-keys -t "$TAPE_SESSION" "$command" Enter
}

# ── tape_key ─────────────────────────────────────────────────
# Send one or more keystrokes. Supports tmux key names:
#   Enter, Up, Down, Left, Right, Tab, Escape, BTab (Shift-Tab)
#   tape_key Up Up Enter
#   tape_key q
tape_key() {
  for key in "$@"; do
    tmux send-keys -t "$TAPE_SESSION" "$key"
    sleep 0.05
  done
}

# ── tape_type ────────────────────────────────────────────────
# Type text char-by-char with realistic delay.
#   tape_type "hello world" 0.08
tape_type() {
  local text="${1:?tape_type requires text}"
  local delay="${2:-0.05}"
  for (( i=0; i<${#text}; i++ )); do
    local char="${text:$i:1}"
    tmux send-keys -t "$TAPE_SESSION" -l "$char"
    sleep "$delay"
  done
}

# ── tape_sleep ───────────────────────────────────────────────
# Pause for N seconds (visible in recording).
#   tape_sleep 2
tape_sleep() {
  sleep "${1:?tape_sleep requires seconds}"
}

# ── tape_wait_for ────────────────────────────────────────────
# Poll tmux pane content until a pattern appears.
# Used to sync with GnuGo engine responses.
#   tape_wait_for "W:" 10
tape_wait_for() {
  local pattern="${1:?tape_wait_for requires a pattern}"
  local timeout="${2:-10}"
  local elapsed=0
  while (( elapsed < timeout )); do
    if tmux capture-pane -t "$TAPE_SESSION" -p | grep -q "$pattern"; then
      return 0
    fi
    sleep 0.3
    elapsed=$(( elapsed + 1 ))
  done
  echo "WARN: tape_wait_for '$pattern' timed out after ${timeout}s" >&2
}

# ── tape_finish ──────────────────────────────────────────────
# Stop recording, kill session, post-process.
tape_finish() {
  # Exit the app gracefully (if still running)
  tape_key q
  sleep 0.5
  tape_key q
  sleep 0.5

  # Stop asciinema (Ctrl-D or exit)
  tmux send-keys -t "$TAPE_SESSION" "exit" Enter
  sleep 1

  # Kill session
  tmux kill-session -t "$TAPE_SESSION" 2>/dev/null || true
  trap - EXIT INT TERM

  # Post-process
  if [[ -f "$CAST_FILE" ]]; then
    echo "Cleaning cast: $CAST_FILE"
    "${REPO_ROOT}/scripts/clean-cast.sh" "$CAST_FILE" --inplace --max-delay 2.0

    echo "Converting to MP4: $MP4_FILE"
    "${REPO_ROOT}/scripts/cast-to-mp4.sh" "$CAST_FILE" -o "$MP4_FILE"
  else
    echo "WARN: cast file not found at $CAST_FILE" >&2
  fi
}

# ── _tape_cleanup (internal) ─────────────────────────────────
_tape_cleanup() {
  tmux kill-session -t "$TAPE_SESSION" 2>/dev/null || true
}
```

### Key Design Decisions

- **Session naming:** `termsuji-tape-<name>-$$` — the PID suffix prevents collisions when running tapes concurrently or re-running after a crash.
- **Trap cleanup:** EXIT/INT/TERM traps ensure tmux sessions don't leak.
- **Auto-build:** Compares `*.go` timestamps against the binary; only rebuilds when source is newer.
- **`tape_wait_for` polling:** Uses `tmux capture-pane -p | grep` at 0.3s intervals. Essential for synchronizing with GnuGo engine responses (the `W:` turn indicator appears after the engine plays).
- **`tape_key` sleep:** A 50ms gap between keystrokes prevents tcell from dropping events.

---

## 2. Orchestrator (`demos/run.sh`)

```bash
#!/usr/bin/env bash
# demos/run.sh — Run all tape scripts (or a subset)
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"

usage() {
  echo "Usage: $0 [options] [tape...]"
  echo ""
  echo "Options:"
  echo "  --cast-only   Generate .cast files but skip MP4 conversion"
  echo "  --list        List available tapes and exit"
  echo "  --clean       Remove all generated casts and output"
  echo ""
  echo "Examples:"
  echo "  $0                        # Run all tapes"
  echo "  $0 01-basic-game          # Run one tape"
  echo "  $0 --cast-only 02-focus   # Cast only, no MP4"
  echo "  $0 --list                 # Show available tapes"
}

# Kill stale tape sessions on entry
tmux list-sessions -F '#{session_name}' 2>/dev/null \
  | grep '^termsuji-tape-' \
  | xargs -I{} tmux kill-session -t {} 2>/dev/null || true

CAST_ONLY=false
TAPES=()

while [[ $# -gt 0 ]]; do
  case "$1" in
    --cast-only) CAST_ONLY=true; shift ;;
    --list)
      echo "Available tapes:"
      for f in "$SCRIPT_DIR"/tapes/*.tape; do
        echo "  $(basename "$f" .tape)"
      done
      exit 0
      ;;
    --clean)
      rm -rf "$SCRIPT_DIR/casts" "$SCRIPT_DIR/output"
      echo "Cleaned demos/casts/ and demos/output/"
      exit 0
      ;;
    --help|-h) usage; exit 0 ;;
    *) TAPES+=("$1"); shift ;;
  esac
done

# Export for tape scripts
export CAST_ONLY

# Find tapes to run
if [[ ${#TAPES[@]} -eq 0 ]]; then
  for f in "$SCRIPT_DIR"/tapes/*.tape; do
    TAPES+=("$(basename "$f" .tape)")
  done
fi

# Run tapes sequentially (tmux sessions must not overlap)
for name in "${TAPES[@]}"; do
  tape_file="$SCRIPT_DIR/tapes/${name}.tape"
  if [[ ! -f "$tape_file" ]]; then
    echo "ERROR: tape not found: $tape_file" >&2
    exit 1
  fi
  echo "=== Running tape: $name ==="
  bash "$tape_file"
  echo "=== Done: $name ==="
  echo ""
done

echo "All tapes complete."
echo "  Casts:  demos/casts/"
echo "  Videos: demos/output/"
```

### Usage

```bash
# Run all tapes
./demos/run.sh

# Run a single tape
./demos/run.sh 01-basic-game

# Generate .cast files only (skip MP4)
CAST_ONLY=true ./demos/run.sh

# List available tapes
./demos/run.sh --list

# Clean generated files
./demos/run.sh --clean
```

---

## 3. Starter Tape Examples

### Setup Screen Navigation

The setup screen has a focus chain navigated with Tab / arrow keys:

| Index | Component | Default |
|-------|-----------|---------|
| 0 | Board size (radio: 9x9 / 13x13 / 19x19) | 19x19 (idx 2) |
| 1 | Your color (radio: Black / White) | Black (idx 0) |
| 2 | Strength (slider: 1–10) | 5 |
| 3 | Komi (input: float) | 6.5 |
| 4 | (P)LAY button | — |
| 5 | HISTORY button | — |
| 6 | COLORS button | — |
| 7 | QUIT button | — |

To select **9x9**: focus is on boardSelect by default, press **Up twice** (moves from idx 2 → 1 → 0).
Hotkey **`p`** starts the game from anywhere except the komi input.

### `01-basic-game.tape`

A quick 9x9 game: select board size, play 3 moves (waiting for GnuGo responses), then quit. ~15 seconds.

```bash
#!/usr/bin/env bash
# demos/tapes/01-basic-game.tape — 9x9 game, 3 moves, quit
set -euo pipefail
SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
source "$SCRIPT_DIR/../lib/tape.sh"

tape_init "01-basic-game" 100 30

# Launch termsuji
tape_run "./termsuji-local"
tape_sleep 1

# Setup screen: select 9x9 (default is 19x19, Up twice)
tape_key Up Up
tape_sleep 0.5

# Start game
tape_key p
tape_sleep 2  # Wait for board + GnuGo to initialize

# Move 1: navigate to center (E5 on 9x9) and play
tape_key Right Right Right Right  # Move cursor to column E
tape_key Down Down Down Down      # Move cursor to row 5
tape_sleep 0.3
tape_key Enter                    # Place stone
tape_wait_for "W:" 10            # Wait for GnuGo response

# Move 2: play at D3
tape_sleep 1
tape_key Left                     # One left to column D
tape_key Up Up                    # Up to row 3
tape_sleep 0.3
tape_key Enter
tape_wait_for "W:" 10

# Move 3: play at F7
tape_sleep 1
tape_key Right Right              # Right to column F
tape_key Down Down Down Down      # Down to row 7
tape_sleep 0.3
tape_key Enter
tape_wait_for "W:" 10

tape_sleep 2  # Pause to show final position

# Quit: q deselects cursor, q again returns to setup
tape_finish
```

### `02-focus-mode.tape`

Start a 9x9 game in normal layout, pause to show the side panels, then toggle focus mode with `f`. ~20 seconds.

```bash
#!/usr/bin/env bash
# demos/tapes/02-focus-mode.tape — Normal → focus toggle
set -euo pipefail
SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
source "$SCRIPT_DIR/../lib/tape.sh"

tape_init "02-focus-mode" 120 35

# Launch and start 9x9 game
tape_run "./termsuji-local"
tape_sleep 1
tape_key Up Up     # Select 9x9
tape_sleep 0.5
tape_key p
tape_sleep 2

# Show normal layout: make a move so there's something on screen
tape_key Right Right Right Right Down Down Down Down
tape_sleep 0.3
tape_key Enter
tape_wait_for "W:" 10
tape_sleep 2  # Pause to show normal layout with side panels

# Toggle focus mode
tape_key f
tape_sleep 3  # Pause to show fullscreen board (hint: "f to toggle")

# Make another move in focus mode
tape_key Left Left Up Up
tape_sleep 0.3
tape_key Enter
tape_wait_for "W:" 10
tape_sleep 2

# Toggle back to normal
tape_key f
tape_sleep 2

tape_finish
```

### `03-planning-mode.tape`

Start a game, play a few moves, enter planning mode, place variations, navigate with `[` / `]`, and show branching with `{` / `}`. ~25 seconds.

```bash
#!/usr/bin/env bash
# demos/tapes/03-planning-mode.tape — Plan mode with navigation
set -euo pipefail
SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
source "$SCRIPT_DIR/../lib/tape.sh"

tape_init "03-planning-mode" 120 35

# Launch and start 9x9 game
tape_run "./termsuji-local"
tape_sleep 1
tape_key Up Up     # Select 9x9
tape_sleep 0.5
tape_key p
tape_sleep 2

# Play 2 real moves to set up a board position
tape_key Right Right Right Right Down Down Down Down
tape_sleep 0.3
tape_key Enter
tape_wait_for "W:" 10
tape_sleep 1

tape_key Left Left Up
tape_sleep 0.3
tape_key Enter
tape_wait_for "W:" 10
tape_sleep 1

# Enter planning mode
tape_key a
tape_sleep 1.5  # Pause to show PLAN indicator

# Place a planned stone (variation 1)
tape_key Right Right Down Down
tape_sleep 0.3
tape_key Enter
tape_sleep 1

# Place another planned stone
tape_key Left Down
tape_sleep 0.3
tape_key Enter
tape_sleep 1

# Navigate back with [
tape_key '['
tape_sleep 1
tape_key '['
tape_sleep 1

# Place a different stone to create a branch (variation 2)
tape_key Up Up Right
tape_sleep 0.3
tape_key Enter
tape_sleep 1

# Navigate forward with ]
tape_key ']'
tape_sleep 1

# Switch between variations with { and }
tape_key '['       # Go back to branch point
tape_sleep 1
tape_key '}'       # Next variation
tape_sleep 1.5
tape_key '{'       # Previous variation
tape_sleep 1.5

# Exit planning mode
tape_key a
tape_sleep 2

tape_finish
```

---

## 4. Gitignore Additions

Add to the project `.gitignore`:

```
# Demo recordings (generated)
demos/casts/
demos/output/
```

---

## 5. Reuse of Existing Scripts

The pipeline calls two scripts already in the repo:

### `scripts/clean-cast.sh`

Called by `tape_finish` with `--max-delay 2.0` (more generous than the default 0.5 since tape recordings have intentional pauses for readability).

```bash
"${REPO_ROOT}/scripts/clean-cast.sh" "$CAST_FILE" --inplace --max-delay 2.0
```

What it does:
- Strips the hostname prefix (`grins@Deans-MacBook-Pro termsuji-local`) from shell prompts
- Caps long delays to the specified maximum
- Preserves the asciicast v3 header and all TUI rendering

### `scripts/cast-to-mp4.sh`

Called by `tape_finish` for MP4 conversion (skipped when `CAST_ONLY=true`).

```bash
"${REPO_ROOT}/scripts/cast-to-mp4.sh" "$CAST_FILE" -o "$MP4_FILE"
```

Requires `agg` and `ffmpeg`. See [SKILLS.md](../../SKILLS.md) for install instructions.

---

## 6. Dependencies

| Tool | Purpose | Install |
|------|---------|---------|
| `tmux` | Session management, keystroke delivery | `brew install tmux` |
| `asciinema` | Terminal recording to `.cast` | `brew install asciinema` |
| `agg` | `.cast` → GIF conversion | `cargo install --git https://github.com/asciinema/agg` |
| `ffmpeg` | GIF → MP4 conversion | `brew install ffmpeg` |
| `gnugo` | Game engine (needed by termsuji) | `brew install gnugo` |

---

## 7. Verification

### Run a single tape

```bash
./demos/run.sh 01-basic-game
```

### Inspect the .cast file

```bash
# Check it's valid asciicast v3 (first line is JSON header)
head -1 demos/casts/01-basic-game.cast | python3 -m json.tool

# Play it back in terminal
asciinema play demos/casts/01-basic-game.cast
```

### Watch the MP4

```bash
open demos/output/01-basic-game.mp4        # macOS
# or
ffplay demos/output/01-basic-game.mp4       # ffmpeg's player
```

### Run all tapes

```bash
./demos/run.sh
```

### Expected output

```
=== Running tape: 01-basic-game ===
Building termsuji-local...
Cleaning cast: demos/casts/01-basic-game.cast
Wrote 847 lines to demos/casts/01-basic-game.cast
Converting to MP4: demos/output/01-basic-game.mp4
Running agg: .cast → .gif
Running ffmpeg: .gif → .mp4
Wrote demos/output/01-basic-game.mp4
=== Done: 01-basic-game ===
```
