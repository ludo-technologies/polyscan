package cfg

// ComplexityContribution represents a single language-specific complexity contribution.
type ComplexityContribution struct {
	Count       int
	Description string // e.g. "logical_and", "ternary", "null_coalescing"
}

// ComplexityContributor provides language-specific additional complexity counts.
// For example, jscan counts logical operators (&&, ||, ??) and ternary expressions.
type ComplexityContributor interface {
	ContributeComplexity(block *BasicBlock) ([]ComplexityContribution, error)
}

// ComplexityConfig configures complexity computation.
type ComplexityConfig struct {
	// Contributor provides language-specific extra complexity contributions.
	// If nil, no extra contributions are added.
	Contributor ComplexityContributor
}

// ComplexityResult holds McCabe cyclomatic complexity analysis results.
type ComplexityResult struct {
	McCabe             int
	DecisionPoints     int
	ExtraContributions int
	Contributions      []ComplexityContribution
	EdgeBreakdown      map[EdgeType]int
}

// ComputeComplexity computes McCabe cyclomatic complexity for a CFG.
// A decision point is a block that has at least one outgoing edge of type
// EdgeCondTrue, EdgeCondFalse, EdgeLoop, or EdgeException. Each such block
// counts as exactly one decision point regardless of how many decision edges
// it has (e.g. an if-else has both EdgeCondTrue and EdgeCondFalse but is one
// decision point). McCabe = DecisionPoints + ExtraContributions + 1.
func ComputeComplexity(c *CFG, config ComplexityConfig) (*ComplexityResult, error) {
	result := &ComplexityResult{
		EdgeBreakdown: make(map[EdgeType]int),
	}

	if c == nil {
		result.McCabe = 1
		return result, nil
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

		if config.Contributor != nil {
			contributions, err := config.Contributor.ContributeComplexity(block)
			if err != nil {
				return nil, err
			}
			for _, contrib := range contributions {
				result.ExtraContributions += contrib.Count
				result.Contributions = append(result.Contributions, contrib)
			}
		}
	}

	result.McCabe = result.DecisionPoints + result.ExtraContributions + 1
	return result, nil
}
