package clone

import (
	"container/heap"
	"container/list"
	"fmt"
	"sort"

	"github.com/ludo-technologies/polyscan/core/domain"
)

// ItemLocation is the source location of a groupable item. The zero value is
// valid; items with equal locations fall back to ItemID ordering.
type ItemLocation struct {
	FilePath  string
	StartLine int
	EndLine   int
	StartCol  int
	EndCol    int
}

// GroupableItem represents an item that can be grouped (e.g. a code fragment
// or clone). Items passed to this package must be non-nil.
type GroupableItem interface {
	ItemID() int
	ItemLocation() ItemLocation
}

// ItemPair represents a pair of items with similarity information.
type ItemPair[T GroupableItem] struct {
	Item1      T
	Item2      T
	Similarity float64
	PairType   domain.CloneType
}

// ItemGroup represents a grouping result.
type ItemGroup[T GroupableItem] struct {
	ID         int
	Items      []T
	GroupType  domain.CloneType
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

// Algorithm-specific constants.
const (
	starMedoidMaxIterations    = 10
	starMedoidConvergenceRatio = 0.01
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
		return NewKCoreGrouping[T](config.Threshold, config.KCoreK)
	case ModeStarMedoid:
		return NewStarMedoidGrouping[T](config.Threshold)
	case ModeCompleteLinkage:
		return NewCompleteLinkageGrouping[T](config.Threshold)
	case ModeCentroid:
		return NewCentroidGrouping[T](config.Threshold)
	default:
		return NewConnectedGrouping[T](config.Threshold)
	}
}

// ---------------------------------------------------------------------------
// Keys, ordering, and shared metadata helpers
// ---------------------------------------------------------------------------

// ItemKey returns a stable identifier for an item based on its location.
func ItemKey[T GroupableItem](item T) string {
	loc := item.ItemLocation()
	return fmt.Sprintf("%s|%d|%d|%d|%d", loc.FilePath, loc.StartLine, loc.EndLine, loc.StartCol, loc.EndCol)
}

// PairKey returns a canonical key for a pair of items, independent of order.
func PairKey[T GroupableItem](a, b T) string {
	ka := ItemKey(a)
	kb := ItemKey(b)
	if ka <= kb {
		return ka + "||" + kb
	}
	return kb + "||" + ka
}

// metadataPairKey identifies a pair by item identity rather than location.
// Distinct analysis items may legitimately refer to the same source range.
func metadataPairKey[T GroupableItem](a, b T) string {
	left, right := a.ItemID(), b.ItemID()
	if left > right {
		left, right = right, left
	}
	return fmt.Sprintf("%d||%d", left, right)
}

func suppressedMemberKey[T GroupableItem](item T) string {
	return fmt.Sprintf("%s||id:%d", ItemKey(item), item.ItemID())
}

// itemLess provides deterministic ordering between two items by location.
func itemLess[T GroupableItem](a, b T) bool {
	al, bl := a.ItemLocation(), b.ItemLocation()
	if al == bl {
		return a.ItemID() < b.ItemID()
	}
	if al.FilePath != bl.FilePath {
		return al.FilePath < bl.FilePath
	}
	if al.StartLine != bl.StartLine {
		return al.StartLine < bl.StartLine
	}
	if al.StartCol != bl.StartCol {
		return al.StartCol < bl.StartCol
	}
	if al.EndLine != bl.EndLine {
		return al.EndLine < bl.EndLine
	}
	return al.EndCol < bl.EndCol
}

// itemSimilarity returns cached similarity, or 0 if not present.
func itemSimilarity[T GroupableItem](sims map[string]float64, a, b T) float64 {
	if a.ItemID() == b.ItemID() {
		return 1.0
	}
	if s, ok := sims[metadataPairKey(a, b)]; ok {
		return s
	}
	return 0.0
}

// averageGroupSimilarity computes average pairwise similarity among members using cache.
// Only pairs that exist in the similarity map are counted (missing pairs are skipped, not treated as 0).
func averageGroupSimilarity[T GroupableItem](sims map[string]float64, members []T) float64 {
	if len(members) < 2 {
		return 1.0
	}
	sum := 0.0
	cnt := 0
	for i := 0; i < len(members); i++ {
		for j := i + 1; j < len(members); j++ {
			key := metadataPairKey(members[i], members[j])
			if sim, ok := sims[key]; ok {
				sum += sim
				cnt++
			}
		}
	}
	if cnt == 0 {
		return 0.0
	}
	return sum / float64(cnt)
}

// majorityType chooses the CloneType of the highest-similarity pair edge in
// members. When several pairs share the maximum similarity, the most strict
// (lowest enum) type wins. This prevents a high-similarity Type-2/Type-4 pair
// from being hidden when lower-similarity Type-3 transitive edges outnumber it
// in the same connected component.
func majorityType[T GroupableItem](typeMap map[string]domain.CloneType, simMap map[string]float64, members []T) domain.CloneType {
	maxSim := -1.0
	var best domain.CloneType
	found := false
	for i := 0; i < len(members); i++ {
		for j := i + 1; j < len(members); j++ {
			key := metadataPairKey(members[i], members[j])
			t, tok := typeMap[key]
			s, sok := simMap[key]
			if !tok || !sok || t == 0 {
				continue
			}
			found = true
			if s > maxSim || (almostEqual(s, maxSim) && t < best) {
				maxSim = s
				best = t
			}
		}
	}
	if !found {
		return domain.Type4Clone // conservative fallback: never report unknown as Type-1
	}
	return best
}

// pairMetadata reduces raw pairs to per-pair-key similarity and type maps,
// keeping the highest-similarity record for duplicate keys.
func pairMetadata[T GroupableItem](pairs []*ItemPair[T]) (map[string]float64, map[string]domain.CloneType) {
	similarities := make(map[string]float64, len(pairs))
	types := make(map[string]domain.CloneType, len(pairs))
	for _, pair := range pairs {
		if pair == nil {
			continue
		}
		key := metadataPairKey(pair.Item1, pair.Item2)
		old, ok := similarities[key]
		if !ok || pair.Similarity > old || (almostEqual(pair.Similarity, old) && pair.PairType < types[key]) {
			similarities[key] = pair.Similarity
			types[key] = pair.PairType
		}
	}
	return similarities, types
}

