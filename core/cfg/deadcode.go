package cfg

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

	// Find unreachable blocks.
	for id := range c.Blocks {
		if !reachResult.Reachable[id] {
			result.Findings = append(result.Findings, &DeadCodeFinding{
				BlockID:  id,
				Severity: SeverityCritical,
				Reason:   "unreachable",
			})
			result.DeadBlocks++
		}
	}

	// For reachable blocks, check for code after terminators.
	if config.Classifier != nil {
		for id, block := range c.Blocks {
			if !reachResult.Reachable[id] {
				continue
			}
			reason := findTerminatorInBlock(block, config.Classifier)
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
