package clone

import (
	"fmt"
	"testing"
)

// testItem implements GroupableItem for testing.
type testItem struct {
	id  int
	key string
}

func (t *testItem) ItemID() int    { return t.id }
func (t *testItem) ItemKey() string { return t.key }

func newItem(id int) *testItem {
	return &testItem{id: id, key: fmt.Sprintf("file%d|%d|%d|0|0", id, id, id+10)}
}

func makePair(a, b *testItem, sim float64, pairType int) *ItemPair[*testItem] {
	return &ItemPair[*testItem]{
		Item1:      a,
		Item2:      b,
		Similarity: sim,
		PairType:   pairType,
	}
}

// --- GroupableItem interface test ---

func TestGroupableItemInterface(t *testing.T) {
	var _ GroupableItem = &testItem{id: 1, key: "a"}
}

// --- Factory test ---

func TestNewGroupingStrategy(t *testing.T) {
	tests := []struct {
		mode GroupingMode
		name string
	}{
		{ModeConnected, "connected"},
		{ModeKCore, "k_core"},
		{ModeStarMedoid, "star_medoid"},
		{ModeCompleteLinkage, "complete_linkage"},
		{ModeCentroid, "centroid"},
		{"", "connected"}, // default
	}

	for _, tt := range tests {
		s := NewGroupingStrategy[*testItem](GroupingConfig{Mode: tt.mode})
		if s.Name() != tt.name {
			t.Errorf("mode %q: expected name %q, got %q", tt.mode, tt.name, s.Name())
		}
	}
}

// --- Empty input test ---

func TestAllStrategiesEmptyInput(t *testing.T) {
	modes := []GroupingMode{ModeConnected, ModeKCore, ModeStarMedoid, ModeCompleteLinkage, ModeCentroid}
	for _, mode := range modes {
		s := NewGroupingStrategy[*testItem](GroupingConfig{Mode: mode, Threshold: 0.5})
		result := s.GroupItems(nil)
		if len(result) != 0 {
			t.Errorf("%s: expected 0 groups for nil input, got %d", mode, len(result))
		}
		result = s.GroupItems([]*ItemPair[*testItem]{})
		if len(result) != 0 {
			t.Errorf("%s: expected 0 groups for empty input, got %d", mode, len(result))
		}
	}
}

// --- ConnectedGrouping tests ---

func TestConnectedGroupingSinglePair(t *testing.T) {
	a, b := newItem(1), newItem(2)
	pairs := []*ItemPair[*testItem]{makePair(a, b, 0.8, 1)}

	s := NewGroupingStrategy[*testItem](GroupingConfig{Mode: ModeConnected, Threshold: 0.5})
	groups := s.GroupItems(pairs)

	if len(groups) != 1 {
		t.Fatalf("expected 1 group, got %d", len(groups))
	}
	if len(groups[0].Items) != 2 {
		t.Fatalf("expected 2 items in group, got %d", len(groups[0].Items))
	}
	if groups[0].ID != 1 {
		t.Fatalf("expected group ID 1, got %d", groups[0].ID)
	}
}

func TestConnectedGroupingTwoComponents(t *testing.T) {
	a, b, c, d := newItem(1), newItem(2), newItem(3), newItem(4)
	pairs := []*ItemPair[*testItem]{
		makePair(a, b, 0.9, 1),
		makePair(c, d, 0.8, 2),
	}

	s := NewGroupingStrategy[*testItem](GroupingConfig{Mode: ModeConnected, Threshold: 0.5})
	groups := s.GroupItems(pairs)

	if len(groups) != 2 {
		t.Fatalf("expected 2 groups, got %d", len(groups))
	}
}

func TestConnectedGroupingThreshold(t *testing.T) {
	a, b := newItem(1), newItem(2)
	pairs := []*ItemPair[*testItem]{makePair(a, b, 0.3, 1)}

	s := NewGroupingStrategy[*testItem](GroupingConfig{Mode: ModeConnected, Threshold: 0.5})
	groups := s.GroupItems(pairs)

	// Pair below threshold, so no groups
	if len(groups) != 0 {
		t.Fatalf("expected 0 groups (below threshold), got %d", len(groups))
	}
}

