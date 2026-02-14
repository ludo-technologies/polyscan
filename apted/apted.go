package apted

import (
	"math"
	"sort"
)

// NormalizationMode controls how similarity is normalized from distance.
type NormalizationMode int

const (
	// NormalizeByMax uses max(size1, size2) — stricter, reduces false positives.
	// Used by pyscn.
	NormalizeByMax NormalizationMode = iota
	// NormalizeBySum uses size1 + size2 (Jaccard-like).
	// Used by jscan.
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
	largeTreeThreshold     = 500
	veryLargeTreeThreshold = 2000
	maxKeyRoots            = 100
	earlyTermSizeFactor    = 0.8
	optimizedMaxDistFactor = 0.5
	approxDepthWeight      = 2.0
	approxSizeWeight       = 0.5
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

func (a *APTEDAnalyzer) computeDistanceOptimized(tree1, tree2 *TreeNode) float64 {
	size1, size2 := tree1.Size(), tree2.Size()
	sizeDiff := math.Abs(float64(size1 - size2))
	maxDistance := math.Max(float64(size1), float64(size2))

	if sizeDiff > maxDistance*earlyTermSizeFactor {
		return sizeDiff
	}

	if size1 > veryLargeTreeThreshold || size2 > veryLargeTreeThreshold {
		return a.computeApproximateDistance(tree1, tree2)
	}

	keyRoots1 := PrepareTreeForAPTED(tree1)
	keyRoots2 := PrepareTreeForAPTED(tree2)

	if len(keyRoots1) > maxKeyRoots {
		keyRoots1 = keyRoots1[:maxKeyRoots]
	}
	if len(keyRoots2) > maxKeyRoots {
		keyRoots2 = keyRoots2[:maxKeyRoots]
	}

	sort.Ints(keyRoots1)
	sort.Ints(keyRoots2)

	return a.aptedOptimized(tree1, tree2, keyRoots1, keyRoots2, maxDistance*optimizedMaxDistFactor)
}

func (a *APTEDAnalyzer) computeApproximateDistance(tree1, tree2 *TreeNode) float64 {
	if tree1 == nil && tree2 == nil {
		return 0.0
	}
	if tree1 == nil {
		return float64(tree2.Size())
	}
	if tree2 == nil {
		return float64(tree1.Size())
	}
	depthDiff := math.Abs(float64(tree1.Height() - tree2.Height()))
	sizeDiff := math.Abs(float64(tree1.Size() - tree2.Size()))
	return (depthDiff * approxDepthWeight) + (sizeDiff * approxSizeWeight)
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
	if tree1 == nil && tree2 == nil {
		return 1.0
	}
	if tree1 == nil || tree2 == nil {
		return 0.0
	}

	distance := a.ComputeDistance(tree1, tree2)
	size1 := float64(tree1.Size())
	size2 := float64(tree2.Size())

	var maxPossible float64
	switch a.normalizationMode {
	case NormalizeBySum:
		maxPossible = size1 + size2
	default: // NormalizeByMax
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
	distance := a.ComputeDistance(tree1, tree2)
	similarity := a.ComputeSimilarity(tree1, tree2)

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
	maxDistance      float64
	enableEarlyStop bool
}

// NewOptimizedAPTEDAnalyzer creates an optimized APTED analyzer.
func NewOptimizedAPTEDAnalyzer(costModel CostModel, maxDistance float64) *OptimizedAPTEDAnalyzer {
	return &OptimizedAPTEDAnalyzer{
		APTEDAnalyzer:   NewAPTEDAnalyzer(costModel),
		maxDistance:      maxDistance,
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
