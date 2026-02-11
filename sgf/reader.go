package sgf

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

// GameInfo holds metadata parsed from an SGF file header.
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

// ParseHeader reads an SGF file and extracts metadata from the root node.
func ParseHeader(filePath string) (*GameInfo, error) {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil, err
	}

	content := string(data)
	props := parseProperties(content)

	boardSize := 19
	if v, ok := props["SZ"]; ok {
		if n, err := strconv.Atoi(v); err == nil {
			boardSize = n
		}
	}

	komi := 0.0
	if v, ok := props["KM"]; ok {
		if f, err := strconv.ParseFloat(v, 64); err == nil {
			komi = f
		}
	}

	info := &GameInfo{
		FilePath:    filePath,
		FileName:    filepath.Base(filePath),
		BoardSize:   boardSize,
		Komi:        komi,
		PlayerBlack: props["PB"],
		PlayerWhite: props["PW"],
		Date:        props["DT"],
		Result:      props["RE"],
		MoveCount:   countMoves(content),
	}

	return info, nil
}

// ReplayToEnd parses an SGF file and replays all moves to produce the final board position.
// Returns the board (board[y][x], 0=empty, 1=black, 2=white), the move count, and any error.
func ReplayToEnd(filePath string) ([][]int, int, error) {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil, 0, err
	}

	content := string(data)
	props := parseProperties(content)

	boardSize := 19
	if v, ok := props["SZ"]; ok {
		if n, err := strconv.Atoi(v); err == nil {
			boardSize = n
		}
	}

	board := MakeBoard(boardSize)
	moveCount := 0

	// Apply setup positions (AB/AW)
	applySetup(content, board, boardSize)

	// Parse and apply each move
	nodes := parseNodes(content)
	for _, node := range nodes {
		color, x, y, ok := parseMoveNode(node)
		if !ok {
			continue
		}
		moveCount++
		if x == -1 && y == -1 {
			continue // pass
		}
		if x < 0 || x >= boardSize || y < 0 || y >= boardSize {
			continue
		}
		board[y][x] = color
		RemoveCaptures(board, boardSize, x, y, color)
	}

	return board, moveCount, nil
}

// MakeBoard creates an empty boardSize x boardSize board.
func MakeBoard(size int) [][]int {
	board := make([][]int, size)
	for i := range board {
		board[i] = make([]int, size)
	}
	return board
}

// parseProperties extracts KEY[value] pairs from the root node of an SGF string.
func parseProperties(content string) map[string]string {
	props := make(map[string]string)

	// Find the root node: starts after "(;"
	start := strings.Index(content, "(;")
	if start == -1 {
		return props
	}
	start += 2 // skip "(;"

	// Root node ends at the next ";" or ")"
	end := len(content)
	for i := start; i < len(content); i++ {
		if content[i] == ';' || content[i] == ')' {
			end = i
			break
		}
	}

	root := content[start:end]
	extractProps(root, props)
	return props
}

// extractProps parses KEY[value] pairs from a node string into the map.
func extractProps(node string, props map[string]string) {
	i := 0
	for i < len(node) {
		// Skip whitespace
		for i < len(node) && (node[i] == ' ' || node[i] == '\n' || node[i] == '\r' || node[i] == '\t') {
			i++
		}
		if i >= len(node) {
			break
		}

		// Read property identifier (uppercase letters)
		keyStart := i
		for i < len(node) && node[i] >= 'A' && node[i] <= 'Z' {
			i++
		}
		if i == keyStart {
			i++
			continue
		}
		key := node[keyStart:i]

		// Read all property values (e.g., AB[aa][bb][cc])
		for i < len(node) && node[i] == '[' {
			i++ // skip '['
			valStart := i
			for i < len(node) && node[i] != ']' {
				if node[i] == '\\' && i+1 < len(node) {
					i++ // skip escaped char
				}
				i++
			}
			val := node[valStart:i]
			if i < len(node) {
				i++ // skip ']'
			}
			props[key] = val // last value wins for simple props
		}
	}
}

// countMoves counts the number of move nodes (;B[...] or ;W[...]) in the SGF.
func countMoves(content string) int {
	count := 0
	i := 0
	for i < len(content) {
		if content[i] == ';' && i+1 < len(content) {
			next := content[i+1]
			if (next == 'B' || next == 'W') && i+2 < len(content) && content[i+2] == '[' {
				count++
			}
		}
		i++
	}
	return count
}

