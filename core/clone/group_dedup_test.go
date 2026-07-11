package clone

import (
	"testing"

	"github.com/ludo-technologies/polyscan/core/domain"
)

var nextTestItemID = 1000

func gi(file string, start, end int) *testItem {
	nextTestItemID++
	return &testItem{
		id:  nextTestItemID,
		loc: ItemLocation{FilePath: file, StartLine: start, EndLine: end},
	}
}

func mkGroup(id int, items ...*testItem) *ItemGroup[*testItem] {
	return &ItemGroup[*testItem]{ID: id, Items: items}
}

func keysOf(g *ItemGroup[*testItem]) []string {
	out := make([]string, 0, len(g.Items))
	for _, item := range g.Items {
		out = append(out, ItemKey(item))
	}
	return out
}

// TestCoveredGroups_Repro reproduces the case where the same duplication is
// emitted as two near-identical groups whose member windows differ by
// exactly one enclosing line. The group with the smaller windows must be
// suppressed.
func TestCoveredGroups_Repro(t *testing.T) {
	inner := mkGroup(6, gi("smoke.js", 299, 315), gi("smoke.js", 318, 334))
	inner.Similarity = 0.9692
	outer := mkGroup(14, gi("smoke.js", 298, 315), gi("smoke.js", 317, 334))
	outer.Similarity = 0.9613

	out := DedupeCoveredGroups([]*ItemGroup[*testItem]{inner, outer})

	if len(out.Groups) != 1 {
		t.Fatalf("expected 1 group after covered-group dedup, got %d", len(out.Groups))
	}
	if out.Groups[0] != outer {
		t.Fatalf("expected the larger-window group to survive, got %v", keysOf(out.Groups[0]))
	}
	if len(out.SuppressedPairs) != 1 {
		t.Fatalf("expected the covered group's pair suppressed, got %d", len(out.SuppressedPairs))
	}
}

// TestCoveredGroups_IdenticalGroupsKeepFirst verifies the deterministic
// tiebreak for mutual coverage: groups with identical member ranges collapse
// to the earlier one in the slice.
func TestCoveredGroups_IdenticalGroupsKeepFirst(t *testing.T) {
	g1 := mkGroup(1, gi("x.js", 1, 10), gi("y.js", 1, 10))
	g2 := mkGroup(2, gi("x.js", 1, 10), gi("y.js", 1, 10))

	out := DedupeCoveredGroups([]*ItemGroup[*testItem]{g1, g2})

	if len(out.Groups) != 1 || out.Groups[0] != g1 {
		t.Fatalf("expected only the first of two identical groups to survive, got %+v", out.Groups)
	}
}

// TestCoveredGroups_ChainCollapsesToOutermost verifies transitivity: with
// g1 ⊂ g2 ⊂ g3 member-wise, only the outermost group survives even though g2
// is itself suppressed.
func TestCoveredGroups_ChainCollapsesToOutermost(t *testing.T) {
	g1 := mkGroup(1, gi("x.js", 3, 8), gi("y.js", 3, 8))
	g2 := mkGroup(2, gi("x.js", 2, 9), gi("y.js", 2, 9))
	g3 := mkGroup(3, gi("x.js", 1, 10), gi("y.js", 1, 10))

	out := DedupeCoveredGroups([]*ItemGroup[*testItem]{g1, g2, g3})

	if len(out.Groups) != 1 || out.Groups[0] != g3 {
		t.Fatalf("expected only outermost group to survive, got %+v", out.Groups)
	}
}

// TestCoveredGroups_PartialCoverageKeptBoth verifies a group is kept when any
// member falls outside the other group's windows: it carries information the
// covering group does not.
func TestCoveredGroups_PartialCoverageKeptBoth(t *testing.T) {
	g1 := mkGroup(1, gi("x.js", 2, 9), gi("z.js", 1, 8))
	g2 := mkGroup(2, gi("x.js", 1, 10), gi("y.js", 1, 10))

	out := DedupeCoveredGroups([]*ItemGroup[*testItem]{g1, g2})

	if len(out.Groups) != 2 {
		t.Fatalf("expected both groups kept, got %d", len(out.Groups))
	}
	if len(out.SuppressedPairs) != 0 {
		t.Fatalf("expected no suppressed pairs, got %d", len(out.SuppressedPairs))
	}
}

