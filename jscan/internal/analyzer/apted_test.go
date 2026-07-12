package analyzer

import (
	"fmt"
	"math"
	"testing"

	"github.com/ludo-technologies/polyscan/jscan/internal/parser"
)

func TestNewTreeNode(t *testing.T) {
	node := NewTreeNode(1, "Test")
	if node.ID != 1 {
		t.Errorf("Expected ID 1, got %d", node.ID)
	}
	if node.Label != "Test" {
		t.Errorf("Expected label 'Test', got %s", node.Label)
	}
	if len(node.Children) != 0 {
		t.Errorf("Expected empty children, got %d", len(node.Children))
	}
}

func TestTreeNodeAddChild(t *testing.T) {
	parent := NewTreeNode(1, "Parent")
	child := NewTreeNode(2, "Child")

	parent.AddChild(child)

	if len(parent.Children) != 1 {
		t.Errorf("Expected 1 child, got %d", len(parent.Children))
	}
	if child.Parent != parent {
		t.Error("Child's parent should be set")
	}
}

func TestTreeNodeIsLeaf(t *testing.T) {
	leaf := NewTreeNode(1, "Leaf")
	if !leaf.IsLeaf() {
		t.Error("Node without children should be a leaf")
	}

	parent := NewTreeNode(2, "Parent")
	child := NewTreeNode(3, "Child")
	parent.AddChild(child)
	if parent.IsLeaf() {
		t.Error("Node with children should not be a leaf")
	}
}

func TestTreeNodeSize(t *testing.T) {
	// Single node
	single := NewTreeNode(1, "Single")
	if single.Size() != 1 {
		t.Errorf("Single node size should be 1, got %d", single.Size())
	}

	// Tree with children
	root := NewTreeNode(1, "Root")
	child1 := NewTreeNode(2, "Child1")
	child2 := NewTreeNode(3, "Child2")
	grandchild := NewTreeNode(4, "Grandchild")

	root.AddChild(child1)
	root.AddChild(child2)
	child1.AddChild(grandchild)

	if root.Size() != 4 {
		t.Errorf("Tree size should be 4, got %d", root.Size())
	}
}

func TestTreeNodeHeight(t *testing.T) {
	// Single node
	single := NewTreeNode(1, "Single")
	if single.Height() != 0 {
		t.Errorf("Single node height should be 0, got %d", single.Height())
	}

	// Tree with depth
	root := NewTreeNode(1, "Root")
	child := NewTreeNode(2, "Child")
	grandchild := NewTreeNode(3, "Grandchild")

	root.AddChild(child)
	child.AddChild(grandchild)

	if root.Height() != 2 {
		t.Errorf("Tree height should be 2, got %d", root.Height())
	}
}

func TestDefaultCostModel(t *testing.T) {
	costModel := NewDefaultCostModel()
	node := NewTreeNode(1, "Test")

	if costModel.Insert(node) != 1.0 {
		t.Error("Insert cost should be 1.0")
	}
	if costModel.Delete(node) != 1.0 {
		t.Error("Delete cost should be 1.0")
	}

	node2 := NewTreeNode(2, "Test")
	if costModel.Rename(node, node2) != 0.0 {
		t.Error("Rename cost for same labels should be 0.0")
	}

	node3 := NewTreeNode(3, "Different")
	if costModel.Rename(node, node3) != 1.0 {
		t.Error("Rename cost for different labels should be 1.0")
	}
}

