package clone

import (
	"fmt"
	"hash/fnv"
	"sort"
	"strings"

	"github.com/ludo-technologies/codescan-core/apted"
)

// FeatureExtractor converts AST trees into feature sets for Jaccard similarity.
type FeatureExtractor interface {
	ExtractFeatures(ast *apted.TreeNode) ([]string, error)
	ExtractSubtreeHashes(ast *apted.TreeNode, maxHeight int) []string
	ExtractNodeSequences(ast *apted.TreeNode, k int) []string
}

// ASTFeatureExtractor implements FeatureExtractor for TreeNode.
type ASTFeatureExtractor struct {
	maxSubtreeHeight int
	kGramSize        int
	includeTypes     bool
	includeLiterals  bool

	// PatternNames is the list of node type names to check for structural patterns.
	// Language-specific: set this to match your AST node types.
	// If nil, no pattern features are extracted.
	PatternNames []string
}

// NewASTFeatureExtractor creates a feature extractor with sensible defaults.
func NewASTFeatureExtractor() *ASTFeatureExtractor {
	return &ASTFeatureExtractor{
		maxSubtreeHeight: 3,
		kGramSize:        4,
		includeTypes:     true,
		includeLiterals:  false,
	}
}

// WithOptions allows overriding defaults.
func (a *ASTFeatureExtractor) WithOptions(maxHeight, k int, includeTypes, includeLiterals bool) *ASTFeatureExtractor {
	if maxHeight > 0 {
		a.maxSubtreeHeight = maxHeight
	}
	if k > 0 {
		a.kGramSize = k
	}
	a.includeTypes = includeTypes
	a.includeLiterals = includeLiterals
	return a
}

// WithPatterns sets the pattern names for structural feature extraction.
func (a *ASTFeatureExtractor) WithPatterns(patterns []string) *ASTFeatureExtractor {
	a.PatternNames = patterns
	return a
}

// ExtractFeatures builds a mixed set of features from the tree.
func (a *ASTFeatureExtractor) ExtractFeatures(ast *apted.TreeNode) ([]string, error) {
	if ast == nil {
		return []string{}, nil
	}

	features := make(map[string]struct{})

	for _, f := range a.ExtractSubtreeHashes(ast, a.maxSubtreeHeight) {
		features[f] = struct{}{}
	}

	for _, f := range a.ExtractNodeSequences(ast, a.kGramSize) {
		features["kgram:"+f] = struct{}{}
	}

	typeCounts := make(map[string]int)
	preorder := a.preorderLabels(ast)
	for _, lbl := range preorder {
		base := a.baseType(lbl)
		if a.includeTypes && base != "" {
			typeCounts[base]++
			features["type:"+base] = struct{}{}
		}
	}
	for t, c := range typeCounts {
		bin := binCount(c)
		features[fmt.Sprintf("typedist:%s:%s", t, bin)] = struct{}{}
	}

	for _, p := range a.extractPatterns(ast) {
		features["pattern:"+p] = struct{}{}
	}

	out := make([]string, 0, len(features))
	for f := range features {
		out = append(out, f)
	}
	sort.Strings(out)
	return out, nil
}

// ExtractSubtreeHashes computes bottom-up hashes of subtrees up to maxHeight.
func (a *ASTFeatureExtractor) ExtractSubtreeHashes(ast *apted.TreeNode, maxHeight int) []string {
	if ast == nil {
		return []string{}
	}
	var feats []string
	var dfs func(n *apted.TreeNode) (uint64, int)
	dfs = func(n *apted.TreeNode) (uint64, int) {
		if n == nil {
			return 0, -1
		}
		childHashes := make([]uint64, 0, len(n.Children))
		maxH := -1
		for _, ch := range n.Children {
			h, height := dfs(ch)
			childHashes = append(childHashes, h)
			if height > maxH {
				maxH = height
			}
		}
		height := maxH + 1
		h := fnv.New64a()
		_, _ = h.Write([]byte(a.canonicalLabel(n.Label)))
		for _, ch := range childHashes {
			var b [8]byte
			for i := 0; i < 8; i++ {
				b[7-i] = byte(ch >> (uint(8 * i)))
			}
			_, _ = h.Write(b[:])
		}
		hv := h.Sum64()
		if height <= maxHeight {
			feats = append(feats, fmt.Sprintf("sub:%d:%016x", height, hv))
		}
		return hv, height
	}
	_, _ = dfs(ast)
	return feats
}

// ExtractNodeSequences returns k-grams from pre-order traversal labels.
func (a *ASTFeatureExtractor) ExtractNodeSequences(ast *apted.TreeNode, k int) []string {
	if ast == nil || k <= 1 {
		return []string{}
	}
	labels := a.preorderLabels(ast)
	if len(labels) < k {
		return []string{}
	}
	grams := make([]string, 0, len(labels)-k+1)
	for i := 0; i <= len(labels)-k; i++ {
		grams = append(grams, strings.Join(labels[i:i+k], ":"))
	}
	return grams
}

func (a *ASTFeatureExtractor) canonicalLabel(lbl string) string {
	if a.includeLiterals {
		return lbl
	}
	if idx := strings.IndexByte(lbl, '('); idx >= 0 {
		return lbl[:idx]
	}
	return lbl
}

func (a *ASTFeatureExtractor) baseType(lbl string) string {
	return a.canonicalLabel(lbl)
}

func (a *ASTFeatureExtractor) preorderLabels(ast *apted.TreeNode) []string {
	labels := []string{}
	var walk func(n *apted.TreeNode)
	walk = func(n *apted.TreeNode) {
		if n == nil {
			return
		}
		labels = append(labels, a.canonicalLabel(n.Label))
		for _, ch := range n.Children {
			walk(ch)
		}
	}
	walk(ast)
	return labels
}

func binCount(c int) string {
	switch {
	case c <= 1:
		return "1"
	case c <= 3:
		return "2-3"
	case c <= 7:
		return "4-7"
	case c <= 15:
		return "8-15"
	default:
		return "16+"
	}
}

func (a *ASTFeatureExtractor) extractPatterns(ast *apted.TreeNode) []string {
	if len(a.PatternNames) == 0 {
		return []string{}
	}
	counts := make(map[string]int)
	var walk func(n *apted.TreeNode)
	walk = func(n *apted.TreeNode) {
		if n == nil {
			return
		}
		b := a.baseType(n.Label)
		counts[b]++
		for _, ch := range n.Children {
			walk(ch)
		}
	}
	walk(ast)

	pats := []string{}
	for _, name := range a.PatternNames {
		if counts[name] > 0 {
			pats = append(pats, name)
		}
	}
	sort.Strings(pats)
	return pats
}
