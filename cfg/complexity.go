package cfg

// ComplexityContributor provides language-specific additional complexity counts.
// For example, jscan counts logical operators (&&, ||, ??) and ternary expressions.
// If a language has no extra contributors, pass nil.
type ComplexityContributor interface {
	ExtraComplexity(block *BasicBlock) int
}

// ComplexityResult holds McCabe cyclomatic complexity analysis results.
type ComplexityResult struct {
	McCabe             int
	DecisionPoints     int
	ExtraContributions int
	EdgeBreakdown      map[EdgeType]int
}

// ComputeComplexity computes McCabe cyclomatic complexity for a CFG.
// A decision point is a block that has at least one outgoing edge of type
// EdgeCondTrue, EdgeCondFalse, EdgeLoop, or EdgeException. Each such block
// counts as exactly one decision point regardless of how many decision edges
// it has (e.g. an if-else has both EdgeCondTrue and EdgeCondFalse but is one
// decision point). McCabe = DecisionPoints + ExtraContributions + 1.
func ComputeComplexity(c *CFG, contributor ComplexityContributor) *ComplexityResult {
	result := &ComplexityResult{
		EdgeBreakdown: make(map[EdgeType]int),
	}

	if c == nil {
		result.McCabe = 1
		return result
	}

	for _, block := range c.Blocks {
		isDecision := false
		for _, edge := range block.Successors {
			result.EdgeBreakdown[edge.Type]++
			switch edge.Type {
			case EdgeCondTrue, EdgeCondFalse, EdgeLoop, EdgeException:
				isDecision = true
			}
		}
		if isDecision {
			result.DecisionPoints++
		}

		if contributor != nil {
			result.ExtraContributions += contributor.ExtraComplexity(block)
		}
	}

	result.McCabe = result.DecisionPoints + result.ExtraContributions + 1
	return result
}