func TestConnectedGroupingChain(t *testing.T) {
	a, b, c := newItem(1), newItem(2), newItem(3)
	pairs := []*ItemPair[*testItem]{
		makePair(a, b, 0.8, 1),
		makePair(b, c, 0.7, 1),
	}

	s := NewGroupingStrategy[*testItem](GroupingConfig{Mode: ModeConnected, Threshold: 0.5})
	groups := s.GroupItems(pairs)

	if len(groups) != 1 {
		t.Fatalf("expected 1 connected group, got %d", len(groups))
	}
	if len(groups[0].Items) != 3 {
		t.Fatalf("expected 3 items in chain group, got %d", len(groups[0].Items))
	}
}

// --- KCoreGrouping tests ---

func TestKCoreGroupingTriangle(t *testing.T) {
	a, b, c := newItem(1), newItem(2), newItem(3)
	pairs := []*ItemPair[*testItem]{
		makePair(a, b, 0.9, 1),
		makePair(b, c, 0.8, 1),
		makePair(a, c, 0.7, 1),
	}

	s := NewGroupingStrategy[*testItem](GroupingConfig{Mode: ModeKCore, Threshold: 0.5, KCoreK: 2})
	groups := s.GroupItems(pairs)

	// Triangle: every node has degree 2, so all survive k=2 peeling.
	if len(groups) != 1 {
		t.Fatalf("expected 1 group, got %d", len(groups))
	}
	if len(groups[0].Items) != 3 {
		t.Fatalf("expected 3 items, got %d", len(groups[0].Items))
	}
}

func TestKCoreGroupingChainPeeled(t *testing.T) {
	a, b, c := newItem(1), newItem(2), newItem(3)
	// Chain: a-b-c, degrees: a=1, b=2, c=1 → peeling removes a and c, then b
	pairs := []*ItemPair[*testItem]{
		makePair(a, b, 0.9, 1),
		makePair(b, c, 0.8, 1),
	}

	s := NewGroupingStrategy[*testItem](GroupingConfig{Mode: ModeKCore, Threshold: 0.5, KCoreK: 2})
	groups := s.GroupItems(pairs)

	// No 2-core in a chain
	if len(groups) != 0 {
		t.Fatalf("expected 0 groups (no 2-core in chain), got %d", len(groups))
	}
}

func TestKCoreGroupingDefaultK(t *testing.T) {
	// Config with KCoreK=0 should default to 2.
	s := NewGroupingStrategy[*testItem](GroupingConfig{Mode: ModeKCore, Threshold: 0.5, KCoreK: 0})
	if s.Name() != "k_core" {
		t.Fatalf("expected k_core, got %s", s.Name())
	}
}

// --- StarMedoidGrouping tests ---

func TestStarMedoidGrouping(t *testing.T) {
	a, b, c := newItem(1), newItem(2), newItem(3)
	pairs := []*ItemPair[*testItem]{
		makePair(a, b, 0.9, 1),
		makePair(a, c, 0.8, 1),
		makePair(b, c, 0.7, 1), // fully connected star
	}

	s := NewGroupingStrategy[*testItem](GroupingConfig{Mode: ModeStarMedoid, Threshold: 0.5})
	groups := s.GroupItems(pairs)

	// a has avg sim (0.9+0.8)/2=0.85, b has (0.9+0.7)/2=0.8, c has (0.8+0.7)/2=0.75
	// Medoid is a (or b depending on tie-break). All connected → 1 star group.
	if len(groups) != 1 {
		t.Fatalf("expected 1 group, got %d", len(groups))
	}
	if len(groups[0].Items) != 3 {
		t.Fatalf("expected 3 items in star, got %d", len(groups[0].Items))
	}
}

func TestStarMedoidGroupingPartial(t *testing.T) {
	a, b, c := newItem(1), newItem(2), newItem(3)
	pairs := []*ItemPair[*testItem]{
		makePair(a, b, 0.9, 1),
		makePair(a, c, 0.8, 1),
		// b and c not directly connected
	}

	s := NewGroupingStrategy[*testItem](GroupingConfig{Mode: ModeStarMedoid, Threshold: 0.5})
	groups := s.GroupItems(pairs)

	// b has avg 0.9 (best medoid), star = {b, a}. Then c is singleton.
	if len(groups) != 2 {
		t.Fatalf("expected 2 groups, got %d", len(groups))
	}
}

