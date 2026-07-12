package cfg

import (
	"sort"
	"strings"
)

// DeadCodeSeverity indicates the severity of a dead code finding.
type DeadCodeSeverity int

const (
	SeverityInfo     DeadCodeSeverity = iota // after_break, after_continue
	SeverityWarning                          // after_return, after_throw
	SeverityCritical                         // unreachable
)

// DeadCodeFinding represents a single dead code detection result.
type DeadCodeFinding struct {
	BlockID  string
	Severity DeadCodeSeverity
	Reason   string // "after_return", "after_break", "after_continue", "after_throw", "unreachable"
}

// DeadCodeResult holds all dead code findings for a CFG.
type DeadCodeResult struct {
	Findings    []*DeadCodeFinding
	TotalBlocks int
	DeadBlocks  int
}

// DeadCodeConfig configures dead code detection.
type DeadCodeConfig struct {
	// Classifier provides language-specific statement classification.
	// If nil, only structural analysis (unreachable blocks) is performed.
	Classifier StatementClassifier
}

// NoOpClassifier is an optional StatementClassifier extension reporting
// statements with no actionable content (e.g. a bare `;` in JS/TS or `pass`
// in Python). Unreachable blocks consisting solely of no-op statements are
// technically dead but carry no signal for the user, so they are not
// reported as findings.
type NoOpClassifier interface {
	IsNoOp(stmt any) bool
}

// DetectDeadCode identifies dead code in a CFG.
// It uses AnalyzeReachability to find unreachable blocks, then examines
// reachable blocks for code after terminators (return/break/continue/throw).
func DetectDeadCode(c *CFG, config DeadCodeConfig) *DeadCodeResult {
	result := &DeadCodeResult{}

	if c == nil {
		return result
	}

	result.TotalBlocks = len(c.Blocks)

	// Run reachability analysis with the classifier so that successors of
	// terminator blocks (return/break/continue/throw) are properly marked
	// unreachable when no alternative path reaches them.
	reachResult := AnalyzeReachability(c, ReachabilityConfig{Classifier: config.Classifier})

	// Iterate blocks in sorted ID order for deterministic finding order.
	blockIDs := make([]string, 0, len(c.Blocks))
	for id := range c.Blocks {
		blockIDs = append(blockIDs, id)
	}
	sort.Strings(blockIDs)

	noOpClassifier, _ := config.Classifier.(NoOpClassifier)

	// Find unreachable blocks.
	for _, id := range blockIDs {
		if reachResult.Reachable[id] {
			continue
		}
		if noOpClassifier != nil && isOnlyNoOpStatements(c.Blocks[id], noOpClassifier) {
			continue
		}
		result.Findings = append(result.Findings, &DeadCodeFinding{
			BlockID:  id,
			Severity: SeverityCritical,
			Reason:   "unreachable",
		})
		result.DeadBlocks++
	}

	// For reachable blocks, check for code after terminators.
	if config.Classifier != nil {
		for _, id := range blockIDs {
			if !reachResult.Reachable[id] {
				continue
			}
			reason := findTerminatorInBlock(c.Blocks[id], config.Classifier)
			if reason != "" {
				severity := SeverityInfo
				if reason == "after_return" || reason == "after_throw" {
					severity = SeverityWarning
				}
				result.Findings = append(result.Findings, &DeadCodeFinding{
					BlockID:  id,
					Severity: severity,
					Reason:   reason,
				})
				result.DeadBlocks++
			}
		}
	}

	return result
}

// isOnlyNoOpStatements reports whether every statement in the block is a
// no-op separator. Blocks without statements return false.
func isOnlyNoOpStatements(block *BasicBlock, classifier NoOpClassifier) bool {
	if block == nil || len(block.Statements) == 0 {
		return false
	}
	for _, stmt := range block.Statements {
		if stmt == nil || !classifier.IsNoOp(stmt) {
			return false
		}
	}
	return true
}

// findTerminatorInBlock checks if a block has statements after a terminator.
// Returns the reason string if dead code is found, empty string otherwise.
func findTerminatorInBlock(block *BasicBlock, classifier StatementClassifier) string {
	for i, stmt := range block.Statements {
		if i == len(block.Statements)-1 {
			// Last statement — no code after it, so no dead code in this block.
			break
		}
		if classifier.IsReturn(stmt) {
			return "after_return"
		}
		if classifier.IsThrow(stmt) {
			return "after_throw"
		}
		if classifier.IsBreak(stmt) {
			return "after_break"
		}
		if classifier.IsContinue(stmt) {
			return "after_continue"
		}
	}
	return ""
}

// ---------------------------------------------------------------------------
// Line-level finding post-processing
// ---------------------------------------------------------------------------

// LineFinding is the language-independent, line-level slice of a dead code
// finding used by post-processing passes. Language adapters convert their
// richer finding types to and from this representation (or embed it).
type LineFinding struct {
	StartLine   int
	EndLine     int
	Reason      string
	Severity    DeadCodeSeverity
	Description string
	Code        string
}

// SortLineFindings sorts findings by start line, then end line, for
// consistent output and as the precondition of MergeContiguousFindings.
func SortLineFindings(findings []*LineFinding) {
	sort.Slice(findings, func(i, j int) bool {
		if findings[i].StartLine != findings[j].StartLine {
			return findings[i].StartLine < findings[j].StartLine
		}
		return findings[i].EndLine < findings[j].EndLine
	})
}

// MergeContiguousFindings collapses findings whose line ranges overlap or are
// directly adjacent (no reachable line between them) and that share the same
// reason into a single finding. Findings must be pre-sorted by StartLine (then
// EndLine); use SortLineFindings. This removes the overlapping/duplicate
// ranges that arise because a compound statement's finding spans its body
// while the body's block emits its own nested finding.
func MergeContiguousFindings(findings []*LineFinding) []*LineFinding {
	if len(findings) <= 1 {
		return findings
	}

	merged := make([]*LineFinding, 0, len(findings))
	current := findings[0]

	for _, next := range findings[1:] {
		// Contiguous if the next finding starts at or before the line right after
		// the current region's end (overlapping or back-to-back lines).
		contiguous := next.StartLine <= current.EndLine+1
		if contiguous && next.Reason == current.Reason {
			if next.EndLine > current.EndLine {
				current.EndLine = next.EndLine
			}
			current.Code = mergeCodeLines(current.Code, next.Code)
			if next.Severity > current.Severity {
				current.Severity = next.Severity
				current.Description = next.Description
			}
			continue
		}
		merged = append(merged, current)
		current = next
	}
	merged = append(merged, current)

	return merged
}

// mergeCodeLines appends the lines of b to a, skipping a leading line of b that
// duplicates the trailing line of a. This keeps the merged snippet readable when
// a nested-body block repeats the line already shown by its enclosing statement.
func mergeCodeLines(a, b string) string {
	if a == "" {
		return b
	}
	if b == "" {
		return a
	}
	aLines := strings.Split(a, "\n")
	bLines := strings.Split(b, "\n")
	if len(aLines) > 0 && len(bLines) > 0 && aLines[len(aLines)-1] == bLines[0] {
		bLines = bLines[1:]
	}
	if len(bLines) == 0 {
		return a
	}
	return a + "\n" + strings.Join(bLines, "\n")
}
