package analyzer

import (
	"strings"
)

// CostModel defines the interface for calculating edit operation costs
type CostModel interface {
	// Insert returns the cost of inserting a node
	Insert(node *TreeNode) float64

	// Delete returns the cost of deleting a node
	Delete(node *TreeNode) float64

	// Rename returns the cost of renaming node1 to node2
	Rename(node1, node2 *TreeNode) float64
}

// DefaultCostModel implements a uniform cost model where all operations cost 1.0
type DefaultCostModel struct{}

// NewDefaultCostModel creates a new default cost model
func NewDefaultCostModel() *DefaultCostModel {
	return &DefaultCostModel{}
}

// Insert returns the cost of inserting a node (always 1.0)
func (c *DefaultCostModel) Insert(node *TreeNode) float64 {
	return 1.0
}

// Delete returns the cost of deleting a node (always 1.0)
func (c *DefaultCostModel) Delete(node *TreeNode) float64 {
	return 1.0
}

// Rename returns the cost of renaming node1 to node2
func (c *DefaultCostModel) Rename(node1, node2 *TreeNode) float64 {
	if node1 == nil || node2 == nil {
		return 1.0
	}

	// If labels are identical, no cost for rename
	if node1.Label == node2.Label {
		return 0.0
	}

	return 1.0
}

// JavaScriptCostModel implements a JavaScript-aware cost model with different costs for different node types
type JavaScriptCostModel struct {
	// Base costs for different operations
	BaseInsertCost float64
	BaseDeleteCost float64
	BaseRenameCost float64

	// Whether to ignore differences in literal values
	IgnoreLiterals bool

	// Whether to ignore differences in identifier names
	IgnoreIdentifiers bool
}

// NewJavaScriptCostModel creates a new JavaScript-aware cost model with default settings
func NewJavaScriptCostModel() *JavaScriptCostModel {
	return &JavaScriptCostModel{
		BaseInsertCost:    1.0,
		BaseDeleteCost:    1.0,
		BaseRenameCost:    1.0,
		IgnoreLiterals:    false,
		IgnoreIdentifiers: false,
	}
}

// NewJavaScriptCostModelWithConfig creates a JavaScript cost model with custom configuration
func NewJavaScriptCostModelWithConfig(ignoreLiterals, ignoreIdentifiers bool) *JavaScriptCostModel {
	return &JavaScriptCostModel{
		BaseInsertCost:    1.0,
		BaseDeleteCost:    1.0,
		BaseRenameCost:    1.0,
		IgnoreLiterals:    ignoreLiterals,
		IgnoreIdentifiers: ignoreIdentifiers,
	}
}

// Insert returns the cost of inserting a node
func (c *JavaScriptCostModel) Insert(node *TreeNode) float64 {
	if node == nil {
		return c.BaseInsertCost
	}

	// Different costs based on node type
	multiplier := c.getNodeTypeMultiplier(node.Label)
	return c.BaseInsertCost * multiplier
}

// Delete returns the cost of deleting a node
func (c *JavaScriptCostModel) Delete(node *TreeNode) float64 {
	if node == nil {
		return c.BaseDeleteCost
	}

	// Different costs based on node type
	multiplier := c.getNodeTypeMultiplier(node.Label)
	return c.BaseDeleteCost * multiplier
}

// Rename returns the cost of renaming node1 to node2
func (c *JavaScriptCostModel) Rename(node1, node2 *TreeNode) float64 {
	if node1 == nil || node2 == nil {
		return c.BaseRenameCost
	}

	// If labels are identical, no cost
	if node1.Label == node2.Label {
		return 0.0
	}

	// Apply ignore patterns
	if c.shouldIgnoreDifference(node1.Label, node2.Label) {
		return 0.0
	}

	// Check if both nodes are of similar types
	similarity := c.calculateLabelSimilarity(node1.Label, node2.Label)

	// Scale rename cost based on similarity
	return c.BaseRenameCost * (1.0 - similarity)
}

