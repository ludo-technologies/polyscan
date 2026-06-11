package analyzer

import (
	"testing"

	"github.com/ludo-technologies/jscan/domain"
)

func TestCreateGroupingStrategy(t *testing.T) {
	testCases := []struct {
		mode     GroupingMode
		expected string
	}{
		{GroupingModeConnected, "Connected Components"},
		{GroupingModeKCore, "2-Core"},
		{GroupingModeStarMedoid, "Star/Medoid"},
		{GroupingModeCompleteLinkage, "Complete Linkage"},
		{GroupingModeCentroid, "Centroid"},
		{"unknown", "Connected Components"}, // Default fallback
	}

	for _, tc := range testCases {
		t.Run(string(tc.mode), func(t *testing.T) {
			config := GroupingConfig{
				Mode:      tc.mode,
				Threshold: 0.8,
				KCoreK:    2,
			}
			strategy := CreateGroupingStrategy(config)
			if strategy.GetName() != tc.expected {
				t.Errorf("Expected strategy name %s, got %s", tc.expected, strategy.GetName())
			}
		})
	}
}

func TestNewConnectedGrouping(t *testing.T) {
	grouping := NewConnectedGrouping(0.8)

	if grouping.threshold != 0.8 {
		t.Errorf("Expected threshold 0.8, got %f", grouping.threshold)
	}
	if grouping.GetName() != "Connected Components" {
		t.Errorf("Expected name 'Connected Components', got %s", grouping.GetName())
	}
}

func TestConnectedGroupingEmpty(t *testing.T) {
	grouping := NewConnectedGrouping(0.8)
	groups := grouping.GroupClones([]*domain.ClonePair{})

	if len(groups) != 0 {
		t.Errorf("Expected 0 groups for empty input, got %d", len(groups))
	}
}

func TestConnectedGroupingSinglePair(t *testing.T) {
	grouping := NewConnectedGrouping(0.8)

	clone1 := &domain.Clone{
		ID:       1,
		Location: &domain.CloneLocation{FilePath: "a.js", StartLine: 1, EndLine: 10},
	}
	clone2 := &domain.Clone{
		ID:       2,
		Location: &domain.CloneLocation{FilePath: "b.js", StartLine: 1, EndLine: 10},
	}

	pairs := []*domain.ClonePair{
		{
			ID:         1,
			Clone1:     clone1,
			Clone2:     clone2,
			Similarity: 0.9,
			Type:       domain.Type1Clone,
		},
	}

	groups := grouping.GroupClones(pairs)

	if len(groups) != 1 {
		t.Errorf("Expected 1 group, got %d", len(groups))
	}
	if groups[0].Size != 2 {
		t.Errorf("Expected group size 2, got %d", groups[0].Size)
	}
}

func TestConnectedGroupingBelowThreshold(t *testing.T) {
	grouping := NewConnectedGrouping(0.9)

	clone1 := &domain.Clone{
		ID:       1,
		Location: &domain.CloneLocation{FilePath: "a.js", StartLine: 1, EndLine: 10},
	}
	clone2 := &domain.Clone{
		ID:       2,
		Location: &domain.CloneLocation{FilePath: "b.js", StartLine: 1, EndLine: 10},
	}

	pairs := []*domain.ClonePair{
		{
			ID:         1,
			Clone1:     clone1,
			Clone2:     clone2,
			Similarity: 0.8, // Below threshold
			Type:       domain.Type2Clone,
		},
	}

	groups := grouping.GroupClones(pairs)

	// Below threshold, should not form a group
	if len(groups) != 0 {
		t.Errorf("Expected 0 groups (below threshold), got %d", len(groups))
	}
}

