package clone

import (
	"math"
	"sort"
)

// GroupableItem represents an item that can be grouped (e.g. CodeFragment or domain.Clone).
type GroupableItem interface {
	ItemID() int
	ItemKey() string // Sorting key: "filepath|startLine|endLine|startCol|endCol"
}

// ItemPair represents a pair of items with similarity information.
type ItemPair[T GroupableItem] struct {
	Item1      T
	Item2      T
	Similarity float64
	PairType   int // Clone type (1-4)
}

// ItemGroup represents a grouping result.
type ItemGroup[T GroupableItem] struct {
	ID         int
	Items      []T
	GroupType  int
	Similarity float64
}

// GroupingStrategy is the interface for grouping algorithms.
type GroupingStrategy[T GroupableItem] interface {
	GroupItems(pairs []*ItemPair[T]) []*ItemGroup[T]
	Name() string
}

// ---------------------------------------------------------------------------
// Configuration
// ---------------------------------------------------------------------------

// GroupingMode selects the grouping algorithm.
type GroupingMode string

const (
	ModeConnected       GroupingMode = "connected"
	ModeKCore           GroupingMode = "k_core"
	ModeStarMedoid      GroupingMode = "star_medoid"
	ModeCompleteLinkage GroupingMode = "complete_linkage"
	ModeCentroid        GroupingMode = "centroid"
)

// GroupingConfig holds configuration for the grouping strategy.
type GroupingConfig struct {
	Mode      GroupingMode
	Threshold float64
	KCoreK    int
}

// NewGroupingStrategy returns the appropriate strategy based on config.Mode.
func NewGroupingStrategy[T GroupableItem](config GroupingConfig) GroupingStrategy[T] {
	switch config.Mode {
	case ModeKCore:
		k := config.KCoreK
		if k <= 0 {
			k = 2
		}
		return &KCoreGrouping[T]{threshold: config.Threshold, k: k}
	case ModeStarMedoid:
		return &StarMedoidGrouping[T]{threshold: config.Threshold}
	case ModeCompleteLinkage:
		return &CompleteLinkageGrouping[T]{threshold: config.Threshold}
	case ModeCentroid:
		return &CentroidGrouping[T]{threshold: config.Threshold}
	default:
		return &ConnectedGrouping[T]{threshold: config.Threshold}
	}
}

// ---------------------------------------------------------------------------
// Helper functions (unexported)
// ---------------------------------------------------------------------------

// pairKey returns a normalized key for a pair of items, smaller key first.
func pairKey[T GroupableItem](a, b T) string {
	ka, kb := a.ItemKey(), b.ItemKey()
	if ka > kb {
		ka, kb = kb, ka
	}
	return ka + "|" + kb
}

// itemLess compares two items by their ItemKey for stable sorting.
func itemLess[T GroupableItem](a, b T) bool {
	return a.ItemKey() < b.ItemKey()
}

// averageGroupSimilarity computes the average similarity of all pairs whose
// both endpoints are in the members set.
func averageGroupSimilarity[T GroupableItem](pairs []*ItemPair[T], members map[int]bool) float64 {
	sum := 0.0
	count := 0
	for _, p := range pairs {
		if members[p.Item1.ItemID()] && members[p.Item2.ItemID()] {
			sum += p.Similarity
			count++
		}
	}
	if count == 0 {
		return 0.0
	}
	return sum / float64(count)
}

// majorityType returns the most common PairType among pairs whose both
// endpoints are in the members set.
func majorityType[T GroupableItem](pairs []*ItemPair[T], members map[int]bool) int {
	counts := make(map[int]int)
	for _, p := range pairs {
		if members[p.Item1.ItemID()] && members[p.Item2.ItemID()] {
			counts[p.PairType]++
		}
	}
	bestType, bestCount := 0, -1
	// Iterate in sorted order for determinism when counts are equal.
	keys := make([]int, 0, len(counts))
	for k := range counts {
		keys = append(keys, k)
	}
	sort.Ints(keys)
	for _, k := range keys {
		if counts[k] > bestCount {
			bestCount = counts[k]
			bestType = k
		}
	}
	return bestType
}

