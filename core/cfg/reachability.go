package cfg

// StatementClassifier classifies statements in a BasicBlock.
// Language-specific implementations check for terminator statements:
// e.g. pyscn checks parser.NodeReturn, jscan checks parser.NodeReturnStatement, etc.
type StatementClassifier interface {
	IsReturn(stmt any) bool
	IsBreak(stmt any) bool
	IsContinue(stmt any) bool
	IsThrow(stmt any) bool
}

// ReachabilityConfig configures reachability analysis.
type ReachabilityConfig struct {
	// Classifier provides language-specific statement classification.
	// If nil, only structural reachability (DFS from entry) is computed.
	Classifier StatementClassifier
}

// ReachabilityResult holds the result of reachability analysis.
type ReachabilityResult struct {
	Reachable        map[string]bool // blockID -> reachable from entry
	ReachableCount   int
	UnreachableCount int
}

// AnalyzeReachability performs reachability analysis on a CFG.
// If config.Classifier is nil, only structural reachability (DFS from entry) is
// computed, following all edges. If config.Classifier is non-nil, normal
// fallthrough edges from blocks containing a terminator
// (return/break/continue/throw) are not followed. Explicit control-transfer
// edges such as exception, return, break, and continue remain traversable.
// Pruned successors are still reachable if another path leads to them.
func AnalyzeReachability(c *CFG, config ReachabilityConfig) *ReachabilityResult {
	result := &ReachabilityResult{
		Reachable: make(map[string]bool),
	}

	if c == nil || c.Entry == nil {
		return result
	}

	visited := make(map[string]bool)
	var dfs func(block *BasicBlock)
	dfs = func(block *BasicBlock) {
		if block == nil || visited[block.ID] {
			return
		}
		visited[block.ID] = true
		result.Reachable[block.ID] = true

		// Check if this block ends with a terminator statement.
		terminates := false
		if config.Classifier != nil && len(block.Statements) > 0 {
			terminates = blockHasTerminator(block, config.Classifier)
		}

		for _, edge := range block.Successors {
			if terminates && edge.Type == EdgeNormal {
				continue
			}
			dfs(edge.To)
		}
	}

	dfs(c.Entry)

	// Count reachable and unreachable blocks.
	for id := range c.Blocks {
		if result.Reachable[id] {
			result.ReachableCount++
		} else {
			result.UnreachableCount++
		}
	}

	return result
}

// blockHasTerminator returns true if any statement in the block is a terminator.
func blockHasTerminator(block *BasicBlock, classifier StatementClassifier) bool {
	for _, stmt := range block.Statements {
		if classifier.IsReturn(stmt) || classifier.IsBreak(stmt) ||
			classifier.IsContinue(stmt) || classifier.IsThrow(stmt) {
			return true
		}
	}
	return false
}
