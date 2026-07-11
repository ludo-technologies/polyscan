package cfg

import "testing"

func TestEdgeType_String(t *testing.T) {
	tests := []struct {
		et       EdgeType
		expected string
	}{
		{EdgeNormal, "normal"},
		{EdgeCondTrue, "true"},
		{EdgeCondFalse, "false"},
		{EdgeException, "exception"},
		{EdgeLoop, "loop"},
		{EdgeBreak, "break"},
		{EdgeContinue, "continue"},
		{EdgeReturn, "return"},
		{EdgeType(99), "unknown"},
	}

	for _, tt := range tests {
		if got := tt.et.String(); got != tt.expected {
			t.Errorf("EdgeType(%d).String() = %q, want %q", tt.et, got, tt.expected)
		}
	}
}

func TestNewBasicBlock(t *testing.T) {
	bb := NewBasicBlock("bb0")
	if bb.ID != "bb0" {
		t.Errorf("ID = %q, want %q", bb.ID, "bb0")
	}
	if len(bb.Statements) != 0 {
		t.Error("Expected empty statements")
	}
	if len(bb.Predecessors) != 0 {
		t.Error("Expected empty predecessors")
	}
	if len(bb.Successors) != 0 {
		t.Error("Expected empty successors")
	}
	if bb.IsEntry || bb.IsExit {
		t.Error("Expected non-entry, non-exit")
	}
}

func TestBasicBlock_AddStatement(t *testing.T) {
	bb := NewBasicBlock("bb0")
	bb.AddStatement("stmt1")
	bb.AddStatement(42)
	bb.AddStatement(nil) // should be ignored

	if len(bb.Statements) != 2 {
		t.Errorf("Expected 2 statements, got %d", len(bb.Statements))
	}
	if bb.Statements[0] != "stmt1" {
		t.Error("First statement mismatch")
	}
	if bb.Statements[1] != 42 {
		t.Error("Second statement mismatch")
	}
}

func TestBasicBlock_AddSuccessor(t *testing.T) {
	bb1 := NewBasicBlock("bb1")
	bb2 := NewBasicBlock("bb2")

	edge := bb1.AddSuccessor(bb2, EdgeCondTrue)

	if len(bb1.Successors) != 1 {
		t.Fatalf("Expected 1 successor, got %d", len(bb1.Successors))
	}
	if len(bb2.Predecessors) != 1 {
		t.Fatalf("Expected 1 predecessor, got %d", len(bb2.Predecessors))
	}
	if edge.From != bb1 || edge.To != bb2 || edge.Type != EdgeCondTrue {
		t.Error("Edge mismatch")
	}
}

func TestBasicBlock_RemoveSuccessor(t *testing.T) {
	bb1 := NewBasicBlock("bb1")
	bb2 := NewBasicBlock("bb2")
	bb3 := NewBasicBlock("bb3")

	bb1.AddSuccessor(bb2, EdgeNormal)
	bb1.AddSuccessor(bb3, EdgeNormal)
	bb1.RemoveSuccessor(bb2)

	if len(bb1.Successors) != 1 {
		t.Errorf("Expected 1 successor after remove, got %d", len(bb1.Successors))
	}
	if bb1.Successors[0].To != bb3 {
		t.Error("Wrong successor remaining")
	}
	if len(bb2.Predecessors) != 0 {
		t.Errorf("Expected 0 predecessors for removed, got %d", len(bb2.Predecessors))
	}
}

func TestBasicBlock_IsEmpty(t *testing.T) {
	bb := NewBasicBlock("bb0")
	if !bb.IsEmpty() {
		t.Error("Expected empty")
	}
	bb.AddStatement("x")
	if bb.IsEmpty() {
		t.Error("Expected non-empty")
	}
}

func TestBasicBlock_String(t *testing.T) {
	bb := NewBasicBlock("bb0")
	bb.Label = "test"
	if s := bb.String(); s != "[test: 0 stmts]" {
		t.Errorf("String = %q, unexpected", s)
	}

	bb.IsEntry = true
	if s := bb.String(); s != "[ENTRY: test]" {
		t.Errorf("Entry String = %q, unexpected", s)
	}

	bb.IsEntry = false
	bb.IsExit = true
	if s := bb.String(); s != "[EXIT: test]" {
		t.Errorf("Exit String = %q, unexpected", s)
	}

	// No label fallback to ID
	bb2 := NewBasicBlock("bb99")
	bb2.AddStatement("x")
	if s := bb2.String(); s != "[bb99: 1 stmts]" {
		t.Errorf("No-label String = %q, unexpected", s)
	}
}

