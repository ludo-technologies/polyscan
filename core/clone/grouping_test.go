package clone

import (
	"testing"

	"github.com/ludo-technologies/polyscan/core/domain"
)

// testItem is a minimal GroupableItem implementation for tests.
type testItem struct {
	id  int
	loc ItemLocation
}

func (t *testItem) ItemID() int                { return t.id }
func (t *testItem) ItemLocation() ItemLocation { return t.loc }

func ti(id int, file string, startLine, endLine int) *testItem {
	return &testItem{id: id, loc: ItemLocation{FilePath: file, StartLine: startLine, EndLine: endLine}}
}

func pair(id1, id2 *testItem, sim float64, t domain.CloneType) *ItemPair[*testItem] {
	return &ItemPair[*testItem]{Item1: id1, Item2: id2, Similarity: sim, PairType: t}
}

func TestNewGroupingStrategy(t *testing.T) {
	testCases := []struct {
		mode     GroupingMode
		expected string
	}{
		{ModeConnected, "connected"},
		{ModeKCore, "k_core"},
		{ModeStarMedoid, "star_medoid"},
		{ModeCompleteLinkage, "complete_linkage"},
		{ModeCentroid, "centroid"},
		{"unknown", "connected"}, // Default fallback
	}

	for _, tc := range testCases {
		t.Run(string(tc.mode), func(t *testing.T) {
			config := GroupingConfig{
				Mode:      tc.mode,
				Threshold: 0.8,
				KCoreK:    2,
			}
			strategy := NewGroupingStrategy[*testItem](config)
			if strategy.Name() != tc.expected {
				t.Errorf("Expected strategy name %s, got %s", tc.expected, strategy.Name())
			}
		})
	}
}

func TestConnectedGroupingEmpty(t *testing.T) {
	grouping := NewConnectedGrouping[*testItem](0.8)
	groups := grouping.GroupItems([]*ItemPair[*testItem]{})

	if len(groups) != 0 {
		t.Errorf("Expected 0 groups for empty input, got %d", len(groups))
	}
}

func TestConnectedGroupingSinglePair(t *testing.T) {
	grouping := NewConnectedGrouping[*testItem](0.8)

	item1 := ti(1, "a.js", 1, 10)
	item2 := ti(2, "b.js", 1, 10)

	groups := grouping.GroupItems([]*ItemPair[*testItem]{
		pair(item1, item2, 0.9, domain.Type1Clone),
	})

	if len(groups) != 1 {
		t.Fatalf("Expected 1 group, got %d", len(groups))
	}
	if len(groups[0].Items) != 2 {
		t.Errorf("Expected group size 2, got %d", len(groups[0].Items))
	}
}

func TestConnectedGroupingBelowThreshold(t *testing.T) {
	grouping := NewConnectedGrouping[*testItem](0.9)

	item1 := ti(1, "a.js", 1, 10)
	item2 := ti(2, "b.js", 1, 10)

	groups := grouping.GroupItems([]*ItemPair[*testItem]{
		pair(item1, item2, 0.8, domain.Type2Clone), // Below threshold
	})

	if len(groups) != 0 {
		t.Errorf("Expected 0 groups (below threshold), got %d", len(groups))
	}
}

func TestConnectedGroupingMultipleComponents(t *testing.T) {
	grouping := NewConnectedGrouping[*testItem](0.8)

	// First component: items 1 and 2. Second component: items 3 and 4.
	item1 := ti(1, "a.js", 1, 10)
	item2 := ti(2, "a.js", 20, 30)
	item3 := ti(3, "b.js", 1, 10)
	item4 := ti(4, "b.js", 20, 30)

	groups := grouping.GroupItems([]*ItemPair[*testItem]{
		pair(item1, item2, 0.9, domain.Type1Clone),
		pair(item3, item4, 0.85, domain.Type2Clone),
	})

	if len(groups) != 2 {
		t.Errorf("Expected 2 groups, got %d", len(groups))
	}
}

func TestConnectedGroupingSortsMembersDeterministically(t *testing.T) {
	grouping := NewConnectedGrouping[*testItem](0.8)

	item1 := ti(1, "b.js", 1, 10)
	item2 := ti(2, "a.js", 1, 10)

	groups := grouping.GroupItems([]*ItemPair[*testItem]{
		pair(item1, item2, 0.9, domain.Type1Clone),
	})

	if len(groups) != 1 {
		t.Fatalf("Expected 1 group, got %d", len(groups))
	}
	if groups[0].Items[0] != item2 {
		t.Error("expected members sorted by location (a.js before b.js)")
	}
}

