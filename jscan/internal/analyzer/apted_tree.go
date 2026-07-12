package analyzer

import (
	"fmt"

	"github.com/ludo-technologies/polyscan/jscan/internal/parser"
)

// TreeNode represents a node in the ordered tree for APTED algorithm
type TreeNode struct {
	// Unique identifier for this node
	ID int

	// Label for the node (typically the node type or value)
	Label string

	// Tree structure
	Children []*TreeNode
	Parent   *TreeNode

	// APTED-specific fields for optimization
	PostOrderID  int  // Post-order traversal position
	LeftMostLeaf int  // Left-most leaf descendant
	KeyRoot      bool // Whether this node is a key root

	// Optional metadata from original AST
	OriginalNode *parser.Node
}

// NewTreeNode creates a new tree node with the given ID and label
func NewTreeNode(id int, label string) *TreeNode {
	return &TreeNode{
		ID:       id,
		Label:    label,
		Children: []*TreeNode{},
	}
}

// AddChild adds a child node to this node
func (t *TreeNode) AddChild(child *TreeNode) {
	if child != nil {
		child.Parent = t
		t.Children = append(t.Children, child)
	}
}

// IsLeaf returns true if this node has no children
func (t *TreeNode) IsLeaf() bool {
	return len(t.Children) == 0
}

// Size returns the size of the subtree rooted at this node
func (t *TreeNode) Size() int {
	return t.SizeWithDepthLimit(1000) // Default recursion limit
}

// SizeWithDepthLimit returns the size with maximum recursion depth limit
func (t *TreeNode) SizeWithDepthLimit(maxDepth int) int {
	if maxDepth <= 0 {
		return 1 // Return 1 to avoid infinite loops, treat as leaf
	}

	size := 1
	for _, child := range t.Children {
		size += child.SizeWithDepthLimit(maxDepth - 1)
	}
	return size
}

// Height returns the height of the subtree rooted at this node
func (t *TreeNode) Height() int {
	return t.HeightWithDepthLimit(1000) // Default recursion limit
}

// HeightWithDepthLimit returns the height with maximum recursion depth limit
func (t *TreeNode) HeightWithDepthLimit(maxDepth int) int {
	if maxDepth <= 0 {
		return 0 // Treat as leaf when depth limit reached
	}

	if t.IsLeaf() {
		return 0
	}

	maxHeight := 0
	for _, child := range t.Children {
		if h := child.HeightWithDepthLimit(maxDepth - 1); h > maxHeight {
			maxHeight = h
		}
	}
	return maxHeight + 1
}

// String returns a string representation of the node
func (t *TreeNode) String() string {
	return fmt.Sprintf("Node{ID: %d, Label: %s, Children: %d}", t.ID, t.Label, len(t.Children))
}

// TreeConverter converts parser AST nodes to APTED tree nodes
type TreeConverter struct {
	nextID int
}

// NewTreeConverter creates a new tree converter
func NewTreeConverter() *TreeConverter {
	return &TreeConverter{nextID: 0}
}

// ConvertAST converts a parser AST node to an APTED tree
func (tc *TreeConverter) ConvertAST(astNode *parser.Node) *TreeNode {
	if astNode == nil {
		return nil
	}

	// Create tree node with simplified label
	label := tc.getNodeLabel(astNode)
	treeNode := NewTreeNode(tc.nextID, label)
	tc.nextID++

	// Store reference to original AST node
	treeNode.OriginalNode = astNode

	for _, child := range parser.OrderedChildren(astNode) {
		if childNode := tc.ConvertAST(child); childNode != nil {
			treeNode.AddChild(childNode)
		}
	}

	return treeNode
}

// getNodeLabel extracts a meaningful label from the AST node
func (tc *TreeConverter) getNodeLabel(astNode *parser.Node) string {
	// Use the node type as the primary label
	label := string(astNode.Type)

	// For some node types, include additional information
	switch astNode.Type {
	case parser.NodeIdentifier:
		if astNode.Name != "" {
			label = fmt.Sprintf("Identifier(%s)", astNode.Name)
		}
	case parser.NodeLiteral, parser.NodeStringLiteral, parser.NodeNumberLiteral:
		if astNode.Value != nil {
			label = fmt.Sprintf("Literal(%v)", astNode.Value)
		}
	case parser.NodeFunction, parser.NodeAsyncFunction, parser.NodeArrowFunction:
		if astNode.Name != "" {
			label = fmt.Sprintf("Function(%s)", astNode.Name)
		}
	case parser.NodeClass, parser.NodeClassExpression:
		if astNode.Name != "" {
			label = fmt.Sprintf("Class(%s)", astNode.Name)
		}
	case parser.NodeBinaryExpression, parser.NodeUnaryExpression, parser.NodeLogicalExpression,
		parser.NodeAssignmentExpression, parser.NodeUpdateExpression:
		if astNode.Operator != "" {
			label = fmt.Sprintf("%s(%s)", astNode.Type, astNode.Operator)
		}
	case parser.NodeVariableDeclaration:
		if astNode.Kind != "" {
			label = fmt.Sprintf("VariableDeclaration(%s)", astNode.Kind)
		}
	}

	return label
}