// almostEqual returns true if |a - b| < 1e-9.
func almostEqual(a, b float64) bool {
	return math.Abs(a-b) < 1e-9
}

// sortGroupItems sorts the items within a group by ItemKey for deterministic output.
func sortGroupItems[T GroupableItem](items []T) {
	sort.Slice(items, func(i, j int) bool {
		return itemLess(items[i], items[j])
	})
}

// ---------------------------------------------------------------------------
// adjacency builds an adjacency list and collects all unique items from pairs
// that meet the similarity threshold.
// ---------------------------------------------------------------------------

type adjacency[T GroupableItem] struct {
	items map[int]T            // id -> item
	adj   map[int]map[int]bool // id -> set of neighbor ids
}

func buildAdjacency[T GroupableItem](pairs []*ItemPair[T], threshold float64) adjacency[T] {
	a := adjacency[T]{
		items: make(map[int]T),
		adj:   make(map[int]map[int]bool),
	}
	for _, p := range pairs {
		if p.Similarity < threshold && !almostEqual(p.Similarity, threshold) {
			continue
		}
		id1, id2 := p.Item1.ItemID(), p.Item2.ItemID()
		a.items[id1] = p.Item1
		a.items[id2] = p.Item2
		if a.adj[id1] == nil {
			a.adj[id1] = make(map[int]bool)
		}
		if a.adj[id2] == nil {
			a.adj[id2] = make(map[int]bool)
		}
		a.adj[id1][id2] = true
		a.adj[id2][id1] = true
	}
	return a
}

// sortedIDs returns all item IDs sorted for deterministic iteration.
func (a *adjacency[T]) sortedIDs() []int {
	ids := make([]int, 0, len(a.items))
	for id := range a.items {
		ids = append(ids, id)
	}
	sort.Ints(ids)
	return ids
}

// ---------------------------------------------------------------------------
// Union-Find
// ---------------------------------------------------------------------------

type unionFind struct {
	parent map[int]int
	rank   map[int]int
}

func newUnionFind() *unionFind {
	return &unionFind{
		parent: make(map[int]int),
		rank:   make(map[int]int),
	}
}

func (u *unionFind) makeSet(x int) {
	if _, ok := u.parent[x]; !ok {
		u.parent[x] = x
		u.rank[x] = 0
	}
}

func (u *unionFind) find(x int) int {
	if u.parent[x] != x {
		u.parent[x] = u.find(u.parent[x]) // path compression
	}
	return u.parent[x]
}

func (u *unionFind) union(x, y int) {
	rx, ry := u.find(x), u.find(y)
	if rx == ry {
		return
	}
	// union by rank
	if u.rank[rx] < u.rank[ry] {
		u.parent[rx] = ry
	} else if u.rank[rx] > u.rank[ry] {
		u.parent[ry] = rx
	} else {
		u.parent[ry] = rx
		u.rank[rx]++
	}
}

// ---------------------------------------------------------------------------
// 1. ConnectedGrouping — Union-Find based connected components
// ---------------------------------------------------------------------------

// ConnectedGrouping groups items by connected components using Union-Find.
type ConnectedGrouping[T GroupableItem] struct {
	threshold float64
}

func (c *ConnectedGrouping[T]) Name() string { return "connected" }

