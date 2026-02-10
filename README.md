# termsuji-local

A beautiful terminal UI for playing Go/Baduk against GnuGo offline.

Fork of [termsuji](https://github.com/lvank/termsuji) that replaces the online-go.com backend with a local GnuGo (GTP) subprocess interface.

https://github.com/user-attachments/assets/7bee8c44-5521-484b-b3cc-4ed2236fe9bd

## Features

- Beautiful terminal-based Go board (inherited from termsuji)
- Play against GnuGo locally, completely offline
- No time limits (untimed games)
- Configurable board sizes (9x9, 13x13, 19x19)
- Adjustable engine difficulty (levels 1-10)
- Choose your color (Black/White)
- Customizable board color

## Requirements

- Go 1.18+
- GnuGo installed and in PATH
- Terminal with Unicode support

### Installing GnuGo

**macOS** (via Homebrew):
```bash
brew install gnu-go
```

**Linux (Ubuntu/Debian)**:
```bash
sudo apt install gnugo
```

**Linux (Fedora)**:
```bash
sudo dnf install gnugo
```

**Linux (Arch)**:
```bash
sudo pacman -S gnugo
```

**Windows**:
1. Download GnuGo from http://www.gnu.org/software/gnugo/download.html
2. Extract to a folder (e.g., `C:\gnugo`)
3. Add the folder to your PATH:
   - Open System Properties → Advanced → Environment Variables
   - Edit PATH and add `C:\gnugo` (or wherever you extracted it)
4. Verify installation: `gnugo --version`

Alternatively on Windows with WSL:
```bash
sudo apt install gnugo
```

## Installation

```bash
git clone git@github.com:JollyGrin/termsuji-local.git
cd termsuji-local
go build
```

This creates a `termsuji-local` executable in the current directory.

## Usage

Simply run the binary:

```bash
./termsuji-local
```

### Command Line Flags

| Flag | Description | Default |
|------|-------------|---------|
| `--boardsize` | Board size (9, 13, or 19) | 19 |
| `--level` | GnuGo difficulty (1-10) | 5 |
| `--komi` | Komi compensation for White | 6.5 |

You'll be presented with a game setup screen where you can configure:
- Board size (9x9, 13x13, 19x19)
- Your color (Black plays first, White plays second)
- GnuGo difficulty level (1-10)
- Komi (compensation for White)

## Controls

| Key | Action |
|-----|--------|
| Arrow keys | Move cursor |
| Enter | Play move at cursor |
| p | Pass turn |
| q | Quit (or deselect cursor) |

## Configuration

Configuration is stored at `~/.config/termsuji-local/config.json`:

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

## Credits

Based on [termsuji](https://github.com/lvank/termsuji) by lvank.

## License

MIT