// getNodeTypeMultiplier returns a cost multiplier based on the node type
func (c *JavaScriptCostModel) getNodeTypeMultiplier(label string) float64 {
	// Structural nodes are more expensive to modify
	if c.isStructuralNode(label) {
		return 1.5
	}

	// Control flow nodes are expensive
	if c.isControlFlowNode(label) {
		return 1.3
	}

	// Expression nodes are less expensive
	if c.isExpressionNode(label) {
		return 0.8
	}

	// Literals and identifiers can be very cheap if configured to ignore
	if c.isLiteralNode(label) && c.IgnoreLiterals {
		return 0.1
	}

	if c.isIdentifierNode(label) && c.IgnoreIdentifiers {
		return 0.2
	}

	return 1.0 // Default multiplier
}

// isStructuralNode checks if a node represents a structural element
func (c *JavaScriptCostModel) isStructuralNode(label string) bool {
	structuralNodes := []string{
		"FunctionDeclaration", "FunctionExpression", "ArrowFunctionExpression",
		"AsyncFunctionDeclaration", "GeneratorFunctionDeclaration",
		"ClassDeclaration", "ClassExpression", "MethodDefinition",
		"Program", "Module",
	}

	for _, nodeType := range structuralNodes {
		if strings.HasPrefix(label, nodeType) || strings.HasPrefix(label, "Function(") || strings.HasPrefix(label, "Class(") {
			return true
		}
	}

	return false
}

// isControlFlowNode checks if a node represents a control flow element
func (c *JavaScriptCostModel) isControlFlowNode(label string) bool {
	controlFlowNodes := []string{
		"IfStatement", "SwitchStatement", "SwitchCase",
		"ForStatement", "ForInStatement", "ForOfStatement",
		"WhileStatement", "DoWhileStatement",
		"TryStatement", "CatchClause", "FinallyClause",
		"BreakStatement", "ContinueStatement", "ReturnStatement", "ThrowStatement",
	}

	for _, nodeType := range controlFlowNodes {
		if strings.HasPrefix(label, nodeType) {
			return true
		}
	}

	return false
}

// isExpressionNode checks if a node represents an expression
func (c *JavaScriptCostModel) isExpressionNode(label string) bool {
	expressionNodes := []string{
		"BinaryExpression", "UnaryExpression", "LogicalExpression",
		"ConditionalExpression", "CallExpression", "MemberExpression",
		"AssignmentExpression", "UpdateExpression", "NewExpression",
		"ArrayExpression", "ObjectExpression", "SequenceExpression",
		"AwaitExpression", "YieldExpression", "SpreadElement",
		"TemplateLiteral",
	}

	for _, nodeType := range expressionNodes {
		if strings.HasPrefix(label, nodeType) {
			return true
		}
	}

	return false
}

// isLiteralNode checks if a node represents a literal value
func (c *JavaScriptCostModel) isLiteralNode(label string) bool {
	return strings.HasPrefix(label, "Literal(") ||
		strings.HasPrefix(label, "StringLiteral") ||
		strings.HasPrefix(label, "NumberLiteral") ||
		strings.HasPrefix(label, "BooleanLiteral") ||
		strings.HasPrefix(label, "NullLiteral") ||
		strings.HasPrefix(label, "RegExpLiteral")
}

// isIdentifierNode checks if a node represents an identifier
func (c *JavaScriptCostModel) isIdentifierNode(label string) bool {
	return strings.HasPrefix(label, "Identifier(")
}

// shouldIgnoreDifference determines if the difference between two labels should be ignored
func (c *JavaScriptCostModel) shouldIgnoreDifference(label1, label2 string) bool {
	// Ignore literal differences if configured
	if c.IgnoreLiterals && c.isLiteralNode(label1) && c.isLiteralNode(label2) {
		return true
	}

	// Ignore identifier differences if configured
	if c.IgnoreIdentifiers && c.isIdentifierNode(label1) && c.isIdentifierNode(label2) {
		return true
	}

	return false
}