func TestNewCFG(t *testing.T) {
	c := NewCFG("myFunc")
	if c.Name != "myFunc" {
		t.Errorf("Name = %q, want %q", c.Name, "myFunc")
	}
	if c.Entry == nil {
		t.Fatal("Entry is nil")
	}
	if c.Exit == nil {
		t.Fatal("Exit is nil")
	}
	if !c.Entry.IsEntry {
		t.Error("Entry block should have IsEntry=true")
	}
	if !c.Exit.IsExit {
		t.Error("Exit block should have IsExit=true")
	}
	// Entry + Exit = 2 blocks
	if c.Size() != 2 {
		t.Errorf("Size = %d, want 2", c.Size())
	}
}

func TestCFG_CreateBlock(t *testing.T) {
	c := NewCFG("test")
	b := c.CreateBlock("myBlock")
	if b.Label != "myBlock" {
		t.Errorf("Label = %q, want %q", b.Label, "myBlock")
	}
	// Entry + Exit + myBlock = 3
	if c.Size() != 3 {
		t.Errorf("Size = %d, want 3", c.Size())
	}
}

func TestCFG_AddBlock(t *testing.T) {
	c := NewCFG("test")
	b := NewBasicBlock("external")
	c.AddBlock(b)
	if c.GetBlock("external") != b {
		t.Error("AddBlock did not register block")
	}

	c.AddBlock(nil) // should not panic
}

func TestCFG_RemoveBlock(t *testing.T) {
	c := NewCFG("test")
	b := c.CreateBlock("removable")

	// Connect: Entry -> b -> Exit
	c.ConnectBlocks(c.Entry, b, EdgeNormal)
	c.ConnectBlocks(b, c.Exit, EdgeNormal)

	c.RemoveBlock(b)
	if c.GetBlock(b.ID) != nil {
		t.Error("Block should be removed")
	}

	// Cannot remove entry/exit
	sizeBefore := c.Size()
	c.RemoveBlock(c.Entry)
	c.RemoveBlock(c.Exit)
	c.RemoveBlock(nil)
	if c.Size() != sizeBefore {
		t.Error("Should not remove entry/exit/nil")
	}
}

func TestCFG_ConnectBlocks(t *testing.T) {
	c := NewCFG("test")
	b := c.CreateBlock("b")

	edge := c.ConnectBlocks(c.Entry, b, EdgeNormal)
	if edge == nil {
		t.Fatal("Expected non-nil edge")
	}
	if edge.From != c.Entry || edge.To != b {
		t.Error("Edge endpoints mismatch")
	}

	// Nil handling
	if e := c.ConnectBlocks(nil, b, EdgeNormal); e != nil {
		t.Error("Expected nil for nil source")
	}
	if e := c.ConnectBlocks(c.Entry, nil, EdgeNormal); e != nil {
		t.Error("Expected nil for nil target")
	}
}

func TestCFG_GetBlock(t *testing.T) {
	c := NewCFG("test")
	b := c.CreateBlock("b")

	if c.GetBlock(b.ID) != b {
		t.Error("GetBlock should return the block")
	}
	if c.GetBlock("nonexistent") != nil {
		t.Error("Expected nil for nonexistent")
	}
}

func TestCFG_String(t *testing.T) {
	c := NewCFG("foo")
	s := c.String()
	if s != "CFG(foo): 2 blocks" {
		t.Errorf("String = %q, unexpected", s)
	}
}

type testVisitor struct {
	blocks []string
	edges  []EdgeType
	stop   bool
}

func (v *testVisitor) VisitBlock(block *BasicBlock) bool {
	v.blocks = append(v.blocks, block.ID)
	return !v.stop
}

func (v *testVisitor) VisitEdge(edge *Edge) bool {
	v.edges = append(v.edges, edge.Type)
	return !v.stop
}

func TestCFG_Walk(t *testing.T) {
	// Entry -> A -> B -> Exit
	c := NewCFG("test")
	a := c.CreateBlock("A")
	b := c.CreateBlock("B")
	c.ConnectBlocks(c.Entry, a, EdgeNormal)
	c.ConnectBlocks(a, b, EdgeNormal)
	c.ConnectBlocks(b, c.Exit, EdgeNormal)

	v := &testVisitor{}
	c.Walk(v)

	if len(v.blocks) != 4 {
		t.Errorf("Walk visited %d blocks, want 4", len(v.blocks))
	}
}

