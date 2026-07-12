package analyzer

import (
	"sort"
	"testing"

	"github.com/ludo-technologies/polyscan/jscan/domain"
)

var nextTestCloneID = 1000

func gc(file string, start, end int) *domain.Clone {
	nextTestCloneID++
	return &domain.Clone{
		ID: nextTestCloneID,
		Location: &domain.CloneLocation{
			FilePath:  file,
			StartLine: start,
			EndLine:   end,
		},
	}
}

func makeGroup(id int, clones ...*domain.Clone) *domain.CloneGroup {
	g := &domain.CloneGroup{ID: id}
	for _, c := range clones {
		g.AddClone(c)
	}
	return g
}

func rangesOf(g *domain.CloneGroup) []string {
	out := make([]string, 0, len(g.Clones))
	for _, c := range g.Clones {
		out = append(out, c.Location.String())
	}
	sort.Strings(out)
	return out
}

// TestCoveredGroups_Repro reproduces the case where the same duplication is
// emitted as two near-identical groups whose member windows differ by
// exactly one enclosing line. The group with the smaller windows must be
// suppressed.
func TestCoveredGroups_Repro(t *testing.T) {
	inner := makeGroup(6, gc("smoke.js", 299, 315), gc("smoke.js", 318, 334))
	inner.Similarity = 0.9692
	outer := makeGroup(14, gc("smoke.js", 298, 315), gc("smoke.js", 317, 334))
	outer.Similarity = 0.9613

	out := dedupeCoveredGroups([]*domain.CloneGroup{inner, outer})

	if len(out.groups) != 1 {
		t.Fatalf("expected 1 group after covered-group dedup, got %d", len(out.groups))
	}
	if out.groups[0] != outer {
		t.Fatalf("expected the larger-window group to survive, got %v", rangesOf(out.groups[0]))
	}
	if len(out.suppressedPairs) != 1 {
		t.Fatalf("expected the covered group's pair suppressed, got %d", len(out.suppressedPairs))
	}
}

// TestCoveredGroups_IdenticalGroupsKeepFirst verifies the deterministic
// tiebreak for mutual coverage: groups with identical member ranges collapse
// to the earlier one in the slice.
func TestCoveredGroups_IdenticalGroupsKeepFirst(t *testing.T) {
	g1 := makeGroup(1, gc("x.js", 1, 10), gc("y.js", 1, 10))
	g2 := makeGroup(2, gc("x.js", 1, 10), gc("y.js", 1, 10))

	out := dedupeCoveredGroups([]*domain.CloneGroup{g1, g2})

	if len(out.groups) != 1 || out.groups[0] != g1 {
		t.Fatalf("expected only the first of two identical groups to survive, got %+v", out.groups)
	}
}

// TestCoveredGroups_ChainCollapsesToOutermost verifies transitivity: with
// g1 ⊂ g2 ⊂ g3 member-wise, only the outermost group survives even though g2
// is itself suppressed.
func TestCoveredGroups_ChainCollapsesToOutermost(t *testing.T) {
	g1 := makeGroup(1, gc("x.js", 3, 8), gc("y.js", 3, 8))
	g2 := makeGroup(2, gc("x.js", 2, 9), gc("y.js", 2, 9))
	g3 := makeGroup(3, gc("x.js", 1, 10), gc("y.js", 1, 10))

	out := dedupeCoveredGroups([]*domain.CloneGroup{g1, g2, g3})

	if len(out.groups) != 1 || out.groups[0] != g3 {
		t.Fatalf("expected only outermost group to survive, got %+v", out.groups)
	}
}

// TestCoveredGroups_PartialCoverageKeptBoth verifies a group is kept when any
// member falls outside the other group's windows: it carries information the
// covering group does not.
func TestCoveredGroups_PartialCoverageKeptBoth(t *testing.T) {
	g1 := makeGroup(1, gc("x.js", 2, 9), gc("z.js", 1, 8))
	g2 := makeGroup(2, gc("x.js", 1, 10), gc("y.js", 1, 10))

	out := dedupeCoveredGroups([]*domain.CloneGroup{g1, g2})

	if len(out.groups) != 2 {
		t.Fatalf("expected both groups kept, got %d", len(out.groups))
	}
	if len(out.suppressedPairs) != 0 {
		t.Fatalf("expected no suppressed pairs, got %d", len(out.suppressedPairs))
	}
}

// TestCoveredGroups_DistinctMembersRequired verifies the injective-matching
// constraint: two disjoint blocks inside ONE member of another group describe
// duplication within that member, which the outer group does not report, so
// the inner group must be kept.
func TestCoveredGroups_DistinctMembersRequired(t *testing.T) {
	inner := makeGroup(1, gc("x.js", 10, 20), gc("x.js", 30, 40))
	outer := makeGroup(2, gc("x.js", 1, 100), gc("y.js", 1, 100))

	out := dedupeCoveredGroups([]*domain.CloneGroup{inner, outer})

	if len(out.groups) != 2 {
		t.Fatalf("expected both groups kept (no injective cover), got %d", len(out.groups))
	}
}

