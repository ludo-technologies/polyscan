package clone

import (
	"strings"
	"testing"

	"github.com/ludo-technologies/codescan-core/apted"
)

func TestNewASTFeatureExtractor(t *testing.T) {
	ext := NewASTFeatureExtractor()
	if ext.maxSubtreeHeight != 3 {
		t.Errorf("maxSubtreeHeight = %d, want 3", ext.maxSubtreeHeight)
	}
	if ext.kGramSize != 4 {
		t.Errorf("kGramSize = %d, want 4", ext.kGramSize)
	}
	if !ext.includeTypes {
		t.Error("Expected includeTypes=true")
	}
	if ext.includeLiterals {
		t.Error("Expected includeLiterals=false")
	}
}

func TestWithOptions(t *testing.T) {
	ext := NewASTFeatureExtractor().WithOptions(5, 6, false, true)
	if ext.maxSubtreeHeight != 5 {
		t.Errorf("maxSubtreeHeight = %d, want 5", ext.maxSubtreeHeight)
	}
	if ext.kGramSize != 6 {
		t.Errorf("kGramSize = %d, want 6", ext.kGramSize)
	}
	if ext.includeTypes {
		t.Error("Expected includeTypes=false")
	}
	if !ext.includeLiterals {
		t.Error("Expected includeLiterals=true")
	}

	// Zero/negative values should not change defaults
	ext2 := NewASTFeatureExtractor().WithOptions(0, -1, true, false)
	if ext2.maxSubtreeHeight != 3 {
		t.Errorf("maxSubtreeHeight should stay default, got %d", ext2.maxSubtreeHeight)
	}
	if ext2.kGramSize != 4 {
		t.Errorf("kGramSize should stay default, got %d", ext2.kGramSize)
	}
}

func TestWithPatterns(t *testing.T) {
	ext := NewASTFeatureExtractor().WithPatterns([]string{"If", "For"})
	if len(ext.PatternNames) != 2 {
		t.Errorf("PatternNames length = %d, want 2", len(ext.PatternNames))
	}
}

func TestExtractFeatures_Nil(t *testing.T) {
	ext := NewASTFeatureExtractor()
	feats, err := ext.ExtractFeatures(nil)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	if len(feats) != 0 {
		t.Errorf("Expected 0 features for nil, got %d", len(feats))
	}
}

func TestExtractFeatures_SingleNode(t *testing.T) {
	ext := NewASTFeatureExtractor()
	node := apted.NewTreeNode(0, "FunctionDef")
	feats, err := ext.ExtractFeatures(node)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	if len(feats) == 0 {
		t.Error("Expected some features")
	}

	// Should contain subtree hash and type
	hasSub := false
	hasType := false
	for _, f := range feats {
		if strings.HasPrefix(f, "sub:") {
			hasSub = true
		}
		if strings.HasPrefix(f, "type:") {
			hasType = true
		}
	}
	if !hasSub {
		t.Error("Expected subtree hash features")
	}
	if !hasType {
		t.Error("Expected type features")
	}
}

func TestExtractFeatures_WithPatterns(t *testing.T) {
	ext := NewASTFeatureExtractor().WithPatterns([]string{"If", "For", "Return"})

	root := apted.NewTreeNode(0, "FunctionDef")
	root.AddChild(apted.NewTreeNode(1, "If"))
	root.AddChild(apted.NewTreeNode(2, "Return"))
	// No "For" node

	feats, err := ext.ExtractFeatures(root)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	hasPatternIf := false
	hasPatternReturn := false
	hasPatternFor := false
	for _, f := range feats {
		if f == "pattern:If" {
			hasPatternIf = true
		}
		if f == "pattern:Return" {
			hasPatternReturn = true
		}
		if f == "pattern:For" {
			hasPatternFor = true
		}
	}
	if !hasPatternIf {
		t.Error("Expected pattern:If")
	}
	if !hasPatternReturn {
		t.Error("Expected pattern:Return")
	}
	if hasPatternFor {
		t.Error("Did not expect pattern:For (no For node in tree)")
	}
}

func TestExtractFeatures_NoPatterns(t *testing.T) {
	ext := NewASTFeatureExtractor() // no PatternNames set

	root := apted.NewTreeNode(0, "If")
	feats, err := ext.ExtractFeatures(root)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	for _, f := range feats {
		if strings.HasPrefix(f, "pattern:") {
			t.Errorf("Expected no pattern features, found: %s", f)
		}
	}
}

func TestExtractFeatures_Determinism(t *testing.T) {
	ext := NewASTFeatureExtractor().WithPatterns([]string{"If", "Return"})

	root := apted.NewTreeNode(0, "FunctionDef")
	root.AddChild(apted.NewTreeNode(1, "If"))
	root.AddChild(apted.NewTreeNode(2, "Return"))

	feats1, _ := ext.ExtractFeatures(root)
	feats2, _ := ext.ExtractFeatures(root)

	if len(feats1) != len(feats2) {
		t.Fatalf("Non-deterministic feature count: %d vs %d", len(feats1), len(feats2))
	}
	for i := range feats1 {
		if feats1[i] != feats2[i] {
			t.Errorf("Non-deterministic at %d: %q vs %q", i, feats1[i], feats2[i])
			break
		}
	}
}

func TestExtractSubtreeHashes_Nil(t *testing.T) {
	ext := NewASTFeatureExtractor()
	hashes, err := ext.ExtractSubtreeHashes(nil, 3)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	if len(hashes) != 0 {
		t.Errorf("Expected 0 hashes for nil, got %d", len(hashes))
	}
}