func TestNewKCoreGroupingMinK(t *testing.T) {
	grouping := NewKCoreGrouping[*testItem](0.8, 1) // k=1 should be bumped to 2

	if grouping.k != 2 {
		t.Errorf("Expected k=2 (minimum), got %d", grouping.k)
	}
}

func TestKCoreGroupingEmpty(t *testing.T) {
	grouping := NewKCoreGrouping[*testItem](0.8, 2)
	groups := grouping.GroupItems([]*ItemPair[*testItem]{})

	if len(groups) != 0 {
		t.Errorf("Expected 0 groups for empty input, got %d", len(groups))
	}
}

func TestKCoreGroupingTriangle(t *testing.T) {
	grouping := NewKCoreGrouping[*testItem](0.8, 2)

	// Three items forming a triangle (each connected to 2 others)
	item1 := ti(1, "a.js", 1, 10)
	item2 := ti(2, "b.js", 1, 10)
	item3 := ti(3, "c.js", 1, 10)

	groups := grouping.GroupItems([]*ItemPair[*testItem]{
		pair(item1, item2, 0.9, domain.Type1Clone),
		pair(item2, item3, 0.9, domain.Type1Clone),
		pair(item1, item3, 0.9, domain.Type1Clone),
	})

	if len(groups) != 1 {
		t.Fatalf("Expected 1 group (triangle is 2-core), got %d", len(groups))
	}
	if len(groups[0].Items) != 3 {
		t.Errorf("Expected group size 3, got %d", len(groups[0].Items))
	}
}

func TestKCoreGroupingRemovesLowDegree(t *testing.T) {
	grouping := NewKCoreGrouping[*testItem](0.8, 2)

	// Item 1 only connected to item 2 (degree 1, will be removed).
	// After removing item 1, we have a triangle: 2-3-4.
	item1 := ti(1, "a.js", 1, 10)
	item2 := ti(2, "b.js", 1, 10)
	item3 := ti(3, "c.js", 1, 10)
	item4 := ti(4, "d.js", 1, 10)

	groups := grouping.GroupItems([]*ItemPair[*testItem]{
		pair(item1, item2, 0.9, domain.Type1Clone),
		pair(item2, item3, 0.9, domain.Type1Clone),
		pair(item2, item4, 0.9, domain.Type1Clone),
		pair(item3, item4, 0.9, domain.Type1Clone),
	})

	if len(groups) != 1 {
		t.Fatalf("Expected 1 group, got %d", len(groups))
	}
	if len(groups[0].Items) != 3 {
		t.Errorf("Expected group size 3 (item 1 should be removed), got %d", len(groups[0].Items))
	}
}

func TestPairKeyCanonical(t *testing.T) {
	item1 := &testItem{id: 1, loc: ItemLocation{FilePath: "a.js", StartLine: 1, EndLine: 10, EndCol: 50}}
	item2 := &testItem{id: 2, loc: ItemLocation{FilePath: "b.js", StartLine: 1, EndLine: 10, EndCol: 50}}

	key1 := PairKey(item1, item2)
	key2 := PairKey(item2, item1)

	// Key should be canonical (same regardless of order)
	if key1 != key2 {
		t.Errorf("Keys should be identical regardless of order: %s vs %s", key1, key2)
	}
}

func TestItemKey(t *testing.T) {
	item := &testItem{id: 1, loc: ItemLocation{FilePath: "test.js", StartLine: 1, EndLine: 10, StartCol: 0, EndCol: 50}}

	key := ItemKey(item)

	expected := "test.js|1|10|0|50"
	if key != expected {
		t.Errorf("Expected key %s, got %s", expected, key)
	}
}