// parseNodes returns all node strings after the root node.
func parseNodes(content string) []string {
	var nodes []string

	// Find first ";" after "(;"
	start := strings.Index(content, "(;")
	if start == -1 {
		return nodes
	}
	start += 2

	// Skip root node to find subsequent ";"
	i := start
	for i < len(content) {
		if content[i] == ';' {
			break
		}
		if content[i] == '[' {
			// Skip value
			i++
			for i < len(content) && content[i] != ']' {
				if content[i] == '\\' && i+1 < len(content) {
					i++
				}
				i++
			}
		}
		i++
	}

	// Now parse subsequent nodes
	for i < len(content) {
		if content[i] == ';' {
			nodeStart := i
			i++
			// Read until next ';' or ')'
			for i < len(content) && content[i] != ';' && content[i] != ')' {
				if content[i] == '[' {
					i++
					for i < len(content) && content[i] != ']' {
						if content[i] == '\\' && i+1 < len(content) {
							i++
						}
						i++
					}
				}
				i++
			}
			nodes = append(nodes, content[nodeStart:i])
		} else {
			i++
		}
	}

	return nodes
}

// parseMoveNode extracts color and coordinates from a move node like ";B[pd]".
// Returns color (1=black, 2=white), x, y, and whether it's a valid move node.
// Pass moves return x=-1, y=-1.
func parseMoveNode(node string) (color, x, y int, ok bool) {
	node = strings.TrimSpace(node)
	if len(node) < 2 || node[0] != ';' {
		return 0, 0, 0, false
	}

	ch := node[1]
	if ch != 'B' && ch != 'W' {
		return 0, 0, 0, false
	}

	color = 1
	if ch == 'W' {
		color = 2
	}

	// Find the value in brackets
	bracketStart := strings.Index(node, "[")
	bracketEnd := strings.Index(node, "]")
	if bracketStart == -1 || bracketEnd == -1 || bracketEnd <= bracketStart {
		return 0, 0, 0, false
	}

	coord := node[bracketStart+1 : bracketEnd]
	if coord == "" {
		// Pass
		return color, -1, -1, true
	}

	if len(coord) != 2 {
		return 0, 0, 0, false
	}

	x = int(coord[0] - 'a')
	y = int(coord[1] - 'a')
	return color, x, y, true
}

// applySetup applies AB[]/AW[] setup properties from the SGF content.
func applySetup(content string, board [][]int, boardSize int) {
	// Find setup node (second node with AB/AW)
	// It could also be in the root node or a subsequent node
	i := strings.Index(content, "(;")
	if i == -1 {
		return
	}

	// Scan through all nodes looking for AB/AW
	for i < len(content) {
		if content[i] == 'A' && i+1 < len(content) && (content[i+1] == 'B' || content[i+1] == 'W') {
			color := 1
			if content[i+1] == 'W' {
				color = 2
			}
			i += 2

			// Read all coordinate values
			for i < len(content) && content[i] == '[' {
				i++ // skip '['
				if i+1 < len(content) && content[i+1] != ']' {
					coordStr := ""
					start := i
					for i < len(content) && content[i] != ']' {
						i++
					}
					coordStr = content[start:i]
					if len(coordStr) == 2 {
						x := int(coordStr[0] - 'a')
						y := int(coordStr[1] - 'a')
						if x >= 0 && x < boardSize && y >= 0 && y < boardSize {
							board[y][x] = color
						}
					}
				}
				if i < len(content) {
					i++ // skip ']'
				}
			}
		} else {
			i++
		}
	}
}

// RemoveCaptures checks and removes any opponent groups adjacent to (x, y) that have zero liberties.
func RemoveCaptures(board [][]int, size, x, y, color int) {
	opponent := 1
	if color == 1 {
		opponent = 2
	}

	// Check all four neighbors
	for _, d := range [][2]int{{0, -1}, {0, 1}, {-1, 0}, {1, 0}} {
		nx, ny := x+d[0], y+d[1]
		if nx < 0 || nx >= size || ny < 0 || ny >= size {
			continue
		}
		if board[ny][nx] == opponent {
			if !hasLiberties(board, size, nx, ny, opponent) {
				removeGroup(board, size, nx, ny, opponent)
			}
		}
	}
}

