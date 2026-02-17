package nesting

import "testing"

type testNode struct {
	isNesting bool
	line      int
	children  []*testNode
}

type testClassifier struct{}

func (c *testClassifier) IsNestingNode(node any) bool {
	if n, ok := node.(*testNode); ok {
		return n.isNesting
	}
	return false
}

func (c *testClassifier) Children(node any) []any {
	if n, ok := node.(*testNode); ok {
		result := make([]any, len(n.children))
		for i, child := range n.children {
			result[i] = child
		}
		return result
	}
	return nil
}

func (c *testClassifier) Location(node any) int {
	if n, ok := node.(*testNode); ok {
		return n.line
	}
	return 0
}

func TestNilRoot(t *testing.T) {
	cls := &testClassifier{}
	result := ComputeMaxDepth(nil, cls)
	if result.MaxDepth != 0 {
		t.Errorf("expected MaxDepth 0, got %d", result.MaxDepth)
	}
	if result.DeepestLine != 0 {
		t.Errorf("expected DeepestLine 0, got %d", result.DeepestLine)
	}
}

func TestNilClassifier(t *testing.T) {
	root := &testNode{isNesting: false, line: 1}
	result := ComputeMaxDepth(root, nil)
	if result.MaxDepth != 0 {
		t.Errorf("expected MaxDepth 0, got %d", result.MaxDepth)
	}
	if result.DeepestLine != 0 {
		t.Errorf("expected DeepestLine 0, got %d", result.DeepestLine)
	}
}

func TestSingleNonNestingNode(t *testing.T) {
	cls := &testClassifier{}
	root := &testNode{isNesting: false, line: 1}
	result := ComputeMaxDepth(root, cls)
	if result.MaxDepth != 0 {
		t.Errorf("expected MaxDepth 0, got %d", result.MaxDepth)
	}
}

func TestSingleNestingNode(t *testing.T) {
	cls := &testClassifier{}
	root := &testNode{isNesting: true, line: 5}
	result := ComputeMaxDepth(root, cls)
	if result.MaxDepth != 1 {
		t.Errorf("expected MaxDepth 1, got %d", result.MaxDepth)
	}
	if result.DeepestLine != 5 {
		t.Errorf("expected DeepestLine 5, got %d", result.DeepestLine)
	}
}

func TestDepthTwo(t *testing.T) {
	cls := &testClassifier{}
	inner := &testNode{isNesting: true, line: 10}
	root := &testNode{isNesting: true, line: 5, children: []*testNode{inner}}
	result := ComputeMaxDepth(root, cls)
	if result.MaxDepth != 2 {
		t.Errorf("expected MaxDepth 2, got %d", result.MaxDepth)
	}
	if result.DeepestLine != 10 {
		t.Errorf("expected DeepestLine 10, got %d", result.DeepestLine)
	}
}

func TestDepthThree(t *testing.T) {
	cls := &testClassifier{}
	innermost := &testNode{isNesting: true, line: 15}
	middle := &testNode{isNesting: true, line: 10, children: []*testNode{innermost}}
	root := &testNode{isNesting: true, line: 5, children: []*testNode{middle}}
	result := ComputeMaxDepth(root, cls)
	if result.MaxDepth != 3 {
		t.Errorf("expected MaxDepth 3, got %d", result.MaxDepth)
	}
	if result.DeepestLine != 15 {
		t.Errorf("expected DeepestLine 15, got %d", result.DeepestLine)
	}
}

func TestNonNestingChildrenDontIncrease(t *testing.T) {
	cls := &testClassifier{}
	child1 := &testNode{isNesting: false, line: 10}
	child2 := &testNode{isNesting: false, line: 15}
	root := &testNode{isNesting: true, line: 5, children: []*testNode{child1, child2}}
	result := ComputeMaxDepth(root, cls)
	if result.MaxDepth != 1 {
		t.Errorf("expected MaxDepth 1, got %d", result.MaxDepth)
	}
	if result.DeepestLine != 5 {
		t.Errorf("expected DeepestLine 5, got %d", result.DeepestLine)
	}
}

func TestMixedNestingAndNonNesting(t *testing.T) {
	cls := &testClassifier{}
	// Build a tree:
	//   root (nesting, line 1)
	//     ├── child1 (non-nesting, line 2)
	//     │     └── grandchild1 (nesting, line 3)  -> depth 2
	//     └── child2 (nesting, line 4)              -> depth 2
	//           └── grandchild2 (nesting, line 5)   -> depth 3
	grandchild1 := &testNode{isNesting: true, line: 3}
	grandchild2 := &testNode{isNesting: true, line: 5}
	child1 := &testNode{isNesting: false, line: 2, children: []*testNode{grandchild1}}
	child2 := &testNode{isNesting: true, line: 4, children: []*testNode{grandchild2}}
	root := &testNode{isNesting: true, line: 1, children: []*testNode{child1, child2}}
	result := ComputeMaxDepth(root, cls)
	if result.MaxDepth != 3 {
		t.Errorf("expected MaxDepth 3, got %d", result.MaxDepth)
	}
	if result.DeepestLine != 5 {
		t.Errorf("expected DeepestLine 5, got %d", result.DeepestLine)
	}
}
