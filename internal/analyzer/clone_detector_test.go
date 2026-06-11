package analyzer

import (
	"context"
	"math"
	"testing"

	"github.com/ludo-technologies/jscan/domain"
	"github.com/ludo-technologies/jscan/internal/parser"
)

func TestDefaultCloneDetectorConfig(t *testing.T) {
	config := DefaultCloneDetectorConfig()

	if config.MinLines != 5 {
		t.Errorf("Expected MinLines 5, got %d", config.MinLines)
	}
	if config.MinNodes != 10 {
		t.Errorf("Expected MinNodes 10, got %d", config.MinNodes)
	}
	if config.CostModelType != "javascript" {
		t.Errorf("Expected CostModelType 'javascript', got %s", config.CostModelType)
	}
	if config.GroupingMode != GroupingModeConnected {
		t.Errorf("Expected GroupingMode 'connected', got %s", config.GroupingMode)
	}
}

func TestNewCloneDetector(t *testing.T) {
	config := DefaultCloneDetectorConfig()
	detector := NewCloneDetector(config)

	if detector == nil {
		t.Fatal("Expected non-nil detector")
	}
	if detector.analyzer == nil {
		t.Error("Expected non-nil analyzer")
	}
	if detector.converter == nil {
		t.Error("Expected non-nil converter")
	}
}

func TestNewCloneDetectorWithDifferentCostModels(t *testing.T) {
	testCases := []struct {
		costModelType string
	}{
		{"default"},
		{"javascript"},
		{"weighted"},
		{"unknown"},
	}

	for _, tc := range testCases {
		t.Run(tc.costModelType, func(t *testing.T) {
			config := DefaultCloneDetectorConfig()
			config.CostModelType = tc.costModelType
			detector := NewCloneDetector(config)
			if detector == nil {
				t.Fatal("Expected non-nil detector")
			}
		})
	}
}

func TestNewCodeFragment(t *testing.T) {
	location := &CodeLocation{
		FilePath:  "test.js",
		StartLine: 1,
		EndLine:   10,
		StartCol:  0,
		EndCol:    50,
	}

	// Create a simple AST node
	node := &parser.Node{
		Type: parser.NodeFunction,
		Children: []*parser.Node{
			{Type: parser.NodeIdentifier},
			{Type: parser.NodeBlockStatement},
		},
	}

	fragment := NewCodeFragment(location, node, "function test() {}")

	if fragment.Location != location {
		t.Error("Location not set correctly")
	}
	if fragment.ASTNode != node {
		t.Error("ASTNode not set correctly")
	}
	if fragment.LineCount != 10 {
		t.Errorf("Expected LineCount 10, got %d", fragment.LineCount)
	}
	if fragment.Size != 3 { // 1 root + 2 children
		t.Errorf("Expected Size 3, got %d", fragment.Size)
	}
}

func TestCodeLocationString(t *testing.T) {
	location := &CodeLocation{
		FilePath:  "test.js",
		StartLine: 10,
		EndLine:   20,
		StartCol:  5,
		EndCol:    30,
	}

	expected := "test.js:10:5-20:30"
	if location.String() != expected {
		t.Errorf("Expected %s, got %s", expected, location.String())
	}
}

func TestIsFragmentCandidate(t *testing.T) {
	config := DefaultCloneDetectorConfig()
	detector := NewCloneDetector(config)

	testCases := []struct {
		nodeType parser.NodeType
		expected bool
	}{
		{parser.NodeFunction, true},
		{parser.NodeArrowFunction, true},
		{parser.NodeClass, true},
		{parser.NodeForStatement, true},
		{parser.NodeWhileStatement, true},
		{parser.NodeIfStatement, true},
		{parser.NodeTryStatement, true},
		{parser.NodeIdentifier, false},
		{parser.NodeCallExpression, false},
		{parser.NodeLiteral, false},
	}

	for _, tc := range testCases {
		t.Run(string(tc.nodeType), func(t *testing.T) {
			node := &parser.Node{Type: tc.nodeType}
			result := detector.isFragmentCandidate(node)
			if result != tc.expected {
				t.Errorf("Expected %v for %s, got %v", tc.expected, tc.nodeType, result)
			}
		})
	}
}

