package clone

import (
	"reflect"
	"testing"

	"github.com/ludo-technologies/polyscan/core/domain"
)

func groupIDs(groups []*ItemGroup[*testItem]) [][]int {
	var ids [][]int
	for _, group := range groups {
		members := []int{group.ID}
		for _, item := range group.Items {
			members = append(members, item.id)
		}
		ids = append(ids, members)
	}
	return ids
}

func TestSplitGroupsByPairs(t *testing.T) {
	a, b, c, d, e := ti(0, "a.js", 0, 5), ti(1, "a.js", 10, 15), ti(2, "a.js", 20, 25), ti(3, "a.js", 30, 35), ti(4, "a.js", 40, 45)
	input := &ItemGroup[*testItem]{ID: 7, Items: []*testItem{a, b, c, d, e}, GroupType: domain.Type3Clone, Similarity: 0.5}
	// Only B-C held A-B and C-D together, and it is gone. E has no pair left.
	pairs := []*ItemPair[*testItem]{
		pair(a, b, 0.96, domain.Type1Clone),
		pair(c, d, 0.90, domain.Type2Clone),
	}

	split := SplitGroupsByPairs([]*ItemGroup[*testItem]{input}, pairs)

	if got, want := groupIDs(split), [][]int{{7, 0, 1}, {8, 2, 3}}; !reflect.DeepEqual(got, want) {
		t.Fatalf("groups (ID, members...) = %v, want %v", got, want)
	}
	if split[0].GroupType != domain.Type1Clone || split[0].Similarity != 0.96 {
		t.Errorf("first group = %v %.2f, want Type-1 0.96", split[0].GroupType, split[0].Similarity)
	}
	if split[1].GroupType != domain.Type2Clone || split[1].Similarity != 0.90 {
		t.Errorf("second group = %v %.2f, want Type-2 0.90", split[1].GroupType, split[1].Similarity)
	}
	if len(input.Items) != 5 || input.GroupType != domain.Type3Clone || input.Similarity != 0.5 {
		t.Errorf("input group was modified: %+v", input)
	}
}

func TestSplitGroupsByPairsDropsUnbackedGroups(t *testing.T) {
	input := &ItemGroup[*testItem]{ID: 0, Items: []*testItem{ti(0, "a.js", 0, 5), ti(1, "a.js", 10, 15)}}

	if split := SplitGroupsByPairs([]*ItemGroup[*testItem]{input}, nil); len(split) != 0 {
		t.Fatalf("got %v, want no group without a backing pair", groupIDs(split))
	}
}
