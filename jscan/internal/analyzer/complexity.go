package analyzer

import (
	"fmt"

	"github.com/ludo-technologies/polyscan/jscan/internal/config"
	"github.com/ludo-technologies/polyscan/jscan/internal/parser"
)

// ComplexityResult holds cyclomatic complexity metrics for a function or method
type ComplexityResult struct {
	// McCabe cyclomatic complexity
	Complexity int

	// Raw CFG metrics
	Edges               int
	Nodes               int
	ConnectedComponents int

	// Function/method information
	FunctionName string
	StartLine    int
	StartCol     int
	EndLine      int

	// Nesting depth
	NestingDepth int

	// Decision points breakdown
	IfStatements      int
	LoopStatements    int
	ExceptionHandlers int
	SwitchCases       int
	LogicalOperators  int // JavaScript-specific: &&, ||, ??
	TernaryOperators  int // JavaScript-specific: ? :

	// Risk assessment based on complexity thresholds
	RiskLevel string // "low", "medium", "high"
}

// Interface methods for reporter compatibility

func (cr *ComplexityResult) GetComplexity() int {
	return cr.Complexity
}

func (cr *ComplexityResult) GetFunctionName() string {
	return cr.FunctionName
}

func (cr *ComplexityResult) GetRiskLevel() string {
	return cr.RiskLevel
}

func (cr *ComplexityResult) GetDetailedMetrics() map[string]int {
	return map[string]int{
		"nodes":              cr.Nodes,
		"edges":              cr.Edges,
		"if_statements":      cr.IfStatements,
		"loop_statements":    cr.LoopStatements,
		"exception_handlers": cr.ExceptionHandlers,
		"switch_cases":       cr.SwitchCases,
		"logical_operators":  cr.LogicalOperators,
		"ternary_operators":  cr.TernaryOperators,
	}
}

// String returns a human-readable representation of the complexity result
func (cr *ComplexityResult) String() string {
	return fmt.Sprintf("Function: %s, Complexity: %d, Risk: %s",
		cr.FunctionName, cr.Complexity, cr.RiskLevel)
}

// complexityVisitor implements CFGVisitor to count edges and nodes
type complexityVisitor struct {
	edgeCount         int
	nodeCount         int
	decisionPoints    map[*BasicBlock]int // Track decision points per block
	loopStatements    int
	exceptionHandlers int
	logicalOperators  int
	ternaryOperators  int
}

// VisitBlock counts nodes and analyzes decision points
func (cv *complexityVisitor) VisitBlock(block *BasicBlock) bool {
	if block == nil {
		return true
	}

	// Count all blocks except entry/exit for accurate complexity
	if !block.IsEntry && !block.IsExit {
		cv.nodeCount++
	}

	// Count logical operators and ternary expressions in statements
	// Skip statements that are nested function nodes — their complexity
	// is calculated in their own separate CFG.
	for _, stmt := range block.Statements {
		if isFunctionNode(stmt) {
			continue
		}
		cv.countJavaScriptComplexity(stmt)
	}

	return true
}

// isFunctionNode returns true if the node represents a function boundary
func isFunctionNode(n *parser.Node) bool {
	switch n.Type {
	case parser.NodeFunction, parser.NodeFunctionExpression, parser.NodeArrowFunction,
		parser.NodeAsyncFunction, parser.NodeGeneratorFunction, parser.NodeMethodDefinition:
		return true
	}
	return false
}

// countJavaScriptComplexity counts JavaScript-specific complexity contributors
func (cv *complexityVisitor) countJavaScriptComplexity(node *parser.Node) {
	if node == nil {
		return
	}

	// Count logical operators (&&, ||, ??)
	if node.Type == parser.NodeLogicalExpression {
		cv.logicalOperators++
	}

	// Count ternary operators (? :)
	if node.Type == parser.NodeConditionalExpression {
		cv.ternaryOperators++
	}

	// Recursively check child nodes, but stop at nested function boundaries
	// to avoid counting inner functions' operators toward the parent's complexity
	node.Walk(func(n *parser.Node) bool {
		if n != node {
			// Don't descend into nested function scopes
			if isFunctionNode(n) {
				return false
			}
			if n.Type == parser.NodeLogicalExpression {
				cv.logicalOperators++
			}
			if n.Type == parser.NodeConditionalExpression {
				cv.ternaryOperators++
			}
		}
		return true
	})
}

// VisitEdge counts edges and categorizes decision points
func (cv *complexityVisitor) VisitEdge(edge *Edge) bool {
	if edge == nil {
		return true
	}

	cv.edgeCount++

	// Count decision points accurately by source block
	// A decision point is a block with multiple outgoing edges
	if edge.From != nil {
		if cv.decisionPoints == nil {
			cv.decisionPoints = make(map[*BasicBlock]int)
		}

		switch edge.Type {
		case EdgeCondTrue, EdgeCondFalse:
			// Mark this block as having conditional edges
			// We only count the block once, regardless of number of edges
			cv.decisionPoints[edge.From] = 1
		case EdgeLoop:
			cv.loopStatements++
		case EdgeException:
			cv.exceptionHandlers++
		}
	}

	return true
}