func (c *ConnectedGrouping[T]) GroupItems(pairs []*ItemPair[T]) []*ItemGroup[T] {
	if len(pairs) == 0 {
		return []*ItemGroup[T]{}
	}

	graph := buildAdjacency[T](pairs, c.threshold)
	if len(graph.items) == 0 {
		return []*ItemGroup[T]{}
	}

	uf := newUnionFind()
	for id := range graph.items {
		uf.makeSet(id)
	}
	for id, neighbors := range graph.adj {
		for nid := range neighbors {
			uf.union(id, nid)
		}
	}

	// Collect components.
	components := make(map[int][]int) // root -> list of item ids
	for _, id := range graph.sortedIDs() {
		root := uf.find(id)
		components[root] = append(components[root], id)
	}

	// Sort component roots for deterministic ordering.
	roots := make([]int, 0, len(components))
	for r := range components {
		roots = append(roots, r)
	}
	sort.Ints(roots)

	groups := make([]*ItemGroup[T], 0, len(components))
	groupID := 1
	for _, root := range roots {
		memberIDs := components[root]
		members := make(map[int]bool, len(memberIDs))
		items := make([]T, 0, len(memberIDs))
		for _, id := range memberIDs {
			members[id] = true
			items = append(items, graph.items[id])
		}
		sortGroupItems(items)

		groups = append(groups, &ItemGroup[T]{
			ID:         groupID,
			Items:      items,
			GroupType:  majorityType(pairs, members),
			Similarity: averageGroupSimilarity(pairs, members),
		})
		groupID++
	}
	return groups
}

// ---------------------------------------------------------------------------
// 2. KCoreGrouping — k-core subgraph decomposition
// ---------------------------------------------------------------------------

// KCoreGrouping finds k-core subgraphs and groups their connected components.
type KCoreGrouping[T GroupableItem] struct {
	threshold float64
	k         int
}

func (kc *KCoreGrouping[T]) Name() string { return "k_core" }

func (kc *KCoreGrouping[T]) GroupItems(pairs []*ItemPair[T]) []*ItemGroup[T] {
	if len(pairs) == 0 {
		return []*ItemGroup[T]{}
	}

	graph := buildAdjacency[T](pairs, kc.threshold)
	if len(graph.items) == 0 {
		return []*ItemGroup[T]{}
	}

	// Copy adjacency so we can mutate it during peeling.
	adj := make(map[int]map[int]bool, len(graph.adj))
	for id, nbrs := range graph.adj {
		cp := make(map[int]bool, len(nbrs))
		for n := range nbrs {
			cp[n] = true
		}
		adj[id] = cp
	}

	// Iteratively remove nodes with degree < k.
	changed := true
	for changed {
		changed = false
		for id := range adj {
			if len(adj[id]) < kc.k {
				// Remove this node.
				for nid := range adj[id] {
					delete(adj[nid], id)
				}
				delete(adj, id)
				changed = true
			}
		}
	}

	if len(adj) == 0 {
		return []*ItemGroup[T]{}
	}

	// Find connected components among remaining nodes via BFS.
	visited := make(map[int]bool)
	var components [][]int

	// Sort remaining IDs for determinism.
	remaining := make([]int, 0, len(adj))
	for id := range adj {
		remaining = append(remaining, id)
	}
	sort.Ints(remaining)

	for _, id := range remaining {
		if visited[id] {
			continue
		}
		// BFS
		queue := []int{id}
		visited[id] = true
		var comp []int
		for len(queue) > 0 {
			cur := queue[0]
			queue = queue[1:]
			comp = append(comp, cur)
			for nid := range adj[cur] {
				if !visited[nid] {
					visited[nid] = true
					queue = append(queue, nid)
				}
			}
		}
		components = append(components, comp)
	}

	groups := make([]*ItemGroup[T], 0, len(components))
	groupID := 1
	for _, comp := range components {
		members := make(map[int]bool, len(comp))
		items := make([]T, 0, len(comp))
		for _, id := range comp {
			members[id] = true
			items = append(items, graph.items[id])
		}
		sortGroupItems(items)

		groups = append(groups, &ItemGroup[T]{
			ID:         groupID,
			Items:      items,
			GroupType:  majorityType(pairs, members),
			Similarity: averageGroupSimilarity(pairs, members),
		})
		groupID++
	}
	return groups
}

// ---------------------------------------------------------------------------
// 3. StarMedoidGrouping — greedy medoid star expansion
// ---------------------------------------------------------------------------

// StarMedoidGrouping builds groups by iteratively selecting the item with the
// highest average similarity to its neighbors (the medoid), grouping it with
// its neighbors, and repeating.
type StarMedoidGrouping[T GroupableItem] struct {
	threshold float64
}

func (s *StarMedoidGrouping[T]) Name() string { return "star_medoid" }