// HasLiberty checks if the group at (x, y) has any liberties using flood fill.
// Exported for use by planning mode's suicide detection.
func HasLiberty(board [][]int, size, x, y, color int) bool {
	return hasLiberties(board, size, x, y, color)
}

// hasLiberties checks if the group at (x, y) has any liberties using flood fill.
func hasLiberties(board [][]int, size, x, y, color int) bool {
	visited := make([][]bool, size)
	for i := range visited {
		visited[i] = make([]bool, size)
	}
	return hasLibertiesDFS(board, visited, size, x, y, color)
}

func hasLibertiesDFS(board [][]int, visited [][]bool, size, x, y, color int) bool {
	if x < 0 || x >= size || y < 0 || y >= size {
		return false
	}
	if visited[y][x] {
		return false
	}
	if board[y][x] == 0 {
		return true // found a liberty
	}
	if board[y][x] != color {
		return false
	}

	visited[y][x] = true
	for _, d := range [][2]int{{0, -1}, {0, 1}, {-1, 0}, {1, 0}} {
		if hasLibertiesDFS(board, visited, size, x+d[0], y+d[1], color) {
			return true
		}
	}
	return false
}

// removeGroup removes all stones in the group at (x, y) of the given color.
func removeGroup(board [][]int, size, x, y, color int) {
	if x < 0 || x >= size || y < 0 || y >= size {
		return
	}
	if board[y][x] != color {
		return
	}
	board[y][x] = 0
	for _, d := range [][2]int{{0, -1}, {0, 1}, {-1, 0}, {1, 0}} {
		removeGroup(board, size, x+d[0], y+d[1], color)
	}
}

// ParseMovesForRecord parses an SGF file and returns moves in the format used by GameRecord.moves
// (e.g., ";B[pd]", ";W[]" for passes).
func ParseMovesForRecord(filePath string) ([]string, error) {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil, err
	}

	content := string(data)
	nodes := parseNodes(content)

	var moves []string
	for _, node := range nodes {
		color, x, y, ok := parseMoveNode(node)
		if !ok {
			continue
		}
		colorChar := "B"
		if color == 2 {
			colorChar = "W"
		}
		if x == -1 && y == -1 {
			moves = append(moves, fmt.Sprintf(";%s[]", colorChar))
		} else {
			moves = append(moves, fmt.Sprintf(";%s[%s]", colorChar, string(rune('a'+x))+string(rune('a'+y))))
		}
	}

	return moves, nil
}

// ParseSetupPositions parses AB[]/AW[] setup positions from an SGF file.
// Returns black coords and white coords in SGF letter-pair format (e.g., "dd", "pp").
func ParseSetupPositions(filePath string) ([]string, []string, error) {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil, nil, err
	}

	content := string(data)
	var blacks, whites []string

	i := strings.Index(content, "(;")
	if i == -1 {
		return blacks, whites, nil
	}

	for i < len(content) {
		if content[i] == 'A' && i+1 < len(content) && (content[i+1] == 'B' || content[i+1] == 'W') {
			isBlack := content[i+1] == 'B'
			i += 2

			for i < len(content) && content[i] == '[' {
				i++ // skip '['
				start := i
				for i < len(content) && content[i] != ']' {
					i++
				}
				coord := content[start:i]
				if i < len(content) {
					i++ // skip ']'
				}
				if len(coord) == 2 {
					if isBlack {
						blacks = append(blacks, coord)
					} else {
						whites = append(whites, coord)
					}
				}
			}
		} else {
			i++
		}
	}

	return blacks, whites, nil
}

// ParseMovesAsEntries returns all moves as (color, x, y) triples.
func ParseMovesAsEntries(filePath string) ([][3]int, error) {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil, err
	}
	nodes := parseNodes(string(data))
	var result [][3]int
	for _, node := range nodes {
		color, x, y, ok := parseMoveNode(node)
		if !ok {
			continue
		}
		result = append(result, [3]int{color, x, y})
	}
	return result, nil
}

// ListGames scans a directory for .sgf files and returns their parsed headers,
// sorted newest-first (by filename, which contains timestamps).
func ListGames(dir string) ([]GameInfo, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("read history dir: %w", err)
	}

	var games []GameInfo
	for i := len(entries) - 1; i >= 0; i-- {
		e := entries[i]
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".sgf") {
			continue
		}
		path := filepath.Join(dir, e.Name())
		info, err := ParseHeader(path)
		if err != nil {
			continue
		}
		games = append(games, *info)
	}

	return games, nil
}