func TestStarMedoidSingleton(t *testing.T) {
	a, b, c := newItem(1), newItem(2), newItem(3)
	// All below threshold — singletons
	pairs := []*ItemPair[*testItem]{
		makePair(a, b, 0.1, 1),
		makePair(b, c, 0.2, 1),
	}

	s := NewGroupingStrategy[*testItem](GroupingConfig{Mode: ModeStarMedoid, Threshold: 0.5})
	groups := s.GroupItems(pairs)

	if len(groups) != 0 {
		t.Fatalf("expected 0 groups (all below threshold), got %d", len(groups))
	}
}

// --- CompleteLinkageGrouping tests ---

func TestCompleteLinkageTriangle(t *testing.T) {
	a, b, c := newItem(1), newItem(2), newItem(3)
	pairs := []*ItemPair[*testItem]{
		makePair(a, b, 0.9, 1),
		makePair(b, c, 0.8, 1),
		makePair(a, c, 0.7, 1),
	}

	s := NewGroupingStrategy[*testItem](GroupingConfig{Mode: ModeCompleteLinkage, Threshold: 0.5})
	groups := s.GroupItems(pairs)

	// Complete triangle is one maximal clique.
	if len(groups) != 1 {
		t.Fatalf("expected 1 group (triangle clique), got %d", len(groups))
	}
	if len(groups[0].Items) != 3 {
		t.Fatalf("expected 3 items in clique, got %d", len(groups[0].Items))
	}
}

func TestCompleteLinkageMissingEdge(t *testing.T) {
	a, b, c := newItem(1), newItem(2), newItem(3)
	// a-b and a-c connected, but b-c not → no triangle clique
	pairs := []*ItemPair[*testItem]{
		makePair(a, b, 0.9, 1),
		makePair(a, c, 0.8, 1),
	}

	s := NewGroupingStrategy[*testItem](GroupingConfig{Mode: ModeCompleteLinkage, Threshold: 0.5})
	groups := s.GroupItems(pairs)

	// Maximal cliques: {a,b} and {a,c}. a assigned to largest (or first).
	// Both have size 2, so a goes to the first one.
	if len(groups) < 1 {
		t.Fatal("expected at least 1 group")
	}
}

// --- CentroidGrouping tests ---

func TestCentroidGroupingTriangle(t *testing.T) {
	a, b, c := newItem(1), newItem(2), newItem(3)
	pairs := []*ItemPair[*testItem]{
		makePair(a, b, 0.9, 1),
		makePair(b, c, 0.8, 1),
		makePair(a, c, 0.7, 1),
	}

	s := NewGroupingStrategy[*testItem](GroupingConfig{Mode: ModeCentroid, Threshold: 0.5})
	groups := s.GroupItems(pairs)

	// All pairs above threshold, all can be expanded into one group.
	if len(groups) != 1 {
		t.Fatalf("expected 1 group, got %d", len(groups))
	}
	if len(groups[0].Items) != 3 {
		t.Fatalf("expected 3 items, got %d", len(groups[0].Items))
	}
}

func TestCentroidGroupingNoExpansion(t *testing.T) {
	a, b, c := newItem(1), newItem(2), newItem(3)
	// a-b high sim, a-c above threshold but b-c below → c can't expand into {a,b}
	pairs := []*ItemPair[*testItem]{
		makePair(a, b, 0.9, 1),
		makePair(a, c, 0.6, 1), // above threshold
		makePair(b, c, 0.3, 1), // below threshold
	}

	s := NewGroupingStrategy[*testItem](GroupingConfig{Mode: ModeCentroid, Threshold: 0.5})
	groups := s.GroupItems(pairs)

	// Seed pair = {a,b} (sim 0.9). c has sim 0.6 to a (ok) but 0.3 to b (below threshold).
	// c can't join {a,b}, becomes separate group. {a,b} + {c} = 2 groups.
	if len(groups) != 2 {
		t.Fatalf("expected 2 groups, got %d", len(groups))
	}
}

