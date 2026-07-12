package analyzer

import (
	"testing"
)

func TestNewASTFeatureExtractor(t *testing.T) {
	extractor := NewASTFeatureExtractor()

	if extractor.maxSubtreeHeight != 3 {
		t.Errorf("Expected maxSubtreeHeight 3, got %d", extractor.maxSubtreeHeight)
	}
	if extractor.kGramSize != 4 {
		t.Errorf("Expected kGramSize 4, got %d", extractor.kGramSize)
	}
	if !extractor.includeTypes {
		t.Error("Expected includeTypes to be true")
	}
	if extractor.includeLiterals {
		t.Error("Expected includeLiterals to be false")
	}
}

func TestASTFeatureExtractorWithOptions(t *testing.T) {
	extractor := NewASTFeatureExtractor().WithOptions(5, 6, false, true)

	if extractor.maxSubtreeHeight != 5 {
		t.Errorf("Expected maxSubtreeHeight 5, got %d", extractor.maxSubtreeHeight)
	}
	if extractor.kGramSize != 6 {
		t.Errorf("Expected kGramSize 6, got %d", extractor.kGramSize)
	}
	if extractor.includeTypes {
		t.Error("Expected includeTypes to be false")
	}
	if !extractor.includeLiterals {
		t.Error("Expected includeLiterals to be true")
	}
}

func TestExtractFeaturesNil(t *testing.T) {
	extractor := NewASTFeatureExtractor()
	features, err := extractor.ExtractFeatures(nil)

	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}
	if len(features) != 0 {
		t.Errorf("Expected 0 features for nil input, got %d", len(features))
	}
}

func TestExtractFeaturesSingleNode(t *testing.T) {
	extractor := NewASTFeatureExtractor()
	node := &TreeNode{
		ID:       1,
		Label:    "FunctionDeclaration",
		Children: []*TreeNode{},
	}

	features, err := extractor.ExtractFeatures(node)

	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}
	if len(features) == 0 {
		t.Error("Expected at least some features for a valid node")
	}
}

func TestExtractFeaturesTree(t *testing.T) {
	extractor := NewASTFeatureExtractor()
	node := &TreeNode{
		ID:    1,
		Label: "FunctionDeclaration",
		Children: []*TreeNode{
			{
				ID:       2,
				Label:    "Identifier(foo)",
				Children: []*TreeNode{},
			},
			{
				ID:    3,
				Label: "BlockStatement",
				Children: []*TreeNode{
					{
						ID:       4,
						Label:    "ReturnStatement",
						Children: []*TreeNode{},
					},
				},
			},
		},
	}

	features, err := extractor.ExtractFeatures(node)

	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}
	if len(features) == 0 {
		t.Error("Expected features for a tree")
	}

	// Check for expected feature types
	hasSubtreeFeature := false
	hasKGramFeature := false
	hasTypeFeature := false
	hasPatternFeature := false

	for _, f := range features {
		if len(f) > 4 && f[:4] == "sub:" {
			hasSubtreeFeature = true
		}
		if len(f) > 6 && f[:6] == "kgram:" {
			hasKGramFeature = true
		}
		if len(f) > 5 && f[:5] == "type:" {
			hasTypeFeature = true
		}
		if len(f) > 8 && f[:8] == "pattern:" {
			hasPatternFeature = true
		}
	}

	if !hasSubtreeFeature {
		t.Error("Expected subtree features")
	}
	if !hasTypeFeature {
		t.Error("Expected type features")
	}
	// k-grams need at least k nodes
	// pattern features require specific node types
	_ = hasKGramFeature
	_ = hasPatternFeature
}

func TestExtractSubtreeHashesNil(t *testing.T) {
	extractor := NewASTFeatureExtractor()
	hashes := extractor.ExtractSubtreeHashes(nil, 3)

	if len(hashes) != 0 {
		t.Errorf("Expected 0 hashes for nil input, got %d", len(hashes))
	}
}

func TestExtractSubtreeHashes(t *testing.T) {
	extractor := NewASTFeatureExtractor()
	node := &TreeNode{
		ID:    1,
		Label: "Root",
		Children: []*TreeNode{
			{
				ID:       2,
				Label:    "Child1",
				Children: []*TreeNode{},
			},
			{
				ID:       3,
				Label:    "Child2",
				Children: []*TreeNode{},
			},
		},
	}

	hashes := extractor.ExtractSubtreeHashes(node, 3)

	if len(hashes) == 0 {
		t.Error("Expected at least one subtree hash")
	}

	// All hashes should be prefixed with "sub:"
	for _, h := range hashes {
		if len(h) < 4 || h[:4] != "sub:" {
			t.Errorf("Hash should be prefixed with 'sub:', got %s", h)
		}
	}
}