func TestJavaScriptCostModel(t *testing.T) {
	costModel := NewJavaScriptCostModel()

	// Test structural node (higher cost)
	funcNode := NewTreeNode(1, "FunctionDeclaration")
	cost := costModel.Insert(funcNode)
	if cost <= 1.0 {
		t.Error("Structural nodes should have higher cost")
	}

	// Test control flow node
	ifNode := NewTreeNode(2, "IfStatement")
	cost = costModel.Insert(ifNode)
	if cost <= 1.0 {
		t.Error("Control flow nodes should have higher cost")
	}

	// Test expression node (lower cost)
	exprNode := NewTreeNode(3, "BinaryExpression")
	cost = costModel.Insert(exprNode)
	if cost >= 1.0 {
		t.Error("Expression nodes should have lower cost")
	}

	// Test rename with same base type
	node1 := NewTreeNode(4, "Identifier(foo)")
	node2 := NewTreeNode(5, "Identifier(bar)")
	renameCost := costModel.Rename(node1, node2)
	if renameCost >= 1.0 {
		t.Error("Rename cost for same base type should be reduced")
	}
}

func TestJavaScriptCostModelIgnore(t *testing.T) {
	// Test with ignore literals
	costModel := NewJavaScriptCostModelWithConfig(true, false)

	lit1 := NewTreeNode(1, "Literal(42)")
	lit2 := NewTreeNode(2, "Literal(100)")
	cost := costModel.Rename(lit1, lit2)
	if cost != 0.0 {
		t.Error("Literal differences should be ignored when configured")
	}

	// Test with ignore identifiers
	costModel2 := NewJavaScriptCostModelWithConfig(false, true)

	id1 := NewTreeNode(3, "Identifier(foo)")
	id2 := NewTreeNode(4, "Identifier(bar)")
	cost = costModel2.Rename(id1, id2)
	if cost != 0.0 {
		t.Error("Identifier differences should be ignored when configured")
	}
}

func TestPrepareTreeForAPTED(t *testing.T) {
	// Build a simple tree
	root := NewTreeNode(1, "Root")
	child1 := NewTreeNode(2, "Child1")
	child2 := NewTreeNode(3, "Child2")
	root.AddChild(child1)
	root.AddChild(child2)

	keyRoots := PrepareTreeForAPTED(root)

	// Verify post-order IDs were assigned
	if child1.PostOrderID >= child2.PostOrderID {
		t.Error("Post-order IDs should be in correct order")
	}

	// Verify key roots were identified
	if len(keyRoots) == 0 {
		t.Error("Key roots should be identified")
	}
}

func TestAPTEDAnalyzerIdenticalTrees(t *testing.T) {
	costModel := NewDefaultCostModel()
	analyzer := NewAPTEDAnalyzer(costModel)

	// Create identical trees
	tree1 := NewTreeNode(1, "Root")
	tree1.AddChild(NewTreeNode(2, "Child"))

	tree2 := NewTreeNode(1, "Root")
	tree2.AddChild(NewTreeNode(2, "Child"))

	distance := analyzer.ComputeDistance(tree1, tree2)
	if distance != 0.0 {
		t.Errorf("Distance between identical trees should be 0, got %f", distance)
	}

	similarity := analyzer.ComputeSimilarity(tree1, tree2)
	if similarity != 1.0 {
		t.Errorf("Similarity between identical trees should be 1.0, got %f", similarity)
	}
}

func TestAPTEDAnalyzerDifferentTrees(t *testing.T) {
	costModel := NewDefaultCostModel()
	analyzer := NewAPTEDAnalyzer(costModel)

	// Create different trees
	tree1 := NewTreeNode(1, "A")
	tree1.AddChild(NewTreeNode(2, "B"))

	tree2 := NewTreeNode(1, "X")
	tree2.AddChild(NewTreeNode(2, "Y"))
	tree2.AddChild(NewTreeNode(3, "Z"))

	distance := analyzer.ComputeDistance(tree1, tree2)
	if distance <= 0.0 {
		t.Error("Distance between different trees should be positive")
	}

	similarity := analyzer.ComputeSimilarity(tree1, tree2)
	if similarity >= 1.0 || similarity < 0.0 {
		t.Errorf("Similarity should be between 0 and 1, got %f", similarity)
	}
}

