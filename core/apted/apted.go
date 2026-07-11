package apted

import (
	"math"
	"sort"
	"strings"
)

// NormalizationMode controls how similarity is normalized from distance.
type NormalizationMode int

const (
	// NormalizeByMax uses max(size1, size2) — stricter, reduces false positives.
	// Used by both pyscn and jscan.
	NormalizeByMax NormalizationMode = iota
	// NormalizeBySum uses size1 + size2 (Jaccard-like).
	NormalizeBySum
)

// APTEDAnalyzer implements the APTED (All Path Tree Edit Distance) algorithm.
// Based on Pawlik & Augsten's optimal O(n^2 log n) algorithm.
type APTEDAnalyzer struct {
	costModel         CostModel
	normalizationMode NormalizationMode
}

// NewAPTEDAnalyzer creates a new APTED analyzer with the given cost model.
// Default normalization mode is NormalizeByMax.
func NewAPTEDAnalyzer(costModel CostModel) *APTEDAnalyzer {
	return &APTEDAnalyzer{
		costModel:         costModel,
		normalizationMode: NormalizeByMax,
	}
}

// NewAPTEDAnalyzerWithNormalization creates an APTED analyzer with a specific normalization mode.
func NewAPTEDAnalyzerWithNormalization(costModel CostModel, mode NormalizationMode) *APTEDAnalyzer {
	return &APTEDAnalyzer{
		costModel:         costModel,
		normalizationMode: mode,
	}
}

const (
	largeTreeThreshold         = 500
	veryLargeTreeThreshold     = 2000
	maxKeyRoots                = 100
	earlyTermSizeFactor        = 0.8
	optimizedMaxDistFactor     = 0.5
	maxSameShapeAlignmentCells = 20000
)

// ComputeDistance computes the tree edit distance between two trees.
func (a *APTEDAnalyzer) ComputeDistance(tree1, tree2 *TreeNode) float64 {
	if tree1 == nil && tree2 == nil {
		return 0.0
	}
	if tree1 == nil {
		return a.computeInsertCost(tree2)
	}
	if tree2 == nil {
		return a.computeDeleteCost(tree1)
	}

	size1, size2 := tree1.Size(), tree2.Size()
	if size1 > largeTreeThreshold || size2 > largeTreeThreshold {
		return a.computeDistanceOptimized(tree1, tree2)
	}

	keyRoots1 := PrepareTreeForAPTED(tree1)
	keyRoots2 := PrepareTreeForAPTED(tree2)

	sort.Ints(keyRoots1)
	sort.Ints(keyRoots2)

	return a.apted(tree1, tree2, keyRoots1, keyRoots2)
}

// ComputeDistanceAndSimilarity computes both APTED distance and normalized
// similarity from one distance pass.
func (a *APTEDAnalyzer) ComputeDistanceAndSimilarity(tree1, tree2 *TreeNode) (float64, float64) {
	distance := a.ComputeDistance(tree1, tree2)
	return distance, a.normalizeSimilarity(distance, tree1, tree2)
}

// computeDistanceOptimized keeps the large-tree clone-detection path fast
// without letting the optimization erase real label or shape differences. This
// is a bounded heuristic for large inputs, not an exact APTED replacement.
func (a *APTEDAnalyzer) computeDistanceOptimized(tree1, tree2 *TreeNode) float64 {
	// Early termination based on size difference
	size1, size2 := tree1.Size(), tree2.Size()
	sizeDiff := math.Abs(float64(size1 - size2))

	// If size difference is too large, return early upper bound
	maxDistance := math.Max(float64(size1), float64(size2))
	if sizeDiff > maxDistance*earlyTermSizeFactor {
		return sizeDiff // Conservative estimate
	}
	if size1 == size2 {
		if sameShapeDistance, ok := a.computeBoundedSameShapeDistance(tree1, tree2); ok {
			return sameShapeDistance
		}
	}

	profileDistance := a.computeApproximateDistanceWithSizes(tree1, tree2, size1, size2)
	if profileDistance >= maxDistance {
		return profileDistance
	}

	// Use a simplified dynamic programming approach for very large trees
	if size1 > veryLargeTreeThreshold || size2 > veryLargeTreeThreshold {
		return profileDistance
	}

	// Use the capped optimized algorithm. The profile distance below is a
	// lower bound that keeps capped key roots from undercounting.
	keyRoots1 := PrepareTreeForAPTED(tree1)
	keyRoots2 := PrepareTreeForAPTED(tree2)

	// Limit key roots to reduce computation
	if len(keyRoots1) > maxKeyRoots {
		keyRoots1 = keyRoots1[:maxKeyRoots]
	}
	if len(keyRoots2) > maxKeyRoots {
		keyRoots2 = keyRoots2[:maxKeyRoots]
	}

	sort.Ints(keyRoots1)
	sort.Ints(keyRoots2)

	optimizedDistance := a.aptedOptimized(tree1, tree2, keyRoots1, keyRoots2, maxDistance*optimizedMaxDistFactor)
	return math.Max(optimizedDistance, profileDistance)
}

