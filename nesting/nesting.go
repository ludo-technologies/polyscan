package nesting

// NestingClassifier abstracts the language-specific logic for determining
// which AST nodes introduce nesting levels. Language adapters implement this
// to classify their own node types.
type NestingClassifier interface {
	// IsNestingNode returns true if the given node introduces a new nesting level
	// (e.g. if, for, while, with, try blocks).
	IsNestingNode(node any) bool

	// Children returns the child nodes of the given node.
	Children(node any) []any

	// Location returns the line number of the given node.
	Location(node any) int
}

// Result holds the nesting depth analysis result.
type Result struct {
	MaxDepth    int
	DeepestLine int
}

// ComputeMaxDepth computes the maximum nesting depth of the given AST tree.
// It uses the NestingClassifier to determine which nodes increase nesting depth.
func ComputeMaxDepth(root any, classifier NestingClassifier) *Result {
	if root == nil || classifier == nil {
		return &Result{MaxDepth: 0, DeepestLine: 0}
	}

	result := &Result{MaxDepth: 0, DeepestLine: 0}

	// Start depth at 0 for the root; if root is a nesting node, it starts at 1
	startDepth := 0
	if classifier.IsNestingNode(root) {
		startDepth = 1
		result.MaxDepth = 1
		result.DeepestLine = classifier.Location(root)
	}

	traverseForNesting(root, startDepth, classifier, result)
	return result
}

// traverseForNesting recursively traverses the tree to find the maximum nesting depth.
func traverseForNesting(node any, currentDepth int, classifier NestingClassifier, result *Result) {
	children := classifier.Children(node)
	for _, child := range children {
		if child == nil {
			continue
		}
		childDepth := currentDepth
		if classifier.IsNestingNode(child) {
			childDepth++
		}
		if childDepth > result.MaxDepth {
			result.MaxDepth = childDepth
			result.DeepestLine = classifier.Location(child)
		}
		traverseForNesting(child, childDepth, classifier, result)
	}
}