// calculateLabelSimilarity calculates similarity between two node labels
func (c *JavaScriptCostModel) calculateLabelSimilarity(label1, label2 string) float64 {
	// Extract base node types (remove parenthetical content)
	baseType1 := c.extractBaseNodeType(label1)
	baseType2 := c.extractBaseNodeType(label2)

	// If base types are identical, high similarity
	if baseType1 == baseType2 {
		return 0.8
	}

	// Check for related node types
	if c.areRelatedNodeTypes(baseType1, baseType2) {
		return 0.5
	}

	// Check for same category
	if c.areSameCategory(baseType1, baseType2) {
		return 0.3
	}

	return 0.0 // No similarity
}

// extractBaseNodeType extracts the base node type from a label
func (c *JavaScriptCostModel) extractBaseNodeType(label string) string {
	if idx := strings.Index(label, "("); idx != -1 {
		return label[:idx]
	}
	return label
}

// areRelatedNodeTypes checks if two node types are related
func (c *JavaScriptCostModel) areRelatedNodeTypes(type1, type2 string) bool {
	relatedPairs := [][2]string{
		{"FunctionDeclaration", "FunctionExpression"},
		{"FunctionDeclaration", "ArrowFunctionExpression"},
		{"FunctionExpression", "ArrowFunctionExpression"},
		{"FunctionDeclaration", "AsyncFunctionDeclaration"},
		{"ClassDeclaration", "ClassExpression"},
		{"ForStatement", "ForInStatement"},
		{"ForStatement", "ForOfStatement"},
		{"ForInStatement", "ForOfStatement"},
		{"WhileStatement", "DoWhileStatement"},
		{"BinaryExpression", "UnaryExpression"},
		{"BinaryExpression", "LogicalExpression"},
		{"ArrayExpression", "ObjectExpression"},
		{"IfStatement", "ConditionalExpression"},
	}

	for _, pair := range relatedPairs {
		if (type1 == pair[0] && type2 == pair[1]) || (type1 == pair[1] && type2 == pair[0]) {
			return true
		}
	}

	return false
}

// areSameCategory checks if two node types belong to the same category
func (c *JavaScriptCostModel) areSameCategory(type1, type2 string) bool {
	if c.isStructuralNode(type1) && c.isStructuralNode(type2) {
		return true
	}

	if c.isControlFlowNode(type1) && c.isControlFlowNode(type2) {
		return true
	}

	if c.isExpressionNode(type1) && c.isExpressionNode(type2) {
		return true
	}

	return false
}

// WeightedCostModel allows custom weights for different operation types
type WeightedCostModel struct {
	InsertWeight  float64
	DeleteWeight  float64
	RenameWeight  float64
	BaseCostModel CostModel
}

// NewWeightedCostModel creates a new weighted cost model
func NewWeightedCostModel(insertWeight, deleteWeight, renameWeight float64, baseCostModel CostModel) *WeightedCostModel {
	return &WeightedCostModel{
		InsertWeight:  insertWeight,
		DeleteWeight:  deleteWeight,
		RenameWeight:  renameWeight,
		BaseCostModel: baseCostModel,
	}
}

// Insert returns the weighted cost of inserting a node
func (c *WeightedCostModel) Insert(node *TreeNode) float64 {
	return c.InsertWeight * c.BaseCostModel.Insert(node)
}

// Delete returns the weighted cost of deleting a node
func (c *WeightedCostModel) Delete(node *TreeNode) float64 {
	return c.DeleteWeight * c.BaseCostModel.Delete(node)
}

// Rename returns the weighted cost of renaming node1 to node2
func (c *WeightedCostModel) Rename(node1, node2 *TreeNode) float64 {
	return c.RenameWeight * c.BaseCostModel.Rename(node1, node2)
}
