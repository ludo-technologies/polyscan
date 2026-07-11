package clone

import (
	"fmt"
	"hash/fnv"
	"sort"
	"strings"

	"github.com/ludo-technologies/polyscan/core/apted"
)

// SimilarityAnalyzer computes similarity between two code fragments.
type SimilarityAnalyzer interface {
	ComputeSimilarity(f1, f2 *CodeFragment) float64
	Name() string
}

// ---------------------------------------------------------------------------
// Structural similarity (APTED tree edit distance)
// ---------------------------------------------------------------------------

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

// ComputeDistanceAndSimilarity computes both APTED distance and normalized
// similarity from one distance pass.
func (s *StructuralAnalyzer) ComputeDistanceAndSimilarity(f1, f2 *CodeFragment) (float64, float64) {
	if f1 == nil || f2 == nil || f1.ASTNode == nil || f2.ASTNode == nil {
		return 0.0, 0.0
	}
	return s.analyzer.ComputeDistanceAndSimilarity(f1.ASTNode, f2.ASTNode)
}

// Name returns the name of this analyzer.
func (s *StructuralAnalyzer) Name() string {
	return "structural"
}

// ---------------------------------------------------------------------------
// Textual similarity (Type-1 gate)
// ---------------------------------------------------------------------------

// CommentStripper removes language-specific comments from source content.
// Each language adapter provides its own implementation (e.g. `//` and
// `/* */` for JS/TS, `#` for Python). A nil stripper keeps comments.
type CommentStripper func(content string) string

// TextualSimilarityAnalyzer computes textual similarity for Type-1 clone detection.
// Type-1 clones are identical code fragments except for whitespace and comments.
type TextualSimilarityAnalyzer struct {
	normalizeWhitespace bool
	stripComments       CommentStripper
}

// TextualSimilarityConfig holds configuration for textual similarity analysis.
type TextualSimilarityConfig struct {
	NormalizeWhitespace bool
	StripComments       CommentStripper
}

// NewTextualSimilarityAnalyzer creates a textual similarity analyzer with
// whitespace normalization enabled and the given language comment stripper.
func NewTextualSimilarityAnalyzer(stripComments CommentStripper) *TextualSimilarityAnalyzer {
	return &TextualSimilarityAnalyzer{
		normalizeWhitespace: true,
		stripComments:       stripComments,
	}
}

// NewTextualSimilarityAnalyzerWithConfig creates a textual similarity analyzer
// with custom configuration.
func NewTextualSimilarityAnalyzerWithConfig(config TextualSimilarityConfig) *TextualSimilarityAnalyzer {
	return &TextualSimilarityAnalyzer{
		normalizeWhitespace: config.NormalizeWhitespace,
		stripComments:       config.StripComments,
	}
}

// ComputeSimilarity computes the textual similarity between two code fragments.
// Returns 1.0 for identical content (after normalization), or a Levenshtein-based
// similarity score for near-matches.
func (t *TextualSimilarityAnalyzer) ComputeSimilarity(f1, f2 *CodeFragment) float64 {
	if f1 == nil || f2 == nil {
		return 0.0
	}

	content1 := t.NormalizeContent(f1.Content)
	content2 := t.NormalizeContent(f2.Content)

	if content1 == "" && content2 == "" {
		return 1.0 // Both empty = identical
	}
	if content1 == "" || content2 == "" {
		return 0.0 // One empty = completely different
	}

	// Quick hash comparison for identical content
	if t.hashContent(content1) == t.hashContent(content2) {
		return 1.0
	}

	// If not identical, compute string similarity using Levenshtein distance
	return t.computeLevenshteinSimilarity(content1, content2)
}

// IsExactMatch reports whether two fragments have identical source text after
// Type-1 normalization. Near matches are deliberately not treated as Type-1.
func (t *TextualSimilarityAnalyzer) IsExactMatch(f1, f2 *CodeFragment) bool {
	if f1 == nil || f2 == nil {
		return false
	}

	content1 := t.NormalizeContent(f1.Content)
	content2 := t.NormalizeContent(f2.Content)
	if content1 == "" || content2 == "" {
		return false
	}

	return content1 == content2
}

// NormalizeContent normalizes source code content for comparison: strips
// comments via the configured language stripper and collapses whitespace.
func (t *TextualSimilarityAnalyzer) NormalizeContent(content string) string {
	if content == "" {
		return ""
	}

	result := content

	if t.stripComments != nil {
		result = t.stripComments(result)
	}

	if t.normalizeWhitespace {
		result = t.normalizeWhitespaceInContent(result)
	}

	return result
}