func TestAPTEDAnalyzerNilTrees(t *testing.T) {
	costModel := NewDefaultCostModel()
	analyzer := NewAPTEDAnalyzer(costModel)

	// Both nil
	distance := analyzer.ComputeDistance(nil, nil)
	if distance != 0.0 {
		t.Error("Distance between two nil trees should be 0")
	}

	// One nil
	tree := NewTreeNode(1, "Root")
	tree.AddChild(NewTreeNode(2, "Child"))

	distance = analyzer.ComputeDistance(tree, nil)
	if distance != 2.0 {
		t.Errorf("Distance should be tree size (2), got %f", distance)
	}

	distance = analyzer.ComputeDistance(nil, tree)
	if distance != 2.0 {
		t.Errorf("Distance should be tree size (2), got %f", distance)
	}
}

func TestAPTEDAnalyzerComputeDetailedDistance(t *testing.T) {
	costModel := NewDefaultCostModel()
	analyzer := NewAPTEDAnalyzer(costModel)

	tree1 := NewTreeNode(1, "A")
	tree1.AddChild(NewTreeNode(2, "B"))

	tree2 := NewTreeNode(1, "A")
	tree2.AddChild(NewTreeNode(2, "C"))

	result := analyzer.ComputeDetailedDistance(tree1, tree2)

	if result.Tree1Size != 2 {
		t.Errorf("Expected tree1 size 2, got %d", result.Tree1Size)
	}
	if result.Tree2Size != 2 {
		t.Errorf("Expected tree2 size 2, got %d", result.Tree2Size)
	}
	if result.Distance < 0 {
		t.Error("Distance should be non-negative")
	}
	if result.Similarity < 0 || result.Similarity > 1 {
		t.Error("Similarity should be between 0 and 1")
	}
}

func TestOptimizedAPTEDAnalyzer(t *testing.T) {
	costModel := NewDefaultCostModel()
	analyzer := NewOptimizedAPTEDAnalyzer(costModel, 5.0)

	tree1 := NewTreeNode(1, "Root")
	tree2 := NewTreeNode(1, "Root")

	distance := analyzer.ComputeDistance(tree1, tree2)
	if distance != 0.0 {
		t.Errorf("Distance between identical trees should be 0, got %f", distance)
	}
}

func TestBatchComputeDistances(t *testing.T) {
	costModel := NewDefaultCostModel()
	analyzer := NewAPTEDAnalyzer(costModel)

	tree1 := NewTreeNode(1, "A")
	tree2 := NewTreeNode(2, "A")
	tree3 := NewTreeNode(3, "B")

	pairs := [][2]*TreeNode{
		{tree1, tree2},
		{tree1, tree3},
		{tree2, tree3},
	}

	distances := analyzer.BatchComputeDistances(pairs)

	if len(distances) != 3 {
		t.Errorf("Expected 3 distances, got %d", len(distances))
	}

	// tree1 and tree2 have same label
	if distances[0] != 0.0 {
		t.Errorf("Distance between identical label trees should be 0, got %f", distances[0])
	}

	// tree1 and tree3 have different labels
	if distances[1] == 0.0 {
		t.Error("Distance between different label trees should not be 0")
	}
}

func TestClusterSimilarTrees(t *testing.T) {
	costModel := NewDefaultCostModel()
	analyzer := NewAPTEDAnalyzer(costModel)

	// Create similar trees
	tree1 := NewTreeNode(1, "A")
	tree1.AddChild(NewTreeNode(2, "B"))

	tree2 := NewTreeNode(1, "A")
	tree2.AddChild(NewTreeNode(2, "B"))

	// Create different tree
	tree3 := NewTreeNode(1, "X")
	tree3.AddChild(NewTreeNode(2, "Y"))
	tree3.AddChild(NewTreeNode(3, "Z"))

	trees := []*TreeNode{tree1, tree2, tree3}
	result := analyzer.ClusterSimilarTrees(trees, 0.8)

	if len(result.Groups) == 0 {
		t.Error("Should have at least one group")
	}
}

