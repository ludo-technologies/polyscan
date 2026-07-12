package analyzer

import (
	"math"
	"testing"
)

func TestNewMinHasher(t *testing.T) {
	// Default value for invalid input
	mh := NewMinHasher(0)
	if mh.NumHashes() != 128 {
		t.Errorf("Expected default 128 hashes, got %d", mh.NumHashes())
	}

	mh = NewMinHasher(-1)
	if mh.NumHashes() != 128 {
		t.Errorf("Expected default 128 hashes for negative input, got %d", mh.NumHashes())
	}

	// Custom value
	mh = NewMinHasher(64)
	if mh.NumHashes() != 64 {
		t.Errorf("Expected 64 hashes, got %d", mh.NumHashes())
	}
}

func TestComputeSignatureEmpty(t *testing.T) {
	mh := NewMinHasher(128)
	sig := mh.ComputeSignature([]string{})

	if sig == nil {
		t.Fatal("Signature should not be nil")
	}
	if len(sig.Signatures()) != 128 {
		t.Errorf("Expected 128 signatures, got %d", len(sig.Signatures()))
	}
}

func TestComputeSignature(t *testing.T) {
	mh := NewMinHasher(128)
	features := []string{"feature1", "feature2", "feature3"}
	sig := mh.ComputeSignature(features)

	if sig == nil {
		t.Fatal("Signature should not be nil")
	}
	if len(sig.Signatures()) != 128 {
		t.Errorf("Expected 128 signatures, got %d", len(sig.Signatures()))
	}

	// Verify signatures are not all MaxUint64 (which would indicate no hashing occurred)
	allMax := true
	for _, s := range sig.Signatures() {
		if s != math.MaxUint64 {
			allMax = false
			break
		}
	}
	if allMax {
		t.Error("Signatures should not all be MaxUint64")
	}
}

func TestComputeSignatureDeduplication(t *testing.T) {
	mh := NewMinHasher(128)

	// Duplicate features should produce same signature
	features1 := []string{"a", "b", "c"}
	features2 := []string{"a", "b", "c", "a", "b"} // with duplicates

	sig1 := mh.ComputeSignature(features1)
	sig2 := mh.ComputeSignature(features2)

	// Signatures should be identical
	for i := 0; i < len(sig1.Signatures()); i++ {
		if sig1.Signatures()[i] != sig2.Signatures()[i] {
			t.Errorf("Signatures differ at index %d: %d vs %d", i, sig1.Signatures()[i], sig2.Signatures()[i])
			break
		}
	}
}

func TestEstimateJaccardSimilarityIdentical(t *testing.T) {
	mh := NewMinHasher(128)
	features := []string{"a", "b", "c", "d", "e"}
	sig := mh.ComputeSignature(features)

	similarity := mh.EstimateJaccardSimilarity(sig, sig)
	if similarity != 1.0 {
		t.Errorf("Identical signatures should have similarity 1.0, got %f", similarity)
	}
}

func TestEstimateJaccardSimilarityDifferent(t *testing.T) {
	mh := NewMinHasher(128)

	features1 := []string{"a", "b", "c"}
	features2 := []string{"x", "y", "z"}

	sig1 := mh.ComputeSignature(features1)
	sig2 := mh.ComputeSignature(features2)

	similarity := mh.EstimateJaccardSimilarity(sig1, sig2)
	if similarity < 0.0 || similarity > 1.0 {
		t.Errorf("Similarity should be between 0 and 1, got %f", similarity)
	}

	// Different sets should have low similarity
	if similarity > 0.5 {
		t.Errorf("Expected low similarity for different sets, got %f", similarity)
	}
}

func TestEstimateJaccardSimilarityOverlap(t *testing.T) {
	mh := NewMinHasher(256) // More hashes for better accuracy

	features1 := []string{"a", "b", "c", "d", "e"}
	features2 := []string{"a", "b", "c", "x", "y"}

	sig1 := mh.ComputeSignature(features1)
	sig2 := mh.ComputeSignature(features2)

	similarity := mh.EstimateJaccardSimilarity(sig1, sig2)

	// True Jaccard: |intersection| / |union| = 3 / 7 ≈ 0.43
	// MinHash should estimate this approximately
	if similarity < 0.2 || similarity > 0.7 {
		t.Errorf("Expected similarity around 0.43, got %f", similarity)
	}
}

func TestEstimateJaccardSimilarityNil(t *testing.T) {
	mh := NewMinHasher(128)
	sig := mh.ComputeSignature([]string{"a", "b"})

	// nil signatures
	similarity := mh.EstimateJaccardSimilarity(nil, sig)
	if similarity != 0.0 {
		t.Errorf("Nil signature should return 0.0, got %f", similarity)
	}

	similarity = mh.EstimateJaccardSimilarity(sig, nil)
	if similarity != 0.0 {
		t.Errorf("Nil signature should return 0.0, got %f", similarity)
	}

	similarity = mh.EstimateJaccardSimilarity(nil, nil)
	if similarity != 0.0 {
		t.Errorf("Both nil should return 0.0, got %f", similarity)
	}
}

func TestMinHashDeterminism(t *testing.T) {
	// Same input should always produce same output
	mh1 := NewMinHasher(128)
	mh2 := NewMinHasher(128)

	features := []string{"test", "features", "for", "determinism"}

	sig1 := mh1.ComputeSignature(features)
	sig2 := mh2.ComputeSignature(features)

	for i := 0; i < len(sig1.Signatures()); i++ {
		if sig1.Signatures()[i] != sig2.Signatures()[i] {
			t.Errorf("MinHash should be deterministic, signatures differ at index %d", i)
			break
		}
	}
}

func TestHash64(t *testing.T) {
	// Same string should produce same hash
	h1 := hash64("test")
	h2 := hash64("test")
	if h1 != h2 {
		t.Error("Same string should produce same hash")
	}

	// Different strings should produce different hashes (with high probability)
	h3 := hash64("different")
	if h1 == h3 {
		t.Error("Different strings should produce different hashes")
	}
}
