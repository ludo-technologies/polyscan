package semantic

import (
	"math"
	"slices"

	"github.com/ludo-technologies/polyscan/core/cfg"
	"github.com/ludo-technologies/polyscan/core/dfa"
	"github.com/ludo-technologies/polyscan/core/domain"
)

// CFGFeatures holds structural features extracted from a CFG.
type CFGFeatures struct {
	BlockCount      int
	EdgeCount       int
	EdgeTypeCounts  map[cfg.EdgeType]int
	CyclomaticNumber int
	BranchingFactor float64
	LoopEdgeCount   int
	ConditionalCount int
}

// ExtractCFGFeatures extracts structural features from a CFG.
func ExtractCFGFeatures(c *cfg.CFG) *CFGFeatures {
	if c == nil {
		return &CFGFeatures{EdgeTypeCounts: make(map[cfg.EdgeType]int)}
	}

	f := &CFGFeatures{
		BlockCount:     len(c.Blocks),
		EdgeTypeCounts: make(map[cfg.EdgeType]int),
	}

	totalSuccessors := 0
	branchingBlocks := 0

	for _, block := range c.Blocks {
		for _, edge := range block.Successors {
			f.EdgeCount++
			f.EdgeTypeCounts[edge.Type]++

			switch edge.Type {
			case cfg.EdgeLoop:
				f.LoopEdgeCount++
			case cfg.EdgeCondTrue, cfg.EdgeCondFalse:
				f.ConditionalCount++
			}
		}

		if len(block.Successors) > 1 {
			totalSuccessors += len(block.Successors)
			branchingBlocks++
		}
	}

	// Cyclomatic complexity: V(G) = E - N + 2P (P=1 for single component)
	f.CyclomaticNumber = f.EdgeCount - f.BlockCount + 2

	// Average branching factor among blocks with >1 successor
	if branchingBlocks > 0 {
		f.BranchingFactor = float64(totalSuccessors) / float64(branchingBlocks)
	}

	return f
}

// Config holds configuration for semantic similarity computation.
type Config struct {
	EnableDFA        bool
	CFGFeatureWeight float64
	DFAFeatureWeight float64
}

// DefaultConfig returns the default semantic similarity configuration.
func DefaultConfig() Config {
	return Config{
		EnableDFA:        true,
		CFGFeatureWeight: domain.DefaultCFGFeatureWeight,
		DFAFeatureWeight: domain.DefaultDFAFeatureWeight,
	}
}

// CompareCFGFeatures computes similarity between two CFG feature sets.
// Returns a value between 0.0 (completely different) and 1.0 (identical).
func CompareCFGFeatures(f1, f2 *CFGFeatures) float64 {
	if f1 == nil || f2 == nil {
		return 0.0
	}

	// Weighted similarity across multiple dimensions
	blockSim := computeCountSimilarity(f1.BlockCount, f2.BlockCount)
	edgeSim := computeCountSimilarity(f1.EdgeCount, f2.EdgeCount)
	ccSim := computeCountSimilarity(f1.CyclomaticNumber, f2.CyclomaticNumber)
	edgeDistSim := compareEdgeDistributions(f1.EdgeTypeCounts, f2.EdgeTypeCounts)
	branchSim := computeFloatSimilarity(f1.BranchingFactor, f2.BranchingFactor)
	loopCondSim := computeCountSimilarity(
		f1.LoopEdgeCount+f1.ConditionalCount,
		f2.LoopEdgeCount+f2.ConditionalCount,
	)

	return blockSim*0.20 + edgeSim*0.15 + ccSim*0.25 + edgeDistSim*0.25 + branchSim*0.10 + loopCondSim*0.05
}

// CompareDFAFeatures computes similarity between two DFA feature sets.
// Returns a value between 0.0 (completely different) and 1.0 (identical).
func CompareDFAFeatures(f1, f2 *dfa.DFAFeatures) float64 {
	if f1 == nil || f2 == nil {
		return 0.0
	}

	pairSim := computeCountSimilarity(f1.PairCount, f2.PairCount)
	chainSim := computeFloatSimilarity(f1.AvgChainLength, f2.AvgChainLength)
	crossSim := computeFloatSimilarity(f1.CrossBlockRatio, f2.CrossBlockRatio)
	defKindSim := compareDefUseKindDistributions(f1.DefKindDist, f2.DefKindDist)
	useKindSim := compareDefUseKindDistributions(f1.UseKindDist, f2.UseKindDist)

	return pairSim*domain.DefaultDFAPairCountWeight +
		chainSim*domain.DefaultDFAChainLengthWeight +
		crossSim*domain.DefaultDFACrossBlockWeight +
		defKindSim*domain.DefaultDFADefKindWeight +
		useKindSim*domain.DefaultDFAUseKindWeight
}

