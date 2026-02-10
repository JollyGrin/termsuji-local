// Package gtp provides a GTP (Go Text Protocol) engine implementation.
package gtp

import (
	"fmt"
	"strconv"
	"strings"
)

// GTP coordinate system:
// - Columns: A-T (skipping I to avoid confusion with 1)
// - Rows: 1-19 (from bottom of board)
// - Example: D4, Q16, K10
//
// Termsuji coordinate system:
// - X: 0-18 (left to right)
// - Y: 0-18 (top to bottom)
// - Example: (3, 15) for D4 on a 19x19 board

// posToGTP converts termsuji coordinates (0-indexed, top-left origin) to GTP notation.
// For a 19x19 board: (0, 18) -> A1, (3, 15) -> D4, (15, 3) -> Q16
func posToGTP(x, y, size int) string {
	// Column: A-T, skipping I
	col := 'A' + rune(x)
	if x >= 8 {
		col++ // Skip 'I'
	}

	// Row: 1-19 from bottom, so invert Y
	row := size - y

	return fmt.Sprintf("%c%d", col, row)
}

// gtpToPos converts GTP notation to termsuji coordinates.
// For a 19x19 board: A1 -> (0, 18), D4 -> (3, 15), Q16 -> (15, 3)
// Returns (-1, -1) for "pass" or "PASS".
func gtpToPos(vertex string, size int) (int, int, error) {
	vertex = strings.TrimSpace(strings.ToUpper(vertex))

	// Handle pass
	if vertex == "PASS" {
		return -1, -1, nil
	}

	// Handle resign
	if vertex == "RESIGN" {
		return -2, -2, nil
	}

	if len(vertex) < 2 {
		return 0, 0, fmt.Errorf("invalid vertex: %s", vertex)
	}

	// Parse column (A-T, no I)
	col := int(vertex[0] - 'A')
	if col < 0 || col > 19 {
		return 0, 0, fmt.Errorf("invalid column in vertex: %s", vertex)
	}
	if col > 7 {
		col-- // Account for skipped 'I'
	}

	// Parse row
	row, err := strconv.Atoi(vertex[1:])
	if err != nil {
		return 0, 0, fmt.Errorf("invalid row in vertex: %s", vertex)
	}

	// Convert row to Y coordinate (invert from bottom-up to top-down)
	y := size - row

	if col < 0 || col >= size || y < 0 || y >= size {
		return 0, 0, fmt.Errorf("vertex out of bounds: %s", vertex)
	}

	return col, y, nil
}

// colorToGTP converts a color (1=black, 2=white) to GTP color string.
func colorToGTP(color int) string {
	if color == 1 {
		return "black"
	}
	return "white"
}

// gtpToColor converts a GTP color string to color int (1=black, 2=white).
func gtpToColor(color string) int {
	color = strings.ToLower(strings.TrimSpace(color))
	if color == "black" || color == "b" {
		return 1
	}
	return 2
}

// oppositeColor returns the opposite color (1->2, 2->1).
func oppositeColor(color int) int {
	if color == 1 {
		return 2
	}
	return 1
}
