package clone

import (
	"github.com/ludo-technologies/codescan-core/apted"
)

// SimilarityAnalyzer computes similarity between two code fragments.
type SimilarityAnalyzer interface {
	ComputeSimilarity(f1, f2 *CodeFragment) float64
	Name() string
}

// StructuralAnalyzer computes structural similarity using APTED tree edit distance.
type StructuralAnalyzer struct {
	analyzer *apted.APTEDAnalyzer
}

// NewStructuralAnalyzer creates a new structural similarity analyzer.
func NewStructuralAnalyzer(costModel apted.CostModel, normMode apted.NormalizationMode) *StructuralAnalyzer {
	return &StructuralAnalyzer{
		analyzer: apted.NewAPTEDAnalyzerWithNormalization(costModel, normMode),
	}
}

// ComputeSimilarity computes the structural similarity between two fragments using APTED.
func (s *StructuralAnalyzer) ComputeSimilarity(f1, f2 *CodeFragment) float64 {
	if f1 == nil || f2 == nil {
		return 0.0
	}
	if f1.ASTNode == nil || f2.ASTNode == nil {
		return 0.0
	}
	return s.analyzer.ComputeSimilarity(f1.ASTNode, f2.ASTNode)
}

// Name returns the name of this analyzer.
func (s *StructuralAnalyzer) Name() string {
	return "structural"
}