func TestClusterEmptyTrees(t *testing.T) {
	costModel := NewDefaultCostModel()
	analyzer := NewAPTEDAnalyzer(costModel)

	// Empty slice
	result := analyzer.ClusterSimilarTrees([]*TreeNode{}, 0.8)
	if len(result.Groups) != 0 {
		t.Error("Empty input should produce empty groups")
	}

	// Single tree
	tree := NewTreeNode(1, "A")
	result = analyzer.ClusterSimilarTrees([]*TreeNode{tree}, 0.8)
	if len(result.Groups) != 1 {
		t.Error("Single tree should produce one group")
	}
}

func TestWeightedCostModel(t *testing.T) {
	baseCost := NewDefaultCostModel()
	weighted := NewWeightedCostModel(2.0, 0.5, 1.5, baseCost)

	node := NewTreeNode(1, "Test")

	if weighted.Insert(node) != 2.0 {
		t.Errorf("Weighted insert cost should be 2.0, got %f", weighted.Insert(node))
	}
	if weighted.Delete(node) != 0.5 {
		t.Errorf("Weighted delete cost should be 0.5, got %f", weighted.Delete(node))
	}

	node2 := NewTreeNode(2, "Different")
	if weighted.Rename(node, node2) != 1.5 {
		t.Errorf("Weighted rename cost should be 1.5, got %f", weighted.Rename(node, node2))
	}
}

func TestTreeConverterConvertAST(t *testing.T) {
	// This test verifies the converter works with nil input
	converter := NewTreeConverter()

	result := converter.ConvertAST(nil)
	if result != nil {
		t.Error("Converting nil AST should return nil")
	}
}

func TestSimilarityBounds(t *testing.T) {
	costModel := NewDefaultCostModel()
	analyzer := NewAPTEDAnalyzer(costModel)

	// Create various tree pairs
	testCases := []struct {
		name  string
		tree1 *TreeNode
		tree2 *TreeNode
	}{
		{"identical", createTestTree(3), createTestTree(3)},
		{"different_size", createTestTree(2), createTestTree(5)},
		{"completely_different", createDifferentTree(3), createDifferentTree(3)},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			sim := analyzer.ComputeSimilarity(tc.tree1, tc.tree2)
			if sim < 0.0 || sim > 1.0 {
				t.Errorf("Similarity must be in [0, 1], got %f", sim)
			}
			if math.IsNaN(sim) || math.IsInf(sim, 0) {
				t.Errorf("Similarity must be a valid number, got %f", sim)
			}
		})
	}
}

// Helper functions for creating test trees
func createTestTree(size int) *TreeNode {
	root := NewTreeNode(0, "Root")
	for i := 1; i < size; i++ {
		root.AddChild(NewTreeNode(i, "Child"))
	}
	return root
}

func createDifferentTree(size int) *TreeNode {
	root := NewTreeNode(0, "Different")
	for i := 1; i < size; i++ {
		root.AddChild(NewTreeNode(i, "Other"))
	}
	return root
}

