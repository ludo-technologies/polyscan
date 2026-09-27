package analyzer

import (
	"reflect"
	"testing"

	coreclone "github.com/ludo-technologies/polyscan/core/clone"
	coredomain "github.com/ludo-technologies/polyscan/core/domain"
	"github.com/ludo-technologies/polyscan/polyscan/internal/js/domain"
)

func splitTestClone(id int) *domain.Clone {
	return &domain.Clone{ID: id, Location: &domain.CloneLocation{FilePath: "a.js", StartLine: id * 10, EndLine: id*10 + 5}}
}

func splitTestPair(a, b *domain.Clone, similarity float64, cloneType coredomain.CloneType) *coreclone.ItemPair[*domain.Clone] {
	return &coreclone.ItemPair[*domain.Clone]{Item1: a, Item2: b, Similarity: similarity, PairType: cloneType}
}

func groupIDs(groups []*coreclone.ItemGroup[*domain.Clone]) [][]int {
	var ids [][]int
	for _, group := range groups {
		members := []int{group.ID}
		for _, item := range group.Items {
			members = append(members, item.ID)
		}
		ids = append(ids, members)
	}
	return ids
}

func TestSplitGroupsByPairs(t *testing.T) {
	a, b, c, d, e := splitTestClone(0), splitTestClone(1), splitTestClone(2), splitTestClone(3), splitTestClone(4)
	input := &coreclone.ItemGroup[*domain.Clone]{ID: 7, Items: []*domain.Clone{a, b, c, d, e}, GroupType: coredomain.Type3Clone, Similarity: 0.5}
	// Only B-C held A-B and C-D together, and it is gone. E has no pair left.
	pairs := []*coreclone.ItemPair[*domain.Clone]{
		splitTestPair(a, b, 0.96, coredomain.Type1Clone),
		splitTestPair(c, d, 0.90, coredomain.Type2Clone),
	}

	split := SplitGroupsByPairs([]*coreclone.ItemGroup[*domain.Clone]{input}, pairs)

	if got, want := groupIDs(split), [][]int{{7, 0, 1}, {8, 2, 3}}; !reflect.DeepEqual(got, want) {
		t.Fatalf("groups (ID, members...) = %v, want %v", got, want)
	}
	if split[0].GroupType != coredomain.Type1Clone || split[0].Similarity != 0.96 {
		t.Errorf("first group = %v %.2f, want Type-1 0.96", split[0].GroupType, split[0].Similarity)
	}
	if split[1].GroupType != coredomain.Type2Clone || split[1].Similarity != 0.90 {
		t.Errorf("second group = %v %.2f, want Type-2 0.90", split[1].GroupType, split[1].Similarity)
	}
	if len(input.Items) != 5 || input.GroupType != coredomain.Type3Clone || input.Similarity != 0.5 {
		t.Errorf("input group was modified: %+v", input)
	}
}

func TestSplitGroupsByPairsDropsUnbackedGroups(t *testing.T) {
	a, b := splitTestClone(0), splitTestClone(1)
	input := &coreclone.ItemGroup[*domain.Clone]{ID: 0, Items: []*domain.Clone{a, b}}

	if split := SplitGroupsByPairs([]*coreclone.ItemGroup[*domain.Clone]{input}, nil); len(split) != 0 {
		t.Fatalf("got %v, want no group without a backing pair", groupIDs(split))
	}
}