func TestItemLess(t *testing.T) {
	item1 := ti(1, "a.js", 1, 10)
	item2 := ti(2, "b.js", 1, 10)
	item3 := ti(3, "a.js", 20, 30)

	// a.js < b.js
	if !itemLess(item1, item2) {
		t.Error("item1 should be less than item2 (file path)")
	}

	// Same file, different start line
	if !itemLess(item1, item3) {
		t.Error("item1 should be less than item3 (start line)")
	}

	// Same element
	if itemLess(item1, item1) {
		t.Error("item should not be less than itself")
	}

	// Equal locations fall back to ItemID
	item4 := ti(4, "a.js", 1, 10)
	if !itemLess(item1, item4) {
		t.Error("equal locations should order by ItemID")
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

func TestAverageGroupSimilarity(t *testing.T) {
	item1 := ti(1, "a.js", 1, 10)
	item2 := ti(2, "b.js", 1, 10)
	item3 := ti(3, "c.js", 1, 10)

	sims := map[string]float64{
		PairKey(item1, item2): 0.9,
		PairKey(item2, item3): 0.8,
		PairKey(item1, item3): 0.85,
	}

	avg := averageGroupSimilarity(sims, []*testItem{item1, item2, item3})

	expected := (0.9 + 0.8 + 0.85) / 3.0
	if !almostEqual(avg, expected) {
		t.Errorf("Expected average %f, got %f", expected, avg)
	}
}

func TestAverageGroupSimilarityEmpty(t *testing.T) {
	avg := averageGroupSimilarity(nil, []*testItem{})
	if avg != 1.0 {
		t.Errorf("Expected 1.0 for empty/single member, got %f", avg)
	}
}

func TestAverageGroupSimilaritySkipsMissingPairs(t *testing.T) {
	item1 := ti(1, "a.js", 1, 10)
	item2 := ti(2, "b.js", 1, 10)
	item3 := ti(3, "c.js", 1, 10)

	// Only one of three member pairs has a cached similarity; missing pairs
	// must be skipped rather than counted as 0.
	sims := map[string]float64{
		PairKey(item1, item2): 0.9,
	}

	avg := averageGroupSimilarity(sims, []*testItem{item1, item2, item3})
	if !almostEqual(avg, 0.9) {
		t.Errorf("Expected 0.9 (missing pairs skipped), got %f", avg)
	}
}

func TestMajorityType_TieBreaksDeterministically(t *testing.T) {
	item1 := ti(1, "a.js", 0, 0)
	item2 := ti(2, "b.js", 0, 0)
	item3 := ti(3, "c.js", 0, 0)

	typeMap := map[string]domain.CloneType{
		PairKey(item1, item2): domain.Type2Clone,
		PairKey(item2, item3): domain.Type2Clone,
		PairKey(item1, item3): domain.Type1Clone,
	}
	simMap := map[string]float64{
		PairKey(item1, item2): 0.9,
		PairKey(item2, item3): 0.9,
		PairKey(item1, item3): 0.9,
	}

	majority := majorityType(typeMap, simMap, []*testItem{item1, item2, item3})

	// All pairs tie on similarity, so the strictest (lowest) type wins deterministically.
	if majority != domain.Type1Clone {
		t.Errorf("Expected Type1Clone deterministic tie break, got %v", majority)
	}
}

// TestMajorityType_PrefersHighSimilarityPair ensures a connected component
// whose strongest edge is a high-similarity Type-2 pair is reported as Type-2
// even when weaker Type-3 transitive edges outnumber it.
func TestMajorityType_PrefersHighSimilarityPair(t *testing.T) {
	item1 := ti(1, "a.js", 0, 0)
	item2 := ti(2, "b.js", 0, 0)
	item3 := ti(3, "c.js", 0, 0)
	item4 := ti(4, "d.js", 0, 0)

	members := []*testItem{item1, item2, item3, item4}
	typeMap := map[string]domain.CloneType{
		PairKey(item1, item2): domain.Type2Clone,
		PairKey(item1, item3): domain.Type3Clone,
		PairKey(item1, item4): domain.Type3Clone,
		PairKey(item2, item3): domain.Type3Clone,
		PairKey(item2, item4): domain.Type3Clone,
		PairKey(item3, item4): domain.Type3Clone,
	}
	simMap := map[string]float64{
		PairKey(item1, item2): 0.96, // high-sim Type-2 pair
		PairKey(item1, item3): 0.85,
		PairKey(item1, item4): 0.85,
		PairKey(item2, item3): 0.85,
		PairKey(item2, item4): 0.85,
		PairKey(item3, item4): 0.85,
	}

	majority := majorityType(typeMap, simMap, members)
	if majority != domain.Type2Clone {
		t.Errorf("Expected Type2Clone from the highest-similarity edge, got %v", majority)
	}
}

func TestMajorityTypeEmpty(t *testing.T) {
	majority := majorityType(nil, nil, []*testItem{})

	// Conservative default fallback: never report unknown as Type-1
	if majority != domain.Type4Clone {
		t.Errorf("Expected Type4Clone as fallback, got %v", majority)
	}
}

// StarMedoidGrouping tests

func TestStarMedoidGroupingEmpty(t *testing.T) {
	grouping := NewStarMedoidGrouping[*testItem](0.8)
	groups := grouping.GroupItems([]*ItemPair[*testItem]{})

	if len(groups) != 0 {
		t.Errorf("Expected 0 groups for empty input, got %d", len(groups))
	}
}

func TestStarMedoidGroupingSinglePair(t *testing.T) {
	grouping := NewStarMedoidGrouping[*testItem](0.8)

	item1 := ti(1, "a.js", 1, 10)
	item2 := ti(2, "b.js", 1, 10)

	groups := grouping.GroupItems([]*ItemPair[*testItem]{
		pair(item1, item2, 0.9, domain.Type1Clone),
	})

	if len(groups) != 1 {
		t.Fatalf("Expected 1 group, got %d", len(groups))
	}
	if len(groups[0].Items) != 2 {
		t.Errorf("Expected group size 2, got %d", len(groups[0].Items))
	}
}

func TestStarMedoidGroupingMedoidSelection(t *testing.T) {
	grouping := NewStarMedoidGrouping[*testItem](0.7)

	// Item 2 should be medoid as it has highest average similarity to others
	item1 := ti(1, "a.js", 1, 10)
	item2 := ti(2, "b.js", 1, 10)
	item3 := ti(3, "c.js", 1, 10)

	groups := grouping.GroupItems([]*ItemPair[*testItem]{
		pair(item1, item2, 0.9, domain.Type1Clone),
		pair(item2, item3, 0.95, domain.Type1Clone),
		pair(item1, item3, 0.75, domain.Type2Clone),
	})

	if len(groups) != 1 {
		t.Fatalf("Expected 1 group, got %d", len(groups))
	}
	if len(groups[0].Items) != 3 {
		t.Errorf("Expected group size 3, got %d", len(groups[0].Items))
	}
}

func TestStarMedoidGroupingBelowThreshold(t *testing.T) {
	grouping := NewStarMedoidGrouping[*testItem](0.9)

	item1 := ti(1, "a.js", 1, 10)
	item2 := ti(2, "b.js", 1, 10)

	groups := grouping.GroupItems([]*ItemPair[*testItem]{
		pair(item1, item2, 0.8, domain.Type2Clone),
	})

	if len(groups) != 0 {
		t.Errorf("Expected 0 groups (below threshold), got %d", len(groups))
	}
}

// CompleteLinkageGrouping tests

func TestCompleteLinkageGroupingEmpty(t *testing.T) {
	grouping := NewCompleteLinkageGrouping[*testItem](0.8)
	groups := grouping.GroupItems([]*ItemPair[*testItem]{})

	if len(groups) != 0 {
		t.Errorf("Expected 0 groups for empty input, got %d", len(groups))
	}
}

func TestCompleteLinkageGroupingTriangle(t *testing.T) {
	grouping := NewCompleteLinkageGrouping[*testItem](0.8)

	// Three items forming a complete triangle (clique)
	item1 := ti(1, "a.js", 1, 10)
	item2 := ti(2, "b.js", 1, 10)
	item3 := ti(3, "c.js", 1, 10)

	groups := grouping.GroupItems([]*ItemPair[*testItem]{
		pair(item1, item2, 0.9, domain.Type1Clone),
		pair(item2, item3, 0.9, domain.Type1Clone),
		pair(item1, item3, 0.9, domain.Type1Clone),
	})

	if len(groups) != 1 {
		t.Fatalf("Expected 1 group (triangle is a clique), got %d", len(groups))
	}
	if len(groups[0].Items) != 3 {
		t.Errorf("Expected group size 3, got %d", len(groups[0].Items))
	}
}

func TestCompleteLinkageGroupingNonClique(t *testing.T) {
	grouping := NewCompleteLinkageGrouping[*testItem](0.8)

	// Three items but only 2 edges: 1-2, 2-3 (not a complete clique)
	item1 := ti(1, "a.js", 1, 10)
	item2 := ti(2, "b.js", 1, 10)
	item3 := ti(3, "c.js", 1, 10)

	groups := grouping.GroupItems([]*ItemPair[*testItem]{
		pair(item1, item2, 0.9, domain.Type1Clone),
		pair(item2, item3, 0.9, domain.Type1Clone),
		// Missing item1-item3 edge
	})

	// Complete-linkage clustering merges one pair first; the chained item
	// cannot join because it lacks an edge to every member. One group remains.
	if len(groups) != 1 {
		t.Fatalf("Expected 1 group (complete-linkage merges one pair), got %d", len(groups))
	}
	if len(groups[0].Items) != 2 {
		t.Errorf("Expected group size 2, got %d", len(groups[0].Items))
	}
}

func TestCompleteLinkageGroupingBelowThreshold(t *testing.T) {
	grouping := NewCompleteLinkageGrouping[*testItem](0.9)

	item1 := ti(1, "a.js", 1, 10)
	item2 := ti(2, "b.js", 1, 10)

	groups := grouping.GroupItems([]*ItemPair[*testItem]{
		pair(item1, item2, 0.8, domain.Type2Clone),
	})

	if len(groups) != 0 {
		t.Errorf("Expected 0 groups (below threshold), got %d", len(groups))
	}
}

// CentroidGrouping tests

func TestCentroidGroupingEmpty(t *testing.T) {
	grouping := NewCentroidGrouping[*testItem](0.8)
	groups := grouping.GroupItems([]*ItemPair[*testItem]{})

	if len(groups) != 0 {
		t.Errorf("Expected 0 groups for empty input, got %d", len(groups))
	}
}

func TestCentroidGroupingSinglePair(t *testing.T) {
	grouping := NewCentroidGrouping[*testItem](0.8)

	item1 := ti(1, "a.js", 1, 10)
	item2 := ti(2, "b.js", 1, 10)

	groups := grouping.GroupItems([]*ItemPair[*testItem]{
		pair(item1, item2, 0.9, domain.Type1Clone),
	})

	if len(groups) != 1 {
		t.Fatalf("Expected 1 group, got %d", len(groups))
	}
	if len(groups[0].Items) != 2 {
		t.Errorf("Expected group size 2, got %d", len(groups[0].Items))
	}
}

func TestCentroidGroupingTransitivityRejection(t *testing.T) {
	grouping := NewCentroidGrouping[*testItem](0.8)

	// A~B, B~C but A is NOT similar to C.
	// Centroid should reject C from joining the group with A and B.
	item1 := ti(1, "a.js", 1, 10)
	item2 := ti(2, "b.js", 1, 10)
	item3 := ti(3, "c.js", 1, 10)

	groups := grouping.GroupItems([]*ItemPair[*testItem]{
		pair(item1, item2, 0.9, domain.Type1Clone),
		pair(item2, item3, 0.9, domain.Type1Clone),
		pair(item1, item3, 0.5, domain.Type3Clone), // Below threshold
	})

	for _, g := range groups {
		if len(g.Items) == 3 {
			t.Errorf("Expected no group of size 3 due to transitive rejection, but found one")
		}
	}
}

func TestCentroidGroupingAllSimilar(t *testing.T) {
	grouping := NewCentroidGrouping[*testItem](0.8)

	// All items similar to each other (complete clique)
	item1 := ti(1, "a.js", 1, 10)
	item2 := ti(2, "b.js", 1, 10)
	item3 := ti(3, "c.js", 1, 10)

	groups := grouping.GroupItems([]*ItemPair[*testItem]{
		pair(item1, item2, 0.9, domain.Type1Clone),
		pair(item2, item3, 0.9, domain.Type1Clone),
		pair(item1, item3, 0.85, domain.Type2Clone),
	})

	if len(groups) != 1 {
		t.Fatalf("Expected 1 group, got %d", len(groups))
	}
	if len(groups[0].Items) != 3 {
		t.Errorf("Expected group size 3, got %d", len(groups[0].Items))
	}
}

func TestCentroidGroupingBelowThreshold(t *testing.T) {
	grouping := NewCentroidGrouping[*testItem](0.9)

	item1 := ti(1, "a.js", 1, 10)
	item2 := ti(2, "b.js", 1, 10)

	groups := grouping.GroupItems([]*ItemPair[*testItem]{
		pair(item1, item2, 0.8, domain.Type2Clone),
	})

	if len(groups) != 0 {
		t.Errorf("Expected 0 groups (below threshold), got %d", len(groups))
	}
}
