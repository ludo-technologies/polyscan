package analyzer

import (
	"container/heap"
	"container/list"
	"fmt"
	"sort"

	"github.com/ludo-technologies/polyscan/jscan/domain"
)

// GroupingMode represents the mode of grouping strategy
type GroupingMode string

const (
	GroupingModeConnected       GroupingMode = "connected"
	GroupingModeKCore           GroupingMode = "k_core"
	GroupingModeStarMedoid      GroupingMode = "star_medoid"
	GroupingModeCompleteLinkage GroupingMode = "complete_linkage"
	GroupingModeCentroid        GroupingMode = "centroid"
)

// Algorithm-specific constants
const (
	starMedoidMaxIterations    = 10
	starMedoidConvergenceRatio = 0.01
)

// GroupingConfig holds configuration for clone grouping
type GroupingConfig struct {
	Mode           GroupingMode
	Threshold      float64
	KCoreK         int
	Type1Threshold float64
	Type2Threshold float64
	Type3Threshold float64
	Type4Threshold float64
}

// GroupingStrategy defines a strategy for grouping clone pairs into clone groups.
type GroupingStrategy interface {
	// GroupClones groups the given clone pairs into clone groups.
	GroupClones(pairs []*domain.ClonePair) []*domain.CloneGroup
	// GetName returns the strategy name.
	GetName() string
}

// CreateGroupingStrategy creates a grouping strategy based on config
func CreateGroupingStrategy(config GroupingConfig) GroupingStrategy {
	switch config.Mode {
	case GroupingModeKCore:
		return NewKCoreGrouping(config.Threshold, config.KCoreK)
	case GroupingModeStarMedoid:
		return NewStarMedoidGrouping(config.Threshold)
	case GroupingModeCompleteLinkage:
		return NewCompleteLinkageGrouping(config.Threshold)
	case GroupingModeCentroid:
		return NewCentroidGrouping(config.Threshold)
	case GroupingModeConnected:
		fallthrough
	default:
		return NewConnectedGrouping(config.Threshold)
	}
}

// ConnectedGrouping wraps transitive grouping logic using Union-Find
type ConnectedGrouping struct {
	threshold float64
}

func NewConnectedGrouping(threshold float64) *ConnectedGrouping {
	return &ConnectedGrouping{threshold: threshold}
}

func (c *ConnectedGrouping) GetName() string { return "Connected Components" }

func (c *ConnectedGrouping) GroupClones(pairs []*domain.ClonePair) []*domain.CloneGroup {
	if len(pairs) == 0 {
		return []*domain.CloneGroup{}
	}

	// Build set of clones and adjacency filtered by threshold
	clones := make([]*domain.Clone, 0)
	seen := make(map[int]struct{})
	simMap := make(map[string]float64)
	typeMap := make(map[string]domain.CloneType)

	addClone := func(clone *domain.Clone) {
		if clone == nil {
			return
		}
		if _, ok := seen[clone.ID]; !ok {
			seen[clone.ID] = struct{}{}
			clones = append(clones, clone)
		}
	}

	for _, p := range pairs {
		if p == nil || p.Clone1 == nil || p.Clone2 == nil {
			continue
		}
		addClone(p.Clone1)
		addClone(p.Clone2)

		// Cache similarity and type for existing pair
		key := clonePairKey(p.Clone1, p.Clone2)
		if old, ok := simMap[key]; !ok || p.Similarity > old {
			simMap[key] = p.Similarity
			typeMap[key] = p.Type
		}
	}

	if len(clones) == 0 {
		return []*domain.CloneGroup{}
	}

	// Union-Find across edges with similarity >= threshold
	parent := make(map[int]int, len(clones))
	rank := make(map[int]int, len(clones))

	var find func(int) int
	find = func(x int) int {
		if parent[x] != x {
			parent[x] = find(parent[x])
		}
		return parent[x]
	}
	union := func(a, b int) {
		ra := find(a)
		rb := find(b)
		if ra == rb {
			return
		}
		if rank[ra] < rank[rb] {
			parent[ra] = rb
		} else if rank[ra] > rank[rb] {
			parent[rb] = ra
		} else {
			parent[rb] = ra
			rank[ra]++
		}
	}
	for _, clone := range clones {
		parent[clone.ID] = clone.ID
		rank[clone.ID] = 0
	}

	// Union only for edges meeting threshold
	for _, p := range pairs {
		if p == nil || p.Clone1 == nil || p.Clone2 == nil {
			continue
		}
		if p.Similarity >= c.threshold {
			union(p.Clone1.ID, p.Clone2.ID)
		}
	}

	// Build components
	comp := make(map[int][]*domain.Clone)
	cloneByID := make(map[int]*domain.Clone)
	for _, clone := range clones {
		cloneByID[clone.ID] = clone
		r := find(clone.ID)
		comp[r] = append(comp[r], clone)
	}

	// Convert to groups, exclude singletons
	groups := make([]*domain.CloneGroup, 0, len(comp))
	groupID := 0
	for _, members := range comp {
		if len(members) < 2 {
			continue
		}
		sort.Slice(members, func(i, j int) bool { return cloneLess(members[i], members[j]) })
		g := &domain.CloneGroup{
			ID:     groupID,
			Clones: make([]*domain.Clone, 0, len(members)),
			Size:   len(members),
		}
		groupID++
		for _, clone := range members {
			g.AddClone(clone)
		}
		// Compute average similarity using cached pairs among members
		g.Similarity = averageGroupSimilarityClones(simMap, members)
		// Determine predominant clone type from within-group available pairs
		g.Type = majorityCloneTypeClones(typeMap, simMap, members)
		groups = append(groups, g)
	}

	// Sort groups by decreasing similarity then size
	sort.Slice(groups, func(i, j int) bool {
		if !almostEqual(groups[i].Similarity, groups[j].Similarity) {
			return groups[i].Similarity > groups[j].Similarity
		}
		if groups[i].Size != groups[j].Size {
			return groups[i].Size > groups[j].Size
		}
		if len(groups[i].Clones) == 0 || len(groups[j].Clones) == 0 {
			return false
		}
		return cloneLess(groups[i].Clones[0], groups[j].Clones[0])
	})

	return groups
}

