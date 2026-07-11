package clone

import (
	"testing"

	"github.com/ludo-technologies/polyscan/core/domain"
)

type mockAnalyzer struct {
	similarity float64
	name       string
}

func (m *mockAnalyzer) ComputeSimilarity(f1, f2 *CodeFragment) float64 { return m.similarity }
func (m *mockAnalyzer) Name() string                                   { return m.name }

func newTestClassifier(sim float64) *Classifier {
	cfg := DefaultClassifierConfig()
	c := NewClassifier(cfg)
	mock := &mockAnalyzer{similarity: sim, name: "mock"}
	c.RegisterAnalyzer(domain.Type1Clone, mock)
	c.RegisterAnalyzer(domain.Type2Clone, mock)
	c.RegisterAnalyzer(domain.Type3Clone, mock)
	c.RegisterAnalyzer(domain.Type4Clone, mock)
	return c
}

func TestClassifyNilFragments(t *testing.T) {
	c := newTestClassifier(0.9)
	f := &CodeFragment{ID: 1, FilePath: "a.go", StartLine: 1, EndLine: 10}

	if r := c.Classify(nil, f); r != nil {
		t.Error("expected nil for nil f1")
	}
	if r := c.Classify(f, nil); r != nil {
		t.Error("expected nil for nil f2")
	}
	if r := c.Classify(nil, nil); r != nil {
		t.Error("expected nil for both nil")
	}
}

func TestClassifyHighSimilarityReturnsType1(t *testing.T) {
	c := newTestClassifier(0.95)
	f1 := &CodeFragment{ID: 1, FilePath: "a.go", StartLine: 1, EndLine: 10}
	f2 := &CodeFragment{ID: 2, FilePath: "b.go", StartLine: 1, EndLine: 10}

	r := c.Classify(f1, f2)
	if r == nil {
		t.Fatal("expected non-nil result")
	}
	if r.CloneType != domain.Type1Clone {
		t.Errorf("CloneType = %v, want Type1Clone", r.CloneType)
	}
	if r.Similarity != 0.95 {
		t.Errorf("Similarity = %f, want 0.95", r.Similarity)
	}
	if r.AnalyzerName != "mock" {
		t.Errorf("AnalyzerName = %q, want %q", r.AnalyzerName, "mock")
	}
}

func TestClassifyMediumSimilarityReturnsType2(t *testing.T) {
	// 0.82 is above Type2 threshold (0.80) but below Type1 threshold (0.85)
	c := newTestClassifier(0.82)
	f1 := &CodeFragment{ID: 1, FilePath: "a.go", StartLine: 1, EndLine: 10}
	f2 := &CodeFragment{ID: 2, FilePath: "b.go", StartLine: 1, EndLine: 10}

	r := c.Classify(f1, f2)
	if r == nil {
		t.Fatal("expected non-nil result")
	}
	if r.CloneType != domain.Type2Clone {
		t.Errorf("CloneType = %v, want Type2Clone", r.CloneType)
	}
}

func TestClassifyType3Similarity(t *testing.T) {
	// 0.75 is above Type3 threshold (0.70) but below Type2 threshold (0.80)
	c := newTestClassifier(0.75)
	f1 := &CodeFragment{ID: 1, FilePath: "a.go", StartLine: 1, EndLine: 10}
	f2 := &CodeFragment{ID: 2, FilePath: "b.go", StartLine: 1, EndLine: 10}

	r := c.Classify(f1, f2)
	if r == nil {
		t.Fatal("expected non-nil result")
	}
	if r.CloneType != domain.Type3Clone {
		t.Errorf("CloneType = %v, want Type3Clone", r.CloneType)
	}
}

func TestClassifyType4Similarity(t *testing.T) {
	// 0.67 is above Type4 threshold (0.65) but below Type3 threshold (0.70)
	c := newTestClassifier(0.67)
	f1 := &CodeFragment{ID: 1, FilePath: "a.go", StartLine: 1, EndLine: 10}
	f2 := &CodeFragment{ID: 2, FilePath: "b.go", StartLine: 1, EndLine: 10}

	r := c.Classify(f1, f2)
	if r == nil {
		t.Fatal("expected non-nil result")
	}
	if r.CloneType != domain.Type4Clone {
		t.Errorf("CloneType = %v, want Type4Clone", r.CloneType)
	}
}

func TestClassifyBelowAllThresholds(t *testing.T) {
	c := newTestClassifier(0.50)
	f1 := &CodeFragment{ID: 1, FilePath: "a.go", StartLine: 1, EndLine: 10}
	f2 := &CodeFragment{ID: 2, FilePath: "b.go", StartLine: 1, EndLine: 10}

	r := c.Classify(f1, f2)
	if r != nil {
		t.Errorf("expected nil for similarity below all thresholds, got %+v", r)
	}
}

func TestClassifyDisabledType(t *testing.T) {
	cfg := DefaultClassifierConfig()
	cfg.EnableType1 = false
	c := NewClassifier(cfg)
	mock := &mockAnalyzer{similarity: 0.95, name: "mock"}
	c.RegisterAnalyzer(domain.Type1Clone, mock)
	c.RegisterAnalyzer(domain.Type2Clone, mock)
	c.RegisterAnalyzer(domain.Type3Clone, mock)
	c.RegisterAnalyzer(domain.Type4Clone, mock)

	f1 := &CodeFragment{ID: 1, FilePath: "a.go", StartLine: 1, EndLine: 10}
	f2 := &CodeFragment{ID: 2, FilePath: "b.go", StartLine: 1, EndLine: 10}

	r := c.Classify(f1, f2)
	if r == nil {
		t.Fatal("expected non-nil result")
	}
	// Type-1 is disabled, so it should fall through to Type-2
	if r.CloneType != domain.Type2Clone {
		t.Errorf("CloneType = %v, want Type2Clone (Type1 disabled)", r.CloneType)
	}
}