func TestShouldIncludeFragment(t *testing.T) {
	config := DefaultCloneDetectorConfig()
	config.MinLines = 5
	config.MinNodes = 10
	detector := NewCloneDetector(config)

	testCases := []struct {
		name      string
		lineCount int
		size      int
		expected  bool
	}{
		{"small_fragment", 3, 5, false},
		{"few_nodes", 10, 5, false},
		{"valid_fragment", 10, 15, true},
		{"exact_minimum", 5, 10, true},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			fragment := &CodeFragment{
				LineCount: tc.lineCount,
				Size:      tc.size,
			}
			result := detector.shouldIncludeFragment(fragment)
			if result != tc.expected {
				t.Errorf("Expected %v, got %v", tc.expected, result)
			}
		})
	}
}

func TestClassifyClonePair(t *testing.T) {
	config := DefaultCloneDetectorConfig()
	detector := NewCloneDetector(config)

	sameFeatures := []string{"type:IfStatement", "type:ReturnStatement"}
	differentFeatures := []string{"type:ForStatement", "type:ThrowStatement"}

	testCases := []struct {
		name       string
		similarity float64
		fragment1  *CodeFragment
		fragment2  *CodeFragment
		expected   domain.CloneType
	}{
		{
			name:       "exact textual match above Type-1 threshold",
			similarity: 0.90,
			fragment1:  &CodeFragment{Content: "if (a) { return 1; }", Features: sameFeatures},
			fragment2:  &CodeFragment{Content: "if (a) { return 1; }", Features: sameFeatures},
			expected:   domain.Type1Clone,
		},
		{
			name:       "no textual match is capped below Type-1, syntactic match gives Type-2",
			similarity: 0.90,
			fragment1:  &CodeFragment{Content: "if (a) { return 1; }", Features: sameFeatures},
			fragment2:  &CodeFragment{Content: "if (b) { return 2; }", Features: sameFeatures},
			expected:   domain.Type2Clone,
		},
		{
			name:       "syntactic mismatch falls through to Type-3",
			similarity: 0.90,
			fragment1:  &CodeFragment{Content: "if (a) { return 1; }", Features: sameFeatures},
			fragment2:  &CodeFragment{Content: "for (;;) { throw x; }", Features: differentFeatures},
			expected:   domain.Type3Clone,
		},
		{
			name:       "structural similarity at Type-4 level",
			similarity: 0.66,
			fragment1:  &CodeFragment{Content: "if (a) { return 1; }", Features: sameFeatures},
			fragment2:  &CodeFragment{Content: "for (;;) { throw x; }", Features: differentFeatures},
			expected:   domain.Type4Clone,
		},
		{
			name:       "below all thresholds is not a clone",
			similarity: 0.50,
			fragment1:  &CodeFragment{Content: "if (a) { return 1; }", Features: sameFeatures},
			fragment2:  &CodeFragment{Content: "for (;;) { throw x; }", Features: differentFeatures},
			expected:   0,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result, _ := detector.classifyClonePair(tc.fragment1, tc.fragment2, tc.similarity)
			if result != tc.expected {
				t.Errorf("For similarity %.2f, expected %v, got %v", tc.similarity, tc.expected, result)
			}
		})
	}
}

func TestDetectClonesEmpty(t *testing.T) {
	config := DefaultCloneDetectorConfig()
	detector := NewCloneDetector(config)

	pairs, groups := detector.DetectClones([]*CodeFragment{})

	if len(pairs) != 0 {
		t.Errorf("Expected 0 pairs for empty input, got %d", len(pairs))
	}
	if len(groups) != 0 {
		t.Errorf("Expected 0 groups for empty input, got %d", len(groups))
	}
}