// ComputeSimilarity computes the combined CFG+DFA semantic similarity.
// Returns 0.0 when either CFG is nil (e.g. parse/build failures) to avoid
// false-positive matches on absent semantic data.
func ComputeSimilarity(cfg1, cfg2 *cfg.CFG, dfa1, dfa2 *dfa.DFAInfo, config Config) float64 {
	if cfg1 == nil || cfg2 == nil {
		return 0.0
	}

	cfgFeatures1 := ExtractCFGFeatures(cfg1)
	cfgFeatures2 := ExtractCFGFeatures(cfg2)
	cfgSim := CompareCFGFeatures(cfgFeatures1, cfgFeatures2)

	if !config.EnableDFA || dfa1 == nil || dfa2 == nil {
		return cfgSim
	}

	dfaFeatures1 := dfa.ExtractDFAFeatures(dfa1)
	dfaFeatures2 := dfa.ExtractDFAFeatures(dfa2)
	dfaSim := CompareDFAFeatures(dfaFeatures1, dfaFeatures2)

	return cfgSim*config.CFGFeatureWeight + dfaSim*config.DFAFeatureWeight
}

// computeCountSimilarity computes similarity between two integer counts.
// Returns 1.0 for identical, decreasing toward 0.0 for larger differences.
func computeCountSimilarity(a, b int) float64 {
	if a == 0 && b == 0 {
		return 1.0
	}
	maxVal := math.Max(float64(a), float64(b))
	if maxVal == 0 {
		return 1.0
	}
	diff := math.Abs(float64(a) - float64(b))
	return 1.0 - diff/maxVal
}

// computeFloatSimilarity computes similarity between two float values.
func computeFloatSimilarity(a, b float64) float64 {
	if a == 0 && b == 0 {
		return 1.0
	}
	maxVal := math.Max(math.Abs(a), math.Abs(b))
	if maxVal == 0 {
		return 1.0
	}
	diff := math.Abs(a - b)
	return 1.0 - diff/maxVal
}

// compareEdgeDistributions computes cosine similarity between two edge type distributions.
func compareEdgeDistributions(d1, d2 map[cfg.EdgeType]int) float64 {
	// Collect all edge types
	allTypes := make(map[cfg.EdgeType]bool)
	for k := range d1 {
		allTypes[k] = true
	}
	for k := range d2 {
		allTypes[k] = true
	}
	if len(allTypes) == 0 {
		return 1.0
	}

	// Sort for deterministic iteration
	types := make([]cfg.EdgeType, 0, len(allTypes))
	for t := range allTypes {
		types = append(types, t)
	}
	slices.Sort(types)

	// Cosine similarity
	var dotProduct, norm1, norm2 float64
	for _, t := range types {
		v1 := float64(d1[t])
		v2 := float64(d2[t])
		dotProduct += v1 * v2
		norm1 += v1 * v1
		norm2 += v2 * v2
	}

	if norm1 == 0 || norm2 == 0 {
		return 0.0
	}
	return dotProduct / (math.Sqrt(norm1) * math.Sqrt(norm2))
}

// compareDefUseKindDistributions computes cosine similarity between two DefUseKind distributions.
func compareDefUseKindDistributions(d1, d2 map[dfa.DefUseKind]int) float64 {
	allKinds := make(map[dfa.DefUseKind]bool)
	for k := range d1 {
		allKinds[k] = true
	}
	for k := range d2 {
		allKinds[k] = true
	}
	if len(allKinds) == 0 {
		return 1.0
	}

	kinds := make([]dfa.DefUseKind, 0, len(allKinds))
	for k := range allKinds {
		kinds = append(kinds, k)
	}
	slices.Sort(kinds)

	var dotProduct, norm1, norm2 float64
	for _, k := range kinds {
		v1 := float64(d1[k])
		v2 := float64(d2[k])
		dotProduct += v1 * v2
		norm1 += v1 * v1
		norm2 += v2 * v2
	}

	if norm1 == 0 || norm2 == 0 {
		return 0.0
	}
	return dotProduct / (math.Sqrt(norm1) * math.Sqrt(norm2))
}