func TestAPTEDAnalyzerLargeTreesPreserveLabelDistance(t *testing.T) {
	for _, size := range []int{501, 2001} {
		t.Run("different_labels", func(t *testing.T) {
			tree1 := createWideTreeWithLabels(size, "left")
			tree2 := createWideTreeWithLabels(size, "right")
			analyzer := NewAPTEDAnalyzer(NewDefaultCostModel())

			distance, similarity := analyzer.ComputeDistanceAndSimilarity(tree1, tree2)

			if distance != float64(size) {
				t.Errorf("every node label differs, so each node needs one rename: want %d, got %f", size, distance)
			}
			if similarity != 0.0 {
				t.Errorf("fully different labels must not produce clone similarity, got %f", similarity)
			}
		})

		t.Run("identical_labels", func(t *testing.T) {
			tree1 := createWideTreeWithLabels(size, "same")
			tree2 := createWideTreeWithLabels(size, "same")
			analyzer := NewAPTEDAnalyzer(NewDefaultCostModel())

			distance, similarity := analyzer.ComputeDistanceAndSimilarity(tree1, tree2)

			if distance != 0.0 {
				t.Errorf("expected zero distance, got %f", distance)
			}
			if similarity != 1.0 {
				t.Errorf("expected similarity 1.0, got %f", similarity)
			}
		})

		t.Run("ignored_identifier_labels", func(t *testing.T) {
			tree1 := createWideIdentifierTree(size, "left")
			tree2 := createWideIdentifierTree(size, "right")
			analyzer := NewAPTEDAnalyzer(NewJavaScriptCostModelWithConfig(false, true))

			distance, similarity := analyzer.ComputeDistanceAndSimilarity(tree1, tree2)

			if distance != 0.0 {
				t.Errorf("expected zero distance with ignored identifiers, got %f", distance)
			}
			if similarity != 1.0 {
				t.Errorf("expected similarity 1.0 with ignored identifiers, got %f", similarity)
			}
		})

		t.Run("weighted_rename_cost", func(t *testing.T) {
			tree1 := createWideTreeWithLabels(size, "left")
			tree2 := createWideTreeWithLabels(size, "right")
			costModel := NewWeightedCostModel(3.0, 3.0, 0.25, NewDefaultCostModel())
			analyzer := NewAPTEDAnalyzer(costModel)

			distance, similarity := analyzer.ComputeDistanceAndSimilarity(tree1, tree2)

			if distance != float64(size)*0.25 {
				t.Errorf("large-tree profiles should use rename cost when it is cheaper: want %f, got %f", float64(size)*0.25, distance)
			}
			if similarity != 0.75 {
				t.Errorf("expected similarity 0.75, got %f", similarity)
			}
		})

		t.Run("shifted_siblings", func(t *testing.T) {
			tree1 := createWideTreeWithShiftedChildren(size, 0)
			tree2 := createWideTreeWithShiftedChildren(size, 1)
			analyzer := NewAPTEDAnalyzer(NewDefaultCostModel())

			distance, similarity := analyzer.ComputeDistanceAndSimilarity(tree1, tree2)

			if distance != 2.0 {
				t.Errorf("same-shape sibling shifts should use delete/insert alignment: want 2.0, got %f", distance)
			}
			expected := 1.0 - (2.0 / float64(size))
			if math.Abs(similarity-expected) > 0.001 {
				t.Errorf("expected similarity %f, got %f", expected, similarity)
			}
		})

		t.Run("reversed_siblings", func(t *testing.T) {
			tree1 := createWideTreeWithLabels(size, "child")
			tree2 := createWideTreeWithReversedChildren(size)
			analyzer := NewAPTEDAnalyzer(NewDefaultCostModel())

			distance, similarity := analyzer.ComputeDistanceAndSimilarity(tree1, tree2)

			if distance != float64(size-1) {
				t.Errorf("complex wide reorders should stay bounded: want %d, got %f", size-1, distance)
			}
			expected := 1.0 - (float64(size-1) / float64(size))
			if math.Abs(similarity-expected) > 0.001 {
				t.Errorf("expected similarity %f, got %f", expected, similarity)
			}
		})
	}

	t.Run("same_labels_different_large_shape", func(t *testing.T) {
		tree1 := createTwoLevelTreeWithLabel(1000, 1, "same")
		tree2 := createTwoLevelTreeWithLabel(500, 3, "same")
		analyzer := NewAPTEDAnalyzer(NewDefaultCostModel())

		distance, similarity := analyzer.ComputeDistanceAndSimilarity(tree1, tree2)

		if distance <= 0.0 {
			t.Errorf("large trees with different shape must not look identical, got distance %f", distance)
		}
		if similarity >= 1.0 {
			t.Errorf("expected similarity < 1.0, got %f", similarity)
		}
	})
}

