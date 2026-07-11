package clone

import (
	"sort"

	"github.com/ludo-technologies/polyscan/core/domain"
)

// ClassifierConfig holds configuration for the clone classifier.
type ClassifierConfig struct {
	Type1Threshold            float64
	Type2Threshold            float64
	Type3Threshold            float64
	Type4Threshold            float64
	EnableType1               bool
	EnableType2               bool
	EnableType3               bool
	EnableType4               bool
	JaccardPreFilterThreshold float64
}

// DefaultClassifierConfig returns the default classifier configuration.
func DefaultClassifierConfig() ClassifierConfig {
	return ClassifierConfig{
		Type1Threshold:            domain.DefaultType1CloneThreshold,
		Type2Threshold:            domain.DefaultType2CloneThreshold,
		Type3Threshold:            domain.DefaultType3CloneThreshold,
		Type4Threshold:            domain.DefaultType4CloneThreshold,
		EnableType1:               true,
		EnableType2:               true,
		EnableType3:               true,
		EnableType4:               true,
		JaccardPreFilterThreshold: 0.0,
	}
}

// ClassificationResult holds the result of classifying a clone pair.
type ClassificationResult struct {
	CloneType    domain.CloneType
	Similarity   float64
	Confidence   float64
	AnalyzerName string
}

// Classifier performs cascade classification of clone pairs.
type Classifier struct {
	config    ClassifierConfig
	analyzers map[domain.CloneType]SimilarityAnalyzer
}

// NewClassifier creates a new classifier with the given configuration.
func NewClassifier(config ClassifierConfig) *Classifier {
	return &Classifier{
		config:    config,
		analyzers: make(map[domain.CloneType]SimilarityAnalyzer),
	}
}

// RegisterAnalyzer registers a similarity analyzer for the given clone type.
func (c *Classifier) RegisterAnalyzer(cloneType domain.CloneType, analyzer SimilarityAnalyzer) {
	c.analyzers[cloneType] = analyzer
}

// Classify classifies a pair of code fragments using cascade classification.
// It tries Type-1 first (highest threshold), then Type-2, Type-3, Type-4.
// Returns nil if similarity is below all thresholds.
func (c *Classifier) Classify(f1, f2 *CodeFragment) *ClassificationResult {
	if f1 == nil || f2 == nil {
		return nil
	}

	// Try each type in cascade order (strictest to most lenient)
	types := []struct {
		ct        domain.CloneType
		threshold float64
		enabled   bool
	}{
		{domain.Type1Clone, c.config.Type1Threshold, c.config.EnableType1},
		{domain.Type2Clone, c.config.Type2Threshold, c.config.EnableType2},
		{domain.Type3Clone, c.config.Type3Threshold, c.config.EnableType3},
		{domain.Type4Clone, c.config.Type4Threshold, c.config.EnableType4},
	}

	for _, t := range types {
		if !t.enabled {
			continue
		}

		analyzer, ok := c.analyzers[t.ct]
		if !ok {
			continue
		}

		sim := analyzer.ComputeSimilarity(f1, f2)
		if sim >= t.threshold {
			// Confidence is how far above the threshold the similarity is,
			// normalized to 0-1 range above the threshold
			confidence := 1.0
			if t.threshold < 1.0 {
				confidence = (sim - t.threshold) / (1.0 - t.threshold)
			}
			if confidence > 1.0 {
				confidence = 1.0
			}

			return &ClassificationResult{
				CloneType:    t.ct,
				Similarity:   sim,
				Confidence:   confidence,
				AnalyzerName: analyzer.Name(),
			}
		}
	}

	return nil
}

// ClassifyBatch classifies multiple pairs and returns the results as ClonePairs.
// Pairs that don't meet any threshold are excluded from the result.
func (c *Classifier) ClassifyBatch(pairs [][2]*CodeFragment) []*ClonePair {
	results := make([]*ClonePair, 0, len(pairs))

	for _, pair := range pairs {
		result := c.Classify(pair[0], pair[1])
		if result == nil {
			continue
		}
		results = append(results, &ClonePair{
			Fragment1:    pair[0],
			Fragment2:    pair[1],
			Similarity:   result.Similarity,
			CloneType:    result.CloneType,
			Confidence:   result.Confidence,
			AnalyzerName: result.AnalyzerName,
		})
	}

	// Sort by similarity descending for deterministic output
	sort.Slice(results, func(i, j int) bool {
		if results[i].Similarity != results[j].Similarity {
			return results[i].Similarity > results[j].Similarity
		}
		return results[i].Fragment1.ItemKey() < results[j].Fragment1.ItemKey()
	})

	return results
}
