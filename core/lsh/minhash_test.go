package lsh

import (
	"math"
	"testing"
)

func TestNewMinHasher_Defaults(t *testing.T) {
	tests := []struct {
		input    int
		expected int
	}{
		{0, 128},
		{-1, 128},
		{64, 64},
		{256, 256},
	}

	for _, tt := range tests {
		mh := NewMinHasher(tt.input)
		if mh.NumHashes() != tt.expected {
			t.Errorf("NewMinHasher(%d).NumHashes() = %d, want %d", tt.input, mh.NumHashes(), tt.expected)
		}
	}
}

func TestComputeSignature_Empty(t *testing.T) {
	mh := NewMinHasher(128)
	sig := mh.ComputeSignature([]string{})
	if sig.NumHashes() != 128 {
		t.Errorf("NumHashes = %d, want 128", sig.NumHashes())
	}
	if len(sig.Signatures()) != 128 {
		t.Errorf("Signatures length = %d, want 128", len(sig.Signatures()))
	}
}

func TestComputeSignature_NonEmpty(t *testing.T) {
	mh := NewMinHasher(128)
	sig := mh.ComputeSignature([]string{"hello", "world"})

	if sig.NumHashes() != 128 {
		t.Errorf("NumHashes = %d, want 128", sig.NumHashes())
	}

	// At least some signatures should not be MaxUint64
	allMax := true
	for _, v := range sig.Signatures() {
		if v != math.MaxUint64 {
			allMax = false
			break
		}
	}
	if allMax {
		t.Error("Expected non-MaxUint64 values in signature")
	}
}

func TestComputeSignature_Deduplication(t *testing.T) {
	mh := NewMinHasher(128)
	sig1 := mh.ComputeSignature([]string{"a", "b", "c"})
	sig2 := mh.ComputeSignature([]string{"a", "b", "c", "a", "b"})

	// Deduplication should produce identical signatures
	for i := range sig1.Signatures() {
		if sig1.Signatures()[i] != sig2.Signatures()[i] {
			t.Errorf("Signature mismatch at index %d after dedup", i)
			break
		}
	}
}

func TestEstimateJaccardSimilarity_Identical(t *testing.T) {
	mh := NewMinHasher(128)
	sig := mh.ComputeSignature([]string{"a", "b", "c"})
	sim := mh.EstimateJaccardSimilarity(sig, sig)
	if sim != 1.0 {
		t.Errorf("Identical signature similarity = %f, want 1.0", sim)
	}
}

func TestEstimateJaccardSimilarity_Different(t *testing.T) {
	mh := NewMinHasher(128)
	sig1 := mh.ComputeSignature([]string{"a", "b", "c"})
	sig2 := mh.ComputeSignature([]string{"x", "y", "z"})
	sim := mh.EstimateJaccardSimilarity(sig1, sig2)
	// Completely different sets should have low similarity
	if sim > 0.3 {
		t.Errorf("Different sets similarity = %f, expected < 0.3", sim)
	}
}

func TestEstimateJaccardSimilarity_Overlap(t *testing.T) {
	mh := NewMinHasher(256)
	sig1 := mh.ComputeSignature([]string{"a", "b", "c", "d"})
	sig2 := mh.ComputeSignature([]string{"a", "b", "c", "e"})
	sim := mh.EstimateJaccardSimilarity(sig1, sig2)
	// Jaccard = |{a,b,c}| / |{a,b,c,d,e}| = 3/5 = 0.6
	if sim < 0.4 || sim > 0.8 {
		t.Errorf("Overlapping sets similarity = %f, expected ~0.6", sim)
	}
}

func TestEstimateJaccardSimilarity_Nil(t *testing.T) {
	mh := NewMinHasher(128)
	sig := mh.ComputeSignature([]string{"a"})

	if sim := mh.EstimateJaccardSimilarity(nil, sig); sim != 0.0 {
		t.Errorf("nil/sig similarity = %f, want 0.0", sim)
	}
	if sim := mh.EstimateJaccardSimilarity(sig, nil); sim != 0.0 {
		t.Errorf("sig/nil similarity = %f, want 0.0", sim)
	}
	if sim := mh.EstimateJaccardSimilarity(nil, nil); sim != 0.0 {
		t.Errorf("nil/nil similarity = %f, want 0.0", sim)
	}
}

func TestMinHash_Determinism(t *testing.T) {
	mh1 := NewMinHasher(128)
	mh2 := NewMinHasher(128)
	features := []string{"foo", "bar", "baz"}

	sig1 := mh1.ComputeSignature(features)
	sig2 := mh2.ComputeSignature(features)

	for i := range sig1.Signatures() {
		if sig1.Signatures()[i] != sig2.Signatures()[i] {
			t.Errorf("Non-deterministic at index %d: %d vs %d", i, sig1.Signatures()[i], sig2.Signatures()[i])
			break
		}
	}
}

func TestHash64(t *testing.T) {
	h1 := Hash64("hello")
	h2 := Hash64("hello")
	h3 := Hash64("world")

	if h1 != h2 {
		t.Error("Same input should produce same hash")
	}
	if h1 == h3 {
		t.Error("Different inputs should produce different hashes")
	}
	if h1 == 0 {
		t.Error("Hash should not be zero for non-empty input")
	}
}

func TestMinInt(t *testing.T) {
	if MinInt(1, 2) != 1 {
		t.Error("MinInt(1, 2) should be 1")
	}
	if MinInt(5, 3) != 3 {
		t.Error("MinInt(5, 3) should be 3")
	}
	if MinInt(4, 4) != 4 {
		t.Error("MinInt(4, 4) should be 4")
	}
}

func BenchmarkComputeSignature(b *testing.B) {
	mh := NewMinHasher(128)
	features := make([]string, 100)
	for i := range features {
		features[i] = string(rune('a' + i%26))
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		mh.ComputeSignature(features)
	}
}
