package lsh

import (
	"hash/fnv"
	"math"
	"math/rand"
)

const (
	defaultNumHashes = 128
	hashSeed         = 0x5eed_1234_cafe_babe
)

// MinHashSignature holds the signature vector.
type MinHashSignature struct {
	signatures []uint64
	numHashes  int
}

// Signatures returns the signature slice.
func (s *MinHashSignature) Signatures() []uint64 {
	return s.signatures
}

// NumHashes returns the number of hash functions used.
func (s *MinHashSignature) NumHashes() int {
	return s.numHashes
}

// HashFunc maps a 64-bit base hash to another 64-bit value.
type HashFunc func(uint64) uint64

// MinHasher computes MinHash signatures for feature sets.
type MinHasher struct {
	numHashes     int
	hashFunctions []HashFunc
}

// NewMinHasher creates a MinHasher with numHashes functions (default 128 if invalid).
func NewMinHasher(numHashes int) *MinHasher {
	if numHashes <= 0 {
		numHashes = defaultNumHashes
	}
	mh := &MinHasher{numHashes: numHashes}
	mh.generateHashFunctions()
	return mh
}

func (m *MinHasher) generateHashFunctions() {
	rng := rand.New(rand.NewSource(hashSeed))
	a := make([]uint64, m.numHashes)
	b := make([]uint64, m.numHashes)
	for i := 0; i < m.numHashes; i++ {
		a[i] = rng.Uint64() | 1 // odd to avoid trivial cycles
		b[i] = rng.Uint64()
	}
	m.hashFunctions = make([]HashFunc, m.numHashes)
	for i := 0; i < m.numHashes; i++ {
		ai, bi := a[i], b[i]
		m.hashFunctions[i] = func(x uint64) uint64 {
			return (ai * x) ^ bi + ai + bi
		}
	}
}

// ComputeSignature computes the MinHash signature for a set of features.
func (m *MinHasher) ComputeSignature(features []string) *MinHashSignature {
	if len(features) == 0 {
		return &MinHashSignature{signatures: make([]uint64, m.numHashes), numHashes: m.numHashes}
	}
	set := make(map[string]struct{}, len(features))
	for _, f := range features {
		set[f] = struct{}{}
	}
	base := make([]uint64, 0, len(set))
	for f := range set {
		base = append(base, Hash64(f))
	}
	sig := make([]uint64, m.numHashes)
	for i := 0; i < m.numHashes; i++ {
		sig[i] = math.MaxUint64
	}
	for i := 0; i < m.numHashes; i++ {
		hi := m.hashFunctions[i]
		minv := uint64(math.MaxUint64)
		for _, x := range base {
			if v := hi(x); v < minv {
				minv = v
			}
		}
		sig[i] = minv
	}
	return &MinHashSignature{signatures: sig, numHashes: m.numHashes}
}

// EstimateJaccardSimilarity estimates Jaccard similarity via signature agreement ratio.
func (m *MinHasher) EstimateJaccardSimilarity(sig1, sig2 *MinHashSignature) float64 {
	if sig1 == nil || sig2 == nil || len(sig1.signatures) == 0 || len(sig2.signatures) == 0 {
		return 0.0
	}
	n := MinInt(len(sig1.signatures), len(sig2.signatures))
	if n == 0 {
		return 0.0
	}
	match := 0
	for i := 0; i < n; i++ {
		if sig1.signatures[i] == sig2.signatures[i] {
			match++
		}
	}
	return float64(match) / float64(n)
}

// NumHashes returns the number of hash functions.
func (m *MinHasher) NumHashes() int { return m.numHashes }

// Hash64 computes a 64-bit FNV-1a hash for a string.
func Hash64(s string) uint64 {
	h := fnv.New64a()
	_, _ = h.Write([]byte(s))
	return h.Sum64()
}

// MinInt returns the smaller of two ints.
func MinInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}
