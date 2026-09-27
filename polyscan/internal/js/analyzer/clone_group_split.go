package analyzer

import (
	coreclone "github.com/ludo-technologies/polyscan/core/clone"
)

// SplitGroupsByPairs restricts each group to the connected components of the
// given pairs. Grouping runs over every detected pair, but later steps drop
// pairs: suppressed members take their pairs with them, and the clone-type
// filter removes whole pair types. A group that only held together through a
// dropped pair would otherwise still be reported as one duplication family.
//
// Components of one are dropped. The first component keeps the group's ID and
// the others get fresh IDs above every input ID. Similarity and type are
// recomputed from the given pairs, and a group no positive-similarity pair
// backs is dropped. The input groups are not modified.
func SplitGroupsByPairs[T coreclone.GroupableItem](groups []*coreclone.ItemGroup[T], pairs []*coreclone.ItemPair[T]) []*coreclone.ItemGroup[T] {
	adjacency := make(map[int][]int, len(pairs))
	for _, pair := range pairs {
		if pair == nil {
			continue
		}
		a, b := pair.Item1.ItemID(), pair.Item2.ItemID()
		adjacency[a] = append(adjacency[a], b)
		adjacency[b] = append(adjacency[b], a)
	}
	nextID := 0
	for _, group := range groups {
		if group != nil && group.ID >= nextID {
			nextID = group.ID + 1
		}
	}

	split := make([]*coreclone.ItemGroup[T], 0, len(groups))
	for _, group := range groups {
		if group == nil {
			continue
		}
		first := true
		for _, members := range connectedComponents(group.Items, adjacency) {
			if len(members) < 2 {
				continue
			}
			id := group.ID
			if !first {
				id = nextID
				nextID++
			}
			first = false
			split = append(split, &coreclone.ItemGroup[T]{ID: id, Items: members})
		}
	}
	return coreclone.FilterGroupsWithoutBackingPairs(split, pairs)
}

// connectedComponents partitions items by the adjacency between them,
// preserving the input order within and across components.
func connectedComponents[T coreclone.GroupableItem](items []T, adjacency map[int][]int) [][]T {
	members := make(map[int]bool, len(items))
	for _, item := range items {
		members[item.ItemID()] = true
	}
	component := make(map[int]int, len(items))
	count := 0
	for _, item := range items {
		if _, seen := component[item.ItemID()]; seen {
			continue
		}
		component[item.ItemID()] = count
		queue := []int{item.ItemID()}
		for i := 0; i < len(queue); i++ {
			for _, neighbor := range adjacency[queue[i]] {
				if _, seen := component[neighbor]; members[neighbor] && !seen {
					component[neighbor] = count
					queue = append(queue, neighbor)
				}
			}
		}
		count++
	}
	components := make([][]T, count)
	for _, item := range items {
		c := component[item.ItemID()]
		components[c] = append(components[c], item)
	}
	return components
}