// TestCoveredGroups_DistinctMembersRequired verifies the injective-matching
// constraint: two disjoint blocks inside ONE member of another group describe
// duplication within that member, which the outer group does not report, so
// the inner group must be kept.
func TestCoveredGroups_DistinctMembersRequired(t *testing.T) {
	inner := mkGroup(1, gi("x.js", 10, 20), gi("x.js", 30, 40))
	outer := mkGroup(2, gi("x.js", 1, 100), gi("y.js", 1, 100))

	out := DedupeCoveredGroups([]*ItemGroup[*testItem]{inner, outer})

	if len(out.Groups) != 2 {
		t.Fatalf("expected both groups kept (no injective cover), got %d", len(out.Groups))
	}
}

// TestCoveredGroups_NestedDistinctFilesSuppressed verifies the general nested
// case: a group of inner blocks each inside a distinct cloned member of a
// larger group, with comparable similarity, is redundant with it.
func TestCoveredGroups_NestedDistinctFilesSuppressed(t *testing.T) {
	inner := mkGroup(1, gi("x.js", 20, 30), gi("y.js", 20, 30))
	inner.Similarity = 0.93
	outer := mkGroup(2, gi("x.js", 1, 100), gi("y.js", 1, 100))
	outer.Similarity = 0.91

	out := DedupeCoveredGroups([]*ItemGroup[*testItem]{inner, outer})

	if len(out.Groups) != 1 || out.Groups[0] != outer {
		t.Fatalf("expected nested group suppressed in favor of covering group, got %+v", out.Groups)
	}
}

// TestCoveredGroups_StrongerInnerFindingKept verifies the similarity guard:
// an inner group that matches much more strongly than its covering group
// (e.g., a near-identical block inside loosely similar functions) is a
// distinct finding and must survive.
func TestCoveredGroups_StrongerInnerFindingKept(t *testing.T) {
	inner := mkGroup(1, gi("x.js", 2, 5), gi("y.js", 2, 5))
	inner.Similarity = 0.95
	outer := mkGroup(2, gi("x.js", 1, 6), gi("y.js", 1, 6))
	outer.Similarity = 0.65

	out := DedupeCoveredGroups([]*ItemGroup[*testItem]{inner, outer})

	if len(out.Groups) != 2 {
		t.Fatalf("expected both groups kept when inner is the stronger finding, got %d", len(out.Groups))
	}
	if len(out.SuppressedPairs) != 0 {
		t.Fatalf("expected no suppressed pairs, got %d", len(out.SuppressedPairs))
	}
}

// TestCoveredGroups_SharedItemKeepsPairs verifies that a pair needed by a
// surviving group is not suppressed.
func TestCoveredGroups_SharedItemKeepsPairs(t *testing.T) {
	shared := gi("x.js", 5, 9)
	covered := mkGroup(1, shared, gi("y.js", 5, 9))
	covering := mkGroup(2, gi("x.js", 1, 10), gi("y.js", 1, 10))
	keeper := mkGroup(3, shared, gi("z.js", 50, 90))

	out := DedupeCoveredGroups([]*ItemGroup[*testItem]{covered, covering, keeper})

	if len(out.Groups) != 2 {
		t.Fatalf("expected covering+keeper groups to survive, got %d", len(out.Groups))
	}
	if _, ok := out.SuppressedPairs[PairKey(shared, keeper.Items[1])]; ok {
		t.Fatalf("pair used by a surviving group must not be suppressed")
	}
	if len(out.SuppressedPairs) != 1 {
		t.Fatalf("expected only the covered group's pair suppressed, got %d", len(out.SuppressedPairs))
	}
}

func TestCoveredGroups_SameLinesDifferentColumnsKept(t *testing.T) {
	leftX := gi("x.js", 1, 1)
	leftX.loc.StartCol, leftX.loc.EndCol = 1, 10
	leftY := gi("y.js", 1, 1)
	leftY.loc.StartCol, leftY.loc.EndCol = 1, 10
	rightX := gi("x.js", 1, 1)
	rightX.loc.StartCol, rightX.loc.EndCol = 20, 30
	rightY := gi("y.js", 1, 1)
	rightY.loc.StartCol, rightY.loc.EndCol = 20, 30

	out := DedupeCoveredGroups([]*ItemGroup[*testItem]{
		mkGroup(1, leftX, leftY),
		mkGroup(2, rightX, rightY),
	})

	if len(out.Groups) != 2 {
		t.Fatalf("expected groups at different columns to remain distinct, got %d", len(out.Groups))
	}
}