func TestCFG_Walk_StopEarly(t *testing.T) {
	c := NewCFG("test")
	a := c.CreateBlock("A")
	c.ConnectBlocks(c.Entry, a, EdgeNormal)
	c.ConnectBlocks(a, c.Exit, EdgeNormal)

	v := &testVisitor{stop: true}
	c.Walk(v)
	// Should stop after first block
	if len(v.blocks) != 1 {
		t.Errorf("Early stop: visited %d blocks, want 1", len(v.blocks))
	}
}

func TestCFG_Walk_NilEntry(t *testing.T) {
	c := &CFG{Blocks: make(map[string]*BasicBlock)}
	v := &testVisitor{}
	c.Walk(v) // should not panic
	if len(v.blocks) != 0 {
		t.Error("Expected no blocks visited for nil entry")
	}
}

func TestCFG_BreadthFirstWalk(t *testing.T) {
	// Entry -> {A, B} -> Exit
	c := NewCFG("test")
	a := c.CreateBlock("A")
	b := c.CreateBlock("B")
	c.ConnectBlocks(c.Entry, a, EdgeCondTrue)
	c.ConnectBlocks(c.Entry, b, EdgeCondFalse)
	c.ConnectBlocks(a, c.Exit, EdgeNormal)
	c.ConnectBlocks(b, c.Exit, EdgeNormal)

	v := &testVisitor{}
	c.BreadthFirstWalk(v)

	if len(v.blocks) != 4 {
		t.Errorf("BFS visited %d blocks, want 4", len(v.blocks))
	}
	// Entry should be first in BFS
	if v.blocks[0] != c.Entry.ID {
		t.Error("BFS should visit Entry first")
	}
}

func TestCFG_BreadthFirstWalk_StopEarly(t *testing.T) {
	c := NewCFG("test")
	a := c.CreateBlock("A")
	c.ConnectBlocks(c.Entry, a, EdgeNormal)

	v := &testVisitor{stop: true}
	c.BreadthFirstWalk(v)
	if len(v.blocks) != 1 {
		t.Errorf("BFS early stop: visited %d blocks, want 1", len(v.blocks))
	}
}

func TestCFG_BreadthFirstWalk_NilEntry(t *testing.T) {
	c := &CFG{Blocks: make(map[string]*BasicBlock)}
	v := &testVisitor{}
	c.BreadthFirstWalk(v) // should not panic
}

func TestCFG_CyclicGraph(t *testing.T) {
	// Entry -> A -> B -> A (loop) and B -> Exit
	c := NewCFG("test")
	a := c.CreateBlock("A")
	b := c.CreateBlock("B")
	c.ConnectBlocks(c.Entry, a, EdgeNormal)
	c.ConnectBlocks(a, b, EdgeNormal)
	c.ConnectBlocks(b, a, EdgeLoop) // back edge
	c.ConnectBlocks(b, c.Exit, EdgeNormal)

	// DFS should not loop infinitely
	v := &testVisitor{}
	c.Walk(v)
	if len(v.blocks) != 4 {
		t.Errorf("Cyclic DFS visited %d blocks, want 4", len(v.blocks))
	}

	// BFS should not loop infinitely
	v2 := &testVisitor{}
	c.BreadthFirstWalk(v2)
	if len(v2.blocks) != 4 {
		t.Errorf("Cyclic BFS visited %d blocks, want 4", len(v2.blocks))
	}
}

func TestBasicBlock_StatementAnyType(t *testing.T) {
	bb := NewBasicBlock("bb0")
	// Statements are []any — verify various types work
	bb.AddStatement("string stmt")
	bb.AddStatement(42)

	type customNode struct{ Kind string }
	bb.AddStatement(&customNode{Kind: "if"})

	if len(bb.Statements) != 3 {
		t.Errorf("Expected 3 statements, got %d", len(bb.Statements))
	}
	if n, ok := bb.Statements[2].(*customNode); !ok || n.Kind != "if" {
		t.Error("Expected customNode as third statement")
	}
}

func TestCFG_FunctionNode_Any(t *testing.T) {
	c := NewCFG("test")
	type myAST struct{ Name string }
	c.FunctionNode = &myAST{Name: "foo"}

	if ast, ok := c.FunctionNode.(*myAST); !ok || ast.Name != "foo" {
		t.Error("FunctionNode should hold custom type")
	}
}