// TestCoveredGroups_NestedDistinctFilesSuppressed verifies the general nested
// case: a group of inner blocks each inside a distinct cloned member of a
// larger group, with comparable similarity, is redundant with it.
func TestCoveredGroups_NestedDistinctFilesSuppressed(t *testing.T) {
	inner := makeGroup(1, gc("x.js", 20, 30), gc("y.js", 20, 30))
	inner.Similarity = 0.93
	outer := makeGroup(2, gc("x.js", 1, 100), gc("y.js", 1, 100))
	outer.Similarity = 0.91

	out := dedupeCoveredGroups([]*domain.CloneGroup{inner, outer})

	if len(out.groups) != 1 || out.groups[0] != outer {
		t.Fatalf("expected nested group suppressed in favor of covering group, got %+v", out.groups)
	}
}

// TestCoveredGroups_StrongerInnerFindingKept verifies the similarity guard:
// an inner group that matches much more strongly than its covering group
// (e.g., a near-identical block inside loosely similar functions) is a
// distinct finding and must survive.
func TestCoveredGroups_StrongerInnerFindingKept(t *testing.T) {
	inner := makeGroup(1, gc("x.js", 2, 5), gc("y.js", 2, 5))
	inner.Similarity = 0.95
	outer := makeGroup(2, gc("x.js", 1, 6), gc("y.js", 1, 6))
	outer.Similarity = 0.65

	out := dedupeCoveredGroups([]*domain.CloneGroup{inner, outer})

	if len(out.groups) != 2 {
		t.Fatalf("expected both groups kept when inner is the stronger finding, got %d", len(out.groups))
	}
	if len(out.suppressedPairs) != 0 {
		t.Fatalf("expected no suppressed pairs, got %d", len(out.suppressedPairs))
	}
}

// TestCoveredGroups_SharedCloneKeepsPairs verifies that a pair needed by a
// surviving group is not suppressed.
func TestCoveredGroups_SharedCloneKeepsPairs(t *testing.T) {
	shared := gc("x.js", 5, 9)
	covered := makeGroup(1, shared, gc("y.js", 5, 9))
	covering := makeGroup(2, gc("x.js", 1, 10), gc("y.js", 1, 10))
	keeper := makeGroup(3, shared, gc("z.js", 50, 90))

	out := dedupeCoveredGroups([]*domain.CloneGroup{covered, covering, keeper})

	if len(out.groups) != 2 {
		t.Fatalf("expected covering+keeper groups to survive, got %d", len(out.groups))
	}
	if _, ok := out.suppressedPairs[clonePairKey(shared, keeper.Clones[1])]; ok {
		t.Fatalf("pair used by a surviving group must not be suppressed")
	}
	if len(out.suppressedPairs) != 1 {
		t.Fatalf("expected only the covered group's pair suppressed, got %d", len(out.suppressedPairs))
	}
}

func TestCoveredGroups_SameLinesDifferentColumnsKept(t *testing.T) {
	leftX := gc("x.js", 1, 1)
	leftX.Location.StartCol, leftX.Location.EndCol = 1, 10
	leftY := gc("y.js", 1, 1)
	leftY.Location.StartCol, leftY.Location.EndCol = 1, 10
	rightX := gc("x.js", 1, 1)
	rightX.Location.StartCol, rightX.Location.EndCol = 20, 30
	rightY := gc("y.js", 1, 1)
	rightY.Location.StartCol, rightY.Location.EndCol = 20, 30

	out := dedupeCoveredGroups([]*domain.CloneGroup{
		makeGroup(1, leftX, leftY),
		makeGroup(2, rightX, rightY),
	})

	if len(out.groups) != 2 {
		t.Fatalf("expected groups at different columns to remain distinct, got %d", len(out.groups))
	}
}

func TestFilterMaximalPerFile_SameLinesDifferentColumnsKept(t *testing.T) {
	left := gc("x.js", 1, 1)
	left.Location.StartCol, left.Location.EndCol = 1, 10
	right := gc("x.js", 1, 1)
	right.Location.StartCol, right.Location.EndCol = 20, 30

	kept, suppressed := filterMaximalPerFile([]*domain.Clone{left, right})
	if len(kept) != 2 || len(suppressed) != 0 {
		t.Fatalf("expected fragments at different columns to remain distinct, got %d kept", len(kept))
	}
}

func TestFilterSuppressedClonePairs_KeepsUnrelatedPair(t *testing.T) {
	a := gc("x.js", 10, 20)
	b := gc("y.js", 10, 20)
	c := gc("z.js", 10, 20)
	coveredPair := &domain.ClonePair{Clone1: a, Clone2: b}
	unrelatedPair := &domain.ClonePair{Clone1: a, Clone2: c}

	out := filterSuppressedClonePairs(
		[]*domain.ClonePair{coveredPair, unrelatedPair},
		map[string]struct{}{clonePairKey(a, b): {}},
	)

	if len(out) != 1 || out[0] != unrelatedPair {
		t.Fatalf("expected unrelated pair sharing one member to survive, got %+v", out)
	}
}

// TestFilterCloneGroupsWithoutBackingPairs_DropsUnbackedGroup verifies that a
// group whose members have no positive-similarity backing pair is dropped,
// while a backed group has its metadata refreshed.
func TestFilterCloneGroupsWithoutBackingPairs_DropsUnbackedGroup(t *testing.T) {
	a := gc("x.js", 1, 10)
	b := gc("x.js", 20, 30)
	c := gc("y.js", 1, 10)
	unbacked := makeGroup(1, a, b)
	backed := makeGroup(2, a, c)
	pairs := []*domain.ClonePair{{Clone1: a, Clone2: c, Similarity: 0.92, Type: domain.Type4Clone}}

	out := filterCloneGroupsWithoutBackingPairs([]*domain.CloneGroup{unbacked, backed}, pairs)

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
