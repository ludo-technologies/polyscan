package analyzer

import (
	"testing"
)

func TestNewLSHIndex(t *testing.T) {
	// Default values
	idx := NewLSHIndex(0, 0)
	if idx.Bands() != 32 {
		t.Errorf("Expected default 32 bands, got %d", idx.Bands())
	}
	if idx.Rows() != 4 {
		t.Errorf("Expected default 4 rows, got %d", idx.Rows())
	}

	// Custom values
	idx = NewLSHIndex(16, 8)
	if idx.Bands() != 16 {
		t.Errorf("Expected 16 bands, got %d", idx.Bands())
	}
	if idx.Rows() != 8 {
		t.Errorf("Expected 8 rows, got %d", idx.Rows())
	}
}

func TestAddFragment(t *testing.T) {
	idx := NewLSHIndex(32, 4)
	mh := NewMinHasher(128)

	sig := mh.ComputeSignature([]string{"a", "b", "c"})
	err := idx.AddFragment(1, sig)
	if err != nil {
		t.Errorf("AddFragment failed: %v", err)
	}

	if idx.Size() != 1 {
		t.Errorf("Expected size 1, got %d", idx.Size())
	}
}

func TestAddFragmentErrors(t *testing.T) {
	idx := NewLSHIndex(32, 4)
	mh := NewMinHasher(128)

	// Empty signature
	err := idx.AddFragment(1, nil)
	if err == nil {
		t.Error("Expected error for nil signature")
	}

	// Negative ID
	sig := mh.ComputeSignature([]string{"a", "b"})
	err = idx.AddFragment(-1, sig)
	if err == nil {
		t.Error("Expected error for negative ID")
	}
}

func TestFindCandidates(t *testing.T) {
	idx := NewLSHIndex(32, 4)
	mh := NewMinHasher(128)

	// Add similar fragments
	sig1 := mh.ComputeSignature([]string{"a", "b", "c", "d", "e"})
	sig2 := mh.ComputeSignature([]string{"a", "b", "c", "x", "y"})
	sig3 := mh.ComputeSignature([]string{"p", "q", "r", "s", "t"})

	_ = idx.AddFragment(1, sig1)
	_ = idx.AddFragment(2, sig2)
	_ = idx.AddFragment(3, sig3)

	// Query with sig1 - should find itself
	candidates := idx.FindCandidates(sig1)

	if len(candidates) == 0 {
		t.Error("Expected at least one candidate")
	}

	// Should include fragment 1 (itself)
	found := false
	for _, c := range candidates {
		if c == 1 {
			found = true
			break
		}
	}
	if !found {
		t.Error("Should find itself as candidate")
	}
}

func TestFindCandidatesEmpty(t *testing.T) {
	idx := NewLSHIndex(32, 4)

	// nil signature
	candidates := idx.FindCandidates(nil)
	if len(candidates) != 0 {
		t.Error("Expected empty result for nil signature")
	}

	// Empty index
	mh := NewMinHasher(128)
	sig := mh.ComputeSignature([]string{"a", "b"})
	candidates = idx.FindCandidates(sig)
	if len(candidates) != 0 {
		t.Error("Expected empty result for empty index")
	}
}

func TestGetSignature(t *testing.T) {
	idx := NewLSHIndex(32, 4)
	mh := NewMinHasher(128)

	sig := mh.ComputeSignature([]string{"a", "b", "c"})
	_ = idx.AddFragment(1, sig)

	retrieved := idx.GetSignature(1)
	if retrieved == nil {
		t.Error("Expected to retrieve signature")
	}
	if len(retrieved.Signatures()) != len(sig.Signatures()) {
		t.Error("Retrieved signature should match original")
	}

	// Non-existent
	retrieved = idx.GetSignature(999)
	if retrieved != nil {
		t.Error("Expected nil for non-existent ID")
	}
}

func TestBuildIndex(t *testing.T) {
	idx := NewLSHIndex(32, 4)

	// BuildIndex is a no-op but should not error
	err := idx.BuildIndex()
	if err != nil {
		t.Errorf("BuildIndex should not error: %v", err)
	}
}

func TestDuplicateAvoidance(t *testing.T) {
	idx := NewLSHIndex(32, 4)
	mh := NewMinHasher(128)

	sig := mh.ComputeSignature([]string{"a", "b", "c"})

	// Add same fragment twice
	_ = idx.AddFragment(1, sig)
	_ = idx.AddFragment(1, sig)

	// Size should still be 1
	if idx.Size() != 1 {
		t.Errorf("Expected size 1 (no duplicates), got %d", idx.Size())
	}

	// Candidates should not have duplicates
	candidates := idx.FindCandidates(sig)
	idCount := make(map[int]int)
	for _, c := range candidates {
		idCount[c]++
	}
	for id, count := range idCount {
		if count > 1 {
			t.Errorf("Found duplicate candidate %d (count: %d)", id, count)
		}
	}
}

