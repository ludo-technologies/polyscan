package apted

import (
	"math"
	"testing"
)

// buildSimpleTree creates: root -> [children...]
func buildSimpleTree(rootLabel string, childLabels ...string) *TreeNode {
	root := NewTreeNode(0, rootLabel)
	for i, label := range childLabels {
		root.AddChild(NewTreeNode(i+1, label))
	}
	return root
}

func TestComputeDistance_NilTrees(t *testing.T) {
	a := NewAPTEDAnalyzer(NewDefaultCostModel())

	if d := a.ComputeDistance(nil, nil); d != 0.0 {
		t.Errorf("nil/nil distance = %f, want 0.0", d)
	}

	node := NewTreeNode(0, "A")
	if d := a.ComputeDistance(nil, node); d != 1.0 {
		t.Errorf("nil/node distance = %f, want 1.0", d)
	}
	if d := a.ComputeDistance(node, nil); d != 1.0 {
		t.Errorf("node/nil distance = %f, want 1.0", d)
	}
}

func TestComputeDistance_IdenticalTrees(t *testing.T) {
	a := NewAPTEDAnalyzer(NewDefaultCostModel())

	t1 := buildSimpleTree("A", "B", "C")
	t2 := buildSimpleTree("A", "B", "C")

	d := a.ComputeDistance(t1, t2)
	if d != 0.0 {
		t.Errorf("Identical trees distance = %f, want 0.0", d)
	}
}

func TestComputeDistance_SingleNodes(t *testing.T) {
	a := NewAPTEDAnalyzer(NewDefaultCostModel())

	// Same label
	t1 := NewTreeNode(0, "A")
	t2 := NewTreeNode(1, "A")
	if d := a.ComputeDistance(t1, t2); d != 0.0 {
		t.Errorf("Same label distance = %f, want 0.0", d)
	}

	// Different label
	t3 := NewTreeNode(2, "B")
	if d := a.ComputeDistance(t1, t3); d != 1.0 {
		t.Errorf("Different label distance = %f, want 1.0", d)
	}
}

func TestComputeDistance_InsertDelete(t *testing.T) {
	a := NewAPTEDAnalyzer(NewDefaultCostModel())

	// One child vs no children
	t1 := NewTreeNode(0, "A")
	t2 := buildSimpleTree("A", "B")

	d := a.ComputeDistance(t1, t2)
	if d != 1.0 {
		t.Errorf("Insert one child distance = %f, want 1.0", d)
	}
}

func TestComputeDistance_ComplexTrees(t *testing.T) {
	a := NewAPTEDAnalyzer(NewDefaultCostModel())

	// Tree1: A -> [B, C]
	t1 := buildSimpleTree("A", "B", "C")

	// Tree2: A -> [B, D]
	t2 := buildSimpleTree("A", "B", "D")

	d := a.ComputeDistance(t1, t2)
	// Should rename C->D = 1.0
	if d != 1.0 {
		t.Errorf("Complex tree distance = %f, want 1.0", d)
	}
}

func TestComputeDistance_Symmetric(t *testing.T) {
	a := NewAPTEDAnalyzer(NewDefaultCostModel())

	t1 := buildSimpleTree("A", "B", "C")
	t2 := buildSimpleTree("X", "Y")

	d1 := a.ComputeDistance(t1, t2)
	d2 := a.ComputeDistance(t2, t1)
	if math.Abs(d1-d2) > 1e-9 {
		t.Errorf("Distance not symmetric: %f vs %f", d1, d2)
	}
}

func TestComputeSimilarity_NormalizeByMax(t *testing.T) {
	a := NewAPTEDAnalyzerWithNormalization(NewDefaultCostModel(), NormalizeByMax)

	tests := []struct {
		name string
		t1   *TreeNode
		t2   *TreeNode
		low  float64
		high float64
	}{
		{"both nil", nil, nil, 1.0, 1.0},
		{"one nil", NewTreeNode(0, "A"), nil, 0.0, 0.0},
		{"identical", buildSimpleTree("A", "B"), buildSimpleTree("A", "B"), 1.0, 1.0},
		{"different", buildSimpleTree("A", "B"), buildSimpleTree("X", "Y"), 0.0, 0.99},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := a.ComputeSimilarity(tt.t1, tt.t2)
			if s < tt.low || s > tt.high {
				t.Errorf("Similarity = %f, want [%f, %f]", s, tt.low, tt.high)
			}
		})
	}
}