// computeApproximateDistance computes an approximate distance for very large trees.
func (a *APTEDAnalyzer) computeApproximateDistance(tree1, tree2 *TreeNode) float64 {
	size1, size2 := 0, 0
	if tree1 != nil {
		size1 = tree1.Size()
	}
	if tree2 != nil {
		size2 = tree2.Size()
	}
	return a.computeApproximateDistanceWithSizes(tree1, tree2, size1, size2)
}

func (a *APTEDAnalyzer) computeApproximateDistanceWithSizes(tree1, tree2 *TreeNode, size1, size2 int) float64 {
	// Handle nil cases
	if tree1 == nil && tree2 == nil {
		return 0.0
	}
	if tree1 == nil {
		return float64(size2)
	}
	if tree2 == nil {
		return float64(size1)
	}

	profile1 := collectTreeProfile(tree1)
	profile2 := collectTreeProfile(tree2)

	labelDistance := a.computeLabelProfileDistance(profile1.labels, profile2.labels)
	if labelDistance > 0 {
		return labelDistance
	}

	return computeShapeProfileDistance(tree1, tree2)
}

func (a *APTEDAnalyzer) computeLabelProfileDistance(labels1, labels2 map[string]*labelProfile) float64 {
	cancelSharedLabels(labels1, labels2)

	targetCandidates := buildLabelProfileIndex(labels2)
	sourceCandidates := buildLabelProfileIndex(labels1)

	deleteDistance := a.unmatchedSourceCost(labels1, targetCandidates)
	insertDistance := a.unmatchedTargetCost(sourceCandidates, labels2)
	return math.Max(deleteDistance, insertDistance)
}

type labelProfile struct {
	node  *TreeNode
	count int
}

type treeProfile struct {
	labels map[string]*labelProfile
}

func collectTreeProfile(root *TreeNode) treeProfile {
	profile := treeProfile{
		labels: make(map[string]*labelProfile),
	}
	if root == nil {
		return profile
	}

	stack := []*TreeNode{root}
	for len(stack) > 0 {
		last := len(stack) - 1
		node := stack[last]
		stack = stack[:last]

		entry := profile.labels[node.Label]
		if entry == nil {
			entry = &labelProfile{node: node}
			profile.labels[node.Label] = entry
		}
		entry.count++
		for _, child := range node.Children {
			if child != nil {
				stack = append(stack, child)
			}
		}
	}

	return profile
}

type shapeProfile struct {
	levelCounts map[int]int
	childCounts map[int]int
}

func computeShapeProfileDistance(tree1, tree2 *TreeNode) float64 {
	profile1 := collectShapeProfile(tree1)
	profile2 := collectShapeProfile(tree2)
	return math.Max(
		profileCountDistance(profile1.levelCounts, profile2.levelCounts),
		profileCountDistance(profile1.childCounts, profile2.childCounts),
	)
}

func collectShapeProfile(root *TreeNode) shapeProfile {
	profile := shapeProfile{
		levelCounts: make(map[int]int),
		childCounts: make(map[int]int),
	}
	if root == nil {
		return profile
	}

	type shapeStackEntry struct {
		node  *TreeNode
		depth int
	}
	stack := []shapeStackEntry{{node: root}}
	for len(stack) > 0 {
		last := len(stack) - 1
		item := stack[last]
		stack = stack[:last]

		node := item.node
		profile.levelCounts[item.depth]++
		profile.childCounts[len(node.Children)]++

		for _, child := range node.Children {
			if child != nil {
				stack = append(stack, shapeStackEntry{node: child, depth: item.depth + 1})
			}
		}
	}

	return profile
}

func profileCountDistance(profile1, profile2 map[int]int) float64 {
	difference := 0
	for key, count1 := range profile1 {
		difference += absInt(count1 - profile2[key])
	}
	for key, count2 := range profile2 {
		if _, ok := profile1[key]; !ok {
			difference += count2
		}
	}
	return float64(difference) * 0.5
}

func absInt(value int) int {
	if value < 0 {
		return -value
	}
	return value
}