func TestFilterMaximalPerFile_SameLinesDifferentColumnsKept(t *testing.T) {
	left := gi("x.js", 1, 1)
	left.loc.StartCol, left.loc.EndCol = 1, 10
	right := gi("x.js", 1, 1)
	right.loc.StartCol, right.loc.EndCol = 20, 30

	kept, suppressed := filterMaximalPerFile([]*testItem{left, right})
	if len(kept) != 2 || len(suppressed) != 0 {
		t.Fatalf("expected fragments at different columns to remain distinct, got %d kept", len(kept))
	}
}

func TestFilterMaximalPerFile_StrictSubsetSuppressed(t *testing.T) {
	outer := gi("x.js", 1, 20)
	inner := gi("x.js", 5, 10)

	kept, suppressed := filterMaximalPerFile([]*testItem{inner, outer})
	if len(kept) != 1 || kept[0] != outer {
		t.Fatalf("expected only the covering fragment to survive, got %d kept", len(kept))
	}
	if _, ok := suppressed[ItemKey(inner)]; !ok {
		t.Fatalf("expected the covered fragment to be reported as suppressed")
	}
}

func TestDedupeStrictSubsetGroupMembers_CollapsesOverlappingWindows(t *testing.T) {
	// A group whose same-file members overlap (one strictly covers the other)
	// collapses to the maximal window; a group reduced below 2 members drops.
	a := gi("x.js", 512, 542)
	b := gi("x.js", 515, 542)
	c := gi("y.js", 1, 30)
	g := mkGroup(1, a, b, c)

	pairs := []*ItemPair[*testItem]{
		pair(a, c, 0.9, domain.Type3Clone),
		pair(b, c, 0.9, domain.Type3Clone),
	}

	out := DedupeStrictSubsetGroupMembers([]*ItemGroup[*testItem]{g}, pairs)

	if len(out.Groups) != 1 {
		t.Fatalf("expected 1 group, got %d", len(out.Groups))
	}
	if len(out.Groups[0].Items) != 2 {
		t.Fatalf("expected 2 members after collapsing overlap, got %d", len(out.Groups[0].Items))
	}
	if _, ok := out.Suppressed[ItemKey(b)]; !ok {
		t.Fatalf("expected the covered member to be reported as suppressed")
	}
}

func TestFilterSuppressedPairs_KeepsUnrelatedPair(t *testing.T) {
	a := gi("x.js", 10, 20)
	b := gi("y.js", 10, 20)
	c := gi("z.js", 10, 20)
	coveredPair := &ItemPair[*testItem]{Item1: a, Item2: b}
	unrelatedPair := &ItemPair[*testItem]{Item1: a, Item2: c}

	out := FilterSuppressedPairs(
		[]*ItemPair[*testItem]{coveredPair, unrelatedPair},
		map[string]struct{}{PairKey(a, b): {}},
	)

	if len(out) != 1 || out[0] != unrelatedPair {
		t.Fatalf("expected unrelated pair sharing one member to survive, got %+v", out)
	}
}

func TestFilterPairsWithSuppressedMembers(t *testing.T) {
	a := gi("x.js", 10, 20)
	b := gi("y.js", 10, 20)
	c := gi("z.js", 10, 20)
	suppressedPair := &ItemPair[*testItem]{Item1: a, Item2: b}
	keptPair := &ItemPair[*testItem]{Item1: b, Item2: c}

	out := FilterPairsWithSuppressedMembers(
		[]*ItemPair[*testItem]{suppressedPair, keptPair},
		map[string]struct{}{ItemKey(a): {}},
	)

	if len(out) != 1 || out[0] != keptPair {
		t.Fatalf("expected only the pair without suppressed members to survive, got %+v", out)
	}
}

// TestFilterGroupsWithoutBackingPairs_DropsUnbackedGroup verifies that a
// group whose members have no positive-similarity backing pair is dropped,
// while a backed group has its metadata refreshed.
func TestFilterGroupsWithoutBackingPairs_DropsUnbackedGroup(t *testing.T) {
	a := gi("x.js", 1, 10)
	b := gi("x.js", 20, 30)
	c := gi("y.js", 1, 10)
	unbacked := mkGroup(1, a, b)
	backed := mkGroup(2, a, c)
	pairs := []*ItemPair[*testItem]{pair(a, c, 0.92, domain.Type4Clone)}

	out := FilterGroupsWithoutBackingPairs([]*ItemGroup[*testItem]{unbacked, backed}, pairs)

	if len(out) != 1 {
		t.Fatalf("expected only the backed group to remain, got %d groups", len(out))
	}
	if out[0] != backed {
		t.Fatalf("expected backed group to remain")
	}
	if !almostEqual(out[0].Similarity, 0.92) {
		t.Fatalf("expected refreshed similarity 0.92, got %.3f", out[0].Similarity)
	}
}
