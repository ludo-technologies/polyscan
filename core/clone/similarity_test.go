package clone

import (
	"math"
	"strings"
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

// --- Textual similarity (Type-1 gate) ---

// testStripLineComments is a simple line-comment stripper ("//" to end of
// line) standing in for a language adapter's CommentStripper.
func testStripLineComments(content string) string {
	var b strings.Builder
	for _, line := range strings.Split(content, "\n") {
		if idx := strings.Index(line, "//"); idx >= 0 {
			line = line[:idx]
		}
		b.WriteString(line)
		b.WriteString("\n")
	}
	return b.String()
}

func TestTextualSimilarityExactMatchAfterNormalization(t *testing.T) {
	ta := NewTextualSimilarityAnalyzer(testStripLineComments)

	f1 := &CodeFragment{ID: 1, Content: "const x = 1;\nreturn x; // result"}
	f2 := &CodeFragment{ID: 2, Content: "const x = 1;   \n\n  return x;"}

	if !ta.IsExactMatch(f1, f2) {
		t.Error("expected exact match after comment/whitespace normalization")
	}
	if sim := ta.ComputeSimilarity(f1, f2); sim != 1.0 {
		t.Errorf("expected similarity 1.0, got %f", sim)
	}
}

func TestTextualSimilarityNearMatchIsNotExact(t *testing.T) {
	ta := NewTextualSimilarityAnalyzer(nil)

	f1 := &CodeFragment{ID: 1, Content: "const alpha = 1;"}
	f2 := &CodeFragment{ID: 2, Content: "const alphb = 1;"}

	if ta.IsExactMatch(f1, f2) {
		t.Error("near matches must not be exact matches")
	}
	sim := ta.ComputeSimilarity(f1, f2)
	if sim <= 0.8 || sim >= 1.0 {
		t.Errorf("expected Levenshtein-based near-match similarity in (0.8, 1.0), got %f", sim)
	}
}

func TestTextualSimilarityEmptyContentNeverExact(t *testing.T) {
	ta := NewTextualSimilarityAnalyzer(nil)

	f1 := &CodeFragment{ID: 1, Content: ""}
	f2 := &CodeFragment{ID: 2, Content: ""}

	if ta.IsExactMatch(f1, f2) {
		t.Error("fragments without source content must not be exact matches")
	}
	if sim := ta.ComputeSimilarity(f1, f2); sim != 1.0 {
		t.Errorf("both-empty similarity should be 1.0, got %f", sim)
	}
}

func TestTextualSimilarityPreservesStringWhitespace(t *testing.T) {
	ta := NewTextualSimilarityAnalyzer(nil)

	f1 := &CodeFragment{ID: 1, Content: `x = "a  b"`}
	f2 := &CodeFragment{ID: 2, Content: `x = "a b"`}

	if ta.IsExactMatch(f1, f2) {
		t.Error("whitespace inside string literals must be preserved")
	}
}

func TestHashFragmentContent(t *testing.T) {
	ta := NewTextualSimilarityAnalyzer(testStripLineComments)

	h1 := ta.HashFragmentContent("const x = 1; // one")
	h2 := ta.HashFragmentContent("const  x =  1;")
	if h1 == "" || h1 != h2 {
		t.Errorf("normalized-equal content must hash equal: %q vs %q", h1, h2)
	}

	if h := ta.HashFragmentContent(""); h != "" {
		t.Errorf("empty content must hash to empty string, got %q", h)
	}
	if h := ta.HashFragmentContent("// only a comment"); h != "" {
		t.Errorf("comment-only content must hash to empty string, got %q", h)
	}
}

func TestTextualSimilarityName(t *testing.T) {
	ta := NewTextualSimilarityAnalyzer(nil)
	if ta.Name() != "textual" {
		t.Errorf("Name() = %q, want textual", ta.Name())
	}
	var _ SimilarityAnalyzer = ta
}

// --- Syntactic similarity (Type-2 gate) ---

func TestSyntacticSimilarityUsesPrecomputedFeatures(t *testing.T) {
	sa := NewSyntacticSimilarityAnalyzer()

	f1 := &CodeFragment{ID: 1, Features: []string{"a", "b", "c"}}
	f2 := &CodeFragment{ID: 2, Features: []string{"a", "b", "c"}}

	if sim := sa.ComputeSimilarity(f1, f2); sim != 1.0 {
		t.Errorf("identical feature sets: got %f, want 1.0", sim)
	}

	f3 := &CodeFragment{ID: 3, Features: []string{"x", "y", "z"}}
	if sim := sa.ComputeSimilarity(f1, f3); sim != 0.0 {
		t.Errorf("disjoint feature sets: got %f, want 0.0", sim)
	}
}

func TestSyntacticSimilarityFromASTNodes(t *testing.T) {
	sa := NewSyntacticSimilarityAnalyzer()

	tree1 := apted.NewTreeNode(0, "FunctionDef")
	tree1.AddChild(apted.NewTreeNode(1, "If"))
	tree1.AddChild(apted.NewTreeNode(2, "Return"))
	tree2 := apted.NewTreeNode(0, "FunctionDef")
	tree2.AddChild(apted.NewTreeNode(1, "If"))
	tree2.AddChild(apted.NewTreeNode(2, "Return"))

	f1 := &CodeFragment{ID: 1, ASTNode: tree1}
	f2 := &CodeFragment{ID: 2, ASTNode: tree2}

	if sim := sa.ComputeSimilarity(f1, f2); sim != 1.0 {
		t.Errorf("identical trees: got %f, want 1.0", sim)
	}
	if sim := sa.ComputeSimilarity(f1, nil); sim != 0.0 {
		t.Errorf("nil fragment: got %f, want 0.0", sim)
	}
	if d := sa.ComputeDistance(f1, f2); d != 0.0 {
		t.Errorf("identical trees distance: got %f, want 0.0", d)
	}
	if sa.Name() != "syntactic" {
		t.Errorf("Name() = %q, want syntactic", sa.Name())
	}
}

// --- Jaccard ---

func TestJaccardSimilarity(t *testing.T) {
	tests := []struct {
		name string
		a, b []string
		want float64
	}{
		{"both empty", nil, nil, 1.0},
		{"one empty", []string{"a"}, nil, 0.0},
		{"identical", []string{"a", "b"}, []string{"a", "b"}, 1.0},
		{"disjoint", []string{"a", "b"}, []string{"c", "d"}, 0.0},
		{"half overlap", []string{"a", "b"}, []string{"b", "c"}, 1.0 / 3.0},
		{"duplicates collapse", []string{"a", "a", "b"}, []string{"a", "b", "b"}, 1.0},
		{"unsorted input", []string{"b", "a"}, []string{"a", "b"}, 1.0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := JaccardSimilarity(tt.a, tt.b)
			if math.Abs(got-tt.want) > 1e-9 {
				t.Errorf("JaccardSimilarity(%v, %v) = %f, want %f", tt.a, tt.b, got, tt.want)
			}
		})
	}
}
