package sgf

// GameNode represents a single position in the game tree.
type GameNode struct {
	Move     string      // ";B[pd]" or "" for root
	Parent   *GameNode
	Children []*GameNode // First child = main line
}

// GameTree tracks an in-memory tree of moves for planning mode exploration.
type GameTree struct {
	Root    *GameNode
	Current *GameNode
}

// NewGameTree creates a new game tree with an empty root node.
func NewGameTree() *GameTree {
	root := &GameNode{}
	return &GameTree{Root: root, Current: root}
}

// AddMove adds a child move to the current node and advances to it.
// If a child with the same move already exists, navigates to it instead of creating a duplicate.
func (t *GameTree) AddMove(move string) *GameNode {
	// Check for existing child with same move
	for _, child := range t.Current.Children {
		if child.Move == move {
			t.Current = child
			return child
		}
	}
	node := &GameNode{
		Move:   move,
		Parent: t.Current,
	}
	t.Current.Children = append(t.Current.Children, node)
	t.Current = node
	return node
}

// Back moves current to its parent. Returns false if already at root.
func (t *GameTree) Back() bool {
	if t.Current == t.Root {
		return false
	}
	t.Current = t.Current.Parent
	return true
}

// Forward moves current to children[idx]. Returns false if no such child.
func (t *GameTree) Forward(idx int) bool {
	if idx < 0 || idx >= len(t.Current.Children) {
		return false
	}
	t.Current = t.Current.Children[idx]
	return true
}

// NextVariation switches to the next sibling (among parent's children). Wraps around.
func (t *GameTree) NextVariation() bool {
	if t.Current.Parent == nil {
		return false
	}
	siblings := t.Current.Parent.Children
	if len(siblings) < 2 {
		return false
	}
	idx := t.childIndex()
	t.Current = siblings[(idx+1)%len(siblings)]
	return true
}

// PrevVariation switches to the previous sibling (among parent's children). Wraps around.
func (t *GameTree) PrevVariation() bool {
	if t.Current.Parent == nil {
		return false
	}
	siblings := t.Current.Parent.Children
	if len(siblings) < 2 {
		return false
	}
	idx := t.childIndex()
	t.Current = siblings[(idx-1+len(siblings))%len(siblings)]
	return true
}

// PathFromRoot returns the slice of move strings from root to current (excluding root's empty move).
func (t *GameTree) PathFromRoot() []string {
	var path []string
	node := t.Current
	for node != t.Root {
		path = append(path, node.Move)
		node = node.Parent
	}
	// Reverse
	for i, j := 0, len(path)-1; i < j; i, j = i+1, j-1 {
		path[i], path[j] = path[j], path[i]
	}
	return path
}

// NumVariations returns the number of siblings at the current node's level.
// Returns 0 if at root.
func (t *GameTree) NumVariations() int {
	if t.Current.Parent == nil {
		return 0
	}
	return len(t.Current.Parent.Children)
}

// VariationIndex returns which child of parent the current node is (0-based).
// Returns -1 if at root.
func (t *GameTree) VariationIndex() int {
	if t.Current.Parent == nil {
		return -1
	}
	return t.childIndex()
}

// HasChildren returns true if the current node has any children.
func (t *GameTree) HasChildren() bool {
	return len(t.Current.Children) > 0
}

// childIndex returns the index of current among its parent's children.
func (t *GameTree) childIndex() int {
	if t.Current.Parent == nil {
		return -1
	}
	for i, child := range t.Current.Parent.Children {
		if child == t.Current {
			return i
		}
	}
	return -1
}