type labelProfileIndex map[string]*labelProfile

func buildLabelProfileIndex(profile map[string]*labelProfile) labelProfileIndex {
	index := make(labelProfileIndex)

	for label, entry := range profile {
		if entry.count <= 0 {
			continue
		}

		key := labelProfileKey(label)
		if betterLabelProfile(entry, index[key]) {
			index[key] = entry
		}
	}

	return index
}

func betterLabelProfile(candidate, current *labelProfile) bool {
	if current == nil {
		return true
	}
	if candidate.count != current.count {
		return candidate.count > current.count
	}
	return candidate.node.Label < current.node.Label
}

func labelProfileKey(label string) string {
	if idx := strings.Index(label, "("); idx >= 0 {
		return label[:idx]
	}
	return label
}

func cancelSharedLabels(profile1, profile2 map[string]*labelProfile) {
	for label, entry1 := range profile1 {
		entry2 := profile2[label]
		if entry2 == nil {
			continue
		}

		shared := minLabelCount(entry1.count, entry2.count)
		entry1.count -= shared
		entry2.count -= shared
	}
}

// sortedProfileLabels returns the profile's labels in sorted order so float
// cost sums are accumulated deterministically regardless of map iteration.
func sortedProfileLabels(profile map[string]*labelProfile) []string {
	labels := make([]string, 0, len(profile))
	for label := range profile {
		labels = append(labels, label)
	}
	sort.Strings(labels)
	return labels
}

func (a *APTEDAnalyzer) unmatchedSourceCost(sources map[string]*labelProfile, targets labelProfileIndex) float64 {
	total := 0.0
	for _, label := range sortedProfileLabels(sources) {
		source := sources[label]
		if source.count <= 0 {
			continue
		}

		best := a.costModel.Delete(source.node)
		if target := targets[labelProfileKey(source.node.Label)]; target != nil {
			best = math.Min(best, a.costModel.Rename(source.node, target.node))
		}
		total += float64(source.count) * best
	}
	return total
}

func (a *APTEDAnalyzer) unmatchedTargetCost(sources labelProfileIndex, targets map[string]*labelProfile) float64 {
	total := 0.0
	for _, label := range sortedProfileLabels(targets) {
		target := targets[label]
		if target.count <= 0 {
			continue
		}

		best := a.costModel.Insert(target.node)
		if source := sources[labelProfileKey(target.node.Label)]; source != nil {
			best = math.Min(best, a.costModel.Rename(source.node, target.node))
		}
		total += float64(target.count) * best
	}
	return total
}