func TestConnectedGroupingMultipleComponents(t *testing.T) {
	grouping := NewConnectedGrouping(0.8)

	// First component: clones 1 and 2
	clone1 := &domain.Clone{ID: 1, Location: &domain.CloneLocation{FilePath: "a.js", StartLine: 1, EndLine: 10}}
	clone2 := &domain.Clone{ID: 2, Location: &domain.CloneLocation{FilePath: "a.js", StartLine: 20, EndLine: 30}}

	// Second component: clones 3 and 4
	clone3 := &domain.Clone{ID: 3, Location: &domain.CloneLocation{FilePath: "b.js", StartLine: 1, EndLine: 10}}
	clone4 := &domain.Clone{ID: 4, Location: &domain.CloneLocation{FilePath: "b.js", StartLine: 20, EndLine: 30}}

	pairs := []*domain.ClonePair{
		{ID: 1, Clone1: clone1, Clone2: clone2, Similarity: 0.9, Type: domain.Type1Clone},
		{ID: 2, Clone1: clone3, Clone2: clone4, Similarity: 0.85, Type: domain.Type2Clone},
	}

	groups := grouping.GroupClones(pairs)

	if len(groups) != 2 {
		t.Errorf("Expected 2 groups, got %d", len(groups))
	}
}

func TestNewKCoreGrouping(t *testing.T) {
	grouping := NewKCoreGrouping(0.8, 3)

	if grouping.threshold != 0.8 {
		t.Errorf("Expected threshold 0.8, got %f", grouping.threshold)
	}
	if grouping.k != 3 {
		t.Errorf("Expected k=3, got %d", grouping.k)
	}
	if grouping.GetName() != "3-Core" {
		t.Errorf("Expected name '3-Core', got %s", grouping.GetName())
	}
}

func TestNewKCoreGroupingMinK(t *testing.T) {
	grouping := NewKCoreGrouping(0.8, 1) // k=1 should be bumped to 2

	if grouping.k != 2 {
		t.Errorf("Expected k=2 (minimum), got %d", grouping.k)
	}
}

func TestKCoreGroupingEmpty(t *testing.T) {
	grouping := NewKCoreGrouping(0.8, 2)
	groups := grouping.GroupClones([]*domain.ClonePair{})

	if len(groups) != 0 {
		t.Errorf("Expected 0 groups for empty input, got %d", len(groups))
	}
}

func TestKCoreGroupingTriangle(t *testing.T) {
	grouping := NewKCoreGrouping(0.8, 2)

	// Three clones forming a triangle (each connected to 2 others)
	clone1 := &domain.Clone{ID: 1, Location: &domain.CloneLocation{FilePath: "a.js", StartLine: 1, EndLine: 10}}
	clone2 := &domain.Clone{ID: 2, Location: &domain.CloneLocation{FilePath: "b.js", StartLine: 1, EndLine: 10}}
	clone3 := &domain.Clone{ID: 3, Location: &domain.CloneLocation{FilePath: "c.js", StartLine: 1, EndLine: 10}}

	pairs := []*domain.ClonePair{
		{ID: 1, Clone1: clone1, Clone2: clone2, Similarity: 0.9, Type: domain.Type1Clone},
		{ID: 2, Clone1: clone2, Clone2: clone3, Similarity: 0.9, Type: domain.Type1Clone},
		{ID: 3, Clone1: clone1, Clone2: clone3, Similarity: 0.9, Type: domain.Type1Clone},
	}

	groups := grouping.GroupClones(pairs)

	if len(groups) != 1 {
		t.Errorf("Expected 1 group (triangle is 2-core), got %d", len(groups))
	}
	if len(groups) > 0 && groups[0].Size != 3 {
		t.Errorf("Expected group size 3, got %d", groups[0].Size)
	}
}