// normalizeWhitespaceInContent normalizes code whitespace while preserving
// whitespace inside string literals, where it changes behavior.
func (t *TextualSimilarityAnalyzer) normalizeWhitespaceInContent(content string) string {
	var b strings.Builder
	b.Grow(len(content))
	var quote byte
	escaped := false
	inWhitespace := false

	for i := 0; i < len(content); i++ {
		ch := content[i]
		if quote != 0 {
			b.WriteByte(ch)
			if escaped {
				escaped = false
			} else if ch == '\\' {
				escaped = true
			} else if ch == quote {
				quote = 0
			}
			continue
		}

		if ch == '\'' || ch == '"' || ch == '`' {
			quote = ch
			inWhitespace = false
			b.WriteByte(ch)
			continue
		}
		if isSourceWhitespace(ch) {
			if !inWhitespace {
				b.WriteByte(' ')
				inWhitespace = true
			}
			continue
		}
		inWhitespace = false
		b.WriteByte(ch)
	}

	return strings.TrimSpace(b.String())
}

func isSourceWhitespace(ch byte) bool {
	switch ch {
	case ' ', '\t', '\n', '\r', '\f':
		return true
	default:
		return false
	}
}

// hashContent computes a FNV-64 hash of the content for quick equality check.
func (t *TextualSimilarityAnalyzer) hashContent(content string) uint64 {
	h := fnv.New64a()
	h.Write([]byte(content))
	return h.Sum64()
}

// HashFragmentContent returns a hex-encoded FNV-64a hash of the Type-1
// normalized content. Two fragments with the same hash are Type-1 clones of
// each other. Returns "" when the content is empty after normalization (e.g.
// the fragment was extracted without source content).
func (t *TextualSimilarityAnalyzer) HashFragmentContent(content string) string {
	normalized := t.NormalizeContent(content)
	if normalized == "" {
		return ""
	}
	return fmt.Sprintf("%016x", t.hashContent(normalized))
}

// computeLevenshteinSimilarity computes similarity based on Levenshtein distance.
// Returns a value between 0.0 and 1.0.
func (t *TextualSimilarityAnalyzer) computeLevenshteinSimilarity(s1, s2 string) float64 {
	distance := t.levenshteinDistance(s1, s2)
	maxLen := max(len(s1), len(s2))

	if maxLen == 0 {
		return 1.0
	}

	similarity := 1.0 - float64(distance)/float64(maxLen)
	if similarity < 0.0 {
		return 0.0
	}
	return similarity
}

// levenshteinDistance computes the Levenshtein edit distance between two strings.
// Uses dynamic programming with O(min(m,n)) space optimization.
func (t *TextualSimilarityAnalyzer) levenshteinDistance(s1, s2 string) int {
	// Ensure s1 is the shorter string for space optimization
	if len(s1) > len(s2) {
		s1, s2 = s2, s1
	}

	m := len(s1)
	n := len(s2)

	if m == 0 {
		return n
	}
	if n == 0 {
		return m
	}

	// Use two rows for space optimization
	prev := make([]int, m+1)
	curr := make([]int, m+1)

	for i := 0; i <= m; i++ {
		prev[i] = i
	}

	for j := 1; j <= n; j++ {
		curr[0] = j
		for i := 1; i <= m; i++ {
			cost := 0
			if s1[i-1] != s2[j-1] {
				cost = 1
			}

			curr[i] = min3(
				prev[i]+1,      // deletion
				curr[i-1]+1,    // insertion
				prev[i-1]+cost, // substitution
			)
		}
		prev, curr = curr, prev
	}

	return prev[m]
}

// Name returns the name of this analyzer.
func (t *TextualSimilarityAnalyzer) Name() string {
	return "textual"
}

// min3 returns the minimum of three integers.
func min3(a, b, c int) int {
	if a <= b && a <= c {
		return a
	}
	if b <= c {
		return b
	}
	return c
}

// ---------------------------------------------------------------------------
// Syntactic similarity (Type-2 gate)
// ---------------------------------------------------------------------------

// SyntacticSimilarityAnalyzer computes syntactic similarity using normalized
// AST hash comparison with Jaccard coefficient. This is used for Type-2 clone
// detection (syntactically identical but with different identifiers/literals).
//
// Unlike an APTED-based approach which measures tree edit distance, this
// implementation compares sets of normalized node hashes. This eliminates
// false positives from structurally similar but semantically different code,
// as only nodes with identical normalized structure contribute to similarity.
type SyntacticSimilarityAnalyzer struct {
	extractor *ASTFeatureExtractor
}

