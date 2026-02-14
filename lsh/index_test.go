package lsh

import (
	"testing"
)

func TestNewLSHIndex_Defaults(t *testing.T) {
	idx := NewLSHIndex(0, 0)
	if idx.Bands() != 32 {
		t.Errorf("Default bands = %d, want 32", idx.Bands())
	}
	if idx.Rows() != 4 {
		t.Errorf("Default rows = %d, want 4", idx.Rows())
	}
}

func TestNewLSHIndex_Custom(t *testing.T) {
	idx := NewLSHIndex(16, 8)
	if idx.Bands() != 16 {
		t.Errorf("Bands = %d, want 16", idx.Bands())
	}
	if idx.Rows() != 8 {
		t.Errorf("Rows = %d, want 8", idx.Rows())
	}
}

func TestAddFragment(t *testing.T) {
	idx := NewLSHIndex(32, 4)
	mh := NewMinHasher(128)
	sig := mh.ComputeSignature([]string{"a", "b", "c"})

	err := idx.AddFragment("frag1", sig)
	if err != nil {
		t.Fatalf("AddFragment failed: %v", err)
	}
	if idx.Size() != 1 {
		t.Errorf("Size = %d, want 1", idx.Size())
	}
}

func TestAddFragment_Errors(t *testing.T) {
	idx := NewLSHIndex(32, 4)
	mh := NewMinHasher(128)
	sig := mh.ComputeSignature([]string{"a"})

	// Empty signature
	if err := idx.AddFragment("id", nil); err == nil {
		t.Error("Expected error for nil signature")
	}

	emptySig := &MinHashSignature{signatures: []uint64{}, numHashes: 0}
	if err := idx.AddFragment("id", emptySig); err == nil {
		t.Error("Expected error for empty signature")
	}

	// Empty ID
	if err := idx.AddFragment("", sig); err == nil {
		t.Error("Expected error for empty ID")
	}
}

func TestFindCandidates(t *testing.T) {
	idx := NewLSHIndex(32, 4)
	mh := NewMinHasher(128)

	// Two similar feature sets
	sigA := mh.ComputeSignature([]string{"a", "b", "c", "d"})
	sigB := mh.ComputeSignature([]string{"a", "b", "c", "e"})
	// One very different
	sigC := mh.ComputeSignature([]string{"x", "y", "z", "w"})

	_ = idx.AddFragment("A", sigA)
	_ = idx.AddFragment("B", sigB)
	_ = idx.AddFragment("C", sigC)

	candidates := idx.FindCandidates(sigA)
	// A should always be found (it's its own candidate)
	foundA := false
	foundB := false
	for _, c := range candidates {
		if c == "A" {
			foundA = true
		}
		if c == "B" {
			foundB = true
		}
	}
	if !foundA {
		t.Error("Expected A to be a candidate for itself")
	}
	if !foundB {
		t.Error("Expected B to be a candidate for A (similar features)")
	}
}

func TestFindCandidates_Empty(t *testing.T) {
	idx := NewLSHIndex(32, 4)
	candidates := idx.FindCandidates(nil)
	if len(candidates) != 0 {
		t.Errorf("Expected 0 candidates for nil, got %d", len(candidates))
	}

	mh := NewMinHasher(128)
	sig := mh.ComputeSignature([]string{"a"})
	candidates = idx.FindCandidates(sig)
	if len(candidates) != 0 {
		t.Errorf("Expected 0 candidates from empty index, got %d", len(candidates))
	}
}

func TestGetSignature(t *testing.T) {
	idx := NewLSHIndex(32, 4)
	mh := NewMinHasher(128)
	sig := mh.ComputeSignature([]string{"a"})

	_ = idx.AddFragment("test", sig)

	retrieved := idx.GetSignature("test")
	if retrieved != sig {
		t.Error("GetSignature returned different object")
	}

	if idx.GetSignature("nonexistent") != nil {
		t.Error("Expected nil for nonexistent ID")
	}
}

func TestBuildIndex(t *testing.T) {
	idx := NewLSHIndex(32, 4)
	if err := idx.BuildIndex(); err != nil {
		t.Errorf("BuildIndex should be no-op, got error: %v", err)
	}
}

func TestDuplicateAvoidance(t *testing.T) {
	idx := NewLSHIndex(32, 4)
	mh := NewMinHasher(128)
	sig := mh.ComputeSignature([]string{"a", "b"})

	_ = idx.AddFragment("dup", sig)
	_ = idx.AddFragment("dup", sig) // Add same ID again

	candidates := idx.FindCandidates(sig)
	count := 0
	for _, c := range candidates {
		if c == "dup" {
			count++
		}
	}
	if count > 1 {
		t.Errorf("Expected 'dup' at most once in candidates, got %d", count)
	}
}

func TestLSH_ManyFragments(t *testing.T) {
	idx := NewLSHIndex(32, 4)
	mh := NewMinHasher(128)

	const n = 100
	for i := 0; i < n; i++ {
		features := []string{string(rune('a' + i%26)), string(rune('A' + i%26))}
		sig := mh.ComputeSignature(features)
		if err := idx.AddFragment(string(rune(i)), sig); err != nil {
			t.Fatalf("Failed to add fragment %d: %v", i, err)
		}
	}

	if idx.Size() != n {
		t.Errorf("Size = %d, want %d", idx.Size(), n)
	}
}

func TestLSH_SimilarItemsFoundMore(t *testing.T) {
	idx := NewLSHIndex(32, 4)
	mh := NewMinHasher(128)

	// Similar pair
	base := []string{"a", "b", "c", "d", "e", "f"}
	similar := []string{"a", "b", "c", "d", "e", "g"}
	// Dissimilar
	different := []string{"w", "x", "y", "z"}

	sigBase := mh.ComputeSignature(base)
	sigSimilar := mh.ComputeSignature(similar)
	sigDiff := mh.ComputeSignature(different)

	_ = idx.AddFragment("base", sigBase)
	_ = idx.AddFragment("similar", sigSimilar)
	_ = idx.AddFragment("different", sigDiff)

	candidates := idx.FindCandidates(sigBase)
	hasSimilar := false
	for _, c := range candidates {
		if c == "similar" {
			hasSimilar = true
		}
	}
	if !hasSimilar {
		t.Error("Expected 'similar' to be a candidate for 'base'")
	}
}

func BenchmarkLSH_AddAndFind(b *testing.B) {
	mh := NewMinHasher(128)
	sigs := make([]*MinHashSignature, 50)
	for i := range sigs {
		sigs[i] = mh.ComputeSignature([]string{string(rune('a' + i%26)), "common"})
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		idx := NewLSHIndex(32, 4)
		for j, sig := range sigs {
			_ = idx.AddFragment(string(rune(j)), sig)
		}
		idx.FindCandidates(sigs[0])
	}
}