func sortItemGroups[T GroupableItem](groups []*ItemGroup[T]) {
	sort.Slice(groups, func(i, j int) bool {
		if !almostEqual(groups[i].Similarity, groups[j].Similarity) {
			return groups[i].Similarity > groups[j].Similarity
		}
		if len(groups[i].Items) != len(groups[j].Items) {
			return len(groups[i].Items) > len(groups[j].Items)
		}
		if len(groups[i].Items) == 0 || len(groups[j].Items) == 0 {
			return false
		}
		return itemLess(groups[i].Items[0], groups[j].Items[0])
	})
}

func almostEqual(a, b float64) bool {
	const eps = 1e-9
	d := a - b
	if d < 0 {
		d = -d
	}
	return d <= eps
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

// collectItems gathers unique items from pairs (insertion order) and reduces
// pair metadata to per-key maps.
func collectItems[T GroupableItem](pairs []*ItemPair[T]) ([]T, map[string]float64, map[string]domain.CloneType) {
	items := make([]T, 0)
	seen := make(map[int]struct{})
	simMap := make(map[string]float64)
	typeMap := make(map[string]domain.CloneType)
	for _, p := range pairs {
		if p == nil {
			continue
		}
		if _, ok := seen[p.Item1.ItemID()]; !ok {
			seen[p.Item1.ItemID()] = struct{}{}
			items = append(items, p.Item1)
		}
		if _, ok := seen[p.Item2.ItemID()]; !ok {
			seen[p.Item2.ItemID()] = struct{}{}
			items = append(items, p.Item2)
		}
		key := metadataPairKey(p.Item1, p.Item2)
		old, ok := simMap[key]
		if !ok || p.Similarity > old || (almostEqual(p.Similarity, old) && p.PairType < typeMap[key]) {
			simMap[key] = p.Similarity
			typeMap[key] = p.PairType
		}
	}
	return items, simMap, typeMap
}

// ---------------------------------------------------------------------------
// 1. ConnectedGrouping — Union-Find based connected components
// ---------------------------------------------------------------------------

// ConnectedGrouping groups items by connected components using Union-Find.
type ConnectedGrouping[T GroupableItem] struct {
	threshold float64
}

func NewConnectedGrouping[T GroupableItem](threshold float64) *ConnectedGrouping[T] {
	return &ConnectedGrouping[T]{threshold: threshold}
}

func (c *ConnectedGrouping[T]) Name() string { return string(ModeConnected) }

func (c *ConnectedGrouping[T]) GroupItems(pairs []*ItemPair[T]) []*ItemGroup[T] {
	if len(pairs) == 0 {
		return []*ItemGroup[T]{}
	}

	items, simMap, typeMap := collectItems(pairs)
	if len(items) == 0 {
		return []*ItemGroup[T]{}
	}

	// Union-Find across edges with similarity >= threshold
	uf := newUnionFind()
	for _, item := range items {
		uf.makeSet(item.ItemID())
	}
	for _, p := range pairs {
		if p == nil {
			continue
		}
		if p.Similarity >= c.threshold {
			uf.union(p.Item1.ItemID(), p.Item2.ItemID())
		}
	}

	// Build components
	comp := make(map[int][]T)
	for _, item := range items {
		r := uf.find(item.ItemID())
		comp[r] = append(comp[r], item)
	}

	// Iterate components in sorted root order for deterministic group IDs.
	roots := make([]int, 0, len(comp))
	for r := range comp {
		roots = append(roots, r)
	}
	sort.Ints(roots)

	// Convert to groups, exclude singletons
	groups := make([]*ItemGroup[T], 0, len(comp))
	groupID := 0
	for _, root := range roots {
		members := comp[root]
		if len(members) < 2 {
			continue
		}
		sort.Slice(members, func(i, j int) bool { return itemLess(members[i], members[j]) })
		groups = append(groups, &ItemGroup[T]{
			ID:         groupID,
			Items:      members,
			GroupType:  majorityType(typeMap, simMap, members),
			Similarity: averageGroupSimilarity(simMap, members),
		})
		groupID++
	}

	sortItemGroups(groups)

	return groups
}

// ---------------------------------------------------------------------------
// 2. KCoreGrouping — k-core subgraph decomposition
// ---------------------------------------------------------------------------

// KCoreGrouping ensures each item has at least k similar neighbors.
type KCoreGrouping[T GroupableItem] struct {
	threshold float64
	k         int
}

func NewKCoreGrouping[T GroupableItem](threshold float64, k int) *KCoreGrouping[T] {
	if k < 2 {
		k = 2 // Minimum meaningful value
	}
	return &KCoreGrouping[T]{threshold: threshold, k: k}
}

func (kg *KCoreGrouping[T]) Name() string { return string(ModeKCore) }

func (kg *KCoreGrouping[T]) GroupItems(pairs []*ItemPair[T]) []*ItemGroup[T] {
	if len(pairs) == 0 {
		return []*ItemGroup[T]{}
	}

	items, simMap, typeMap := collectItems(pairs)
	if len(items) == 0 {
		return []*ItemGroup[T]{}
	}

	// Build adjacency with edges meeting threshold
	adj := make(map[int]map[int]float64, len(items))
	for _, item := range items {
		adj[item.ItemID()] = make(map[int]float64)
	}
	for _, p := range pairs {
		if p == nil {
			continue
		}
		if p.Similarity >= kg.threshold {
			adj[p.Item1.ItemID()][p.Item2.ItemID()] = p.Similarity
			adj[p.Item2.ItemID()][p.Item1.ItemID()] = p.Similarity
		}
	}

	// Build item ID to item map
	itemByID := make(map[int]T)
	for _, item := range items {
		itemByID[item.ItemID()] = item
	}

	// Compute initial degrees
	degree := make(map[int]int, len(items))
	for id, nbrs := range adj {
		degree[id] = len(nbrs)
	}

	// Queue for items with degree < k
	q := list.New()
	inQueue := make(map[int]bool)
	for id, d := range degree {
		if d < kg.k {
			q.PushBack(id)
			inQueue[id] = true
		}
	}

	// Iteratively remove low-degree items
	removed := make(map[int]bool)
	for q.Len() > 0 {
		e := q.Front()
		q.Remove(e)
		v := e.Value.(int)
		if removed[v] {
			continue
		}
		removed[v] = true
		// Decrease degree of neighbors
		for u := range adj[v] {
			if removed[u] {
				continue
			}
			degree[u]--
			delete(adj[u], v)
			if degree[u] < kg.k && !inQueue[u] {
				q.PushBack(u)
				inQueue[u] = true
			}
		}
		// Clear v's adjacency
		delete(adj, v)
	}

	// Remaining items form the k-core subgraph
	// Now find connected components among remaining items
	groups := make([]*ItemGroup[T], 0)
	visited := make(map[int]bool)
	groupID := 0

	// Build deterministic order
	sort.Slice(items, func(i, j int) bool { return itemLess(items[i], items[j]) })

	for _, start := range items {
		if removed[start.ItemID()] || visited[start.ItemID()] || adj[start.ItemID()] == nil {
			continue
		}
		// BFS/DFS to collect component
		stack := []int{start.ItemID()}
		component := make([]T, 0)
		visited[start.ItemID()] = true
		for len(stack) > 0 {
			v := stack[len(stack)-1]
			stack = stack[:len(stack)-1]
			component = append(component, itemByID[v])
			for u := range adj[v] {
				if !removed[u] && !visited[u] {
					visited[u] = true
					stack = append(stack, u)
				}
			}
		}
		if len(component) < 2 {
			continue
		}
		sort.Slice(component, func(i, j int) bool { return itemLess(component[i], component[j]) })
		groups = append(groups, &ItemGroup[T]{
			ID:         groupID,
			Items:      component,
			GroupType:  majorityType(typeMap, simMap, component),
			Similarity: averageGroupSimilarity(simMap, component),
		})
		groupID++
	}

	sortItemGroups(groups)

	return groups
}

// ---------------------------------------------------------------------------
// 3. StarMedoidGrouping — iterative medoid refinement
// ---------------------------------------------------------------------------

// StarMedoidGrouping uses iterative medoid optimization for balanced precision/recall.
type StarMedoidGrouping[T GroupableItem] struct {
	threshold float64
}

func NewStarMedoidGrouping[T GroupableItem](threshold float64) *StarMedoidGrouping[T] {
	return &StarMedoidGrouping[T]{threshold: threshold}
}

func (s *StarMedoidGrouping[T]) Name() string { return string(ModeStarMedoid) }

func (s *StarMedoidGrouping[T]) GroupItems(pairs []*ItemPair[T]) []*ItemGroup[T] {
	if len(pairs) == 0 {
		return []*ItemGroup[T]{}
	}

	items, simMap, typeMap := collectItems(pairs)
	if len(items) == 0 {
		return []*ItemGroup[T]{}
	}

	// Build item ID to item map
	itemByID := make(map[int]T)
	for _, item := range items {
		itemByID[item.ItemID()] = item
	}

	// Phase 1: Initial clustering using Union-Find (same as ConnectedGrouping)
	uf := newUnionFind()
	for _, item := range items {
		uf.makeSet(item.ItemID())
	}
	for _, p := range pairs {
		if p == nil {
			continue
		}
		if p.Similarity >= s.threshold {
			uf.union(p.Item1.ItemID(), p.Item2.ItemID())
		}
	}

	// Build initial components
	comp := make(map[int][]T)
	for _, item := range items {
		r := uf.find(item.ItemID())
		comp[r] = append(comp[r], item)
	}

	// Convert to groups (singletons excluded)
	type groupData struct {
		members   []T
		medoid    T
		hasMedoid bool
	}
	groups := make([]*groupData, 0)
	componentRoots := make([]int, 0, len(comp))
	for root := range comp {
		componentRoots = append(componentRoots, root)
	}
	sort.Slice(componentRoots, func(i, j int) bool {
		left, right := comp[componentRoots[i]], comp[componentRoots[j]]
		sort.Slice(left, func(a, b int) bool { return itemLess(left[a], left[b]) })
		sort.Slice(right, func(a, b int) bool { return itemLess(right[a], right[b]) })
		return itemLess(left[0], right[0])
	})
	for _, root := range componentRoots {
		members := comp[root]
		if len(members) < 2 {
			continue
		}
		groups = append(groups, &groupData{members: members})
	}

	if len(groups) == 0 {
		return []*ItemGroup[T]{}
	}

	// Phase 2: Iterative medoid refinement
	for iter := 0; iter < starMedoidMaxIterations; iter++ {
		// Find medoid for each group
		for _, g := range groups {
			g.medoid, g.hasMedoid = s.findMedoid(g.members, simMap)
		}

		// Build item-to-group map for O(1) lookup
		itemToGroup := make(map[int]int)
		for gi, g := range groups {
			for _, m := range g.members {
				itemToGroup[m.ItemID()] = gi
			}
		}

		// Reassign items to closest medoid
		newAssignment := make(map[int]int) // item ID -> group index
		changed := 0

		for _, item := range items {
			bestGroup := -1
			bestSim := -1.0

			for gi, g := range groups {
				if !g.hasMedoid {
					continue
				}
				sim := itemSimilarity(simMap, item, g.medoid)
				if sim >= s.threshold && sim > bestSim {
					bestSim = sim
					bestGroup = gi
				}
			}

			// Find current group using O(1) lookup
			currentGroup, inGroup := itemToGroup[item.ItemID()]

			if bestGroup >= 0 {
				newAssignment[item.ItemID()] = bestGroup
				if bestGroup != currentGroup {
					changed++
				}
			} else if inGroup {
				// Item doesn't match any medoid above threshold, keep in current group
				newAssignment[item.ItemID()] = currentGroup
			}
		}

		// Rebuild groups from new assignments
		newGroups := make([]*groupData, len(groups))
		for i := range newGroups {
			newGroups[i] = &groupData{members: make([]T, 0)}
		}
		for _, item := range items {
			if gi, ok := newAssignment[item.ItemID()]; ok {
				newGroups[gi].members = append(newGroups[gi].members, itemByID[item.ItemID()])
			}
		}

		// Filter empty groups
		filteredGroups := make([]*groupData, 0)
		for _, g := range newGroups {
			if len(g.members) >= 2 {
				filteredGroups = append(filteredGroups, g)
			}
		}
		groups = filteredGroups

		if len(groups) == 0 {
			return []*ItemGroup[T]{}
		}

		// Check convergence
		if float64(changed)/float64(len(items)) < starMedoidConvergenceRatio {
			break
		}
	}

	// Phase 3: Finalize groups
	result := make([]*ItemGroup[T], 0, len(groups))
	groupID := 0
	for _, g := range groups {
		if len(g.members) < 2 {
			continue
		}
		sort.Slice(g.members, func(i, j int) bool { return itemLess(g.members[i], g.members[j]) })
		result = append(result, &ItemGroup[T]{
			ID:         groupID,
			Items:      g.members,
			GroupType:  majorityType(typeMap, simMap, g.members),
			Similarity: averageGroupSimilarity(simMap, g.members),
		})
		groupID++
	}

	sortItemGroups(result)
	for i, group := range result {
		group.ID = i
	}

	return result
}

// findMedoid returns the item with highest average similarity to all other members.
func (s *StarMedoidGrouping[T]) findMedoid(members []T, simMap map[string]float64) (T, bool) {
	var zero T
	if len(members) == 0 {
		return zero, false
	}
	if len(members) == 1 {
		return members[0], true
	}

	bestMedoid := zero
	found := false
	bestAvgSim := -1.0

	for _, candidate := range members {
		sumSim := 0.0
		for _, other := range members {
			if candidate.ItemID() != other.ItemID() {
				sumSim += itemSimilarity(simMap, candidate, other)
			}
		}
		avgSim := sumSim / float64(len(members)-1)
		if avgSim > bestAvgSim {
			bestAvgSim = avgSim
			bestMedoid = candidate
			found = true
		}
	}

	return bestMedoid, found
}

// ---------------------------------------------------------------------------
// 4. CompleteLinkageGrouping — agglomerative complete-linkage clustering
// ---------------------------------------------------------------------------

// CompleteLinkageGrouping ensures all pairs within a group have similarity above threshold.
type CompleteLinkageGrouping[T GroupableItem] struct {
	threshold float64
}

func NewCompleteLinkageGrouping[T GroupableItem](threshold float64) *CompleteLinkageGrouping[T] {
	return &CompleteLinkageGrouping[T]{threshold: threshold}
}

func (c *CompleteLinkageGrouping[T]) Name() string { return string(ModeCompleteLinkage) }

func (c *CompleteLinkageGrouping[T]) GroupItems(pairs []*ItemPair[T]) []*ItemGroup[T] {
	input := c.collectInput(pairs)
	if len(input.items) < 2 {
		return []*ItemGroup[T]{}
	}

	clusterer := newCompleteLinkageClusterer(input.items, input.edges)
	clusterer.mergeUntilStable()

	return c.buildGroups(clusterer.activeClusters(), input.similarities, input.types)
}

type completeLinkageInput[T GroupableItem] struct {
	items        []T
	similarities map[string]float64
	types        map[string]domain.CloneType
	edges        []completeLinkageEdge
}

type completeLinkagePairRecord[T GroupableItem] struct {
	left       T
	right      T
	similarity float64
	cloneType  domain.CloneType
}

type completeLinkageEdge struct {
	leftID  int
	rightID int
	score   float64
}

func (c *CompleteLinkageGrouping[T]) collectInput(pairs []*ItemPair[T]) completeLinkageInput[T] {
	input := completeLinkageInput[T]{
		items:        make([]T, 0),
		similarities: make(map[string]float64),
		types:        make(map[string]domain.CloneType),
	}

	seen := make(map[int]struct{})
	pairRecords := make(map[string]completeLinkagePairRecord[T])
	for _, pair := range pairs {
		if pair == nil {
			continue
		}

		if _, ok := seen[pair.Item1.ItemID()]; !ok {
			seen[pair.Item1.ItemID()] = struct{}{}
			input.items = append(input.items, pair.Item1)
		}
		if _, ok := seen[pair.Item2.ItemID()]; !ok {
			seen[pair.Item2.ItemID()] = struct{}{}
			input.items = append(input.items, pair.Item2)
		}

		key := metadataPairKey(pair.Item1, pair.Item2)
		record, ok := pairRecords[key]
		if !ok || pair.Similarity > record.similarity || (almostEqual(pair.Similarity, record.similarity) && pair.PairType < record.cloneType) {
			pairRecords[key] = completeLinkagePairRecord[T]{
				left:       pair.Item1,
				right:      pair.Item2,
				similarity: pair.Similarity,
				cloneType:  pair.PairType,
			}
			input.similarities[key] = pair.Similarity
			input.types[key] = pair.PairType
		}
	}

	itemIndexes := make(map[int]int, len(input.items))
	for index, item := range input.items {
		itemIndexes[item.ItemID()] = index
	}

	input.edges = make([]completeLinkageEdge, 0, len(pairRecords))
	for _, record := range pairRecords {
		if record.similarity < c.threshold {
			continue
		}

		leftID, rightID := orderClusterIDs(itemIndexes[record.left.ItemID()], itemIndexes[record.right.ItemID()])
		input.edges = append(input.edges, completeLinkageEdge{
			leftID:  leftID,
			rightID: rightID,
			score:   record.similarity,
		})
	}

	return input
}

func (c *CompleteLinkageGrouping[T]) buildGroups(activeClusters []*completeLinkageCluster[T], similarities map[string]float64, types map[string]domain.CloneType) []*ItemGroup[T] {
	groups := make([]*ItemGroup[T], 0, len(activeClusters))
	groupID := 0
	for _, cluster := range activeClusters {
		members := cluster.members
		if len(members) < 2 {
			continue
		}

		// Keep a final safety check so the optimized clusterer cannot return a
		// non-clique even if an internal update regresses later.
		valid := true
		for i := 0; i < len(members) && valid; i++ {
			for j := i + 1; j < len(members); j++ {
				if itemSimilarity(similarities, members[i], members[j]) < c.threshold {
					valid = false
					break
				}
			}
		}
		if !valid {
			continue
		}

		sortedMembers := append([]T(nil), members...)
		sort.Slice(sortedMembers, func(i, j int) bool { return itemLess(sortedMembers[i], sortedMembers[j]) })

		groups = append(groups, &ItemGroup[T]{
			ID:         groupID,
			Items:      sortedMembers,
			GroupType:  majorityType(types, similarities, sortedMembers),
			Similarity: averageGroupSimilarity(similarities, sortedMembers),
		})
		groupID++
	}

	sortItemGroups(groups)

	return groups
}

// completeLinkageClusterer stores only threshold-qualified inter-cluster edges.
// That keeps sparse workloads sparse while still supporting exact complete-linkage
// merges, since a merged cluster can stay adjacent to C only if both source
// clusters already had qualifying edges to C.
type completeLinkageClusterer[T GroupableItem] struct {
	clusters      []completeLinkageCluster[T]
	bestNeighbors *completeLinkageBestNeighborHeap
}

type completeLinkageCluster[T GroupableItem] struct {
	members   []T
	neighbors map[int]float64
	active    bool
}

func newCompleteLinkageClusterer[T GroupableItem](items []T, edges []completeLinkageEdge) *completeLinkageClusterer[T] {
	clusterer := &completeLinkageClusterer[T]{
		clusters:      make([]completeLinkageCluster[T], len(items)),
		bestNeighbors: newCompleteLinkageBestNeighborHeap(len(items)),
	}

	for clusterID, item := range items {
		clusterer.clusters[clusterID] = completeLinkageCluster[T]{
			members:   []T{item},
			neighbors: make(map[int]float64),
			active:    true,
		}
	}

	for _, edge := range edges {
		clusterer.clusters[edge.leftID].neighbors[edge.rightID] = edge.score
		clusterer.clusters[edge.rightID].neighbors[edge.leftID] = edge.score
	}

	for clusterID := range clusterer.clusters {
		clusterer.recomputeBestNeighbor(clusterID)
	}

	return clusterer
}

func (c *completeLinkageClusterer[T]) mergeUntilStable() {
	for {
		bestNeighbor, ok := c.bestNeighbors.popBest()
		if !ok {
			return
		}

		targetID, sourceID := orderClusterIDs(bestNeighbor.clusterID, bestNeighbor.neighborID)
		if !c.clusters[targetID].active || !c.clusters[sourceID].active {
			continue
		}

		c.mergeClusters(targetID, sourceID)
	}
}

func (c *completeLinkageClusterer[T]) mergeClusters(targetID, sourceID int) {
	target := &c.clusters[targetID]
	source := &c.clusters[sourceID]
	target.members = append(target.members, source.members...)

	affected := make(map[int]struct{}, len(target.neighbors)+len(source.neighbors))
	for neighborID := range target.neighbors {
		affected[neighborID] = struct{}{}
	}
	for neighborID := range source.neighbors {
		affected[neighborID] = struct{}{}
	}
	delete(affected, targetID)
	delete(affected, sourceID)

	newTargetNeighbors := make(map[int]float64)
	for neighborID := range affected {
		if !c.clusters[neighborID].active {
			continue
		}

		neighbor := &c.clusters[neighborID]
		delete(neighbor.neighbors, sourceID)

		targetScore, targetOK := target.neighbors[neighborID]
		sourceScore, sourceOK := source.neighbors[neighborID]
		if !targetOK || !sourceOK {
			delete(neighbor.neighbors, targetID)
			continue
		}

		mergedScore := targetScore
		if sourceScore < mergedScore {
			mergedScore = sourceScore
		}
		newTargetNeighbors[neighborID] = mergedScore
		neighbor.neighbors[targetID] = mergedScore
	}

	target.neighbors = newTargetNeighbors
	source.active = false
	source.members = nil
	source.neighbors = nil

	c.bestNeighbors.remove(sourceID)
	for neighborID := range affected {
		if c.clusters[neighborID].active {
			c.recomputeBestNeighbor(neighborID)
		}
	}
	c.recomputeBestNeighbor(targetID)
}

func (c *completeLinkageClusterer[T]) recomputeBestNeighbor(clusterID int) {
	cluster := &c.clusters[clusterID]
	if !cluster.active {
		c.bestNeighbors.remove(clusterID)
		return
	}

	bestNeighborID, bestScore, ok := c.findBestNeighbor(clusterID)
	if !ok {
		c.bestNeighbors.remove(clusterID)
		return
	}

	c.bestNeighbors.set(clusterID, bestNeighborID, bestScore)
}

func (c *completeLinkageClusterer[T]) findBestNeighbor(clusterID int) (int, float64, bool) {
	cluster := &c.clusters[clusterID]
	bestNeighborID := -1
	bestScore := 0.0
	for neighborID, score := range cluster.neighbors {
		if !c.clusters[neighborID].active {
			continue
		}
		if bestNeighborID == -1 || betterCompleteLinkageNeighbor(clusterID, neighborID, score, bestNeighborID, bestScore) {
			bestNeighborID = neighborID
			bestScore = score
		}
	}
	if bestNeighborID == -1 {
		return 0, 0.0, false
	}

	return bestNeighborID, bestScore, true
}

func betterCompleteLinkageNeighbor(clusterID, candidateNeighborID int, candidateScore float64, bestNeighborID int, bestScore float64) bool {
	if !almostEqual(candidateScore, bestScore) {
		return candidateScore > bestScore
	}

	candidateLeft, candidateRight := orderClusterIDs(clusterID, candidateNeighborID)
	bestLeft, bestRight := orderClusterIDs(clusterID, bestNeighborID)
	if candidateLeft != bestLeft {
		return candidateLeft < bestLeft
	}
	return candidateRight < bestRight
}

func (c *completeLinkageClusterer[T]) activeClusters() []*completeLinkageCluster[T] {
	activeClusters := make([]*completeLinkageCluster[T], 0)
	for clusterID := range c.clusters {
		if c.clusters[clusterID].active {
			activeClusters = append(activeClusters, &c.clusters[clusterID])
		}
	}
	return activeClusters
}

type completeLinkageBestNeighbor struct {
	clusterID  int
	neighborID int
	score      float64
}

type completeLinkageBestNeighborHeap struct {
	entries   []completeLinkageBestNeighbor
	positions []int
}

func newCompleteLinkageBestNeighborHeap(clusterCount int) *completeLinkageBestNeighborHeap {
	positions := make([]int, clusterCount)
	for i := range positions {
		positions[i] = -1
	}
	return &completeLinkageBestNeighborHeap{positions: positions}
}

func (h *completeLinkageBestNeighborHeap) Len() int { return len(h.entries) }

func (h *completeLinkageBestNeighborHeap) Less(i, j int) bool {
	if !almostEqual(h.entries[i].score, h.entries[j].score) {
		return h.entries[i].score > h.entries[j].score
	}

	leftI, rightI := orderClusterIDs(h.entries[i].clusterID, h.entries[i].neighborID)
	leftJ, rightJ := orderClusterIDs(h.entries[j].clusterID, h.entries[j].neighborID)
	if leftI != leftJ {
		return leftI < leftJ
	}
	if rightI != rightJ {
		return rightI < rightJ
	}
	return h.entries[i].clusterID < h.entries[j].clusterID
}

func (h *completeLinkageBestNeighborHeap) Swap(i, j int) {
	h.entries[i], h.entries[j] = h.entries[j], h.entries[i]
	h.positions[h.entries[i].clusterID] = i
	h.positions[h.entries[j].clusterID] = j
}

func (h *completeLinkageBestNeighborHeap) Push(x any) {
	entry := x.(completeLinkageBestNeighbor)
	h.positions[entry.clusterID] = len(h.entries)
	h.entries = append(h.entries, entry)
}

func (h *completeLinkageBestNeighborHeap) Pop() any {
	last := len(h.entries) - 1
	entry := h.entries[last]
	h.entries = h.entries[:last]
	h.positions[entry.clusterID] = -1
	return entry
}

func (h *completeLinkageBestNeighborHeap) set(clusterID, neighborID int, score float64) {
	if position := h.positions[clusterID]; position >= 0 {
		h.entries[position].neighborID = neighborID
		h.entries[position].score = score
		heap.Fix(h, position)
		return
	}

	heap.Push(h, completeLinkageBestNeighbor{
		clusterID:  clusterID,
		neighborID: neighborID,
		score:      score,
	})
}

func (h *completeLinkageBestNeighborHeap) remove(clusterID int) {
	position := h.positions[clusterID]
	if position < 0 {
		return
	}
	heap.Remove(h, position)
}

func (h *completeLinkageBestNeighborHeap) popBest() (completeLinkageBestNeighbor, bool) {
	if h.Len() == 0 {
		return completeLinkageBestNeighbor{}, false
	}
	return heap.Pop(h).(completeLinkageBestNeighbor), true
}

func orderClusterIDs(firstID, secondID int) (int, int) {
	if firstID < secondID {
		return firstID, secondID
	}
	return secondID, firstID
}

// ---------------------------------------------------------------------------
// 5. CentroidGrouping — BFS expansion with strict similarity to all members
// ---------------------------------------------------------------------------

// CentroidGrouping uses BFS expansion with strict similarity to all existing members.
type CentroidGrouping[T GroupableItem] struct {
	threshold float64
}

func NewCentroidGrouping[T GroupableItem](threshold float64) *CentroidGrouping[T] {
	return &CentroidGrouping[T]{threshold: threshold}
}

func (cg *CentroidGrouping[T]) Name() string { return string(ModeCentroid) }

func (cg *CentroidGrouping[T]) GroupItems(pairs []*ItemPair[T]) []*ItemGroup[T] {
	if len(pairs) == 0 {
		return []*ItemGroup[T]{}
	}

	items, simMap, typeMap := collectItems(pairs)
	if len(items) == 0 {
		return []*ItemGroup[T]{}
	}

	// Build adjacency with edges meeting threshold
	neighbors := make(map[int][]int, len(items)) // item ID -> neighbor IDs above threshold
	for _, item := range items {
		neighbors[item.ItemID()] = make([]int, 0)
	}
	for _, p := range pairs {
		if p == nil {
			continue
		}
		if p.Similarity >= cg.threshold {
			neighbors[p.Item1.ItemID()] = append(neighbors[p.Item1.ItemID()], p.Item2.ItemID())
			neighbors[p.Item2.ItemID()] = append(neighbors[p.Item2.ItemID()], p.Item1.ItemID())
		}
	}

	// Sort items for deterministic processing
	sort.Slice(items, func(i, j int) bool { return itemLess(items[i], items[j]) })

	// Build item ID to item map
	itemByID := make(map[int]T)
	for _, item := range items {
		itemByID[item.ItemID()] = item
	}

	// Sort neighbors for deterministic BFS traversal
	for id := range neighbors {
		sort.Ints(neighbors[id])
	}

	// BFS expansion from each unassigned item
	assigned := make(map[int]bool)
	groups := make([]*ItemGroup[T], 0)
	groupID := 0

	for _, seed := range items {
		if assigned[seed.ItemID()] {
			continue
		}

		// Start new group with seed
		members := []T{seed}
		assigned[seed.ItemID()] = true

		// BFS queue: neighbor IDs to consider
		queue := list.New()
		visited := make(map[int]bool)
		visited[seed.ItemID()] = true

		// Add seed's neighbors to queue
		for _, nid := range neighbors[seed.ItemID()] {
			if !visited[nid] && !assigned[nid] {
				queue.PushBack(nid)
				visited[nid] = true
			}
		}

		// BFS expansion
		for queue.Len() > 0 {
			e := queue.Front()
			queue.Remove(e)
			candidateID := e.Value.(int)

			if assigned[candidateID] {
				continue
			}

			candidate, ok := itemByID[candidateID]
			if !ok {
				continue
			}

			// Check if candidate is similar to ALL current members
			if cg.isSimilarToAll(candidate, members, simMap) {
				members = append(members, candidate)
				assigned[candidateID] = true

				// Add candidate's neighbors to queue
				for _, nid := range neighbors[candidateID] {
					if !visited[nid] && !assigned[nid] {
						queue.PushBack(nid)
						visited[nid] = true
					}
				}
			}
		}

		// Only keep groups with at least 2 members
		if len(members) >= 2 {
			sort.Slice(members, func(i, j int) bool { return itemLess(members[i], members[j]) })
			groups = append(groups, &ItemGroup[T]{
				ID:         groupID,
				Items:      members,
				GroupType:  majorityType(typeMap, simMap, members),
				Similarity: averageGroupSimilarity(simMap, members),
			})
			groupID++
		}
	}

	sortItemGroups(groups)

	return groups
}

// isSimilarToAll checks if candidate is similar to all members above threshold.
func (cg *CentroidGrouping[T]) isSimilarToAll(candidate T, members []T, simMap map[string]float64) bool {
	for _, member := range members {
		if itemSimilarity(simMap, candidate, member) < cg.threshold {
			return false
		}
	}
	return true
}

// ---------------------------------------------------------------------------
// Group deduplication post-passes
// ---------------------------------------------------------------------------

// GroupDedupeResult carries the surviving groups plus the keys of suppressed
// members (by location and ItemID) and suppressed pairs (by PairKey).
type GroupDedupeResult[T GroupableItem] struct {
	Groups          []*ItemGroup[T]
	Suppressed      map[string]struct{} // keyed by location and ItemID
	SuppressedPairs map[string]struct{} // keyed by PairKey
}

// DedupeStrictSubsetGroupMembers removes group members whose source range is a
// strict subset of (or identical to) another member's range in the same file.
// Groups reduced to fewer than two members are dropped.
//
// Why this exists: pair-detection paths typically reject *direct* pairs
// between overlapping same-file fragments, so pairs cannot contain a same-file
// `(A, B)` where one strictly covers the other. Union-Find grouping, however,
// still merges such fragments into one group via a shared distinct-file
// neighbor — e.g., pairs `(A=x.ts:512-542, C=y.ts:1-30)` and
// `(B=x.ts:515-542, C=y.ts:1-30)` are both legal yet transitively connect A
// and B. This post-pass collapses those overlapping windows back to the
// maximal one per file.
//
// For exactly-equal ranges (which UF can produce in the same way), the first
// occurrence is kept; later duplicates are suppressed for deterministic output.
func DedupeStrictSubsetGroupMembers[T GroupableItem](groups []*ItemGroup[T], pairs []*ItemPair[T]) GroupDedupeResult[T] {
	result := GroupDedupeResult[T]{
		Groups:     groups,
		Suppressed: make(map[string]struct{}),
	}
	if len(groups) == 0 {
		return result
	}

	out := make([]*ItemGroup[T], 0, len(groups))
	var similarities map[string]float64
	var types map[string]domain.CloneType
	metadataReady := false
	anyChanged := false
	for _, g := range groups {
		if g == nil {
			continue
		}
		kept, suppressed := filterMaximalPerFile(g.Items)
		for key := range suppressed {
			result.Suppressed[key] = struct{}{}
		}
		groupChanged := len(suppressed) > 0
		anyChanged = anyChanged || groupChanged
		if len(kept) < 2 {
			continue
		}
		g.Items = kept
		if groupChanged {
			if !metadataReady {
				similarities, types = pairMetadata(pairs)
				metadataReady = true
			}
			g.Similarity = averageGroupSimilarity(similarities, g.Items)
			g.GroupType = majorityType(types, similarities, g.Items)
		}
		out = append(out, g)
	}
	if anyChanged {
		sortItemGroups(out)
	}
	result.Groups = out
	return result
}

// coveredGroupSimilarityTolerance bounds how much weaker (in average
// similarity) a covering group may be while still subsuming a covered group.
// Overlapping windows of the same duplication shift similarity only slightly;
// a covered group that matches much more strongly than its covering group is
// a distinct, sharper finding (e.g., a near-identical inner block inside
// loosely similar functions) and is kept.
const coveredGroupSimilarityTolerance = 0.05

// DedupeCoveredGroups suppresses whole groups that are covered by another
// group: every member of the covered group fits inside a distinct member of
// the covering group (same file, containing line range), and the covering
// group's similarity is comparable or better. Such groups describe the same
// duplication relationship through slightly smaller windows and double-count
// it.
//
// Why DedupeStrictSubsetGroupMembers does not catch this: that pass compares
// members *within* one group. Here the overlapping windows sit in *different*
// groups, which stay disconnected when detection forbids the direct same-file
// pair that would have linked them.
//
// The group with the larger (covering) windows is kept, mirroring the
// maximal-window policy of filterMaximalPerFile. When two groups cover each
// other (identical member ranges), the earlier one in the slice wins.
func DedupeCoveredGroups[T GroupableItem](groups []*ItemGroup[T]) GroupDedupeResult[T] {
	result := GroupDedupeResult[T]{
		Groups:          groups,
		Suppressed:      make(map[string]struct{}),
		SuppressedPairs: make(map[string]struct{}),
	}
	if len(groups) < 2 {
		return result
	}

	suppressed := make([]bool, len(groups))
	order := make([]int, 0, len(groups))
	for i, group := range groups {
		if group != nil {
			order = append(order, i)
		}
	}
	// Consider outer groups first. Only a surviving group can suppress another,
	// because the similarity tolerance is intentionally not transitive.
	sort.SliceStable(order, func(a, b int) bool {
		i, j := order[a], order[b]
		iCoveredByJ := groupCoveredBy(groups[i], groups[j])
		jCoveredByI := groupCoveredBy(groups[j], groups[i])
		if iCoveredByJ != jCoveredByI {
			return jCoveredByI
		}
		return i < j
	})
	kept := make([]int, 0, len(order))
	for _, i := range order {
		for _, j := range kept {
			if !groupCoveredBy(groups[i], groups[j]) {
				continue
			}
			if groupCoveredBy(groups[j], groups[i]) || groups[i].Similarity <= groups[j].Similarity+coveredGroupSimilarityTolerance {
				suppressed[i] = true
				break
			}
		}
		if !suppressed[i] {
			kept = append(kept, i)
		}
	}

	out := make([]*ItemGroup[T], 0, len(groups))
	keptPairs := make(map[string]struct{})
	for i, g := range groups {
		if g == nil || suppressed[i] {
			continue
		}
		out = append(out, g)
		for first := 0; first < len(g.Items); first++ {
			for second := first + 1; second < len(g.Items); second++ {
				keptPairs[PairKey(g.Items[first], g.Items[second])] = struct{}{}
			}
		}
	}
	for i, g := range groups {
		if g == nil || !suppressed[i] {
			continue
		}
		for first := 0; first < len(g.Items); first++ {
			for second := first + 1; second < len(g.Items); second++ {
				key := PairKey(g.Items[first], g.Items[second])
				if _, needed := keptPairs[key]; !needed {
					result.SuppressedPairs[key] = struct{}{}
				}
			}
		}
	}
	result.Groups = out
	return result
}

// groupCoveredBy reports whether every member of inner can be matched to a
// distinct member of outer that covers it (same file, containing range,
// equality included). Distinctness matters: two disjoint inner blocks inside
// one outer member describe duplication *within* that member, which the outer
// group does not report.
func groupCoveredBy[T GroupableItem](inner, outer *ItemGroup[T]) bool {
	n := len(inner.Items)
	if n == 0 || n > len(outer.Items) {
		return false
	}
	candidates := make([][]int, n)
	for i, c := range inner.Items {
		for j, oc := range outer.Items {
			if locationCovers(oc.ItemLocation(), c.ItemLocation()) {
				candidates[i] = append(candidates[i], j)
			}
		}
		if len(candidates[i]) == 0 {
			return false
		}
	}
	// Bipartite matching via augmenting paths; group sizes are small.
	matchedInner := make([]int, len(outer.Items))
	for j := range matchedInner {
		matchedInner[j] = -1
	}
	var assign func(i int, visited []bool) bool
	assign = func(i int, visited []bool) bool {
		for _, j := range candidates[i] {
			if visited[j] {
				continue
			}
			visited[j] = true
			if matchedInner[j] == -1 || assign(matchedInner[j], visited) {
				matchedInner[j] = i
				return true
			}
		}
		return false
	}
	for i := 0; i < n; i++ {
		if !assign(i, make([]bool, len(outer.Items))) {
			return false
		}
	}
	return true
}

// locationCovers reports whether outer contains inner (same file, inclusive
// line and column ranges; equal ranges count as covered).
func locationCovers(outer, inner ItemLocation) bool {
	if outer.FilePath != inner.FilePath {
		return false
	}
	startsBefore := outer.StartLine < inner.StartLine ||
		(outer.StartLine == inner.StartLine && outer.StartCol <= inner.StartCol)
	endsAfter := outer.EndLine > inner.EndLine ||
		(outer.EndLine == inner.EndLine && outer.EndCol >= inner.EndCol)
	return startsBefore && endsAfter
}

// FilterGroupsWithoutBackingPairs drops groups whose refreshed metadata shows
// no positive-similarity pair actually backs them (e.g. every member pair was
// filtered out upstream), which would otherwise surface a group with zero
// similarity.
func FilterGroupsWithoutBackingPairs[T GroupableItem](groups []*ItemGroup[T], pairs []*ItemPair[T]) []*ItemGroup[T] {
	if len(groups) == 0 {
		return groups
	}
	similarities, types := pairMetadata(pairs)
	out := make([]*ItemGroup[T], 0, len(groups))
	for _, group := range groups {
		if group == nil || len(group.Items) < 2 {
			continue
		}
		group.Similarity = averageGroupSimilarity(similarities, group.Items)
		group.GroupType = majorityType(types, similarities, group.Items)
		if group.Similarity <= 0 {
			continue
		}
		out = append(out, group)
	}
	return out
}

// filterMaximalPerFile returns the subset of items that are maximal under the
// same-file containment order. An item is suppressed if any other kept item in
// the same file strictly covers it, or if it duplicates an earlier item's
// range exactly.
func filterMaximalPerFile[T GroupableItem](items []T) ([]T, map[string]struct{}) {
	suppressedItems := make(map[string]struct{})
	n := len(items)
	if n <= 1 {
		return items, suppressedItems
	}
	suppressed := make([]bool, n)
	for i := 0; i < n; i++ {
		if suppressed[i] {
			continue
		}
		for j := 0; j < n; j++ {
			if i == j || suppressed[j] {
				continue
			}
			if covers(items[i].ItemLocation(), items[j].ItemLocation(), i, j) {
				suppressed[j] = true
			}
		}
	}
	out := make([]T, 0, n)
	for i, item := range items {
		if suppressed[i] {
			suppressedItems[suppressedMemberKey(item)] = struct{}{}
			continue
		}
		out = append(out, item)
	}
	return out, suppressedItems
}

// covers reports whether outer (at index iOuter) covers inner (at index
// iInner) in the same file. Strict coverage suppresses inner outright;
// identical ranges suppress only the later index so that exactly one survives.
func covers(outer, inner ItemLocation, iOuter, iInner int) bool {
	if !locationCovers(outer, inner) {
		return false
	}
	if outer == inner {
		return iOuter < iInner
	}
	return true
}

// FilterPairsWithSuppressedMembers removes pairs that reference a suppressed
// member. Identity keys returned by DedupeStrictSubsetGroupMembers distinguish
// equal-location items; location-only ItemKey values remain accepted when a
// caller intentionally wants to suppress every item at a location.
func FilterPairsWithSuppressedMembers[T GroupableItem](pairs []*ItemPair[T], suppressed map[string]struct{}) []*ItemPair[T] {
	if len(pairs) == 0 || len(suppressed) == 0 {
		return pairs
	}
	out := make([]*ItemPair[T], 0, len(pairs))
	for _, pair := range pairs {
		if pair == nil {
			continue
		}
		_, item1Suppressed := suppressed[suppressedMemberKey(pair.Item1)]
		_, item1LocationSuppressed := suppressed[ItemKey(pair.Item1)]
		if item1Suppressed || item1LocationSuppressed {
			continue
		}
		_, item2Suppressed := suppressed[suppressedMemberKey(pair.Item2)]
		_, item2LocationSuppressed := suppressed[ItemKey(pair.Item2)]
		if item2Suppressed || item2LocationSuppressed {
			continue
		}
		out = append(out, pair)
	}
	return out
}

// FilterSuppressedPairs removes pairs whose PairKey is in the suppressed set.
func FilterSuppressedPairs[T GroupableItem](pairs []*ItemPair[T], suppressed map[string]struct{}) []*ItemPair[T] {
	if len(pairs) == 0 || len(suppressed) == 0 {
		return pairs
	}
	out := make([]*ItemPair[T], 0, len(pairs))
	for _, pair := range pairs {
		if pair == nil {
			continue
		}
		if _, ok := suppressed[PairKey(pair.Item1, pair.Item2)]; ok {
			continue
		}
		out = append(out, pair)
	}
	return out
}