// NewSyntacticSimilarityAnalyzer creates a new syntactic similarity analyzer
// using normalized AST hash comparison that ignores identifier and literal differences.
func NewSyntacticSimilarityAnalyzer() *SyntacticSimilarityAnalyzer {
	// Use ASTFeatureExtractor with includeLiterals=false to normalize
	// identifiers and literals, focusing only on structural patterns.
	extractor := NewASTFeatureExtractor().WithOptions(
		3,     // maxSubtreeHeight
		4,     // kGramSize
		true,  // includeTypes
		false, // includeLiterals - ignore literal values for Type-2
	)
	return &SyntacticSimilarityAnalyzer{extractor: extractor}
}

// ComputeSimilarity computes the syntactic similarity between two code fragments
// using Jaccard coefficient of normalized AST hash sets. It ignores differences
// in identifier names and literal values, focusing only on the structural
// syntax pattern. Pre-computed fragment features are used when available.
func (s *SyntacticSimilarityAnalyzer) ComputeSimilarity(f1, f2 *CodeFragment) float64 {
	if f1 == nil || f2 == nil {
		return 0.0
	}

	// Use pre-computed features if available (avoids redundant tree traversal)
	if len(f1.Features) > 0 && len(f2.Features) > 0 {
		return JaccardSimilarity(f1.Features, f2.Features)
	}

	if f1.ASTNode == nil || f2.ASTNode == nil {
		return 0.0
	}

	features1, err1 := s.extractor.ExtractFeatures(f1.ASTNode)
	features2, err2 := s.extractor.ExtractFeatures(f2.ASTNode)
	if err1 != nil || err2 != nil {
		return 0.0
	}

	return JaccardSimilarity(features1, features2)
}

// ComputeDistance computes the syntactic distance between two code fragments.
// Returns 1 - similarity, so distance ranges from 0 (identical) to 1 (completely
// different). Returns 0.0 for nil inputs (no distance can be computed).
func (s *SyntacticSimilarityAnalyzer) ComputeDistance(f1, f2 *CodeFragment) float64 {
	if f1 == nil || f2 == nil {
		return 0.0
	}
	return 1.0 - s.ComputeSimilarity(f1, f2)
}

// Name returns the name of this analyzer.
func (s *SyntacticSimilarityAnalyzer) Name() string {
	return "syntactic"
}

// JaccardSimilarity computes the Jaccard coefficient between two string slices:
// Jaccard(A, B) = |A ∩ B| / |A ∪ B|. Sorted inputs (as produced by
// ASTFeatureExtractor.ExtractFeatures) are processed with an O(n+m) merge-join;
// unsorted inputs are sorted into a copy first so the result stays correct.
func JaccardSimilarity(set1, set2 []string) float64 {
	if len(set1) == 0 && len(set2) == 0 {
		return 1.0 // Both empty = identical
	}
	if len(set1) == 0 || len(set2) == 0 {
		return 0.0 // One empty = no similarity
	}

	set1 = ensureSorted(set1)
	set2 = ensureSorted(set2)

	// Merge-join on sorted slices: count unique elements and intersection
	i, j := 0, 0
	intersection := 0
	union := 0

	for i < len(set1) && j < len(set2) {
		if set1[i] == set2[j] {
			intersection++
			union++
			// Skip duplicates in both
			val := set1[i]
			for i < len(set1) && set1[i] == val {
				i++
			}
			for j < len(set2) && set2[j] == val {
				j++
			}
		} else if set1[i] < set2[j] {
			union++
			val := set1[i]
			for i < len(set1) && set1[i] == val {
				i++
			}
		} else {
			union++
			val := set2[j]
			for j < len(set2) && set2[j] == val {
				j++
			}
		}
	}
	// Count remaining unique elements
	for i < len(set1) {
		union++
		val := set1[i]
		for i < len(set1) && set1[i] == val {
			i++
		}
	}
	for j < len(set2) {
		union++
		val := set2[j]
		for j < len(set2) && set2[j] == val {
			j++
		}
	}

	if union == 0 {
		return 0.0
	}
	return float64(intersection) / float64(union)
}

func ensureSorted(values []string) []string {
	if sort.StringsAreSorted(values) {
		return values
	}
	sorted := append([]string(nil), values...)
	sort.Strings(sorted)
	return sorted
}