func TestKCoreGroupingRemovesLowDegree(t *testing.T) {
	grouping := NewKCoreGrouping(0.8, 2)

	// Clone 1 only connected to clone 2 (degree 1, will be removed)
	// Clone 2 connected to 1, 3, 4 (degree 3, will stay)
	// Clone 3 connected to 2, 4 (degree 2, will stay)
	// Clone 4 connected to 2, 3 (degree 2, will stay)
	// After removing clone 1, we have a triangle: 2-3-4

	clone1 := &domain.Clone{ID: 1, Location: &domain.CloneLocation{FilePath: "a.js", StartLine: 1, EndLine: 10}}
	clone2 := &domain.Clone{ID: 2, Location: &domain.CloneLocation{FilePath: "b.js", StartLine: 1, EndLine: 10}}
	clone3 := &domain.Clone{ID: 3, Location: &domain.CloneLocation{FilePath: "c.js", StartLine: 1, EndLine: 10}}
	clone4 := &domain.Clone{ID: 4, Location: &domain.CloneLocation{FilePath: "d.js", StartLine: 1, EndLine: 10}}

	pairs := []*domain.ClonePair{
		{ID: 1, Clone1: clone1, Clone2: clone2, Similarity: 0.9, Type: domain.Type1Clone},
		{ID: 2, Clone1: clone2, Clone2: clone3, Similarity: 0.9, Type: domain.Type1Clone},
		{ID: 3, Clone1: clone2, Clone2: clone4, Similarity: 0.9, Type: domain.Type1Clone},
		{ID: 4, Clone1: clone3, Clone2: clone4, Similarity: 0.9, Type: domain.Type1Clone},
	}

	groups := grouping.GroupClones(pairs)

	if len(groups) != 1 {
		t.Errorf("Expected 1 group, got %d", len(groups))
	}
	if len(groups) > 0 && groups[0].Size != 3 {
		t.Errorf("Expected group size 3 (clone 1 should be removed), got %d", groups[0].Size)
	}
}

func TestClonePairKey(t *testing.T) {
	clone1 := &domain.Clone{
		ID:       1,
		Location: &domain.CloneLocation{FilePath: "a.js", StartLine: 1, EndLine: 10, StartCol: 0, EndCol: 50},
	}
	clone2 := &domain.Clone{
		ID:       2,
		Location: &domain.CloneLocation{FilePath: "b.js", StartLine: 1, EndLine: 10, StartCol: 0, EndCol: 50},
	}

	key1 := clonePairKey(clone1, clone2)
	key2 := clonePairKey(clone2, clone1)

	// Key should be canonical (same regardless of order)
	if key1 != key2 {
		t.Errorf("Keys should be identical regardless of order: %s vs %s", key1, key2)
	}
}

func TestCloneID(t *testing.T) {
	clone := &domain.Clone{
		ID:       1,
		Location: &domain.CloneLocation{FilePath: "test.js", StartLine: 1, EndLine: 10, StartCol: 0, EndCol: 50},
	}

	id := cloneID(clone)

	expected := "test.js|1|10|0|50"
	if id != expected {
		t.Errorf("Expected id %s, got %s", expected, id)
	}
}

func TestCloneIDNil(t *testing.T) {
	// Test nil clone
	id := cloneID(nil)
	if id == "" {
		t.Error("ID for nil clone should not be empty")
	}

	// Test clone with nil location
	clone := &domain.Clone{ID: 1}
	id = cloneID(clone)
	if id == "" {
		t.Error("ID for clone with nil location should not be empty")
	}
}

func TestCloneLess(t *testing.T) {
	clone1 := &domain.Clone{
		ID:       1,
		Location: &domain.CloneLocation{FilePath: "a.js", StartLine: 1, EndLine: 10},
	}
	clone2 := &domain.Clone{
		ID:       2,
		Location: &domain.CloneLocation{FilePath: "b.js", StartLine: 1, EndLine: 10},
	}
	clone3 := &domain.Clone{
		ID:       3,
		Location: &domain.CloneLocation{FilePath: "a.js", StartLine: 20, EndLine: 30},
	}

	// a.js < b.js
	if !cloneLess(clone1, clone2) {
		t.Error("clone1 should be less than clone2 (file path)")
	}

	// Same file, different start line
	if !cloneLess(clone1, clone3) {
		t.Error("clone1 should be less than clone3 (start line)")
	}

	// Same element
	if cloneLess(clone1, clone1) {
		t.Error("clone should not be less than itself")
	}

	// Nil handling
	if !cloneLess(nil, clone1) {
		t.Error("nil should be less than non-nil")
	}
	if cloneLess(clone1, nil) {
		t.Error("non-nil should not be less than nil")
	}
}

func TestAlmostEqual(t *testing.T) {
	if !almostEqual(1.0, 1.0) {
		t.Error("1.0 should equal 1.0")
	}
	if !almostEqual(1.0, 1.0+1e-10) {
		t.Error("1.0 should be almost equal to 1.0+1e-10")
	}
	if almostEqual(1.0, 1.1) {
		t.Error("1.0 should not be almost equal to 1.1")
	}
}