func TestAPTEDAnalyzerSameShapeDistanceMatchesExactAPTED(t *testing.T) {
	tests := []struct {
		name      string
		costModel CostModel
		tree1     *TreeNode
		tree2     *TreeNode
	}{
		{
			name:      "default",
			costModel: NewDefaultCostModel(),
			tree1:     createWideTreeWithLabels(31, "left"),
			tree2:     createWideTreeWithLabels(31, "right"),
		},
		{
			name:      "weighted",
			costModel: NewWeightedCostModel(3.0, 3.0, 0.25, NewDefaultCostModel()),
			tree1:     createWideTreeWithLabels(31, "left"),
			tree2:     createWideTreeWithLabels(31, "right"),
		},
		{
			name:      "ignored_identifiers",
			costModel: NewJavaScriptCostModelWithConfig(false, true),
			tree1:     createWideIdentifierTree(31, "left"),
			tree2:     createWideIdentifierTree(31, "right"),
		},
		{
			name:      "shifted_siblings",
			costModel: NewDefaultCostModel(),
			tree1:     createWideTreeWithShiftedChildren(31, 0),
			tree2:     createWideTreeWithShiftedChildren(31, 1),
		},
		{
			name:      "reversed_siblings",
			costModel: NewDefaultCostModel(),
			tree1:     createWideTreeWithLabels(31, "child"),
			tree2:     createWideTreeWithReversedChildren(31),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			analyzer := NewAPTEDAnalyzer(tt.costModel)
			exactDistance := analyzer.ComputeDistance(tt.tree1, tt.tree2)

			sameShapeDistance, ok := analyzer.computeBoundedSameShapeDistance(tt.tree1, tt.tree2)

			if !ok {
				t.Fatal("expected same-shape distance to be computed")
			}
			if sameShapeDistance != exactDistance {
				t.Errorf("same-shape distance %f should match exact APTED %f", sameShapeDistance, exactDistance)
			}
		})
	}

	t.Run("shape_mismatch", func(t *testing.T) {
		analyzer := NewAPTEDAnalyzer(NewDefaultCostModel())

		_, ok := analyzer.computeBoundedSameShapeDistance(
			createTwoLevelTreeWithLabel(5, 1, "same"),
			createTwoLevelTreeWithLabel(3, 3, "same"),
		)

		if ok {
			t.Error("expected shape mismatch to be rejected")
		}
	})

	t.Run("budget_exhaustion_keeps_positional_child_cost", func(t *testing.T) {
		analyzer := NewAPTEDAnalyzer(NewDefaultCostModel())
		// The large-tree same-shape path is a bounded clone-detection heuristic:
		// when alignment budget is gone, it returns the positional signal rather
		// than pretending the children are identical.
		state := &sameShapeDistanceState{
			distances:               make(map[nodePair]float64),
			deleteCosts:             make(map[*TreeNode]float64),
			insertCosts:             make(map[*TreeNode]float64),
			alignmentCellsRemaining: 0,
		}
		left := []*TreeNode{
			NewTreeNode(1, "A"),
			NewTreeNode(2, "B"),
			NewTreeNode(3, "C"),
		}
		right := []*TreeNode{
			NewTreeNode(4, "A"),
		}

		distance := analyzer.sameShapeChildrenDistance(left, right, state)

		if distance != 2.0 {
			t.Errorf("expected positional child cost 2.0, got %f", distance)
		}
	})
}

func createWideTreeWithLabels(nodeCount int, labelPrefix string) *TreeNode {
	root := NewTreeNode(1, labelPrefix+"_root")
	for i := 2; i <= nodeCount; i++ {
		root.AddChild(NewTreeNode(i, fmt.Sprintf("%s_%d", labelPrefix, i)))
	}
	return root
}