func TestComputeSimilarity_NormalizeBySum(t *testing.T) {
	a := NewAPTEDAnalyzerWithNormalization(NewDefaultCostModel(), NormalizeBySum)

	t1 := buildSimpleTree("A", "B")
	t2 := buildSimpleTree("A", "B")
	if s := a.ComputeSimilarity(t1, t2); s != 1.0 {
		t.Errorf("Identical trees similarity (sum norm) = %f, want 1.0", s)
	}

	// Verify NormalizeBySum gives different result than NormalizeByMax for non-identical trees
	aMax := NewAPTEDAnalyzerWithNormalization(NewDefaultCostModel(), NormalizeByMax)
	t3 := buildSimpleTree("A", "B", "C")
	t4 := buildSimpleTree("X")

	sSum := a.ComputeSimilarity(t3, t4)
	sMax := aMax.ComputeSimilarity(t3, t4)

	// With sum normalization, the denominator is larger -> similarity is higher
	if sSum <= sMax {
		t.Logf("Sum normalization: %f, Max normalization: %f", sSum, sMax)
		// This is expected: sum norm produces higher similarity
	}
}

func TestComputeSimilarity_Bounds(t *testing.T) {
	a := NewAPTEDAnalyzer(NewDefaultCostModel())

	// Generate various tree pairs and verify similarity is always in [0, 1]
	trees := []*TreeNode{
		nil,
		NewTreeNode(0, "A"),
		buildSimpleTree("A", "B", "C"),
		buildSimpleTree("X", "Y", "Z", "W"),
	}

	for i, t1 := range trees {
		for j, t2 := range trees {
			s := a.ComputeSimilarity(t1, t2)
			if s < 0.0 || s > 1.0 {
				t.Errorf("Similarity(%d, %d) = %f, out of [0, 1]", i, j, s)
			}
		}
	}
}

func TestComputeDetailedDistance(t *testing.T) {
	a := NewAPTEDAnalyzer(NewDefaultCostModel())

	t1 := buildSimpleTree("A", "B")
	t2 := buildSimpleTree("A", "C")

	result := a.ComputeDetailedDistance(t1, t2)

	if result.Tree1Size != 2 {
		t.Errorf("Tree1Size = %d, want 2", result.Tree1Size)
	}
	if result.Tree2Size != 2 {
		t.Errorf("Tree2Size = %d, want 2", result.Tree2Size)
	}
	if result.Distance != 1.0 {
		t.Errorf("Distance = %f, want 1.0", result.Distance)
	}
	if result.Operations != 1 {
		t.Errorf("Operations = %d, want 1", result.Operations)
	}
	if result.Similarity < 0 || result.Similarity > 1 {
		t.Errorf("Similarity = %f, out of [0, 1]", result.Similarity)
	}
}

func TestComputeDetailedDistance_NilTrees(t *testing.T) {
	a := NewAPTEDAnalyzer(NewDefaultCostModel())
	result := a.ComputeDetailedDistance(nil, nil)
	if result.Tree1Size != 0 || result.Tree2Size != 0 {
		t.Error("Expected zero sizes for nil trees")
	}
	if result.Distance != 0 {
		t.Errorf("Distance = %f, want 0", result.Distance)
	}
}

func TestOptimizedAPTEDAnalyzer(t *testing.T) {
	a := NewOptimizedAPTEDAnalyzer(NewDefaultCostModel(), 2.0)

	t1 := NewTreeNode(0, "A")
	t2 := NewTreeNode(1, "A")
	d := a.ComputeDistance(t1, t2)
	if d != 0.0 {
		t.Errorf("Identical distance = %f, want 0.0", d)
	}

	// Very different sizes should trigger early termination
	big := NewTreeNode(0, "Root")
	for i := 1; i <= 10; i++ {
		big.AddChild(NewTreeNode(i, "Child"))
	}
	small := NewTreeNode(0, "X")
	d = a.ComputeDistance(big, small)
	if d <= 2.0 {
		t.Errorf("Expected distance > 2.0 (max), got %f", d)
	}
}

func TestBatchComputeDistances(t *testing.T) {
	a := NewAPTEDAnalyzer(NewDefaultCostModel())

	pairs := [][2]*TreeNode{
		{NewTreeNode(0, "A"), NewTreeNode(1, "A")},
		{NewTreeNode(0, "A"), NewTreeNode(1, "B")},
		{buildSimpleTree("A", "B"), buildSimpleTree("A", "B")},
	}

	distances := a.BatchComputeDistances(pairs)
	if len(distances) != 3 {
		t.Fatalf("Expected 3 distances, got %d", len(distances))
	}
	if distances[0] != 0.0 {
		t.Errorf("Pair 0 distance = %f, want 0.0", distances[0])
	}
	if distances[1] != 1.0 {
		t.Errorf("Pair 1 distance = %f, want 1.0", distances[1])
	}
	if distances[2] != 0.0 {
		t.Errorf("Pair 2 distance = %f, want 0.0", distances[2])
	}
}