func TestAverageGroupSimilarityClones(t *testing.T) {
	clone1 := &domain.Clone{ID: 1, Location: &domain.CloneLocation{FilePath: "a.js", StartLine: 1, EndLine: 10}}
	clone2 := &domain.Clone{ID: 2, Location: &domain.CloneLocation{FilePath: "b.js", StartLine: 1, EndLine: 10}}
	clone3 := &domain.Clone{ID: 3, Location: &domain.CloneLocation{FilePath: "c.js", StartLine: 1, EndLine: 10}}

	sims := map[string]float64{
		clonePairKey(clone1, clone2): 0.9,
		clonePairKey(clone2, clone3): 0.8,
		clonePairKey(clone1, clone3): 0.85,
	}

	avg := averageGroupSimilarityClones(sims, []*domain.Clone{clone1, clone2, clone3})

	// Average of 0.9, 0.8, 0.85 = 0.85
	expected := (0.9 + 0.8 + 0.85) / 3.0
	if !almostEqual(avg, expected) {
		t.Errorf("Expected average %f, got %f", expected, avg)
	}
}

func TestAverageGroupSimilarityClonesEmpty(t *testing.T) {
	avg := averageGroupSimilarityClones(nil, []*domain.Clone{})
	if avg != 1.0 {
		t.Errorf("Expected 1.0 for empty/single member, got %f", avg)
	}
}

func TestMajorityCloneTypeClones(t *testing.T) {
	clone1 := &domain.Clone{ID: 1, Location: &domain.CloneLocation{FilePath: "a.js"}}
	clone2 := &domain.Clone{ID: 2, Location: &domain.CloneLocation{FilePath: "b.js"}}
	clone3 := &domain.Clone{ID: 3, Location: &domain.CloneLocation{FilePath: "c.js"}}

	typeMap := map[string]domain.CloneType{
		clonePairKey(clone1, clone2): domain.Type2Clone,
		clonePairKey(clone2, clone3): domain.Type2Clone,
		clonePairKey(clone1, clone3): domain.Type1Clone,
	}

	majority := majorityCloneTypeClones(typeMap, []*domain.Clone{clone1, clone2, clone3})

	// Type2Clone appears twice, Type1Clone once
	if majority != domain.Type2Clone {
		t.Errorf("Expected Type2Clone as majority, got %v", majority)
	}
}

func TestMajorityCloneTypeClonesEmpty(t *testing.T) {
	majority := majorityCloneTypeClones(nil, []*domain.Clone{})

	// Conservative default fallback: never report unknown as Type-1
	if majority != domain.Type4Clone {
		t.Errorf("Expected Type4Clone as fallback, got %v", majority)
	}
}

// StarMedoidGrouping tests

func TestNewStarMedoidGrouping(t *testing.T) {
	grouping := NewStarMedoidGrouping(0.8)

	if grouping.threshold != 0.8 {
		t.Errorf("Expected threshold 0.8, got %f", grouping.threshold)
	}
	if grouping.GetName() != "Star/Medoid" {
		t.Errorf("Expected name 'Star/Medoid', got %s", grouping.GetName())
	}
}

func TestStarMedoidGroupingEmpty(t *testing.T) {
	grouping := NewStarMedoidGrouping(0.8)
	groups := grouping.GroupClones([]*domain.ClonePair{})

	if len(groups) != 0 {
		t.Errorf("Expected 0 groups for empty input, got %d", len(groups))
	}
}

func TestStarMedoidGroupingSinglePair(t *testing.T) {
	grouping := NewStarMedoidGrouping(0.8)

	clone1 := &domain.Clone{
		ID:       1,
		Location: &domain.CloneLocation{FilePath: "a.js", StartLine: 1, EndLine: 10},
	}
	clone2 := &domain.Clone{
		ID:       2,
		Location: &domain.CloneLocation{FilePath: "b.js", StartLine: 1, EndLine: 10},
	}

	pairs := []*domain.ClonePair{
		{
			ID:         1,
			Clone1:     clone1,
			Clone2:     clone2,
			Similarity: 0.9,
			Type:       domain.Type1Clone,
		},
	}

	groups := grouping.GroupClones(pairs)

	if len(groups) != 1 {
		t.Errorf("Expected 1 group, got %d", len(groups))
	}
	if groups[0].Size != 2 {
		t.Errorf("Expected group size 2, got %d", groups[0].Size)
	}
}