// PostOrderTraversal performs post-order traversal and assigns post-order IDs
func PostOrderTraversal(root *TreeNode) {
	if root == nil {
		return
	}

	postOrderID := 0
	postOrderTraversalRecursive(root, &postOrderID)
}

// postOrderTraversalRecursive recursively performs post-order traversal
func postOrderTraversalRecursive(node *TreeNode, postOrderID *int) {
	if node == nil {
		return
	}

	// Visit children first
	for _, child := range node.Children {
		postOrderTraversalRecursive(child, postOrderID)
	}

	// Then visit this node
	node.PostOrderID = *postOrderID
	*postOrderID++
}

// ComputeLeftMostLeaves computes left-most leaf descendants for all nodes
func ComputeLeftMostLeaves(root *TreeNode) {
	if root == nil {
		return
	}
	computeLeftMostLeavesRecursive(root)
}

// computeLeftMostLeavesRecursive recursively computes left-most leaf descendants
func computeLeftMostLeavesRecursive(node *TreeNode) int {
	if node.IsLeaf() || len(node.Children) == 0 {
		node.LeftMostLeaf = node.PostOrderID
		return node.LeftMostLeaf
	}

	// Get left-most leaf from first child
	leftMostLeaf := computeLeftMostLeavesRecursive(node.Children[0])
	node.LeftMostLeaf = leftMostLeaf

	// Process remaining children
	for i := 1; i < len(node.Children); i++ {
		computeLeftMostLeavesRecursive(node.Children[i])
	}

	return leftMostLeaf
}

// ComputeKeyRoots identifies key roots for path decomposition
func ComputeKeyRoots(root *TreeNode) []int {
	if root == nil {
		return []int{}
	}

	keyRoots := []int{}
	visited := make(map[int]bool)

	computeKeyRootsRecursive(root, &keyRoots, visited)

	return keyRoots
}

// computeKeyRootsRecursive recursively identifies key roots
func computeKeyRootsRecursive(node *TreeNode, keyRoots *[]int, visited map[int]bool) {
	if node == nil {
		return
	}

	// A node is a key root if its left-most leaf hasn't been visited
	if !visited[node.LeftMostLeaf] {
		node.KeyRoot = true
		*keyRoots = append(*keyRoots, node.PostOrderID)
		visited[node.LeftMostLeaf] = true
	}

	// Process children
	for _, child := range node.Children {
		computeKeyRootsRecursive(child, keyRoots, visited)
	}
}

// PrepareTreeForAPTED prepares a tree for APTED algorithm by computing all necessary indices
func PrepareTreeForAPTED(root *TreeNode) []int {
	if root == nil {
		return []int{}
	}

	// Step 1: Assign post-order IDs
	PostOrderTraversal(root)

	// Step 2: Compute left-most leaf descendants
	ComputeLeftMostLeaves(root)

	// Step 3: Identify key roots
	keyRoots := ComputeKeyRoots(root)

	return keyRoots
}

// GetNodeByPostOrderID finds a node by its post-order ID
func GetNodeByPostOrderID(root *TreeNode, postOrderID int) *TreeNode {
	if root == nil {
		return nil
	}

	if root.PostOrderID == postOrderID {
		return root
	}

	for _, child := range root.Children {
		if node := GetNodeByPostOrderID(child, postOrderID); node != nil {
			return node
		}
	}

	return nil
}

// GetSubtreeNodes returns all nodes in the subtree rooted at the given node
func GetSubtreeNodes(root *TreeNode) []*TreeNode {
	return GetSubtreeNodesWithDepthLimit(root, 1000) // Default recursion limit
}

// GetSubtreeNodesWithDepthLimit returns all nodes with maximum recursion depth limit
func GetSubtreeNodesWithDepthLimit(root *TreeNode, maxDepth int) []*TreeNode {
	if root == nil || maxDepth <= 0 {
		return []*TreeNode{}
	}

	nodes := []*TreeNode{root}
	for _, child := range root.Children {
		nodes = append(nodes, GetSubtreeNodesWithDepthLimit(child, maxDepth-1)...)
	}

	return nodes
}
