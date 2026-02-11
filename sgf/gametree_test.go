package sgf

import (
	"testing"
)

func TestNewGameTree(t *testing.T) {
	tree := NewGameTree()
	if tree.Root == nil {
		t.Fatal("root should not be nil")
	}
	if tree.Current != tree.Root {
		t.Fatal("current should be root")
	}
	if tree.Root.Move != "" {
		t.Fatalf("root move should be empty, got %q", tree.Root.Move)
	}
}

func TestGameTreeAddMove(t *testing.T) {
	tree := NewGameTree()
	node := tree.AddMove(";B[pd]")
	if node.Move != ";B[pd]" {
		t.Fatalf("expected ;B[pd], got %q", node.Move)
	}
	if tree.Current != node {
		t.Fatal("current should advance to new node")
	}
	if node.Parent != tree.Root {
		t.Fatal("parent should be root")
	}
	if len(tree.Root.Children) != 1 {
		t.Fatalf("root should have 1 child, got %d", len(tree.Root.Children))
	}
}

func TestAddMoveDedup(t *testing.T) {
	tree := NewGameTree()
	node1 := tree.AddMove(";B[pd]")
	tree.Back()
	node2 := tree.AddMove(";B[pd]") // same move, should navigate not create
	if node1 != node2 {
		t.Fatal("duplicate move should navigate to existing node, not create new one")
	}
	if len(tree.Root.Children) != 1 {
		t.Fatalf("root should still have 1 child, got %d", len(tree.Root.Children))
	}
}

func TestAddMoveBranching(t *testing.T) {
	tree := NewGameTree()
	tree.AddMove(";B[pd]")
	tree.Back()
	tree.AddMove(";B[dd]") // different move, should create branch
	if len(tree.Root.Children) != 2 {
		t.Fatalf("root should have 2 children, got %d", len(tree.Root.Children))
	}
	if tree.Root.Children[0].Move != ";B[pd]" {
		t.Fatalf("first child should be ;B[pd], got %q", tree.Root.Children[0].Move)
	}
	if tree.Root.Children[1].Move != ";B[dd]" {
		t.Fatalf("second child should be ;B[dd], got %q", tree.Root.Children[1].Move)
	}
}

func TestBack(t *testing.T) {
	tree := NewGameTree()
	// Back at root should return false
	if tree.Back() {
		t.Fatal("back at root should return false")
	}

	tree.AddMove(";B[pd]")
	if !tree.Back() {
		t.Fatal("back should return true")
	}
	if tree.Current != tree.Root {
		t.Fatal("should be back at root")
	}
}

func TestForward(t *testing.T) {
	tree := NewGameTree()
	// Forward with no children
	if tree.Forward(0) {
		t.Fatal("forward with no children should return false")
	}

	tree.AddMove(";B[pd]")
	tree.AddMove(";W[dp]")
	tree.Back()
	tree.Back()

	// Forward to first child
	if !tree.Forward(0) {
		t.Fatal("forward should return true")
	}
	if tree.Current.Move != ";B[pd]" {
		t.Fatalf("expected ;B[pd], got %q", tree.Current.Move)
	}

	// Forward again
	if !tree.Forward(0) {
		t.Fatal("forward should return true")
	}
	if tree.Current.Move != ";W[dp]" {
		t.Fatalf("expected ;W[dp], got %q", tree.Current.Move)
	}

	// Forward with out of bounds index
	if tree.Forward(1) {
		t.Fatal("forward with invalid index should return false")
	}
}

func TestVariationSwitching(t *testing.T) {
	tree := NewGameTree()
	tree.AddMove(";B[pd]")
	tree.Back()
	tree.AddMove(";B[dd]")
	tree.Back()
	tree.AddMove(";B[pp]")
	// Now root has 3 children, current is at ;B[pp] (index 2)

	// NextVariation should wrap to first
	if !tree.NextVariation() {
		t.Fatal("NextVariation should return true")
	}
	if tree.Current.Move != ";B[pd]" {
		t.Fatalf("expected ;B[pd], got %q", tree.Current.Move)
	}

	// NextVariation again
	tree.NextVariation()
	if tree.Current.Move != ";B[dd]" {
		t.Fatalf("expected ;B[dd], got %q", tree.Current.Move)
	}

	// PrevVariation
	tree.PrevVariation()
	if tree.Current.Move != ";B[pd]" {
		t.Fatalf("expected ;B[pd], got %q", tree.Current.Move)
	}

	// PrevVariation wraps to last
	tree.PrevVariation()
	if tree.Current.Move != ";B[pp]" {
		t.Fatalf("expected ;B[pp], got %q", tree.Current.Move)
	}
}