func TestExtractSubtreeHashes_SingleNode(t *testing.T) {
	ext := NewASTFeatureExtractor()
	node := apted.NewTreeNode(0, "A")
	hashes, err := ext.ExtractSubtreeHashes(node, 3)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	if len(hashes) != 1 {
		t.Errorf("Expected 1 hash for leaf, got %d", len(hashes))
	}
	if !strings.HasPrefix(hashes[0], "sub:0:") {
		t.Errorf("Expected sub:0: prefix, got %q", hashes[0])
	}
}

func TestExtractSubtreeHashes_OrderSensitivity(t *testing.T) {
	ext := NewASTFeatureExtractor()

	// Tree1: A -> [B, C]
	t1 := apted.NewTreeNode(0, "A")
	t1.AddChild(apted.NewTreeNode(1, "B"))
	t1.AddChild(apted.NewTreeNode(2, "C"))

	// Tree2: A -> [C, B] (reversed children)
	t2 := apted.NewTreeNode(0, "A")
	t2.AddChild(apted.NewTreeNode(1, "C"))
	t2.AddChild(apted.NewTreeNode(2, "B"))

	h1, err := ext.ExtractSubtreeHashes(t1, 3)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	h2, err := ext.ExtractSubtreeHashes(t2, 3)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	// Root-level hashes should differ due to different child order
	// Find the height=1 hash in each
	var root1, root2 string
	for _, h := range h1 {
		if strings.HasPrefix(h, "sub:1:") {
			root1 = h
		}
	}
	for _, h := range h2 {
		if strings.HasPrefix(h, "sub:1:") {
			root2 = h
		}
	}
	if root1 == root2 {
		t.Error("Expected different hashes for different child order")
	}
}

func TestExtractNodeSequences_Nil(t *testing.T) {
	ext := NewASTFeatureExtractor()
	seqs, err := ext.ExtractNodeSequences(nil, 3)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	if len(seqs) != 0 {
		t.Errorf("Expected 0 sequences for nil, got %d", len(seqs))
	}
}

func TestExtractNodeSequences_KTooLarge(t *testing.T) {
	ext := NewASTFeatureExtractor()
	node := apted.NewTreeNode(0, "A")
	seqs, err := ext.ExtractNodeSequences(node, 5) // Only 1 label, k=5
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	if len(seqs) != 0 {
		t.Errorf("Expected 0 sequences when k > labels, got %d", len(seqs))
	}
}

func TestExtractNodeSequences_KEqualsOne(t *testing.T) {
	ext := NewASTFeatureExtractor()
	node := apted.NewTreeNode(0, "A")
	seqs, err := ext.ExtractNodeSequences(node, 1)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	if len(seqs) != 0 {
		t.Errorf("Expected 0 for k=1 (k<=1 returns empty), got %d", len(seqs))
	}
}

func TestExtractNodeSequences_Valid(t *testing.T) {
	ext := NewASTFeatureExtractor()
	// Pre-order: A, B, C
	root := apted.NewTreeNode(0, "A")
	root.AddChild(apted.NewTreeNode(1, "B"))
	root.AddChild(apted.NewTreeNode(2, "C"))

	seqs, err := ext.ExtractNodeSequences(root, 2)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	// Expected: ["A:B", "B:C"]
	if len(seqs) != 2 {
		t.Fatalf("Expected 2 sequences, got %d", len(seqs))
	}
	if seqs[0] != "A:B" {
		t.Errorf("Seq[0] = %q, want %q", seqs[0], "A:B")
	}
	if seqs[1] != "B:C" {
		t.Errorf("Seq[1] = %q, want %q", seqs[1], "B:C")
	}
}

func TestExtractNodeSequences_StripsPayload(t *testing.T) {
	ext := NewASTFeatureExtractor() // includeLiterals=false

	root := apted.NewTreeNode(0, "Name(foo)")
	root.AddChild(apted.NewTreeNode(1, "Constant(42)"))

	seqs, err := ext.ExtractNodeSequences(root, 2)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	if len(seqs) != 1 {
		t.Fatalf("Expected 1 sequence, got %d", len(seqs))
	}
	// Should strip payload
	if seqs[0] != "Name:Constant" {
		t.Errorf("Seq[0] = %q, want %q", seqs[0], "Name:Constant")
	}
}

func TestBinCount(t *testing.T) {
	tests := []struct {
		input    int
		expected string
	}{
		{0, "1"},
		{1, "1"},
		{2, "2-3"},
		{3, "2-3"},
		{4, "4-7"},
		{7, "4-7"},
		{8, "8-15"},
		{15, "8-15"},
		{16, "16+"},
		{100, "16+"},
	}

	for _, tt := range tests {
		if got := binCount(tt.input); got != tt.expected {
			t.Errorf("binCount(%d) = %q, want %q", tt.input, got, tt.expected)
		}
	}
}

func TestFeatureExtractorInterface(t *testing.T) {
	// Verify ASTFeatureExtractor implements FeatureExtractor
	var _ FeatureExtractor = NewASTFeatureExtractor()
}

func BenchmarkExtractFeatures(b *testing.B) {
	ext := NewASTFeatureExtractor().WithPatterns([]string{"If", "For", "Return", "Call"})

	root := apted.NewTreeNode(0, "FunctionDef")
	for i := 1; i <= 20; i++ {
		child := apted.NewTreeNode(i, "If")
		child.AddChild(apted.NewTreeNode(i*100, "Call"))
		child.AddChild(apted.NewTreeNode(i*100+1, "Return"))
		root.AddChild(child)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		ext.ExtractFeatures(root)
	}
}