func createWideIdentifierTree(nodeCount int, namePrefix string) *TreeNode {
	root := NewTreeNode(1, "Program")
	for i := 2; i <= nodeCount; i++ {
		root.AddChild(NewTreeNode(i, fmt.Sprintf("Identifier(%s_%d)", namePrefix, i)))
	}
	return root
}

func createWideTreeWithShiftedChildren(nodeCount, shift int) *TreeNode {
	root := NewTreeNode(1, "root")
	if nodeCount <= 1 {
		return root
	}

	childCount := nodeCount - 1
	for i := 0; i < childCount; i++ {
		labelIndex := ((i + shift) % childCount) + 2
		root.AddChild(NewTreeNode(i+2, fmt.Sprintf("child_%d", labelIndex)))
	}
	return root
}

func createWideTreeWithReversedChildren(nodeCount int) *TreeNode {
	root := NewTreeNode(1, "child_root")
	for i := nodeCount; i >= 2; i-- {
		root.AddChild(NewTreeNode(i, fmt.Sprintf("child_%d", i)))
	}
	return root
}

func createTwoLevelTreeWithLabel(parentCount, childrenPerParent int, label string) *TreeNode {
	root := NewTreeNode(1, label)
	nextID := 2
	for i := 0; i < parentCount; i++ {
		parent := NewTreeNode(nextID, label)
		nextID++
		root.AddChild(parent)
		for j := 0; j < childrenPerParent; j++ {
			parent.AddChild(NewTreeNode(nextID, label))
			nextID++
		}
	}
	return root
}

// TestConvertASTIncludesExpressionFieldsInAPTEDDistance verifies that
// expression fields (Test, Left, Right, etc.) are part of the converted tree,
// so two fragments differing only in those fields have non-zero distance.
func TestConvertASTIncludesExpressionFieldsInAPTEDDistance(t *testing.T) {
	buildIf := func(operator string) *parser.Node {
		test := parser.NewNode(parser.NodeBinaryExpression)
		test.Operator = operator
		left := parser.NewNode(parser.NodeIdentifier)
		left.Name = "x"
		right := parser.NewNode(parser.NodeIdentifier)
		right.Name = "y"
		test.Left = left
		test.Right = right

		ifStmt := parser.NewNode(parser.NodeIfStatement)
		ifStmt.Test = test
		consequent := parser.NewNode(parser.NodeBlockStatement)
		consequent.Body = append(consequent.Body, parser.NewNode(parser.NodeReturnStatement))
		ifStmt.Consequent = consequent
		return ifStmt
	}

	converter := NewTreeConverter()
	tree1 := converter.ConvertAST(buildIf("<"))
	tree2 := converter.ConvertAST(buildIf(">"))

	if tree1.Size() <= 2 {
		t.Fatalf("expected expression fields to be converted, tree size %d", tree1.Size())
	}

	analyzer := NewAPTEDAnalyzer(NewDefaultCostModel())
	distance := analyzer.ComputeDistance(tree1, tree2)
	if distance <= 0.0 {
		t.Errorf("fragments differing only in expression operators should have non-zero distance, got %f", distance)
	}
}

func TestComputeApproximateDistanceNilTrees(t *testing.T) {
	analyzer := NewAPTEDAnalyzer(NewDefaultCostModel())

	// Both nil
	if distance := analyzer.computeApproximateDistance(nil, nil); distance != 0.0 {
		t.Errorf("Approximate distance between two nil trees should be 0, got %f", distance)
	}

	// Create a test tree for comparison
	tree := NewTreeNode(1, "test")
	tree.AddChild(NewTreeNode(2, "child"))

	// First nil
	if distance := analyzer.computeApproximateDistance(nil, tree); distance != float64(tree.Size()) {
		t.Errorf("Approximate distance from nil should equal tree size, got %f", distance)
	}

	// Second nil
	if distance := analyzer.computeApproximateDistance(tree, nil); distance != float64(tree.Size()) {
		t.Errorf("Approximate distance to nil should equal tree size, got %f", distance)
	}
}