func TestLSHSensitivity(t *testing.T) {
	// Test that similar items are more likely to be found than dissimilar
	mh := NewMinHasher(128)

	// Create base signature
	baseFeatures := []string{"a", "b", "c", "d", "e", "f", "g", "h"}
	baseSig := mh.ComputeSignature(baseFeatures)

	// Similar features (7/9 overlap)
	similarFeatures := []string{"a", "b", "c", "d", "e", "f", "g", "x", "y"}
	similarSig := mh.ComputeSignature(similarFeatures)

	// Different features (0 overlap)
	differentFeatures := []string{"1", "2", "3", "4", "5", "6", "7", "8"}
	differentSig := mh.ComputeSignature(differentFeatures)

	// Create index and add fragments
	idx := NewLSHIndex(32, 4)
	_ = idx.AddFragment(1, baseSig)
	_ = idx.AddFragment(2, similarSig)
	_ = idx.AddFragment(3, differentSig)

	// Query with similar signature
	candidates := idx.FindCandidates(similarSig)

	// Count how many of base vs different are found
	hasBase := false
	hasDifferent := false
	for _, c := range candidates {
		if c == 1 {
			hasBase = true
		}
		if c == 3 {
			hasDifferent = true
		}
	}

	// Similar items should have higher chance of matching bands
	// This is probabilistic, but we expect base to be found more often
	t.Logf("Candidates for similar: %v, hasBase: %v, hasDifferent: %v", candidates, hasBase, hasDifferent)
}

func TestLSHIndexWithManyFragments(t *testing.T) {
	idx := NewLSHIndex(32, 4)
	mh := NewMinHasher(128)

	// Add many fragments
	for i := 0; i < 100; i++ {
		features := []string{
			"feature_" + string(rune('a'+i%26)),
			"feature_" + string(rune('A'+i%26)),
			"common",
		}
		sig := mh.ComputeSignature(features)
		_ = idx.AddFragment(i, sig)
	}

	if idx.Size() != 100 {
		t.Errorf("Expected 100 fragments, got %d", idx.Size())
	}
}

func TestLSHIndexFindCandidatesKeepsOversizedBucketCandidates(t *testing.T) {
	mh := NewMinHasher(128)
	sig := mh.ComputeSignature([]string{"same", "feature", "set"})

	lsh := NewLSHIndex(32, 4).WithMaxCandidates(2)
	for _, id := range []int{1, 2, 3} {
		if err := lsh.AddFragment(id, sig); err != nil {
			t.Fatalf("add %d: %v", id, err)
		}
	}

	cands := lsh.FindCandidates(sig)
	if len(cands) != 2 {
		t.Fatalf("expected oversized bucket candidates capped at 2; got %v", cands)
	}
}

func TestLSHIndexFindCandidatesCapsTotalCandidates(t *testing.T) {
	mh := NewMinHasher(128)
	query := mh.ComputeSignature([]string{"shared", "query", "features"})

	lsh := NewLSHIndex(32, 4).WithMaxCandidates(2)
	keys := lsh.computeBandKeys(query)
	lsh.buckets[keys[0]] = []int{1}
	lsh.buckets[keys[1]] = []int{2}
	lsh.buckets[keys[2]] = []int{3}

	cands := lsh.FindCandidates(query)
	if len(cands) > 2 {
		t.Fatalf("expected candidates to be capped at 2; got %v", cands)
	}
}

func TestLSHIndexFindCandidatesUsesDefaultCapAtBoundary(t *testing.T) {
	mh := NewMinHasher(128)
	sig := mh.ComputeSignature([]string{"same", "feature", "set"})

	lsh := NewLSHIndex(32, 4)
	for id := 0; id <= defaultLSHMaxCandidates; id++ {
		if err := lsh.AddFragment(id, sig); err != nil {
			t.Fatalf("add %d: %v", id, err)
		}
	}

	cands := lsh.FindCandidates(sig)
	if len(cands) != defaultLSHMaxCandidates {
		t.Fatalf("candidate count mismatch: want %d got %d", defaultLSHMaxCandidates, len(cands))
	}
	for i, id := range cands {
		if id != i {
			t.Fatalf("candidate order mismatch at %d: got %d", i, id)
		}
	}
}

func TestLSHIndexFindCandidatesReturnsDeterministicIndexes(t *testing.T) {
	mh := NewMinHasher(128)
	sig := mh.ComputeSignature([]string{"same", "feature", "set"})

	lsh := NewLSHIndex(32, 4)
	for _, id := range []int{4, 2, 3, 1} {
		if err := lsh.AddFragment(id, sig); err != nil {
			t.Fatalf("add %d: %v", id, err)
		}
	}

	cands := lsh.FindCandidates(sig)
	want := []int{1, 2, 3, 4}
	if len(cands) != len(want) {
		t.Fatalf("candidate count mismatch: want %v got %v", want, cands)
	}
	for i := range want {
		if cands[i] != want[i] {
			t.Fatalf("candidate order mismatch: want %v got %v", want, cands)
		}
	}
}