func minLabelCount(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func (a *APTEDAnalyzer) computeBoundedSameShapeDistance(tree1, tree2 *TreeNode) (float64, bool) {
	if !sameTreeShape(tree1, tree2) {
		return 0, false
	}
	if shiftDistance, ok := a.singleNodeChainShiftDistance(tree1, tree2); ok {
		return shiftDistance, true
	}

	state := sameShapeDistanceState{
		distances:               make(map[nodePair]float64),
		deleteCosts:             make(map[*TreeNode]float64),
		insertCosts:             make(map[*TreeNode]float64),
		alignmentCellsRemaining: maxSameShapeAlignmentCells,
	}
	return a.sameShapeNodeDistance(tree1, tree2, &state), true
}

func (a *APTEDAnalyzer) singleNodeChainShiftDistance(tree1, tree2 *TreeNode) (float64, bool) {
	left, leftIsChain := linearChainNodes(tree1)
	right, rightIsChain := linearChainNodes(tree2)
	if !leftIsChain || !rightIsChain || len(left) != len(right) || len(left) < 2 {
		return 0, false
	}

	start := 0
	for start < len(left) && a.nodeDistance(left[start], right[start]) == 0 {
		start++
	}
	if start == len(left) {
		return 0, true
	}

	end := len(left) - 1
	for end > start && a.nodeDistance(left[end], right[end]) == 0 {
		end--
	}
	if start == end {
		return 0, false
	}

	forwardMatches := true
	backwardMatches := true
	for i := start; i < end; i++ {
		forwardMatches = forwardMatches && a.nodeDistance(left[i+1], right[i]) == 0
		backwardMatches = backwardMatches && a.nodeDistance(left[i], right[i+1]) == 0
	}

	best := math.Inf(1)
	if forwardMatches {
		best = a.costModel.Delete(left[start]) + a.costModel.Insert(right[end])
	}
	if backwardMatches {
		best = math.Min(best, a.costModel.Delete(left[end])+a.costModel.Insert(right[start]))
	}
	return best, !math.IsInf(best, 1)
}

func linearChainNodes(root *TreeNode) ([]*TreeNode, bool) {
	nodes := make([]*TreeNode, 0)
	for root != nil {
		nodes = append(nodes, root)
		if len(root.Children) == 0 {
			return nodes, true
		}
		if len(root.Children) != 1 || root.Children[0] == nil {
			return nil, false
		}
		root = root.Children[0]
	}
	return nodes, true
}

func (a *APTEDAnalyzer) nodeDistance(left, right *TreeNode) float64 {
	return math.Min(
		a.costModel.Rename(left, right),
		a.costModel.Delete(left)+a.costModel.Insert(right),
	)
}

func sameTreeShape(tree1, tree2 *TreeNode) bool {
	stack := [][2]*TreeNode{{tree1, tree2}}
	for len(stack) > 0 {
		last := len(stack) - 1
		pair := stack[last]
		stack = stack[:last]

		left := pair[0]
		right := pair[1]
		if left == nil || right == nil {
			return left == right
		}
		if len(left.Children) != len(right.Children) {
			return false
		}

		for i := range left.Children {
			stack = append(stack, [2]*TreeNode{left.Children[i], right.Children[i]})
		}
	}

	return true
}

type nodePair struct {
	left  *TreeNode
	right *TreeNode
}

type sameShapeDistanceState struct {
	distances               map[nodePair]float64
	deleteCosts             map[*TreeNode]float64
	insertCosts             map[*TreeNode]float64
	alignmentCellsRemaining int
}

func (a *APTEDAnalyzer) sameShapeNodeDistance(left, right *TreeNode, state *sameShapeDistanceState) float64 {
	if left == nil && right == nil {
		return 0
	}
	if left == nil {
		return a.cachedInsertCost(right, state)
	}
	if right == nil {
		return a.cachedDeleteCost(left, state)
	}

	if len(left.Children) == 0 && len(right.Children) == 0 {
		return math.Min(
			a.costModel.Rename(left, right),
			a.costModel.Delete(left)+a.costModel.Insert(right),
		)
	}

	key := nodePair{left: left, right: right}
	if distance, ok := state.distances[key]; ok {
		return distance
	}

	rootCost := math.Min(
		a.costModel.Rename(left, right),
		a.costModel.Delete(left)+a.costModel.Insert(right),
	)
	childrenCost := a.sameShapeChildrenDistance(left.Children, right.Children, state)
	distance := rootCost + childrenCost
	state.distances[key] = distance
	return distance
}

func (a *APTEDAnalyzer) sameShapeChildrenDistance(left, right []*TreeNode, state *sameShapeDistanceState) float64 {
	if len(left) == 0 && len(right) == 0 {
		return 0
	}

	positionalDistance := a.positionalChildrenDistance(left, right, state)
	if positionalDistance == 0 || !canRealignChildren(left, right) {
		return positionalDistance
	}
	if len(left) == len(right) {
		if shiftDistance, exact := a.singleChildShiftDistance(left, right, state); exact && shiftDistance < positionalDistance {
			return shiftDistance
		}
	}

	if !state.reserveAlignmentCells(len(left) * len(right)) {
		// Keep the large-tree path bounded. This may overestimate exact APTED
		// for complex sibling reorders, but avoids hiding differences as zero.
		return positionalDistance
	}
	alignedDistance := a.alignChildSequences(left, right, state)
	if len(left) == len(right) {
		return math.Min(positionalDistance, alignedDistance)
	}
	return alignedDistance
}

func (a *APTEDAnalyzer) positionalChildrenDistance(left, right []*TreeNode, state *sameShapeDistanceState) float64 {
	total := 0.0
	shared := minLabelCount(len(left), len(right))
	for i := 0; i < shared; i++ {
		total += a.sameShapeNodeDistance(left[i], right[i], state)
	}
	for _, child := range left[shared:] {
		total += a.cachedDeleteCost(child, state)
	}
	for _, child := range right[shared:] {
		total += a.cachedInsertCost(child, state)
	}
	return total
}

func canRealignChildren(left, right []*TreeNode) bool {
	if len(left) < 2 || len(right) < 2 {
		return false
	}

	rightLabels := make(map[string]childPosition, len(right))
	needsProfileKeys := false
	for i, child := range right {
		rightLabels[child.Label] = addChildPosition(rightLabels[child.Label], i)
		needsProfileKeys = needsProfileKeys || labelProfileKey(child.Label) != child.Label
	}
	for i, child := range left {
		position, ok := rightLabels[child.Label]
		if ok && (position.count > 1 || position.index != i) {
			return true
		}
		needsProfileKeys = needsProfileKeys || labelProfileKey(child.Label) != child.Label
	}

	if !needsProfileKeys {
		return false
	}

	rightKeys := make(map[string]childPosition, len(right))
	for i, child := range right {
		key := labelProfileKey(child.Label)
		rightKeys[key] = addChildPosition(rightKeys[key], i)
	}
	for i, child := range left {
		key := labelProfileKey(child.Label)
		position, ok := rightKeys[key]
		if ok && (position.count > 1 || position.index != i) {
			return true
		}
	}

	return false
}

type childPosition struct {
	index int
	count int
}

func addChildPosition(position childPosition, index int) childPosition {
	if position.count == 0 {
		position.index = index
	}
	position.count++
	return position
}

func (a *APTEDAnalyzer) singleChildShiftDistance(left, right []*TreeNode, state *sameShapeDistanceState) (float64, bool) {
	if len(left) != len(right) || len(left) < 2 {
		return 0, false
	}

	start := 0
	for start < len(left) && a.sameShapeNodeDistance(left[start], right[start], state) == 0 {
		start++
	}
	if start == len(left) {
		return 0, true
	}

	end := len(left) - 1
	for end > start && a.sameShapeNodeDistance(left[end], right[end], state) == 0 {
		end--
	}
	if start == end {
		return 0, false
	}

	forwardCost, forwardExact := a.childShiftDistance(left, right, start, end, true, state)
	backwardCost, backwardExact := a.childShiftDistance(left, right, start, end, false, state)
	if forwardExact && (!backwardExact || forwardCost <= backwardCost) {
		return forwardCost, true
	}
	if backwardExact {
		return backwardCost, true
	}
	return 0, false
}

func (a *APTEDAnalyzer) childShiftDistance(
	left, right []*TreeNode,
	start, end int,
	leftDeletion bool,
	state *sameShapeDistanceState,
) (float64, bool) {
	matchedDistance := 0.0
	if leftDeletion {
		for i := start; i < end; i++ {
			matchedDistance += a.sameShapeNodeDistance(left[i+1], right[i], state)
		}
		return a.cachedDeleteCost(left[start], state) + a.cachedInsertCost(right[end], state) + matchedDistance, matchedDistance == 0
	}

	for i := start; i < end; i++ {
		matchedDistance += a.sameShapeNodeDistance(left[i], right[i+1], state)
	}
	return a.cachedDeleteCost(left[end], state) + a.cachedInsertCost(right[start], state) + matchedDistance, matchedDistance == 0
}

func (state *sameShapeDistanceState) reserveAlignmentCells(cells int) bool {
	if cells <= 0 {
		return true
	}
	if cells > state.alignmentCellsRemaining {
		return false
	}
	state.alignmentCellsRemaining -= cells
	return true
}

func (a *APTEDAnalyzer) alignChildSequences(left, right []*TreeNode, state *sameShapeDistanceState) float64 {
	prev := make([]float64, len(right)+1)
	for j, child := range right {
		prev[j+1] = prev[j] + a.cachedInsertCost(child, state)
	}

	for _, leftChild := range left {
		curr := make([]float64, len(right)+1)
		curr[0] = prev[0] + a.cachedDeleteCost(leftChild, state)

		for j, rightChild := range right {
			deleteCost := prev[j+1] + a.cachedDeleteCost(leftChild, state)
			insertCost := curr[j] + a.cachedInsertCost(rightChild, state)
			matchCost := prev[j] + a.sameShapeNodeDistance(leftChild, rightChild, state)
			curr[j+1] = math.Min(deleteCost, math.Min(insertCost, matchCost))
		}

		prev = curr
	}

	return prev[len(right)]
}

func (a *APTEDAnalyzer) cachedDeleteCost(node *TreeNode, state *sameShapeDistanceState) float64 {
	if node == nil {
		return 0
	}
	if cost, ok := state.deleteCosts[node]; ok {
		return cost
	}

	cost := a.costModel.Delete(node)
	for _, child := range node.Children {
		cost += a.cachedDeleteCost(child, state)
	}
	state.deleteCosts[node] = cost
	return cost
}

func (a *APTEDAnalyzer) cachedInsertCost(node *TreeNode, state *sameShapeDistanceState) float64 {
	if node == nil {
		return 0
	}
	if cost, ok := state.insertCosts[node]; ok {
		return cost
	}

	cost := a.costModel.Insert(node)
	for _, child := range node.Children {
		cost += a.cachedInsertCost(child, state)
	}
	state.insertCosts[node] = cost
	return cost
}

func (a *APTEDAnalyzer) aptedOptimized(tree1, tree2 *TreeNode, keyRoots1, keyRoots2 []int, maxDistance float64) float64 {
	nodes1 := a.getPostOrderNodes(tree1)
	nodes2 := a.getPostOrderNodes(tree2)
	size1 := len(nodes1)
	size2 := len(nodes2)

	td := make([][]float64, size1+1)
	for i := range td {
		td[i] = make([]float64, size2+1)
	}

	for _, i := range keyRoots1 {
		for _, j := range keyRoots2 {
			a.computeForestDistanceOptimized(nodes1, nodes2, i, j, td, maxDistance)
			if td[size1][size2] > maxDistance {
				return td[size1][size2]
			}
		}
	}
	return td[size1][size2]
}

func (a *APTEDAnalyzer) apted(tree1, tree2 *TreeNode, keyRoots1, keyRoots2 []int) float64 {
	nodes1 := a.getPostOrderNodes(tree1)
	nodes2 := a.getPostOrderNodes(tree2)
	size1 := len(nodes1)
	size2 := len(nodes2)

	td := make([][]float64, size1+1)
	for i := range td {
		td[i] = make([]float64, size2+1)
	}

	for _, i := range keyRoots1 {
		for _, j := range keyRoots2 {
			a.computeForestDistance(nodes1, nodes2, i, j, td)
		}
	}
	return td[size1][size2]
}

func (a *APTEDAnalyzer) computeForestDistanceOptimized(nodes1, nodes2 []*TreeNode, i, j int, td [][]float64, maxDistance float64) {
	if i < 0 || i >= len(nodes1) || j < 0 || j >= len(nodes2) {
		return
	}

	lml_i := nodes1[i].LeftMostLeaf
	lml_j := nodes2[j].LeftMostLeaf

	fd := make([][]float64, i+2)
	for k := range fd {
		fd[k] = make([]float64, j+2)
	}

	for x := lml_i; x <= i; x++ {
		if x < 0 || x >= len(nodes1) {
			continue
		}
		fd[x+1][lml_j] = fd[x][lml_j] + a.costModel.Delete(nodes1[x])
		if fd[x+1][lml_j] > maxDistance {
			return
		}
	}
	for y := lml_j; y <= j; y++ {
		if y < 0 || y >= len(nodes2) {
			continue
		}
		fd[lml_i][y+1] = fd[lml_i][y] + a.costModel.Insert(nodes2[y])
		if fd[lml_i][y+1] > maxDistance {
			return
		}
	}

	for x := lml_i; x <= i; x++ {
		if x < 0 || x >= len(nodes1) {
			continue
		}
		for y := lml_j; y <= j; y++ {
			if y < 0 || y >= len(nodes2) {
				continue
			}
			a.computeCell(nodes1, nodes2, x, y, lml_i, lml_j, fd, td)
			if fd[x+1][y+1] > maxDistance {
				return
			}
		}
	}
}

func (a *APTEDAnalyzer) computeForestDistance(nodes1, nodes2 []*TreeNode, i, j int, td [][]float64) {
	if i < 0 || i >= len(nodes1) || j < 0 || j >= len(nodes2) {
		return
	}

	lml_i := nodes1[i].LeftMostLeaf
	lml_j := nodes2[j].LeftMostLeaf

	fd := make([][]float64, i+2)
	for k := range fd {
		fd[k] = make([]float64, j+2)
	}

	for x := lml_i; x <= i; x++ {
		if x < 0 || x >= len(nodes1) {
			continue
		}
		fd[x+1][lml_j] = fd[x][lml_j] + a.costModel.Delete(nodes1[x])
	}
	for y := lml_j; y <= j; y++ {
		if y < 0 || y >= len(nodes2) {
			continue
		}
		fd[lml_i][y+1] = fd[lml_i][y] + a.costModel.Insert(nodes2[y])
	}

	for x := lml_i; x <= i; x++ {
		if x < 0 || x >= len(nodes1) {
			continue
		}
		for y := lml_j; y <= j; y++ {
			if y < 0 || y >= len(nodes2) {
				continue
			}
			a.computeCell(nodes1, nodes2, x, y, lml_i, lml_j, fd, td)
		}
	}
}

// computeCell computes a single cell in the forest distance matrix.
func (a *APTEDAnalyzer) computeCell(nodes1, nodes2 []*TreeNode, x, y, lml_i, lml_j int, fd, td [][]float64) {
	lml_x := nodes1[x].LeftMostLeaf
	lml_y := nodes2[y].LeftMostLeaf

	deleteCost := fd[x][y+1] + a.costModel.Delete(nodes1[x])
	insertCost := fd[x+1][y] + a.costModel.Insert(nodes2[y])

	if lml_x == lml_i && lml_y == lml_j {
		renameCost := fd[x][y] + a.costModel.Rename(nodes1[x], nodes2[y])
		fd[x+1][y+1] = math.Min(deleteCost, math.Min(insertCost, renameCost))
		td[x+1][y+1] = fd[x+1][y+1]
	} else {
		// td[x+1][y+1] was already computed during a previous key root iteration
		subtreeCost := fd[lml_x][lml_y] + td[x+1][y+1]
		fd[x+1][y+1] = math.Min(deleteCost, math.Min(insertCost, subtreeCost))
	}
}

func (a *APTEDAnalyzer) getPostOrderNodes(root *TreeNode) []*TreeNode {
	if root == nil {
		return []*TreeNode{}
	}
	var nodes []*TreeNode
	a.postOrderTraversal(root, &nodes, defaultMaxDepth)
	sort.Slice(nodes, func(i, j int) bool {
		return nodes[i].PostOrderID < nodes[j].PostOrderID
	})
	return nodes
}

func (a *APTEDAnalyzer) postOrderTraversal(node *TreeNode, nodes *[]*TreeNode, maxDepth int) {
	if node == nil || maxDepth <= 0 {
		return
	}
	for _, child := range node.Children {
		a.postOrderTraversal(child, nodes, maxDepth-1)
	}
	*nodes = append(*nodes, node)
}

func (a *APTEDAnalyzer) computeInsertCost(root *TreeNode) float64 {
	return a.computeSubtreeCost(root, defaultMaxDepth, true)
}

func (a *APTEDAnalyzer) computeDeleteCost(root *TreeNode) float64 {
	return a.computeSubtreeCost(root, defaultMaxDepth, false)
}

func (a *APTEDAnalyzer) computeSubtreeCost(root *TreeNode, maxDepth int, insert bool) float64 {
	if root == nil || maxDepth <= 0 {
		return 0.0
	}
	var cost float64
	if insert {
		cost = a.costModel.Insert(root)
	} else {
		cost = a.costModel.Delete(root)
	}
	for _, child := range root.Children {
		cost += a.computeSubtreeCost(child, maxDepth-1, insert)
	}
	return cost
}

// ComputeSimilarity computes similarity score between two trees (0.0 to 1.0).
// The normalization mode is controlled by the analyzer's NormalizationMode setting.
func (a *APTEDAnalyzer) ComputeSimilarity(tree1, tree2 *TreeNode) float64 {
	_, similarity := a.ComputeDistanceAndSimilarity(tree1, tree2)
	return similarity
}

// normalizeSimilarity converts a distance into a [0, 1] similarity using the
// analyzer's normalization mode.
func (a *APTEDAnalyzer) normalizeSimilarity(distance float64, tree1, tree2 *TreeNode) float64 {
	if tree1 == nil && tree2 == nil {
		return 1.0 // Identical (both empty)
	}
	if tree1 == nil || tree2 == nil {
		return 0.0 // Completely different (one empty)
	}

	size1 := float64(tree1.Size())
	size2 := float64(tree2.Size())

	var maxPossible float64
	switch a.normalizationMode {
	case NormalizeBySum:
		maxPossible = size1 + size2
	default:
		// NormalizeByMax: a tree can be transformed into another with at most
		// max(size1, size2) operations when we delete one tree and insert the
		// other optimally. Stricter than size1+size2, reduces false positives.
		maxPossible = math.Max(size1, size2)
	}

	if maxPossible == 0 {
		return 1.0
	}

	normalizedDistance := math.Min(distance, maxPossible) / maxPossible
	similarity := 1.0 - normalizedDistance
	return math.Max(0.0, math.Min(1.0, similarity))
}

// TreeEditResult holds the result of tree edit distance computation.
type TreeEditResult struct {
	Distance   float64
	Similarity float64
	Tree1Size  int
	Tree2Size  int
	Operations int
}

// ComputeDetailedDistance computes detailed tree edit distance information.
func (a *APTEDAnalyzer) ComputeDetailedDistance(tree1, tree2 *TreeNode) *TreeEditResult {
	distance, similarity := a.ComputeDistanceAndSimilarity(tree1, tree2)

	var size1, size2 int
	if tree1 != nil {
		size1 = tree1.Size()
	}
	if tree2 != nil {
		size2 = tree2.Size()
	}

	return &TreeEditResult{
		Distance:   distance,
		Similarity: similarity,
		Tree1Size:  size1,
		Tree2Size:  size2,
		Operations: int(distance),
	}
}

// OptimizedAPTEDAnalyzer extends APTEDAnalyzer with early stopping.
type OptimizedAPTEDAnalyzer struct {
	*APTEDAnalyzer
	maxDistance     float64
	enableEarlyStop bool
}

// NewOptimizedAPTEDAnalyzer creates an optimized APTED analyzer.
func NewOptimizedAPTEDAnalyzer(costModel CostModel, maxDistance float64) *OptimizedAPTEDAnalyzer {
	return &OptimizedAPTEDAnalyzer{
		APTEDAnalyzer:   NewAPTEDAnalyzer(costModel),
		maxDistance:     maxDistance,
		enableEarlyStop: maxDistance > 0,
	}
}

// ComputeDistance computes tree edit distance with early stopping optimization.
func (a *OptimizedAPTEDAnalyzer) ComputeDistance(tree1, tree2 *TreeNode) float64 {
	if a.enableEarlyStop {
		sizeDiff := math.Abs(float64(tree1.Size() - tree2.Size()))
		if sizeDiff > a.maxDistance {
			return a.maxDistance + 1.0
		}
	}
	distance := a.APTEDAnalyzer.ComputeDistance(tree1, tree2)
	if a.enableEarlyStop && distance > a.maxDistance {
		return a.maxDistance + 1.0
	}
	return distance
}

// BatchComputeDistances computes distances between multiple tree pairs.
func (a *APTEDAnalyzer) BatchComputeDistances(pairs [][2]*TreeNode) []float64 {
	distances := make([]float64, len(pairs))
	for i, pair := range pairs {
		distances[i] = a.ComputeDistance(pair[0], pair[1])
	}
	return distances
}

// ClusterResult represents the result of tree clustering.
type ClusterResult struct {
	Groups    [][]int
	Distances [][]float64
	Threshold float64
}

// ClusterSimilarTrees clusters trees based on similarity threshold.
func (a *APTEDAnalyzer) ClusterSimilarTrees(trees []*TreeNode, similarityThreshold float64) *ClusterResult {
	if len(trees) == 0 {
		return &ClusterResult{Groups: [][]int{}, Distances: [][]float64{}, Threshold: similarityThreshold}
	}

	validTrees := make([]*TreeNode, 0, len(trees))
	originalIndices := make([]int, 0, len(trees))
	for i, tree := range trees {
		if tree != nil {
			validTrees = append(validTrees, tree)
			originalIndices = append(originalIndices, i)
		}
	}

	if len(validTrees) == 0 {
		return &ClusterResult{Groups: [][]int{}, Distances: [][]float64{}, Threshold: similarityThreshold}
	}
	if len(validTrees) == 1 {
		return &ClusterResult{Groups: [][]int{{originalIndices[0]}}, Distances: [][]float64{{0.0}}, Threshold: similarityThreshold}
	}

	n := len(validTrees)
	distances := make([][]float64, n)
	for i := 0; i < n; i++ {
		distances[i] = make([]float64, n)
		for j := 0; j < n; j++ {
			if i != j {
				distances[i][j] = math.Inf(1)
			}
		}
	}

	for i := 0; i < n; i++ {
		for j := i + 1; j < n; j++ {
			if validTrees[i] != nil && validTrees[j] != nil {
				dist := a.ComputeDistance(validTrees[i], validTrees[j])
				distances[i][j] = dist
				distances[j][i] = dist
			}
		}
	}

	visited := make([]bool, n)
	var groups [][]int

	for i := 0; i < n; i++ {
		if visited[i] {
			continue
		}
		cluster := []int{originalIndices[i]}
		visited[i] = true

		for j := i + 1; j < n; j++ {
			if !visited[j] && distances[i][j] != math.Inf(1) {
				maxSize := math.Max(float64(validTrees[i].Size()), float64(validTrees[j].Size()))
				if maxSize > 0 {
					similarity := 1.0 - distances[i][j]/maxSize
					if similarity >= similarityThreshold {
						cluster = append(cluster, originalIndices[j])
						visited[j] = true
					}
				}
			}
		}
		groups = append(groups, cluster)
	}

	return &ClusterResult{Groups: groups, Distances: distances, Threshold: similarityThreshold}
}