func TestDetectClonesSingleFragment(t *testing.T) {
	config := DefaultCloneDetectorConfig()
	detector := NewCloneDetector(config)

	fragment := &CodeFragment{
		Location: &CodeLocation{
			FilePath:  "test.js",
			StartLine: 1,
			EndLine:   10,
		},
		ASTNode: &parser.Node{Type: parser.NodeFunction},
		Size:    15,
	}

	pairs, _ := detector.DetectClones([]*CodeFragment{fragment})

	if len(pairs) != 0 {
		t.Errorf("Expected 0 pairs for single fragment, got %d", len(pairs))
	}
}

func TestDetectClonesWithContext(t *testing.T) {
	config := DefaultCloneDetectorConfig()
	detector := NewCloneDetector(config)

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	pairs, groups := detector.DetectClonesWithContext(ctx, []*CodeFragment{})

	if len(pairs) != 0 {
		t.Errorf("Expected 0 pairs after cancellation, got %d", len(pairs))
	}
	if len(groups) != 0 {
		t.Errorf("Expected 0 groups after cancellation, got %d", len(groups))
	}
}

func TestIsSameLocation(t *testing.T) {
	config := DefaultCloneDetectorConfig()
	detector := NewCloneDetector(config)

	loc1 := &CodeLocation{
		FilePath:  "test.js",
		StartLine: 1,
		EndLine:   10,
	}
	loc2 := &CodeLocation{
		FilePath:  "test.js",
		StartLine: 1,
		EndLine:   10,
	}
	loc3 := &CodeLocation{
		FilePath:  "other.js",
		StartLine: 1,
		EndLine:   10,
	}

	if !detector.isSameLocation(loc1, loc2) {
		t.Error("Expected same location")
	}
	if detector.isSameLocation(loc1, loc3) {
		t.Error("Expected different location")
	}
}

func TestCalculateConfidence(t *testing.T) {
	config := DefaultCloneDetectorConfig()
	detector := NewCloneDetector(config)

	fragment1 := &CodeFragment{
		Size:       100,
		Complexity: 10,
	}
	fragment2 := &CodeFragment{
		Size:       100,
		Complexity: 10,
	}

	confidence := detector.calculateConfidence(fragment1, fragment2, 0.9)

	// Base confidence (0.9) + size bonus + complexity bonus
	if confidence < 0.9 || confidence > 1.0 {
		t.Errorf("Confidence should be between 0.9 and 1.0, got %f", confidence)
	}
}

func TestGetStatistics(t *testing.T) {
	config := DefaultCloneDetectorConfig()
	detector := NewCloneDetector(config)

	stats := detector.GetStatistics()

	if stats["total_fragments"] != 0 {
		t.Errorf("Expected 0 fragments, got %v", stats["total_fragments"])
	}
	if stats["total_clone_pairs"] != 0 {
		t.Errorf("Expected 0 clone pairs, got %v", stats["total_clone_pairs"])
	}
	if stats["total_clone_groups"] != 0 {
		t.Errorf("Expected 0 clone groups, got %v", stats["total_clone_groups"])
	}
}

func TestSetUseLSH(t *testing.T) {
	config := DefaultCloneDetectorConfig()
	detector := NewCloneDetector(config)

	if detector.cloneDetectorConfig.UseLSH {
		t.Error("Expected UseLSH to be false by default")
	}

	detector.SetUseLSH(true)

	if !detector.cloneDetectorConfig.UseLSH {
		t.Error("Expected UseLSH to be true after setting")
	}
}

func TestSetBatchSizeLarge(t *testing.T) {
	config := DefaultCloneDetectorConfig()
	detector := NewCloneDetector(config)

	detector.SetBatchSizeLarge(50)

	if detector.cloneDetectorConfig.BatchSizeLarge != 50 {
		t.Errorf("Expected BatchSizeLarge 50, got %d", detector.cloneDetectorConfig.BatchSizeLarge)
	}
}