// KCoreGrouping ensures each clone has at least k similar neighbors
type KCoreGrouping struct {
	threshold float64
	k         int
}

func NewKCoreGrouping(threshold float64, k int) *KCoreGrouping {
	if k < 2 {
		k = 2 // Minimum meaningful value
	}
	return &KCoreGrouping{threshold: threshold, k: k}
}

func (kg *KCoreGrouping) GetName() string { return fmt.Sprintf("%d-Core", kg.k) }

func (kg *KCoreGrouping) GroupClones(pairs []*domain.ClonePair) []*domain.CloneGroup {
	if len(pairs) == 0 {
		return []*domain.CloneGroup{}
	}

	// Collect unique clones and build adjacency with edges meeting threshold
	clones := make([]*domain.Clone, 0)
	seen := make(map[int]struct{})
	adj := make(map[int]map[int]float64)
	simMap := make(map[string]float64)
	typeMap := make(map[string]domain.CloneType)

	addClone := func(clone *domain.Clone) {
		if clone == nil {
			return
		}
		if _, ok := seen[clone.ID]; !ok {
			seen[clone.ID] = struct{}{}
			clones = append(clones, clone)
			adj[clone.ID] = make(map[int]float64)
		}
	}

	for _, p := range pairs {
		if p == nil || p.Clone1 == nil || p.Clone2 == nil {
			continue
		}
		addClone(p.Clone1)
		addClone(p.Clone2)
		key := clonePairKey(p.Clone1, p.Clone2)
		if old, ok := simMap[key]; !ok || p.Similarity > old {
			simMap[key] = p.Similarity
			typeMap[key] = p.Type
		}
		if p.Similarity >= kg.threshold {
			adj[p.Clone1.ID][p.Clone2.ID] = p.Similarity
			adj[p.Clone2.ID][p.Clone1.ID] = p.Similarity
		}
	}

	if len(clones) == 0 {
		return []*domain.CloneGroup{}
	}

	// Build clone ID to clone map
	cloneByID := make(map[int]*domain.Clone)
	for _, clone := range clones {
		cloneByID[clone.ID] = clone
	}

	// Compute initial degrees
	degree := make(map[int]int, len(clones))
	for id, nbrs := range adj {
		degree[id] = len(nbrs)
	}

	// Queue for clones with degree < k
	q := list.New()
	inQueue := make(map[int]bool)
	for id, d := range degree {
		if d < kg.k {
			q.PushBack(id)
			inQueue[id] = true
		}
	}

	// Iteratively remove low-degree clones
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

	// Remaining clones form the k-core subgraph
	// Now find connected components among remaining clones
	groups := make([]*domain.CloneGroup, 0)
	visited := make(map[int]bool)
	groupID := 0

	// Build deterministic order
	sort.Slice(clones, func(i, j int) bool { return cloneLess(clones[i], clones[j]) })

	for _, start := range clones {
		if removed[start.ID] || visited[start.ID] || adj[start.ID] == nil {
			continue
		}
		// BFS/DFS to collect component
		stack := []int{start.ID}
		component := make([]*domain.Clone, 0)
		visited[start.ID] = true
		for len(stack) > 0 {
			v := stack[len(stack)-1]
			stack = stack[:len(stack)-1]
			component = append(component, cloneByID[v])
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
		sort.Slice(component, func(i, j int) bool { return cloneLess(component[i], component[j]) })
		g := &domain.CloneGroup{
			ID:     groupID,
			Clones: make([]*domain.Clone, 0, len(component)),
			Size:   len(component),
		}
		groupID++
		for _, clone := range component {
			g.AddClone(clone)
		}
		g.Similarity = averageGroupSimilarityClones(simMap, component)
		g.Type = majorityCloneTypeClones(typeMap, simMap, component)
		groups = append(groups, g)
	}

	// Sort groups by similarity then size
	sort.Slice(groups, func(i, j int) bool {
		if !almostEqual(groups[i].Similarity, groups[j].Similarity) {
			return groups[i].Similarity > groups[j].Similarity
		}
		if groups[i].Size != groups[j].Size {
			return groups[i].Size > groups[j].Size
		}
		if len(groups[i].Clones) == 0 || len(groups[j].Clones) == 0 {
			return false
		}
		return cloneLess(groups[i].Clones[0], groups[j].Clones[0])
	})

	return groups
}

// StarMedoidGrouping uses iterative medoid optimization for balanced precision/recall
type StarMedoidGrouping struct {
	threshold float64
}

func NewStarMedoidGrouping(threshold float64) *StarMedoidGrouping {
	return &StarMedoidGrouping{threshold: threshold}
}

func (s *StarMedoidGrouping) GetName() string { return "Star/Medoid" }

func (s *StarMedoidGrouping) GroupClones(pairs []*domain.ClonePair) []*domain.CloneGroup {
	if len(pairs) == 0 {
		return []*domain.CloneGroup{}
	}

	// Build set of clones and similarity map
	clones := make([]*domain.Clone, 0)
	seen := make(map[int]struct{})
	simMap := make(map[string]float64)
	typeMap := make(map[string]domain.CloneType)

	addClone := func(clone *domain.Clone) {
		if clone == nil {
			return
		}
		if _, ok := seen[clone.ID]; !ok {
			seen[clone.ID] = struct{}{}
			clones = append(clones, clone)
		}
	}

	for _, p := range pairs {
		if p == nil || p.Clone1 == nil || p.Clone2 == nil {
			continue
		}
		addClone(p.Clone1)
		addClone(p.Clone2)
		key := clonePairKey(p.Clone1, p.Clone2)
		if old, ok := simMap[key]; !ok || p.Similarity > old {
			simMap[key] = p.Similarity
			typeMap[key] = p.Type
		}
	}

	if len(clones) == 0 {
		return []*domain.CloneGroup{}
	}

	// Build clone ID to clone map
	cloneByID := make(map[int]*domain.Clone)
	for _, clone := range clones {
		cloneByID[clone.ID] = clone
	}

	// Phase 1: Initial clustering using Union-Find (same as ConnectedGrouping)
	parent := make(map[int]int, len(clones))
	rank := make(map[int]int, len(clones))

	var find func(int) int
	find = func(x int) int {
		if parent[x] != x {
			parent[x] = find(parent[x])
		}
		return parent[x]
	}
	union := func(a, b int) {
		ra := find(a)
		rb := find(b)
		if ra == rb {
			return
		}
		if rank[ra] < rank[rb] {
			parent[ra] = rb
		} else if rank[ra] > rank[rb] {
			parent[rb] = ra
		} else {
			parent[rb] = ra
			rank[ra]++
		}
	}
	for _, clone := range clones {
		parent[clone.ID] = clone.ID
		rank[clone.ID] = 0
	}

	// Union only for edges meeting threshold
	for _, p := range pairs {
		if p == nil || p.Clone1 == nil || p.Clone2 == nil {
			continue
		}
		if p.Similarity >= s.threshold {
			union(p.Clone1.ID, p.Clone2.ID)
		}
	}

	// Build initial components
	comp := make(map[int][]*domain.Clone)
	for _, clone := range clones {
		r := find(clone.ID)
		comp[r] = append(comp[r], clone)
	}

	// Convert to groups (including singletons for now, we'll filter later)
	type groupData struct {
		members []*domain.Clone
		medoid  *domain.Clone
	}
	groups := make([]*groupData, 0)
	for _, members := range comp {
		if len(members) < 2 {
			continue
		}
		g := &groupData{members: members}
		groups = append(groups, g)
	}

	if len(groups) == 0 {
		return []*domain.CloneGroup{}
	}

	// Phase 2: Iterative medoid refinement
	for iter := 0; iter < starMedoidMaxIterations; iter++ {
		// Find medoid for each group
		for _, g := range groups {
			g.medoid = s.findMedoid(g.members, simMap)
		}

		// Build clone-to-group map for O(1) lookup
		cloneToGroup := make(map[int]int)
		for gi, g := range groups {
			for _, m := range g.members {
				cloneToGroup[m.ID] = gi
			}
		}

		// Reassign clones to closest medoid
		newAssignment := make(map[int]int) // clone ID -> group index
		changed := 0

		for _, clone := range clones {
			bestGroup := -1
			bestSim := -1.0

			for gi, g := range groups {
				if g.medoid == nil {
					continue
				}
				sim := cloneSimilarity(simMap, clone, g.medoid)
				if sim >= s.threshold && sim > bestSim {
					bestSim = sim
					bestGroup = gi
				}
			}

			// Find current group using O(1) lookup
			currentGroup, inGroup := cloneToGroup[clone.ID]

			if bestGroup >= 0 {
				newAssignment[clone.ID] = bestGroup
				if bestGroup != currentGroup {
					changed++
				}
			} else if inGroup {
				// Clone doesn't match any medoid above threshold, keep in current group
				newAssignment[clone.ID] = currentGroup
			}
		}

		// Rebuild groups from new assignments
		newGroups := make([]*groupData, len(groups))
		for i := range newGroups {
			newGroups[i] = &groupData{members: make([]*domain.Clone, 0)}
		}
		for cloneID, gi := range newAssignment {
			newGroups[gi].members = append(newGroups[gi].members, cloneByID[cloneID])
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
			return []*domain.CloneGroup{}
		}

		// Check convergence
		if float64(changed)/float64(len(clones)) < starMedoidConvergenceRatio {
			break
		}
	}

	// Phase 3: Finalize groups
	result := make([]*domain.CloneGroup, 0, len(groups))
	groupID := 0
	for _, g := range groups {
		if len(g.members) < 2 {
			continue
		}
		sort.Slice(g.members, func(i, j int) bool { return cloneLess(g.members[i], g.members[j]) })
		cg := &domain.CloneGroup{
			ID:     groupID,
			Clones: make([]*domain.Clone, 0, len(g.members)),
			Size:   len(g.members),
		}
		groupID++
		for _, clone := range g.members {
			cg.AddClone(clone)
		}
		cg.Similarity = averageGroupSimilarityClones(simMap, g.members)
		cg.Type = majorityCloneTypeClones(typeMap, simMap, g.members)
		result = append(result, cg)
	}

	// Sort groups by similarity then size
	sort.Slice(result, func(i, j int) bool {
		if !almostEqual(result[i].Similarity, result[j].Similarity) {
			return result[i].Similarity > result[j].Similarity
		}
		if result[i].Size != result[j].Size {
			return result[i].Size > result[j].Size
		}
		if len(result[i].Clones) == 0 || len(result[j].Clones) == 0 {
			return false
		}
		return cloneLess(result[i].Clones[0], result[j].Clones[0])
	})

	return result
}

// findMedoid returns the clone with highest average similarity to all other members
func (s *StarMedoidGrouping) findMedoid(members []*domain.Clone, simMap map[string]float64) *domain.Clone {
	if len(members) == 0 {
		return nil
	}
	if len(members) == 1 {
		return members[0]
	}

	var bestMedoid *domain.Clone
	bestAvgSim := -1.0

	for _, candidate := range members {
		sumSim := 0.0
		for _, other := range members {
			if candidate.ID != other.ID {
				sumSim += cloneSimilarity(simMap, candidate, other)
			}
		}
		avgSim := sumSim / float64(len(members)-1)
		if avgSim > bestAvgSim {
			bestAvgSim = avgSim
			bestMedoid = candidate
		}
	}

	return bestMedoid
}

// CompleteLinkageGrouping ensures all pairs within a group have similarity above threshold
type CompleteLinkageGrouping struct {
	threshold float64
}

func NewCompleteLinkageGrouping(threshold float64) *CompleteLinkageGrouping {
	return &CompleteLinkageGrouping{threshold: threshold}
}

func (c *CompleteLinkageGrouping) GetName() string { return "Complete Linkage" }

func (c *CompleteLinkageGrouping) GroupClones(pairs []*domain.ClonePair) []*domain.CloneGroup {
	input := c.collectInput(pairs)
	if len(input.clones) < 2 {
		return []*domain.CloneGroup{}
	}

	clusterer := newCompleteLinkageClusterer(input.clones, input.edges)
	clusterer.mergeUntilStable()

	return c.buildGroups(clusterer.activeClusters(), input.similarities, input.types)
}

type completeLinkageInput struct {
	clones       []*domain.Clone
	similarities map[string]float64
	types        map[string]domain.CloneType
	edges        []completeLinkageEdge
}

type completeLinkagePairRecord struct {
	left       *domain.Clone
	right      *domain.Clone
	similarity float64
	cloneType  domain.CloneType
}

type completeLinkageEdge struct {
	leftID  int
	rightID int
	score   float64
}

func (c *CompleteLinkageGrouping) collectInput(pairs []*domain.ClonePair) completeLinkageInput {
	input := completeLinkageInput{
		clones:       make([]*domain.Clone, 0),
		similarities: make(map[string]float64),
		types:        make(map[string]domain.CloneType),
	}

	seen := make(map[int]struct{})
	pairRecords := make(map[string]completeLinkagePairRecord)
	for _, pair := range pairs {
		if pair == nil || pair.Clone1 == nil || pair.Clone2 == nil {
			continue
		}

		if _, ok := seen[pair.Clone1.ID]; !ok {
			seen[pair.Clone1.ID] = struct{}{}
			input.clones = append(input.clones, pair.Clone1)
		}
		if _, ok := seen[pair.Clone2.ID]; !ok {
			seen[pair.Clone2.ID] = struct{}{}
			input.clones = append(input.clones, pair.Clone2)
		}

		key := clonePairKey(pair.Clone1, pair.Clone2)
		record, ok := pairRecords[key]
		if !ok || pair.Similarity > record.similarity {
			pairRecords[key] = completeLinkagePairRecord{
				left:       pair.Clone1,
				right:      pair.Clone2,
				similarity: pair.Similarity,
				cloneType:  pair.Type,
			}
			input.similarities[key] = pair.Similarity
			input.types[key] = pair.Type
		}
	}

	cloneIndexes := make(map[int]int, len(input.clones))
	for index, clone := range input.clones {
		cloneIndexes[clone.ID] = index
	}

	input.edges = make([]completeLinkageEdge, 0, len(pairRecords))
	for _, record := range pairRecords {
		if record.similarity < c.threshold {
			continue
		}

		leftID, rightID := orderClusterIDs(cloneIndexes[record.left.ID], cloneIndexes[record.right.ID])
		input.edges = append(input.edges, completeLinkageEdge{
			leftID:  leftID,
			rightID: rightID,
			score:   record.similarity,
		})
	}

	return input
}

func (c *CompleteLinkageGrouping) buildGroups(activeClusters []*completeLinkageCluster, similarities map[string]float64, types map[string]domain.CloneType) []*domain.CloneGroup {
	groups := make([]*domain.CloneGroup, 0, len(activeClusters))
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
				if cloneSimilarity(similarities, members[i], members[j]) < c.threshold {
					valid = false
					break
				}
			}
		}
		if !valid {
			continue
		}

		sortedMembers := append([]*domain.Clone(nil), members...)
		sort.Slice(sortedMembers, func(i, j int) bool { return cloneLess(sortedMembers[i], sortedMembers[j]) })

		g := &domain.CloneGroup{
			ID:     groupID,
			Clones: make([]*domain.Clone, 0, len(sortedMembers)),
			Size:   len(sortedMembers),
		}
		groupID++
		for _, clone := range sortedMembers {
			g.AddClone(clone)
		}
		g.Similarity = averageGroupSimilarityClones(similarities, sortedMembers)
		g.Type = majorityCloneTypeClones(types, similarities, sortedMembers)
		groups = append(groups, g)
	}

	sortCloneGroups(groups)

	return groups
}