func TestStarMedoidGroupingMedoidSelection(t *testing.T) {
	grouping := NewStarMedoidGrouping(0.7)

	// Clone 2 should be medoid as it has highest average similarity to others
	clone1 := &domain.Clone{ID: 1, Location: &domain.CloneLocation{FilePath: "a.js", StartLine: 1, EndLine: 10}}
	clone2 := &domain.Clone{ID: 2, Location: &domain.CloneLocation{FilePath: "b.js", StartLine: 1, EndLine: 10}}
	clone3 := &domain.Clone{ID: 3, Location: &domain.CloneLocation{FilePath: "c.js", StartLine: 1, EndLine: 10}}

	pairs := []*domain.ClonePair{
		{ID: 1, Clone1: clone1, Clone2: clone2, Similarity: 0.9, Type: domain.Type1Clone},
		{ID: 2, Clone1: clone2, Clone2: clone3, Similarity: 0.95, Type: domain.Type1Clone},
		{ID: 3, Clone1: clone1, Clone2: clone3, Similarity: 0.75, Type: domain.Type2Clone},
	}

	groups := grouping.GroupClones(pairs)

	if len(groups) != 1 {
		t.Errorf("Expected 1 group, got %d", len(groups))
	}
	if len(groups) > 0 && groups[0].Size != 3 {
		t.Errorf("Expected group size 3, got %d", groups[0].Size)
	}
}

func TestStarMedoidGroupingBelowThreshold(t *testing.T) {
	grouping := NewStarMedoidGrouping(0.9)

	clone1 := &domain.Clone{ID: 1, Location: &domain.CloneLocation{FilePath: "a.js", StartLine: 1, EndLine: 10}}
	clone2 := &domain.Clone{ID: 2, Location: &domain.CloneLocation{FilePath: "b.js", StartLine: 1, EndLine: 10}}

	pairs := []*domain.ClonePair{
		{ID: 1, Clone1: clone1, Clone2: clone2, Similarity: 0.8, Type: domain.Type2Clone},
	}

	groups := grouping.GroupClones(pairs)

	if len(groups) != 0 {
		t.Errorf("Expected 0 groups (below threshold), got %d", len(groups))
	}
}

// CompleteLinkageGrouping tests

func TestNewCompleteLinkageGrouping(t *testing.T) {
	grouping := NewCompleteLinkageGrouping(0.8)

	if grouping.threshold != 0.8 {
		t.Errorf("Expected threshold 0.8, got %f", grouping.threshold)
	}
	if grouping.GetName() != "Complete Linkage" {
		t.Errorf("Expected name 'Complete Linkage', got %s", grouping.GetName())
	}
}

func TestCompleteLinkageGroupingEmpty(t *testing.T) {
	grouping := NewCompleteLinkageGrouping(0.8)
	groups := grouping.GroupClones([]*domain.ClonePair{})

	if len(groups) != 0 {
		t.Errorf("Expected 0 groups for empty input, got %d", len(groups))
	}
}

func TestCompleteLinkageGroupingTriangle(t *testing.T) {
	grouping := NewCompleteLinkageGrouping(0.8)

	// Three clones forming a complete triangle (clique)
	clone1 := &domain.Clone{ID: 1, Location: &domain.CloneLocation{FilePath: "a.js", StartLine: 1, EndLine: 10}}
	clone2 := &domain.Clone{ID: 2, Location: &domain.CloneLocation{FilePath: "b.js", StartLine: 1, EndLine: 10}}
	clone3 := &domain.Clone{ID: 3, Location: &domain.CloneLocation{FilePath: "c.js", StartLine: 1, EndLine: 10}}

	pairs := []*domain.ClonePair{
		{ID: 1, Clone1: clone1, Clone2: clone2, Similarity: 0.9, Type: domain.Type1Clone},
		{ID: 2, Clone1: clone2, Clone2: clone3, Similarity: 0.9, Type: domain.Type1Clone},
		{ID: 3, Clone1: clone1, Clone2: clone3, Similarity: 0.9, Type: domain.Type1Clone},
	}

	groups := grouping.GroupClones(pairs)

	if len(groups) != 1 {
		t.Errorf("Expected 1 group (triangle is a clique), got %d", len(groups))
	}
	if len(groups) > 0 && groups[0].Size != 3 {
		t.Errorf("Expected group size 3, got %d", groups[0].Size)
	}
}