// --- Helper function tests ---

func TestAlmostEqual(t *testing.T) {
	if !almostEqual(1.0, 1.0) {
		t.Fatal("expected 1.0 == 1.0")
	}
	if !almostEqual(0.1+0.2, 0.3) {
		t.Fatal("expected 0.1+0.2 ≈ 0.3")
	}
	if almostEqual(1.0, 2.0) {
		t.Fatal("expected 1.0 != 2.0")
	}
}

func TestPairKey(t *testing.T) {
	a := &testItem{id: 1, key: "b"}
	b := &testItem{id: 2, key: "a"}

	// Should normalize: smaller key first
	k := pairKey(a, b)
	if k != "a|b" {
		t.Fatalf("expected 'a|b', got %s", k)
	}
	k2 := pairKey(b, a)
	if k != k2 {
		t.Fatal("pairKey should be symmetric")
	}
}

func TestItemLess(t *testing.T) {
	a := &testItem{id: 1, key: "alpha"}
	b := &testItem{id: 2, key: "beta"}

	if !itemLess(a, b) {
		t.Fatal("expected alpha < beta")
	}
	if itemLess(b, a) {
		t.Fatal("expected beta > alpha")
	}
}

func TestGroupSimilarity(t *testing.T) {
	a, b, c := newItem(1), newItem(2), newItem(3)
	pairs := []*ItemPair[*testItem]{
		makePair(a, b, 0.8, 1),
		makePair(b, c, 0.6, 1),
		makePair(a, c, 0.7, 1),
	}
	members := map[int]bool{1: true, 2: true, 3: true}
	avg := averageGroupSimilarity(pairs, members)
	expected := (0.8 + 0.6 + 0.7) / 3.0
	if !almostEqual(avg, expected) {
		t.Fatalf("expected avg %.4f, got %.4f", expected, avg)
	}
}

func TestMajorityType(t *testing.T) {
	a, b, c := newItem(1), newItem(2), newItem(3)
	pairs := []*ItemPair[*testItem]{
		makePair(a, b, 0.8, 1),
		makePair(b, c, 0.7, 2),
		makePair(a, c, 0.6, 1),
	}
	members := map[int]bool{1: true, 2: true, 3: true}
	mt := majorityType(pairs, members)
	if mt != 1 {
		t.Fatalf("expected majority type 1, got %d", mt)
	}
}

// --- Group ordering test ---

func TestGroupIDsSequential(t *testing.T) {
	items := make([]*testItem, 6)
	for i := range items {
		items[i] = newItem(i + 1)
	}

	pairs := []*ItemPair[*testItem]{
		makePair(items[0], items[1], 0.9, 1),
		makePair(items[2], items[3], 0.8, 1),
		makePair(items[4], items[5], 0.7, 1),
	}

	s := NewGroupingStrategy[*testItem](GroupingConfig{Mode: ModeConnected, Threshold: 0.5})
	groups := s.GroupItems(pairs)

	if len(groups) != 3 {
		t.Fatalf("expected 3 groups, got %d", len(groups))
	}
	for i, g := range groups {
		if g.ID != i+1 {
			t.Errorf("expected group ID %d, got %d", i+1, g.ID)
		}
	}
}

// --- Items sorted within groups ---

func TestItemsSortedWithinGroup(t *testing.T) {
	a := &testItem{id: 1, key: "z_file|1|10|0|0"}
	b := &testItem{id: 2, key: "a_file|1|10|0|0"}
	pairs := []*ItemPair[*testItem]{makePair(a, b, 0.9, 1)}

	s := NewGroupingStrategy[*testItem](GroupingConfig{Mode: ModeConnected})
	groups := s.GroupItems(pairs)

	if len(groups) != 1 || len(groups[0].Items) != 2 {
		t.Fatal("expected 1 group with 2 items")
	}
	if groups[0].Items[0].ItemKey() > groups[0].Items[1].ItemKey() {
		t.Fatal("items should be sorted by ItemKey within group")
	}
}