// CentroidGrouping uses BFS expansion with strict similarity to all existing members
type CentroidGrouping struct {
	threshold float64
}

func NewCentroidGrouping(threshold float64) *CentroidGrouping {
	return &CentroidGrouping{threshold: threshold}
}

func (cg *CentroidGrouping) GetName() string { return "Centroid" }

func (cg *CentroidGrouping) GroupClones(pairs []*domain.ClonePair) []*domain.CloneGroup {
	if len(pairs) == 0 {
		return []*domain.CloneGroup{}
	}

	// Build similarity map and adjacency
	clones := make([]*domain.Clone, 0)
	seen := make(map[int]struct{})
	simMap := make(map[string]float64)
	typeMap := make(map[string]domain.CloneType)
	neighbors := make(map[int][]int) // clone ID -> neighbor IDs above threshold

	addClone := func(clone *domain.Clone) {
		if clone == nil {
			return
		}
		if _, ok := seen[clone.ID]; !ok {
			seen[clone.ID] = struct{}{}
			clones = append(clones, clone)
			neighbors[clone.ID] = make([]int, 0)
		}
	}

	for _, p := range pairs {
		if p == nil || p.Clone1 == nil || p.Clone2 == nil {
			continue
		}
		addClone(p.Clone1)
		addClone(p.Clone2)
		key := clonePairKey(p.Clone1, p.Clone2)
		if old, ok := simMap[key]; !ok || p.Similarity > old {
			simMap[key] = p.Similarity
			typeMap[key] = p.Type
		}
		if p.Similarity >= cg.threshold {
			neighbors[p.Clone1.ID] = append(neighbors[p.Clone1.ID], p.Clone2.ID)
			neighbors[p.Clone2.ID] = append(neighbors[p.Clone2.ID], p.Clone1.ID)
		}
	}

	if len(clones) == 0 {
		return []*domain.CloneGroup{}
	}

	// Sort clones for deterministic processing
	sort.Slice(clones, func(i, j int) bool { return cloneLess(clones[i], clones[j]) })

	// Build clone ID to clone map
	cloneByID := make(map[int]*domain.Clone)
	for _, clone := range clones {
		cloneByID[clone.ID] = clone
	}

	// Sort neighbors for deterministic BFS traversal
	for id := range neighbors {
		sort.Ints(neighbors[id])
	}

	// BFS expansion from each unassigned clone
	assigned := make(map[int]bool)
	groups := make([]*domain.CloneGroup, 0)
	groupID := 0

	for _, seed := range clones {
		if assigned[seed.ID] {
			continue
		}

		// Start new group with seed
		members := []*domain.Clone{seed}
		assigned[seed.ID] = true

		// BFS queue: neighbor IDs to consider
		queue := list.New()
		visited := make(map[int]bool)
		visited[seed.ID] = true

		// Add seed's neighbors to queue
		for _, nid := range neighbors[seed.ID] {
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

			candidate := cloneByID[candidateID]
			if candidate == nil {
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
			sort.Slice(members, func(i, j int) bool { return cloneLess(members[i], members[j]) })
			g := &domain.CloneGroup{
				ID:     groupID,
				Clones: make([]*domain.Clone, 0, len(members)),
				Size:   len(members),
			}
			groupID++
			for _, clone := range members {
				g.AddClone(clone)
			}
			g.Similarity = averageGroupSimilarityClones(simMap, members)
			g.Type = majorityCloneTypeClones(typeMap, simMap, members)
			groups = append(groups, g)
		}
	}

	// Sort groups by similarity then size
	sort.Slice(groups, func(i, j int) bool {
		if !almostEqual(groups[i].Similarity, groups[j].Similarity) {
			return groups[i].Similarity > groups[j].Similarity
		}
		if groups[i].Size != groups[j].Size {
			return groups[i].Size > groups[j].Size
		}
		if len(groups[i].Clones) == 0 || len(groups[j].Clones) == 0 {
			return false
		}
		return cloneLess(groups[i].Clones[0], groups[j].Clones[0])
	})

	return groups
}

// isSimilarToAll checks if candidate is similar to all members above threshold
func (cg *CentroidGrouping) isSimilarToAll(candidate *domain.Clone, members []*domain.Clone, simMap map[string]float64) bool {
	for _, member := range members {
		if cloneSimilarity(simMap, candidate, member) < cg.threshold {
			return false
		}
	}
	return true
}

// Helper functions

// clonePairKey creates a canonical key for a pair of clones
func clonePairKey(a, b *domain.Clone) string {
	ka := cloneID(a)
	kb := cloneID(b)
	if ka <= kb {
		return ka + "||" + kb
	}
	return kb + "||" + ka
}

// cloneID returns a stable identifier for a clone based on its location
func cloneID(c *domain.Clone) string {
	if c == nil || c.Location == nil {
		return fmt.Sprintf("%p", c)
	}
	loc := c.Location
	return fmt.Sprintf("%s|%d|%d|%d|%d", loc.FilePath, loc.StartLine, loc.EndLine, loc.StartCol, loc.EndCol)
}

// cloneLess provides deterministic ordering between two clones by location
func cloneLess(a, b *domain.Clone) bool {
	if a == b {
		return false
	}
	if a == nil {
		return true
	}
	if b == nil {
		return false
	}
	al, bl := a.Location, b.Location
	if al == nil && bl == nil {
		return a.ID < b.ID
	}
	if al == nil {
		return true
	}
	if bl == nil {
		return false
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

// similarity returns cached similarity, or 0 if not present
func cloneSimilarity(sims map[string]float64, a, b *domain.Clone) float64 {
	if a == nil || b == nil {
		return 0.0
	}
	if a == b || a.ID == b.ID {
		return 1.0
	}
	key := clonePairKey(a, b)
	if s, ok := sims[key]; ok {
		return s
	}
	return 0.0
}

// averageGroupSimilarityClones computes average pairwise similarity among clones using cache.
// Only pairs that exist in the similarity map are counted (missing pairs are skipped, not treated as 0).
func averageGroupSimilarityClones(sims map[string]float64, members []*domain.Clone) float64 {
	if len(members) < 2 {
		return 1.0
	}
	sum := 0.0
	cnt := 0
	for i := 0; i < len(members); i++ {
		for j := i + 1; j < len(members); j++ {
			key := clonePairKey(members[i], members[j])
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

// majorityCloneTypeClones chooses the CloneType of the highest-similarity pair edge in
// members. When several pairs share the maximum similarity, the most strict
// (lowest enum) type wins. This prevents a high-similarity Type-2/Type-4 pair
// from being hidden when lower-similarity Type-3 transitive edges outnumber it
// in the same connected component.
func majorityCloneTypeClones(typeMap map[string]domain.CloneType, simMap map[string]float64, members []*domain.Clone) domain.CloneType {
	maxSim := -1.0
	var best domain.CloneType
	found := false
	for i := 0; i < len(members); i++ {
		for j := i + 1; j < len(members); j++ {
			key := clonePairKey(members[i], members[j])
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

// completeLinkageClusterer stores only threshold-qualified inter-cluster edges.
// That keeps sparse workloads sparse while still supporting exact complete-linkage
// merges, since a merged cluster can stay adjacent to C only if both source
// clusters already had qualifying edges to C.
type completeLinkageClusterer struct {
	clusters      []completeLinkageCluster
	bestNeighbors *completeLinkageBestNeighborHeap
}

type completeLinkageCluster struct {
	members   []*domain.Clone
	neighbors map[int]float64
	active    bool
}

func newCompleteLinkageClusterer(clones []*domain.Clone, edges []completeLinkageEdge) *completeLinkageClusterer {
	clusterer := &completeLinkageClusterer{
		clusters:      make([]completeLinkageCluster, len(clones)),
		bestNeighbors: newCompleteLinkageBestNeighborHeap(len(clones)),
	}

	for clusterID, clone := range clones {
		clusterer.clusters[clusterID] = completeLinkageCluster{
			members:   []*domain.Clone{clone},
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

func (c *completeLinkageClusterer) mergeUntilStable() {
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

func (c *completeLinkageClusterer) mergeClusters(targetID, sourceID int) {
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

func (c *completeLinkageClusterer) recomputeBestNeighbor(clusterID int) {
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

func (c *completeLinkageClusterer) findBestNeighbor(clusterID int) (int, float64, bool) {
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

func (c *completeLinkageClusterer) activeClusters() []*completeLinkageCluster {
	activeClusters := make([]*completeLinkageCluster, 0)
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

type groupDedupeResult struct {
	groups          []*domain.CloneGroup
	suppressed      map[string]struct{} // keyed by cloneID location key
	suppressedPairs map[string]struct{} // keyed by clonePairKey
}

// dedupeStrictSubsetGroupMembers removes clone-group members whose source
// range is a strict subset of (or identical to) another member's range in the
// same file. Groups reduced to fewer than two members are dropped.
//
// Why this exists: the pair-detection paths already reject *direct* pairs
// between overlapping same-file fragments (see isOverlappingLocation), so
// clone pairs cannot contain a same-file `(A, B)` where one strictly covers
// the other. Union-Find grouping, however, still merges such fragments into
// one group via a shared distinct-file neighbor — e.g., pairs
// `(A=x.ts:512-542, C=y.ts:1-30)` and `(B=x.ts:515-542, C=y.ts:1-30)` are both
// legal yet transitively connect A and B. This post-pass collapses those
// overlapping windows back to the maximal one per file.
//
// For exactly-equal ranges (which UF can produce in the same way), the first
// occurrence is kept; later duplicates are suppressed for deterministic output.
func dedupeStrictSubsetGroupMembers(groups []*domain.CloneGroup, pairs []*domain.ClonePair) groupDedupeResult {
	result := groupDedupeResult{
		groups:     groups,
		suppressed: make(map[string]struct{}),
	}
	if len(groups) == 0 {
		return result
	}

	out := make([]*domain.CloneGroup, 0, len(groups))
	var similarities map[string]float64
	var cloneTypes map[string]domain.CloneType
	metadataReady := false
	anyChanged := false
	for _, g := range groups {
		if g == nil {
			continue
		}
		kept, suppressed := filterMaximalPerFile(g.Clones)
		for key := range suppressed {
			result.suppressed[key] = struct{}{}
		}
		groupChanged := len(suppressed) > 0
		anyChanged = anyChanged || groupChanged
		if len(kept) < 2 {
			continue
		}
		g.Clones = kept
		g.Size = len(kept)
		if groupChanged {
			if !metadataReady {
				similarities, cloneTypes = clonePairMetadata(pairs)
				metadataReady = true
			}
			g.Similarity = averageGroupSimilarityClones(similarities, g.Clones)
			g.Type = majorityCloneTypeClones(cloneTypes, similarities, g.Clones)
		}
		out = append(out, g)
	}
	if anyChanged {
		sortCloneGroups(out)
	}
	result.groups = out
	return result
}

// coveredGroupSimilarityTolerance bounds how much weaker (in average
// similarity) a covering group may be while still subsuming a covered group.
// Overlapping windows of the same duplication shift similarity only slightly;
// a covered group that matches much more strongly than its covering group is
// a distinct, sharper finding (e.g., a near-identical inner block inside
// loosely similar functions) and is kept.
const coveredGroupSimilarityTolerance = 0.05

// dedupeCoveredGroups suppresses whole clone groups that are covered by
// another group: every member of the covered group fits inside a distinct
// member of the covering group (same file, containing line range), and the
// covering group's similarity is comparable or better. Such groups describe
// the same duplication relationship through slightly smaller windows and
// double-count it.
//
// Why dedupeStrictSubsetGroupMembers does not catch this: that pass compares
// members *within* one group. Here the overlapping windows sit in *different*
// groups, which stay disconnected because isOverlappingLocation forbids the
// direct same-file pair that would have linked them.
//
// The group with the larger (covering) windows is kept, mirroring the
// maximal-window policy of filterMaximalPerFile. When two groups cover each
// other (identical member ranges), the earlier one in the slice wins.
func dedupeCoveredGroups(groups []*domain.CloneGroup) groupDedupeResult {
	result := groupDedupeResult{
		groups:          groups,
		suppressed:      make(map[string]struct{}),
		suppressedPairs: make(map[string]struct{}),
	}
	if len(groups) < 2 {
		return result
	}

	suppressed := make([]bool, len(groups))
	for i, gi := range groups {
		if gi == nil {
			continue
		}
		// Compare against every other group, including already-suppressed
		// ones: coverage is transitive, so a chain g1 ⊂ g2 ⊂ g3 still
		// resolves to keeping only g3.
		for j, gj := range groups {
			if i == j || gj == nil {
				continue
			}
			if !groupCoveredBy(gi, gj) {
				continue
			}
			if groupCoveredBy(gj, gi) {
				if j > i {
					continue // mutual coverage: the earlier group survives
				}
			} else if gi.Similarity > gj.Similarity+coveredGroupSimilarityTolerance {
				continue // gi is a distinctly stronger match than its cover
			}
			suppressed[i] = true
			break
		}
	}

	out := make([]*domain.CloneGroup, 0, len(groups))
	keptPairs := make(map[string]struct{})
	for i, g := range groups {
		if g == nil || suppressed[i] {
			continue
		}
		out = append(out, g)
		for first := 0; first < len(g.Clones); first++ {
			for second := first + 1; second < len(g.Clones); second++ {
				keptPairs[clonePairKey(g.Clones[first], g.Clones[second])] = struct{}{}
			}
		}
	}
	for i, g := range groups {
		if g == nil || !suppressed[i] {
			continue
		}
		for first := 0; first < len(g.Clones); first++ {
			for second := first + 1; second < len(g.Clones); second++ {
				key := clonePairKey(g.Clones[first], g.Clones[second])
				if _, needed := keptPairs[key]; !needed {
					result.suppressedPairs[key] = struct{}{}
				}
			}
		}
	}
	result.groups = out
	return result
}

// groupCoveredBy reports whether every member of inner can be matched to a
// distinct member of outer that covers it (same file, containing range,
// equality included). Distinctness matters: two disjoint inner blocks inside
// one outer member describe duplication *within* that member, which the outer
// group does not report.
func groupCoveredBy(inner, outer *domain.CloneGroup) bool {
	n := len(inner.Clones)
	if n == 0 || n > len(outer.Clones) {
		return false
	}
	candidates := make([][]int, n)
	for i, c := range inner.Clones {
		if c == nil || c.Location == nil {
			return false
		}
		for j, oc := range outer.Clones {
			if oc == nil || oc.Location == nil {
				continue
			}
			if locationCovers(oc.Location, c.Location) {
				candidates[i] = append(candidates[i], j)
			}
		}
		if len(candidates[i]) == 0 {
			return false
		}
	}
	// Bipartite matching via augmenting paths; group sizes are small.
	matchedInner := make([]int, len(outer.Clones))
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
		if !assign(i, make([]bool, len(outer.Clones))) {
			return false
		}
	}
	return true
}

// locationCovers reports whether outer contains inner (same file, inclusive
// line and column ranges; equal ranges count as covered).
func locationCovers(outer, inner *domain.CloneLocation) bool {
	if outer.FilePath != inner.FilePath {
		return false
	}
	startsBefore := outer.StartLine < inner.StartLine ||
		(outer.StartLine == inner.StartLine && outer.StartCol <= inner.StartCol)
	endsAfter := outer.EndLine > inner.EndLine ||
		(outer.EndLine == inner.EndLine && outer.EndCol >= inner.EndCol)
	return startsBefore && endsAfter
}

// filterCloneGroupsWithoutBackingPairs drops groups whose refreshed metadata
// shows no positive-similarity pair actually backs them (e.g. every member
// pair was filtered out upstream), which would otherwise surface a group with
// zero similarity.
func filterCloneGroupsWithoutBackingPairs(groups []*domain.CloneGroup, pairs []*domain.ClonePair) []*domain.CloneGroup {
	if len(groups) == 0 {
		return groups
	}
	similarities, cloneTypes := clonePairMetadata(pairs)
	out := make([]*domain.CloneGroup, 0, len(groups))
	for _, group := range groups {
		if group == nil || len(group.Clones) < 2 {
			continue
		}
		group.Similarity = averageGroupSimilarityClones(similarities, group.Clones)
		group.Type = majorityCloneTypeClones(cloneTypes, similarities, group.Clones)
		if group.Similarity <= 0 {
			continue
		}
		out = append(out, group)
	}
	return out
}

// filterMaximalPerFile returns the subset of clones that are maximal under
// the same-file containment order. A clone is suppressed if any other kept
// clone in the same file strictly covers it, or if it duplicates an earlier
// clone's range exactly.
func filterMaximalPerFile(clones []*domain.Clone) ([]*domain.Clone, map[string]struct{}) {
	suppressedClones := make(map[string]struct{})
	n := len(clones)
	if n <= 1 {
		return clones, suppressedClones
	}
	suppressed := make([]bool, n)
	for i := 0; i < n; i++ {
		if suppressed[i] || clones[i] == nil || clones[i].Location == nil {
			continue
		}
		for j := 0; j < n; j++ {
			if i == j || suppressed[j] || clones[j] == nil || clones[j].Location == nil {
				continue
			}
			if covers(clones[i].Location, clones[j].Location, i, j) {
				suppressed[j] = true
			}
		}
	}
	out := make([]*domain.Clone, 0, n)
	for i, c := range clones {
		if suppressed[i] {
			if c != nil {
				suppressedClones[cloneID(c)] = struct{}{}
			}
			continue
		}
		out = append(out, c)
	}
	return out, suppressedClones
}

// covers reports whether outer (at index iOuter) covers inner (at index
// iInner) in the same file. Strict coverage suppresses inner outright;
// identical ranges suppress only the later index so that exactly one survives.
func covers(outer, inner *domain.CloneLocation, iOuter, iInner int) bool {
	if !locationCovers(outer, inner) {
		return false
	}
	if outer.StartLine == inner.StartLine && outer.StartCol == inner.StartCol &&
		outer.EndLine == inner.EndLine && outer.EndCol == inner.EndCol {
		return iOuter < iInner
	}
	return true
}

func clonePairMetadata(pairs []*domain.ClonePair) (map[string]float64, map[string]domain.CloneType) {
	similarities := make(map[string]float64, len(pairs))
	cloneTypes := make(map[string]domain.CloneType, len(pairs))
	for _, pair := range pairs {
		if pair == nil || pair.Clone1 == nil || pair.Clone2 == nil {
			continue
		}
		key := clonePairKey(pair.Clone1, pair.Clone2)
		if old, ok := similarities[key]; !ok || pair.Similarity > old {
			similarities[key] = pair.Similarity
			cloneTypes[key] = pair.Type
		}
	}
	return similarities, cloneTypes
}

func filterClonePairsWithSuppressedMembers(pairs []*domain.ClonePair, suppressed map[string]struct{}) []*domain.ClonePair {
	if len(pairs) == 0 || len(suppressed) == 0 {
		return pairs
	}
	out := make([]*domain.ClonePair, 0, len(pairs))
	for _, pair := range pairs {
		if pair == nil {
			continue
		}
		if _, ok := suppressed[cloneID(pair.Clone1)]; ok {
			continue
		}
		if _, ok := suppressed[cloneID(pair.Clone2)]; ok {
			continue
		}
		out = append(out, pair)
	}
	return out
}

func filterSuppressedClonePairs(pairs []*domain.ClonePair, suppressed map[string]struct{}) []*domain.ClonePair {
	if len(pairs) == 0 || len(suppressed) == 0 {
		return pairs
	}
	out := make([]*domain.ClonePair, 0, len(pairs))
	for _, pair := range pairs {
		if pair == nil {
			continue
		}
		if _, ok := suppressed[clonePairKey(pair.Clone1, pair.Clone2)]; ok {
			continue
		}
		out = append(out, pair)
	}
	return out
}

func sortCloneGroups(groups []*domain.CloneGroup) {
	sort.Slice(groups, func(i, j int) bool {
		if !almostEqual(groups[i].Similarity, groups[j].Similarity) {
			return groups[i].Similarity > groups[j].Similarity
		}
		if groups[i].Size != groups[j].Size {
			return groups[i].Size > groups[j].Size
		}
		if len(groups[i].Clones) == 0 || len(groups[j].Clones) == 0 {
			return false
		}
		return cloneLess(groups[i].Clones[0], groups[j].Clones[0])
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