func TestCompleteLinkageGroupingNonClique(t *testing.T) {
	grouping := NewCompleteLinkageGrouping(0.8)

	// Three clones but only 2 edges: 1-2, 2-3 (not a complete clique)
	clone1 := &domain.Clone{ID: 1, Location: &domain.CloneLocation{FilePath: "a.js", StartLine: 1, EndLine: 10}}
	clone2 := &domain.Clone{ID: 2, Location: &domain.CloneLocation{FilePath: "b.js", StartLine: 1, EndLine: 10}}
	clone3 := &domain.Clone{ID: 3, Location: &domain.CloneLocation{FilePath: "c.js", StartLine: 1, EndLine: 10}}

	pairs := []*domain.ClonePair{
		{ID: 1, Clone1: clone1, Clone2: clone2, Similarity: 0.9, Type: domain.Type1Clone},
		{ID: 2, Clone1: clone2, Clone2: clone3, Similarity: 0.9, Type: domain.Type1Clone},
		// Missing clone1-clone3 edge
	}

	groups := grouping.GroupClones(pairs)

	// Complete-linkage clustering merges one pair first; the chained clone
	// cannot join because it lacks an edge to every member. One group remains.
	if len(groups) != 1 {
		t.Errorf("Expected 1 group (complete-linkage merges one pair), got %d", len(groups))
	}
	if len(groups) > 0 && groups[0].Size != 2 {
		t.Errorf("Expected group size 2, got %d", groups[0].Size)
	}
}

func TestCompleteLinkageGroupingBelowThreshold(t *testing.T) {
	grouping := NewCompleteLinkageGrouping(0.9)

	clone1 := &domain.Clone{ID: 1, Location: &domain.CloneLocation{FilePath: "a.js", StartLine: 1, EndLine: 10}}
	clone2 := &domain.Clone{ID: 2, Location: &domain.CloneLocation{FilePath: "b.js", StartLine: 1, EndLine: 10}}

	pairs := []*domain.ClonePair{
		{ID: 1, Clone1: clone1, Clone2: clone2, Similarity: 0.8, Type: domain.Type2Clone},
	}

	groups := grouping.GroupClones(pairs)

	if len(groups) != 0 {
		t.Errorf("Expected 0 groups (below threshold), got %d", len(groups))
	}
}

// CentroidGrouping tests

func TestNewCentroidGrouping(t *testing.T) {
	grouping := NewCentroidGrouping(0.8)

	if grouping.threshold != 0.8 {
		t.Errorf("Expected threshold 0.8, got %f", grouping.threshold)
	}
	if grouping.GetName() != "Centroid" {
		t.Errorf("Expected name 'Centroid', got %s", grouping.GetName())
	}
}

func TestCentroidGroupingEmpty(t *testing.T) {
	grouping := NewCentroidGrouping(0.8)
	groups := grouping.GroupClones([]*domain.ClonePair{})

	if len(groups) != 0 {
		t.Errorf("Expected 0 groups for empty input, got %d", len(groups))
	}
}

func TestCentroidGroupingSinglePair(t *testing.T) {
	grouping := NewCentroidGrouping(0.8)

	clone1 := &domain.Clone{
		ID:       1,
		Location: &domain.CloneLocation{FilePath: "a.js", StartLine: 1, EndLine: 10},
	}
	clone2 := &domain.Clone{
		ID:       2,
		Location: &domain.CloneLocation{FilePath: "b.js", StartLine: 1, EndLine: 10},
	}

	pairs := []*domain.ClonePair{
		{
			ID:         1,
			Clone1:     clone1,
			Clone2:     clone2,
			Similarity: 0.9,
			Type:       domain.Type1Clone,
		},
	}

	groups := grouping.GroupClones(pairs)

	if len(groups) != 1 {
		t.Errorf("Expected 1 group, got %d", len(groups))
	}
	if groups[0].Size != 2 {
		t.Errorf("Expected group size 2, got %d", groups[0].Size)
	}
}