func TestVariationSwitchingAtRoot(t *testing.T) {
	tree := NewGameTree()
	if tree.NextVariation() {
		t.Fatal("NextVariation at root should return false")
	}
	if tree.PrevVariation() {
		t.Fatal("PrevVariation at root should return false")
	}
}

func TestVariationSwitchingSingleChild(t *testing.T) {
	tree := NewGameTree()
	tree.AddMove(";B[pd]")
	if tree.NextVariation() {
		t.Fatal("NextVariation with single sibling should return false")
	}
	if tree.PrevVariation() {
		t.Fatal("PrevVariation with single sibling should return false")
	}
}

func TestPathFromRoot(t *testing.T) {
	tree := NewGameTree()
	// Path at root should be empty
	path := tree.PathFromRoot()
	if len(path) != 0 {
		t.Fatalf("path at root should be empty, got %v", path)
	}

	tree.AddMove(";B[pd]")
	tree.AddMove(";W[dp]")
	tree.AddMove(";B[pp]")

	path = tree.PathFromRoot()
	expected := []string{";B[pd]", ";W[dp]", ";B[pp]"}
	if len(path) != len(expected) {
		t.Fatalf("path length should be %d, got %d", len(expected), len(path))
	}
	for i, m := range expected {
		if path[i] != m {
			t.Fatalf("path[%d] should be %q, got %q", i, m, path[i])
		}
	}
}

func TestNumVariations(t *testing.T) {
	tree := NewGameTree()
	if tree.NumVariations() != 0 {
		t.Fatalf("NumVariations at root should be 0, got %d", tree.NumVariations())
	}

	tree.AddMove(";B[pd]")
	tree.Back()
	tree.AddMove(";B[dd]")
	// Current is ;B[dd], parent (root) has 2 children
	if tree.NumVariations() != 2 {
		t.Fatalf("expected 2 variations, got %d", tree.NumVariations())
	}
}

func TestVariationIndex(t *testing.T) {
	tree := NewGameTree()
	if tree.VariationIndex() != -1 {
		t.Fatalf("VariationIndex at root should be -1, got %d", tree.VariationIndex())
	}

	tree.AddMove(";B[pd]")
	if tree.VariationIndex() != 0 {
		t.Fatalf("expected index 0, got %d", tree.VariationIndex())
	}

	tree.Back()
	tree.AddMove(";B[dd]")
	if tree.VariationIndex() != 1 {
		t.Fatalf("expected index 1, got %d", tree.VariationIndex())
	}
}

func TestHasChildren(t *testing.T) {
	tree := NewGameTree()
	if tree.HasChildren() {
		t.Fatal("root should have no children initially")
	}
	tree.AddMove(";B[pd]")
	tree.Back()
	if !tree.HasChildren() {
		t.Fatal("root should have children after AddMove")
	}
}

func TestDeepTree(t *testing.T) {
	tree := NewGameTree()
	// Build a 10-move main line
	for i := 0; i < 10; i++ {
		color := "B"
		if i%2 == 1 {
			color = "W"
		}
		tree.AddMove(";"+color+"[aa]")
	}
	path := tree.PathFromRoot()
	if len(path) != 10 {
		t.Fatalf("expected path length 10, got %d", len(path))
	}

	// Navigate all the way back
	for i := 0; i < 10; i++ {
		if !tree.Back() {
			t.Fatalf("back should succeed at step %d", i)
		}
	}
	if tree.Current != tree.Root {
		t.Fatal("should be back at root")
	}
	if tree.Back() {
		t.Fatal("back at root should return false")
	}
}