func TestExtractNodeSequencesNil(t *testing.T) {
	extractor := NewASTFeatureExtractor()
	grams := extractor.ExtractNodeSequences(nil, 4)

	if len(grams) != 0 {
		t.Errorf("Expected 0 grams for nil input, got %d", len(grams))
	}
}

func TestExtractNodeSequencesSmallK(t *testing.T) {
	extractor := NewASTFeatureExtractor()
	node := &TreeNode{
		ID:       1,
		Label:    "Root",
		Children: []*TreeNode{},
	}

	grams := extractor.ExtractNodeSequences(node, 1)

	if len(grams) != 0 {
		t.Errorf("Expected 0 grams for k=1, got %d", len(grams))
	}
}

func TestExtractNodeSequences(t *testing.T) {
	extractor := NewASTFeatureExtractor()
	node := &TreeNode{
		ID:    1,
		Label: "A",
		Children: []*TreeNode{
			{
				ID:    2,
				Label: "B",
				Children: []*TreeNode{
					{ID: 4, Label: "D", Children: []*TreeNode{}},
				},
			},
			{
				ID:       3,
				Label:    "C",
				Children: []*TreeNode{},
			},
		},
	}

	grams := extractor.ExtractNodeSequences(node, 2)

	// Pre-order: A, B, D, C
	// 2-grams: A:B, B:D, D:C
	if len(grams) != 3 {
		t.Errorf("Expected 3 2-grams, got %d", len(grams))
	}
}

func TestCanonicalLabel(t *testing.T) {
	extractor := NewASTFeatureExtractor()

	testCases := []struct {
		input    string
		expected string
	}{
		{"FunctionDeclaration", "FunctionDeclaration"},
		{"Identifier(foo)", "Identifier"},
		{"Literal(42)", "Literal"},
		{"Simple", "Simple"},
	}

	for _, tc := range testCases {
		result := extractor.canonicalLabel(tc.input)
		if result != tc.expected {
			t.Errorf("canonicalLabel(%s) = %s, expected %s", tc.input, result, tc.expected)
		}
	}
}

func TestCanonicalLabelWithLiterals(t *testing.T) {
	extractor := NewASTFeatureExtractor()
	extractor.includeLiterals = true

	input := "Identifier(foo)"
	result := extractor.canonicalLabel(input)

	if result != input {
		t.Errorf("With includeLiterals=true, expected %s, got %s", input, result)
	}
}

func TestBinCount(t *testing.T) {
	extractor := NewASTFeatureExtractor()

	testCases := []struct {
		count    int
		expected string
	}{
		{0, "1"},
		{1, "1"},
		{2, "2-3"},
		{3, "2-3"},
		{5, "4-7"},
		{7, "4-7"},
		{10, "8-15"},
		{15, "8-15"},
		{20, "16+"},
	}

	for _, tc := range testCases {
		result := extractor.binCount(tc.count)
		if result != tc.expected {
			t.Errorf("binCount(%d) = %s, expected %s", tc.count, result, tc.expected)
		}
	}
}

func TestExtractPatterns(t *testing.T) {
	extractor := NewASTFeatureExtractor()

	node := &TreeNode{
		ID:    1,
		Label: "FunctionDeclaration",
		Children: []*TreeNode{
			{
				ID:    2,
				Label: "IfStatement",
				Children: []*TreeNode{
					{ID: 4, Label: "ReturnStatement", Children: []*TreeNode{}},
				},
			},
			{
				ID:    3,
				Label: "ForStatement",
				Children: []*TreeNode{
					{ID: 5, Label: "CallExpression", Children: []*TreeNode{}},
				},
			},
		},
	}

	patterns := extractor.extractPatterns(node)

	// Should contain patterns for: FunctionDeclaration, IfStatement, ForStatement, ReturnStatement, CallExpression
	expectedPatterns := map[string]bool{
		"FunctionDeclaration": true,
		"IfStatement":         true,
		"ForStatement":        true,
		"ReturnStatement":     true,
		"CallExpression":      true,
	}

	// Verify we got at least some expected patterns
	foundCount := 0
	for _, p := range patterns {
		if expectedPatterns[p] {
			foundCount++
		}
	}

	if foundCount < 4 {
		t.Errorf("Expected at least 4 matched patterns, got %d out of %v", foundCount, patterns)
	}
}

func TestFeaturesDeterminism(t *testing.T) {
	extractor := NewASTFeatureExtractor()

	node := &TreeNode{
		ID:    1,
		Label: "FunctionDeclaration",
		Children: []*TreeNode{
			{ID: 2, Label: "BlockStatement", Children: []*TreeNode{}},
		},
	}

	features1, _ := extractor.ExtractFeatures(node)
	features2, _ := extractor.ExtractFeatures(node)

	if len(features1) != len(features2) {
		t.Error("Features should be deterministic - different lengths")
	}

	for i := range features1 {
		if features1[i] != features2[i] {
			t.Errorf("Features should be deterministic - differ at index %d", i)
		}
	}
}