func TestCentroidGroupingTransitivityRejection(t *testing.T) {
	grouping := NewCentroidGrouping(0.8)

	// A~B, B~C but A is NOT similar to C
	// Centroid should reject C from joining the group with A and B
	clone1 := &domain.Clone{ID: 1, Location: &domain.CloneLocation{FilePath: "a.js", StartLine: 1, EndLine: 10}}
	clone2 := &domain.Clone{ID: 2, Location: &domain.CloneLocation{FilePath: "b.js", StartLine: 1, EndLine: 10}}
	clone3 := &domain.Clone{ID: 3, Location: &domain.CloneLocation{FilePath: "c.js", StartLine: 1, EndLine: 10}}

	pairs := []*domain.ClonePair{
		{ID: 1, Clone1: clone1, Clone2: clone2, Similarity: 0.9, Type: domain.Type1Clone},
		{ID: 2, Clone1: clone2, Clone2: clone3, Similarity: 0.9, Type: domain.Type1Clone},
		{ID: 3, Clone1: clone1, Clone2: clone3, Similarity: 0.5, Type: domain.Type3Clone}, // Below threshold
	}

	groups := grouping.GroupClones(pairs)

	// Clone 3 should NOT be in the same group as clone 1 because similarity(1,3) < threshold
	// Depending on processing order, we may have different results, but clone 1-2-3 should NOT be in one group
	for _, g := range groups {
		if g.Size == 3 {
			t.Errorf("Expected no group of size 3 due to transitive rejection, but found one")
		}
	}
}

func TestCentroidGroupingAllSimilar(t *testing.T) {
	grouping := NewCentroidGrouping(0.8)

	// All clones similar to each other (complete clique)
	clone1 := &domain.Clone{ID: 1, Location: &domain.CloneLocation{FilePath: "a.js", StartLine: 1, EndLine: 10}}
	clone2 := &domain.Clone{ID: 2, Location: &domain.CloneLocation{FilePath: "b.js", StartLine: 1, EndLine: 10}}
	clone3 := &domain.Clone{ID: 3, Location: &domain.CloneLocation{FilePath: "c.js", StartLine: 1, EndLine: 10}}

	pairs := []*domain.ClonePair{
		{ID: 1, Clone1: clone1, Clone2: clone2, Similarity: 0.9, Type: domain.Type1Clone},
		{ID: 2, Clone1: clone2, Clone2: clone3, Similarity: 0.9, Type: domain.Type1Clone},
		{ID: 3, Clone1: clone1, Clone2: clone3, Similarity: 0.85, Type: domain.Type2Clone},
	}

	groups := grouping.GroupClones(pairs)

	if len(groups) != 1 {
		t.Errorf("Expected 1 group, got %d", len(groups))
	}
	if len(groups) > 0 && groups[0].Size != 3 {
		t.Errorf("Expected group size 3, got %d", groups[0].Size)
	}
}

func TestCentroidGroupingBelowThreshold(t *testing.T) {
	grouping := NewCentroidGrouping(0.9)

	clone1 := &domain.Clone{ID: 1, Location: &domain.CloneLocation{FilePath: "a.js", StartLine: 1, EndLine: 10}}
	clone2 := &domain.Clone{ID: 2, Location: &domain.CloneLocation{FilePath: "b.js", StartLine: 1, EndLine: 10}}

	pairs := []*domain.ClonePair{
		{ID: 1, Clone1: clone1, Clone2: clone2, Similarity: 0.8, Type: domain.Type2Clone},
	}

	groups := grouping.GroupClones(pairs)

	if len(groups) != 0 {
		t.Errorf("Expected 0 groups (below threshold), got %d", len(groups))
	}
}