func (s *StarMedoidGrouping[T]) GroupItems(pairs []*ItemPair[T]) []*ItemGroup[T] {
	if len(pairs) == 0 {
		return []*ItemGroup[T]{}
	}

	graph := buildAdjacency[T](pairs, s.threshold)
	if len(graph.items) == 0 {
		return []*ItemGroup[T]{}
	}

	// Build a similarity lookup: (id1, id2) -> similarity.
	simMap := make(map[[2]int]float64)
	for _, p := range pairs {
		if p.Similarity < s.threshold && !almostEqual(p.Similarity, s.threshold) {
			continue
		}
		id1, id2 := p.Item1.ItemID(), p.Item2.ItemID()
		simMap[[2]int{id1, id2}] = p.Similarity
		simMap[[2]int{id2, id1}] = p.Similarity
	}

	remaining := make(map[int]bool, len(graph.items))
	for id := range graph.items {
		remaining[id] = true
	}

	groups := make([]*ItemGroup[T], 0)
	groupID := 1

	for len(remaining) > 0 {
		// Compute average similarity for each remaining item to its remaining neighbors.
		bestID := -1
		bestAvg := -1.0

		// Sort remaining IDs for deterministic tie-breaking (lowest ID wins).
		remIDs := make([]int, 0, len(remaining))
		for id := range remaining {
			remIDs = append(remIDs, id)
		}
		sort.Ints(remIDs)

		for _, id := range remIDs {
			nbrs := graph.adj[id]
			sum := 0.0
			count := 0
			for nid := range nbrs {
				if remaining[nid] {
					sum += simMap[[2]int{id, nid}]
					count++
				}
			}
			if count == 0 {
				continue
			}
			avg := sum / float64(count)
			if avg > bestAvg || (almostEqual(avg, bestAvg) && (bestID == -1 || id < bestID)) {
				bestAvg = avg
				bestID = id
			}
		}

		if bestID == -1 {
			// Remaining items have no neighbors among remaining items. Add each as singleton.
			for _, id := range remIDs {
				items := []T{graph.items[id]}
				members := map[int]bool{id: true}
				groups = append(groups, &ItemGroup[T]{
					ID:         groupID,
					Items:      items,
					GroupType:  majorityType(pairs, members),
					Similarity: 0.0,
				})
				groupID++
				delete(remaining, id)
			}
			break
		}

		// Collect the medoid and its remaining neighbors into a group.
		memberIDs := []int{bestID}
		for nid := range graph.adj[bestID] {
			if remaining[nid] {
				memberIDs = append(memberIDs, nid)
			}
		}

		members := make(map[int]bool, len(memberIDs))
		items := make([]T, 0, len(memberIDs))
		for _, id := range memberIDs {
			members[id] = true
			items = append(items, graph.items[id])
			delete(remaining, id)
		}
		sortGroupItems(items)

		groups = append(groups, &ItemGroup[T]{
			ID:         groupID,
			Items:      items,
			GroupType:  majorityType(pairs, members),
			Similarity: averageGroupSimilarity(pairs, members),
		})
		groupID++
	}

	return groups
}

// ---------------------------------------------------------------------------
// 4. CompleteLinkageGrouping — Bron-Kerbosch maximal cliques
// ---------------------------------------------------------------------------

// CompleteLinkageGrouping finds maximal cliques using the Bron-Kerbosch
// algorithm with pivoting. Each clique becomes a group. If an item appears
// in multiple cliques, it is assigned to the largest one.
type CompleteLinkageGrouping[T GroupableItem] struct {
	threshold float64
}

func (cl *CompleteLinkageGrouping[T]) Name() string { return "complete_linkage" }