func TestCalculateBatchSize(t *testing.T) {
	config := DefaultCloneDetectorConfig()
	config.BatchSizeThreshold = 50
	config.BatchSizeLarge = 100
	config.BatchSizeSmall = 50
	config.LargeProjectSize = 500
	detector := NewCloneDetector(config)

	testCases := []struct {
		fragmentCount int
		expected      int
	}{
		{30, 30},   // Below threshold, no batching
		{100, 100}, // Normal project
		{1000, 50}, // Large project
	}

	for _, tc := range testCases {
		result := detector.calculateBatchSize(tc.fragmentCount)
		if result != tc.expected {
			t.Errorf("For %d fragments, expected batch size %d, got %d", tc.fragmentCount, tc.expected, result)
		}
	}
}

func TestShouldCompareFragments(t *testing.T) {
	config := DefaultCloneDetectorConfig()
	detector := NewCloneDetector(config)

	testCases := []struct {
		name     string
		size1    int
		size2    int
		lines1   int
		lines2   int
		expected bool
	}{
		{"similar_size", 100, 100, 50, 50, true},
		{"size_diff_50_percent", 100, 50, 50, 50, false},
		{"line_diff_large", 100, 100, 100, 30, false},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			f1 := &CodeFragment{Size: tc.size1, LineCount: tc.lines1}
			f2 := &CodeFragment{Size: tc.size2, LineCount: tc.lines2}
			result := detector.shouldCompareFragments(f1, f2)
			if result != tc.expected {
				t.Errorf("Expected %v, got %v", tc.expected, result)
			}
		})
	}
}

func TestHelperFunctions(t *testing.T) {
	// Test absInt
	if absInt(-5) != 5 {
		t.Error("absInt(-5) should be 5")
	}
	if absInt(5) != 5 {
		t.Error("absInt(5) should be 5")
	}

	// Test maxInt
	if maxInt(3, 5) != 5 {
		t.Error("maxInt(3, 5) should be 5")
	}
	if maxInt(5, 3) != 5 {
		t.Error("maxInt(5, 3) should be 5")
	}

	// Test minInt
	if minInt(3, 5) != 3 {
		t.Error("minInt(3, 5) should be 3")
	}
	if minInt(5, 3) != 3 {
		t.Error("minInt(5, 3) should be 3")
	}
}

func TestComputeDistanceAndSimilarity_MatchesAnalyzerFormula(t *testing.T) {
	analyzer := NewAPTEDAnalyzer(NewJavaScriptCostModel())

	treeA := &TreeNode{
		Label: "FunctionDeclaration",
		Children: []*TreeNode{
			{Label: "Identifier"},
			{
				Label: "BlockStatement",
				Children: []*TreeNode{
					{Label: "ReturnStatement"},
				},
			},
		},
	}
	treeB := &TreeNode{
		Label: "FunctionDeclaration",
		Children: []*TreeNode{
			{Label: "Identifier"},
			{
				Label: "BlockStatement",
				Children: []*TreeNode{
					{Label: "ExpressionStatement"},
				},
			},
		},
	}
	PrepareTreeForAPTED(treeA)
	PrepareTreeForAPTED(treeB)

	wantDistance := analyzer.ComputeDistance(treeA, treeB)
	gotDistance, gotSimilarity := analyzer.ComputeDistanceAndSimilarity(treeA, treeB)
	wantSimilarity := analyzer.ComputeSimilarity(treeA, treeB)

	if math.Abs(gotDistance-wantDistance) > 1e-12 {
		t.Fatalf("distance mismatch: got %.12f, want %.12f", gotDistance, wantDistance)
	}
	if math.Abs(gotSimilarity-wantSimilarity) > 1e-12 {
		t.Fatalf("similarity mismatch: got %.12f, want %.12f", gotSimilarity, wantSimilarity)
	}
}
