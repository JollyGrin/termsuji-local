#!/bin/bash

CONFIG_DIR="$HOME/.config/termsuji-local"
CONFIG_FILE="$CONFIG_DIR/config.json"

if [ -f "$CONFIG_FILE" ]; then
    echo "Config already exists at $CONFIG_FILE"
    exit 0
fi

mkdir -p "$CONFIG_DIR"

cat > "$CONFIG_FILE" << 'EOF'
{
  "theme": {
    "draw_stone_bg": false,
    "draw_cursor_bg": true,
    "draw_last_played_bg": true,
    "fullwidth_letters": false,
    "use_grid_lines": true,
    "colors": {
      "board": 180,
      "board_alt": 180,
      "black": 232,
      "black_alt": 232,
      "white": 255,
      "white_alt": 255,
      "line": 94,
      "cursor_fg": 2,
      "cursor_bg": 4,
      "last_played_bg": 2
    },
    "symbols": {
      "black": "●",
      "white": "●",
      "board": "┼",
      "cursor": "┼",
      "last_played": "┼"
    }
  },
  "gnugo": {
    "gnugo_path": "gnugo",
    "default_board_size": 19,
    "default_komi": 6.5,
    "default_level": 5
  }
}
EOF

echo "Created config at $CONFIG_FILE"