func (cl *CompleteLinkageGrouping[T]) GroupItems(pairs []*ItemPair[T]) []*ItemGroup[T] {
	if len(pairs) == 0 {
		return []*ItemGroup[T]{}
	}

	graph := buildAdjacency[T](pairs, cl.threshold)
	if len(graph.items) == 0 {
		return []*ItemGroup[T]{}
	}

	// Find all maximal cliques using Bron-Kerbosch with pivot.
	var cliques [][]int
	allIDs := graph.sortedIDs()

	pSet := make(map[int]bool, len(allIDs))
	for _, id := range allIDs {
		pSet[id] = true
	}

	var bronKerbosch func(r, p, x map[int]bool)
	bronKerbosch = func(r, p, x map[int]bool) {
		if len(p) == 0 && len(x) == 0 {
			// R is a maximal clique.
			clique := make([]int, 0, len(r))
			for id := range r {
				clique = append(clique, id)
			}
			sort.Ints(clique)
			cliques = append(cliques, clique)
			return
		}

		// Choose pivot: node in P ∪ X with the most neighbors in P.
		pivot := -1
		pivotDeg := -1
		union := make([]int, 0, len(p)+len(x))
		for id := range p {
			union = append(union, id)
		}
		for id := range x {
			union = append(union, id)
		}
		sort.Ints(union)
		for _, u := range union {
			deg := 0
			for nid := range graph.adj[u] {
				if p[nid] {
					deg++
				}
			}
			if deg > pivotDeg || (deg == pivotDeg && (pivot == -1 || u < pivot)) {
				pivotDeg = deg
				pivot = u
			}
		}

		// Iterate over P \ N(pivot) in sorted order.
		candidates := make([]int, 0, len(p))
		for id := range p {
			if !graph.adj[pivot][id] {
				candidates = append(candidates, id)
			}
		}
		sort.Ints(candidates)

		for _, v := range candidates {
			newR := make(map[int]bool, len(r)+1)
			for id := range r {
				newR[id] = true
			}
			newR[v] = true

			newP := make(map[int]bool)
			for id := range p {
				if graph.adj[v][id] {
					newP[id] = true
				}
			}

			newX := make(map[int]bool)
			for id := range x {
				if graph.adj[v][id] {
					newX[id] = true
				}
			}

			bronKerbosch(newR, newP, newX)

			delete(p, v)
			x[v] = true
		}
	}

	bronKerbosch(
		make(map[int]bool),
		pSet,
		make(map[int]bool),
	)

	if len(cliques) == 0 {
		return []*ItemGroup[T]{}
	}

	// Sort cliques: largest first, then by first element for determinism.
	sort.Slice(cliques, func(i, j int) bool {
		if len(cliques[i]) != len(cliques[j]) {
			return len(cliques[i]) > len(cliques[j])
		}
		// Same size: compare element by element.
		for k := 0; k < len(cliques[i]) && k < len(cliques[j]); k++ {
			if cliques[i][k] != cliques[j][k] {
				return cliques[i][k] < cliques[j][k]
			}
		}
		return false
	})

	// Assign each item to the largest clique it belongs to.
	assigned := make(map[int]bool)
	groups := make([]*ItemGroup[T], 0)
	groupID := 1

	for _, clique := range cliques {
		// Filter out already-assigned items.
		var memberIDs []int
		for _, id := range clique {
			if !assigned[id] {
				memberIDs = append(memberIDs, id)
			}
		}
		if len(memberIDs) == 0 {
			continue
		}

		members := make(map[int]bool, len(memberIDs))
		items := make([]T, 0, len(memberIDs))
		for _, id := range memberIDs {
			members[id] = true
			assigned[id] = true
			items = append(items, graph.items[id])
		}
		sortGroupItems(items)

		groups = append(groups, &ItemGroup[T]{
			ID:         groupID,
			Items:      items,
			GroupType:  majorityType(pairs, members),
			Similarity: averageGroupSimilarity(pairs, members),
		})
		groupID++
	}

	return groups
}

// ---------------------------------------------------------------------------
// 5. CentroidGrouping — greedy BFS expansion from highest-similarity pairs
// ---------------------------------------------------------------------------

// CentroidGrouping starts with the highest-similarity pair, forms a group,
// then expands by adding items that have similarity >= threshold to ALL
// existing group members.
type CentroidGrouping[T GroupableItem] struct {
	threshold float64
}

func (cg *CentroidGrouping[T]) Name() string { return "centroid" }