func TestClassifyBatch(t *testing.T) {
	cfg := DefaultClassifierConfig()
	c := NewClassifier(cfg)

	highMock := &mockAnalyzer{similarity: 0.90, name: "high"}
	lowMock := &mockAnalyzer{similarity: 0.50, name: "low"}

	// Register high analyzer for Type1, low for all others
	c.RegisterAnalyzer(domain.Type1Clone, highMock)
	c.RegisterAnalyzer(domain.Type2Clone, lowMock)
	c.RegisterAnalyzer(domain.Type3Clone, lowMock)
	c.RegisterAnalyzer(domain.Type4Clone, lowMock)

	f1 := &CodeFragment{ID: 1, FilePath: "a.go", StartLine: 1, EndLine: 10}
	f2 := &CodeFragment{ID: 2, FilePath: "b.go", StartLine: 1, EndLine: 10}
	f3 := &CodeFragment{ID: 3, FilePath: "c.go", StartLine: 1, EndLine: 10}

	pairs := [][2]*CodeFragment{
		{f1, f2}, // high similarity -> Type1
		{f2, f3}, // high similarity -> Type1
		{f1, f3}, // high similarity -> Type1
	}

	results := c.ClassifyBatch(pairs)
	if len(results) != 3 {
		t.Fatalf("expected 3 results, got %d", len(results))
	}

	for _, r := range results {
		if r.CloneType != domain.Type1Clone {
			t.Errorf("expected Type1Clone, got %v", r.CloneType)
		}
	}
}

func TestClassifyBatchFiltersBelowThreshold(t *testing.T) {
	cfg := DefaultClassifierConfig()
	c := NewClassifier(cfg)

	lowMock := &mockAnalyzer{similarity: 0.30, name: "low"}
	c.RegisterAnalyzer(domain.Type1Clone, lowMock)
	c.RegisterAnalyzer(domain.Type2Clone, lowMock)
	c.RegisterAnalyzer(domain.Type3Clone, lowMock)
	c.RegisterAnalyzer(domain.Type4Clone, lowMock)

	f1 := &CodeFragment{ID: 1, FilePath: "a.go", StartLine: 1, EndLine: 10}
	f2 := &CodeFragment{ID: 2, FilePath: "b.go", StartLine: 1, EndLine: 10}

	pairs := [][2]*CodeFragment{{f1, f2}}
	results := c.ClassifyBatch(pairs)
	if len(results) != 0 {
		t.Errorf("expected 0 results for below-threshold pairs, got %d", len(results))
	}
}

func TestDefaultClassifierConfig(t *testing.T) {
	cfg := DefaultClassifierConfig()

	if cfg.Type1Threshold != domain.DefaultType1CloneThreshold {
		t.Errorf("Type1Threshold = %f, want %f", cfg.Type1Threshold, domain.DefaultType1CloneThreshold)
	}
	if cfg.Type2Threshold != domain.DefaultType2CloneThreshold {
		t.Errorf("Type2Threshold = %f, want %f", cfg.Type2Threshold, domain.DefaultType2CloneThreshold)
	}
	if cfg.Type3Threshold != domain.DefaultType3CloneThreshold {
		t.Errorf("Type3Threshold = %f, want %f", cfg.Type3Threshold, domain.DefaultType3CloneThreshold)
	}
	if cfg.Type4Threshold != domain.DefaultType4CloneThreshold {
		t.Errorf("Type4Threshold = %f, want %f", cfg.Type4Threshold, domain.DefaultType4CloneThreshold)
	}
	if !cfg.EnableType1 || !cfg.EnableType2 || !cfg.EnableType3 || !cfg.EnableType4 {
		t.Error("all clone types should be enabled by default")
	}
	if cfg.JaccardPreFilterThreshold != 0.0 {
		t.Errorf("JaccardPreFilterThreshold = %f, want 0.0", cfg.JaccardPreFilterThreshold)
	}
}

func TestClassifyConfidence(t *testing.T) {
	// With similarity=1.0 and threshold=0.85, confidence should be 1.0
	c := newTestClassifier(1.0)
	f1 := &CodeFragment{ID: 1, FilePath: "a.go", StartLine: 1, EndLine: 10}
	f2 := &CodeFragment{ID: 2, FilePath: "b.go", StartLine: 1, EndLine: 10}

	r := c.Classify(f1, f2)
	if r == nil {
		t.Fatal("expected non-nil result")
	}
	if r.Confidence != 1.0 {
		t.Errorf("Confidence = %f, want 1.0", r.Confidence)
	}
}

func TestClassifyNoAnalyzerRegistered(t *testing.T) {
	cfg := DefaultClassifierConfig()
	c := NewClassifier(cfg)
	// No analyzers registered

	f1 := &CodeFragment{ID: 1, FilePath: "a.go", StartLine: 1, EndLine: 10}
	f2 := &CodeFragment{ID: 2, FilePath: "b.go", StartLine: 1, EndLine: 10}

	r := c.Classify(f1, f2)
	if r != nil {
		t.Errorf("expected nil with no analyzers registered, got %+v", r)
	}
}