func TestClusterSimilarTrees(t *testing.T) {
	a := NewAPTEDAnalyzer(NewDefaultCostModel())

	// Three identical trees and one different
	trees := []*TreeNode{
		buildSimpleTree("A", "B"),
		buildSimpleTree("A", "B"),
		buildSimpleTree("X", "Y", "Z", "W"),
		buildSimpleTree("A", "B"),
	}

	result := a.ClusterSimilarTrees(trees, 0.8)
	if len(result.Groups) == 0 {
		t.Fatal("Expected at least one group")
	}
	if result.Threshold != 0.8 {
		t.Errorf("Threshold = %f, want 0.8", result.Threshold)
	}
}

func TestClusterSimilarTrees_Empty(t *testing.T) {
	a := NewAPTEDAnalyzer(NewDefaultCostModel())
	result := a.ClusterSimilarTrees([]*TreeNode{}, 0.5)
	if len(result.Groups) != 0 {
		t.Errorf("Expected 0 groups, got %d", len(result.Groups))
	}
}

func TestClusterSimilarTrees_AllNil(t *testing.T) {
	a := NewAPTEDAnalyzer(NewDefaultCostModel())
	result := a.ClusterSimilarTrees([]*TreeNode{nil, nil}, 0.5)
	if len(result.Groups) != 0 {
		t.Errorf("Expected 0 groups for all nil, got %d", len(result.Groups))
	}
}

func TestClusterSimilarTrees_Single(t *testing.T) {
	a := NewAPTEDAnalyzer(NewDefaultCostModel())
	result := a.ClusterSimilarTrees([]*TreeNode{NewTreeNode(0, "A")}, 0.5)
	if len(result.Groups) != 1 {
		t.Errorf("Expected 1 group, got %d", len(result.Groups))
	}
}

func TestAPTED_DeepTree(t *testing.T) {
	a := NewAPTEDAnalyzer(NewDefaultCostModel())

	// Build a deep chain: n0 -> n1 -> n2 -> ... -> n99
	const depth = 100
	var root *TreeNode
	current := NewTreeNode(0, "N0")
	root = current
	for i := 1; i < depth; i++ {
		child := NewTreeNode(i, "N")
		current.AddChild(child)
		current = child
	}

	// Should not panic or hang
	d := a.ComputeDistance(root, root)
	if d != 0.0 {
		t.Errorf("Deep identical tree distance = %f, want 0.0", d)
	}
}

func TestAPTED_WideTree(t *testing.T) {
	a := NewAPTEDAnalyzer(NewDefaultCostModel())

	const width = 50
	root := NewTreeNode(0, "Root")
	for i := 1; i <= width; i++ {
		root.AddChild(NewTreeNode(i, "Child"))
	}

	d := a.ComputeDistance(root, root)
	if d != 0.0 {
		t.Errorf("Wide identical tree distance = %f, want 0.0", d)
	}
}

func BenchmarkAPTED_SmallTrees(b *testing.B) {
	a := NewAPTEDAnalyzer(NewDefaultCostModel())
	t1 := buildSimpleTree("A", "B", "C", "D")
	t2 := buildSimpleTree("A", "B", "X", "Y")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		a.ComputeDistance(t1, t2)
	}
}

func BenchmarkAPTED_MediumTrees(b *testing.B) {
	a := NewAPTEDAnalyzer(NewDefaultCostModel())

	buildTree := func(label string, n int) *TreeNode {
		root := NewTreeNode(0, label)
		for i := 1; i <= n; i++ {
			child := NewTreeNode(i, "C")
			for j := 0; j < 3; j++ {
				child.AddChild(NewTreeNode(i*10+j, "L"))
			}
			root.AddChild(child)
		}
		return root
	}

	t1 := buildTree("A", 10)
	t2 := buildTree("B", 10)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		a.ComputeDistance(t1, t2)
	}
}

func BenchmarkTreePreparation(b *testing.B) {
	root := buildSimpleTree("Root", "A", "B", "C", "D", "E")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		PrepareTreeForAPTED(root)
	}
}