func (cg *CentroidGrouping[T]) GroupItems(pairs []*ItemPair[T]) []*ItemGroup[T] {
	if len(pairs) == 0 {
		return []*ItemGroup[T]{}
	}

	// Build similarity lookup.
	type idPair struct{ a, b int }
	simMap := make(map[idPair]float64)
	itemMap := make(map[int]T)

	// Collect qualifying pairs.
	qualifying := make([]*ItemPair[T], 0, len(pairs))
	for _, p := range pairs {
		if p.Similarity < cg.threshold && !almostEqual(p.Similarity, cg.threshold) {
			continue
		}
		qualifying = append(qualifying, p)
		id1, id2 := p.Item1.ItemID(), p.Item2.ItemID()
		simMap[idPair{id1, id2}] = p.Similarity
		simMap[idPair{id2, id1}] = p.Similarity
		itemMap[id1] = p.Item1
		itemMap[id2] = p.Item2
	}

	if len(qualifying) == 0 {
		return []*ItemGroup[T]{}
	}

	// Sort pairs by similarity descending, then by pairKey for determinism.
	sorted := make([]*ItemPair[T], len(qualifying))
	copy(sorted, qualifying)
	sort.Slice(sorted, func(i, j int) bool {
		if !almostEqual(sorted[i].Similarity, sorted[j].Similarity) {
			return sorted[i].Similarity > sorted[j].Similarity
		}
		return pairKey(sorted[i].Item1, sorted[i].Item2) < pairKey(sorted[j].Item1, sorted[j].Item2)
	})

	remaining := make(map[int]bool, len(itemMap))
	for id := range itemMap {
		remaining[id] = true
	}

	groups := make([]*ItemGroup[T], 0)
	groupID := 1

	for len(remaining) > 0 {
		// Find the highest-similarity pair among remaining items.
		var seedPair *ItemPair[T]
		for _, p := range sorted {
			id1, id2 := p.Item1.ItemID(), p.Item2.ItemID()
			if remaining[id1] && remaining[id2] {
				seedPair = p
				break
			}
		}

		if seedPair == nil {
			// No more pairs among remaining items. Each remaining item is isolated.
			remIDs := make([]int, 0, len(remaining))
			for id := range remaining {
				remIDs = append(remIDs, id)
			}
			sort.Ints(remIDs)
			for _, id := range remIDs {
				items := []T{itemMap[id]}
				members := map[int]bool{id: true}
				groups = append(groups, &ItemGroup[T]{
					ID:         groupID,
					Items:      items,
					GroupType:  majorityType(pairs, members),
					Similarity: 0.0,
				})
				groupID++
				delete(remaining, id)
			}
			break
		}

		// Start group with the seed pair.
		groupMembers := map[int]bool{
			seedPair.Item1.ItemID(): true,
			seedPair.Item2.ItemID(): true,
		}

		// Try to expand: check each remaining item (not in group) for compatibility.
		// An item is compatible if it has similarity >= threshold to ALL group members.
		changed := true
		for changed {
			changed = false
			candidates := make([]int, 0)
			for id := range remaining {
				if groupMembers[id] {
					continue
				}
				candidates = append(candidates, id)
			}
			sort.Ints(candidates)

			for _, cid := range candidates {
				compatible := true
				for mid := range groupMembers {
					sim, exists := simMap[idPair{cid, mid}]
					if !exists || (sim < cg.threshold && !almostEqual(sim, cg.threshold)) {
						compatible = false
						break
					}
				}
				if compatible {
					groupMembers[cid] = true
					changed = true
				}
			}
		}

		items := make([]T, 0, len(groupMembers))
		for id := range groupMembers {
			items = append(items, itemMap[id])
			delete(remaining, id)
		}
		sortGroupItems(items)

		groups = append(groups, &ItemGroup[T]{
			ID:         groupID,
			Items:      items,
			GroupType:  majorityType(pairs, groupMembers),
			Similarity: averageGroupSimilarity(pairs, groupMembers),
		})
		groupID++
	}

	return groups
}
