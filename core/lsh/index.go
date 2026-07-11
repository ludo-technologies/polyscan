package lsh

import (
	"encoding/binary"
	"fmt"
	"hash/fnv"
)

const (
	defaultBands = 32
	defaultRows  = 4
)

// LSHIndex implements MinHash LSH with banding.
type LSHIndex struct {
	bands      int
	rows       int
	buckets    map[string][]string
	signatures map[string]*MinHashSignature
}

// NewLSHIndex creates an index with banding parameters.
func NewLSHIndex(bands, rows int) *LSHIndex {
	if bands <= 0 {
		bands = defaultBands
	}
	if rows <= 0 {
		rows = defaultRows
	}
	return &LSHIndex{
		bands:      bands,
		rows:       rows,
		buckets:    make(map[string][]string),
		signatures: make(map[string]*MinHashSignature),
	}
}

// AddFragment inserts a fragment signature into the index.
func (idx *LSHIndex) AddFragment(id string, signature *MinHashSignature) error {
	if signature == nil || len(signature.signatures) == 0 {
		return fmt.Errorf("empty signature for id %s", id)
	}
	if id == "" {
		return fmt.Errorf("empty fragment id")
	}
	idx.signatures[id] = signature
	idx.addToBuckets(id, signature)
	return nil
}

// BuildIndex is a no-op for incremental building (kept for API symmetry).
func (idx *LSHIndex) BuildIndex() error { return nil }

// FindCandidates retrieves candidate fragment IDs that share at least one band bucket.
func (idx *LSHIndex) FindCandidates(signature *MinHashSignature) []string {
	if signature == nil || len(signature.signatures) == 0 {
		return []string{}
	}
	ids := make(map[string]struct{})
	bands := idx.computeBandKeys(signature)
	for _, key := range bands {
		if bucket, ok := idx.buckets[key]; ok {
			for _, id := range bucket {
				ids[id] = struct{}{}
			}
		}
	}
	out := make([]string, 0, len(ids))
	for id := range ids {
		out = append(out, id)
	}
	return out
}

// GetSignature returns the stored signature for a fragment ID.
func (idx *LSHIndex) GetSignature(id string) *MinHashSignature {
	return idx.signatures[id]
}

// Size returns the number of fragments in the index.
func (idx *LSHIndex) Size() int {
	return len(idx.signatures)
}

// Bands returns the number of bands.
func (idx *LSHIndex) Bands() int {
	return idx.bands
}

// Rows returns the number of rows per band.
func (idx *LSHIndex) Rows() int {
	return idx.rows
}

func (idx *LSHIndex) addToBuckets(id string, sig *MinHashSignature) {
	keys := idx.computeBandKeys(sig)
	for _, k := range keys {
		cur := idx.buckets[k]
		exists := false
		for _, v := range cur {
			if v == id {
				exists = true
				break
			}
		}
		if !exists {
			idx.buckets[k] = append(cur, id)
		}
	}
}

func (idx *LSHIndex) computeBandKeys(sig *MinHashSignature) []string {
	total := len(sig.signatures)
	r := idx.rows
	b := idx.bands
	if r <= 0 {
		r = defaultRows
	}
	if b <= 0 {
		b = defaultBands
	}
	maxBands := total / r
	if b > maxBands {
		b = maxBands
	}
	keys := make([]string, 0, b)
	for band := 0; band < b; band++ {
		start := band * r
		end := start + r
		if end > total {
			end = total
		}
		part := sig.signatures[start:end]
		h := fnv.New64a()
		buf := make([]byte, 8)
		for _, v := range part {
			binary.BigEndian.PutUint64(buf, v)
			_, _ = h.Write(buf)
		}
		key := fmt.Sprintf("b:%d:%016x", band, h.Sum64())
		keys = append(keys, key)
	}
	return keys
}