// CalculateComplexity computes McCabe cyclomatic complexity for a CFG using default thresholds
func CalculateComplexity(cfg *CFG) *ComplexityResult {
	defaultConfig := config.DefaultConfig()
	return CalculateComplexityWithConfig(cfg, &defaultConfig.Complexity)
}

// CalculateComplexityWithConfig computes McCabe cyclomatic complexity using provided configuration
func CalculateComplexityWithConfig(cfg *CFG, complexityConfig *config.ComplexityConfig) *ComplexityResult {
	if cfg == nil {
		return &ComplexityResult{
			Complexity: 0,
			RiskLevel:  "low",
		}
	}

	visitor := &complexityVisitor{
		decisionPoints: make(map[*BasicBlock]int),
	}
	cfg.Walk(visitor)

	// Primary method: count decision points + 1
	// This is more reliable for CFGs with entry/exit nodes
	decisionPoints := countDecisionPoints(visitor)
	complexity := decisionPoints + 1

	// JavaScript-specific: Add logical operators and ternary expressions
	// These add to complexity as they create decision points
	complexity += visitor.logicalOperators
	complexity += visitor.ternaryOperators

	// Ensure minimum complexity of 1 for any function
	if complexity < 1 {
		complexity = 1
	}

	// Determine risk level based on thresholds
	riskLevel := determineRiskLevel(complexity, complexityConfig)

	result := &ComplexityResult{
		Complexity:        complexity,
		Edges:             visitor.edgeCount,
		Nodes:             visitor.nodeCount,
		IfStatements:      len(visitor.decisionPoints), // Approximation
		LoopStatements:    visitor.loopStatements,
		ExceptionHandlers: visitor.exceptionHandlers,
		LogicalOperators:  visitor.logicalOperators,
		TernaryOperators:  visitor.ternaryOperators,
		RiskLevel:         riskLevel,
		FunctionName:      cfg.Name,
	}

	if cfg.FunctionNode != nil {
		result.StartLine = cfg.FunctionNode.Location.StartLine
		result.StartCol = cfg.FunctionNode.Location.StartCol
		result.EndLine = cfg.FunctionNode.Location.EndLine
	}

	return result
}

// countDecisionPoints counts the total number of decision points
func countDecisionPoints(visitor *complexityVisitor) int {
	total := 0
	for _, count := range visitor.decisionPoints {
		total += count
	}
	return total
}

// determineRiskLevel determines the risk level based on complexity thresholds
func determineRiskLevel(complexity int, cfg *config.ComplexityConfig) string {
	if complexity > cfg.MediumThreshold {
		return "high"
	} else if complexity > cfg.LowThreshold {
		return "medium"
	}
	return "low"
}

// CalculateNestingDepth calculates the maximum nesting depth of a function
func CalculateNestingDepth(node *parser.Node) int {
	if node == nil {
		return 0
	}

	maxDepth := 0
	currentDepth := 0

	node.Walk(func(n *parser.Node) bool {
		// Increment depth for control structures
		if isControlStructure(n) {
			currentDepth++
			if currentDepth > maxDepth {
				maxDepth = currentDepth
			}
		}

		return true
	})

	return maxDepth
}

// isControlStructure checks if a node is a control structure
func isControlStructure(node *parser.Node) bool {
	switch node.Type {
	case parser.NodeIfStatement, parser.NodeSwitchStatement,
		parser.NodeForStatement, parser.NodeForInStatement, parser.NodeForOfStatement,
		parser.NodeWhileStatement, parser.NodeDoWhileStatement,
		parser.NodeTryStatement, parser.NodeCatchClause:
		return true
	}
	return false
}

// ComplexityAnalyzer analyzes complexity for multiple functions
type ComplexityAnalyzer struct {
	cfg *config.ComplexityConfig
}

// NewComplexityAnalyzer creates a new complexity analyzer
func NewComplexityAnalyzer(cfg *config.ComplexityConfig) *ComplexityAnalyzer {
	return &ComplexityAnalyzer{
		cfg: cfg,
	}
}

// AnalyzeFile analyzes complexity for all functions in a file
func (ca *ComplexityAnalyzer) AnalyzeFile(ast *parser.Node) ([]*ComplexityResult, error) {
	if ast == nil {
		return nil, fmt.Errorf("AST is nil")
	}

	// Build CFGs for all functions
	builder := NewCFGBuilder()
	cfgs, err := builder.BuildAll(ast)
	if err != nil {
		return nil, fmt.Errorf("failed to build CFGs: %w", err)
	}

	// Calculate complexity for each function
	var results []*ComplexityResult
	for _, cfg := range cfgs {
		result := CalculateComplexityWithConfig(cfg, ca.cfg)
		results = append(results, result)
	}

	return results, nil
}
