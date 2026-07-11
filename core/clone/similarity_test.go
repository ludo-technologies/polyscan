package clone

import (
	"testing"

	"github.com/ludo-technologies/polyscan/core/apted"
)

func TestStructuralAnalyzerNilFragments(t *testing.T) {
	sa := NewStructuralAnalyzer(apted.NewDefaultCostModel(), apted.NormalizeByMax)
	f := &CodeFragment{ID: 1, ASTNode: apted.NewTreeNode(0, "A")}

	if sim := sa.ComputeSimilarity(nil, f); sim != 0.0 {
		t.Errorf("nil f1: got %f, want 0.0", sim)
	}
	if sim := sa.ComputeSimilarity(f, nil); sim != 0.0 {
		t.Errorf("nil f2: got %f, want 0.0", sim)
	}
	if sim := sa.ComputeSimilarity(nil, nil); sim != 0.0 {
		t.Errorf("both nil: got %f, want 0.0", sim)
	}
}

func TestStructuralAnalyzerNilASTNodes(t *testing.T) {
	sa := NewStructuralAnalyzer(apted.NewDefaultCostModel(), apted.NormalizeByMax)
	f1 := &CodeFragment{ID: 1}
	f2 := &CodeFragment{ID: 2, ASTNode: apted.NewTreeNode(0, "A")}

	if sim := sa.ComputeSimilarity(f1, f2); sim != 0.0 {
		t.Errorf("nil ASTNode f1: got %f, want 0.0", sim)
	}
	if sim := sa.ComputeSimilarity(f2, f1); sim != 0.0 {
		t.Errorf("nil ASTNode f2: got %f, want 0.0", sim)
	}
}

func TestStructuralAnalyzerIdenticalTrees(t *testing.T) {
	sa := NewStructuralAnalyzer(apted.NewDefaultCostModel(), apted.NormalizeByMax)

	tree1 := apted.NewTreeNode(0, "FunctionDef")
	tree1.AddChild(apted.NewTreeNode(1, "If"))
	tree1.AddChild(apted.NewTreeNode(2, "Return"))

	tree2 := apted.NewTreeNode(0, "FunctionDef")
	tree2.AddChild(apted.NewTreeNode(1, "If"))
	tree2.AddChild(apted.NewTreeNode(2, "Return"))

	f1 := &CodeFragment{ID: 1, ASTNode: tree1}
	f2 := &CodeFragment{ID: 2, ASTNode: tree2}

	sim := sa.ComputeSimilarity(f1, f2)
	if sim != 1.0 {
		t.Errorf("identical trees: got %f, want 1.0", sim)
	}
}

func TestStructuralAnalyzerDifferentTrees(t *testing.T) {
	sa := NewStructuralAnalyzer(apted.NewDefaultCostModel(), apted.NormalizeByMax)

	tree1 := apted.NewTreeNode(0, "FunctionDef")
	tree1.AddChild(apted.NewTreeNode(1, "If"))

	tree2 := apted.NewTreeNode(0, "FunctionDef")
	tree2.AddChild(apted.NewTreeNode(1, "For"))
	tree2.AddChild(apted.NewTreeNode(2, "Return"))

	f1 := &CodeFragment{ID: 1, ASTNode: tree1}
	f2 := &CodeFragment{ID: 2, ASTNode: tree2}

	sim := sa.ComputeSimilarity(f1, f2)
	if sim >= 1.0 {
		t.Errorf("different trees: got %f, want < 1.0", sim)
	}
	if sim <= 0.0 {
		t.Errorf("different trees: got %f, want > 0.0", sim)
	}
}

func TestStructuralAnalyzerName(t *testing.T) {
	sa := NewStructuralAnalyzer(apted.NewDefaultCostModel(), apted.NormalizeByMax)
	if name := sa.Name(); name != "structural" {
		t.Errorf("Name() = %q, want %q", name, "structural")
	}
}

func TestStructuralAnalyzerImplementsInterface(t *testing.T) {
	var _ SimilarityAnalyzer = &StructuralAnalyzer{}
}
