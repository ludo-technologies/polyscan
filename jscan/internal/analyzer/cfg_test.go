package analyzer

import (
	"testing"

	"github.com/ludo-technologies/polyscan/jscan/internal/parser"
)

func TestEdgeType_String(t *testing.T) {
	tests := []struct {
		edgeType EdgeType
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
		{EdgeType(100), "unknown"},
	}

	for _, tc := range tests {
		result := tc.edgeType.String()
		if result != tc.expected {
			t.Errorf("EdgeType(%d).String() = %s, expected %s", tc.edgeType, result, tc.expected)
		}
	}
}

func TestNewBasicBlock(t *testing.T) {
	block := NewBasicBlock("test_block")

	if block.ID != "test_block" {
		t.Errorf("Expected ID 'test_block', got %s", block.ID)
	}
	if len(block.Statements) != 0 {
		t.Errorf("Expected empty statements, got %d", len(block.Statements))
	}
	if len(block.Predecessors) != 0 {
		t.Errorf("Expected empty predecessors, got %d", len(block.Predecessors))
	}
	if len(block.Successors) != 0 {
		t.Errorf("Expected empty successors, got %d", len(block.Successors))
	}
	if block.IsEntry {
		t.Error("Expected IsEntry to be false")
	}
	if block.IsExit {
		t.Error("Expected IsExit to be false")
	}
}

func TestBasicBlock_AddStatement(t *testing.T) {
	block := NewBasicBlock("test")

	// Add nil statement - should be ignored
	block.AddStatement(nil)
	if len(block.Statements) != 0 {
		t.Error("Nil statement should be ignored")
	}

	// Add valid statement
	stmt := &parser.Node{Type: parser.NodeExpressionStatement}
	block.AddStatement(stmt)
	if len(block.Statements) != 1 {
		t.Errorf("Expected 1 statement, got %d", len(block.Statements))
	}
	if block.Statements[0] != stmt {
		t.Error("Statement not added correctly")
	}
}

func TestBasicBlock_AddSuccessor(t *testing.T) {
	block1 := NewBasicBlock("block1")
	block2 := NewBasicBlock("block2")

	edge := block1.AddSuccessor(block2, EdgeNormal)

	// Check edge properties
	if edge.From != block1 {
		t.Error("Edge From should be block1")
	}
	if edge.To != block2 {
		t.Error("Edge To should be block2")
	}
	if edge.Type != EdgeNormal {
		t.Errorf("Edge type should be EdgeNormal, got %s", edge.Type)
	}

	// Check block1 successors
	if len(block1.Successors) != 1 {
		t.Errorf("block1 should have 1 successor, got %d", len(block1.Successors))
	}

	// Check block2 predecessors
	if len(block2.Predecessors) != 1 {
		t.Errorf("block2 should have 1 predecessor, got %d", len(block2.Predecessors))
	}
}

func TestBasicBlock_RemoveSuccessor(t *testing.T) {
	block1 := NewBasicBlock("block1")
	block2 := NewBasicBlock("block2")
	block3 := NewBasicBlock("block3")

	block1.AddSuccessor(block2, EdgeNormal)
	block1.AddSuccessor(block3, EdgeCondTrue)

	// Remove block2 as successor
	block1.RemoveSuccessor(block2)

	if len(block1.Successors) != 1 {
		t.Errorf("block1 should have 1 successor after removal, got %d", len(block1.Successors))
	}
	if block1.Successors[0].To != block3 {
		t.Error("Remaining successor should be block3")
	}
	if len(block2.Predecessors) != 0 {
		t.Errorf("block2 should have no predecessors after removal, got %d", len(block2.Predecessors))
	}
}

func TestBasicBlock_IsEmpty(t *testing.T) {
	block := NewBasicBlock("test")

	if !block.IsEmpty() {
		t.Error("New block should be empty")
	}

	block.AddStatement(&parser.Node{Type: parser.NodeExpressionStatement})

	if block.IsEmpty() {
		t.Error("Block with statements should not be empty")
	}
}

func TestBasicBlock_String(t *testing.T) {
	// Entry block
	entry := NewBasicBlock("bb0")
	entry.IsEntry = true
	entry.Label = "ENTRY"
	if entry.String() != "[ENTRY: ENTRY]" {
		t.Errorf("Entry block string incorrect: %s", entry.String())
	}

	// Exit block
	exit := NewBasicBlock("bb1")
	exit.IsExit = true
	exit.Label = "EXIT"
	if exit.String() != "[EXIT: EXIT]" {
		t.Errorf("Exit block string incorrect: %s", exit.String())
	}

	// Regular block with label
	regular := NewBasicBlock("bb2")
	regular.Label = "test_label"
	regular.AddStatement(&parser.Node{Type: parser.NodeExpressionStatement})
	regular.AddStatement(&parser.Node{Type: parser.NodeExpressionStatement})
	if regular.String() != "[test_label: 2 stmts]" {
		t.Errorf("Regular block string incorrect: %s", regular.String())
	}

	// Block without label
	noLabel := NewBasicBlock("bb3")
	if noLabel.String() != "[bb3: 0 stmts]" {
		t.Errorf("No label block string incorrect: %s", noLabel.String())
	}
}

func TestNewCFG(t *testing.T) {
	cfg := NewCFG("test_function")

	if cfg.Name != "test_function" {
		t.Errorf("Expected name 'test_function', got %s", cfg.Name)
	}
	if cfg.Entry == nil {
		t.Error("Entry block should not be nil")
	}
	if cfg.Exit == nil {
		t.Error("Exit block should not be nil")
	}
	if !cfg.Entry.IsEntry {
		t.Error("Entry block should have IsEntry=true")
	}
	if !cfg.Exit.IsExit {
		t.Error("Exit block should have IsExit=true")
	}
	if cfg.Entry.Label != "ENTRY" {
		t.Errorf("Entry label should be 'ENTRY', got %s", cfg.Entry.Label)
	}
	if cfg.Exit.Label != "EXIT" {
		t.Errorf("Exit label should be 'EXIT', got %s", cfg.Exit.Label)
	}
	if cfg.Size() != 2 {
		t.Errorf("New CFG should have 2 blocks (entry+exit), got %d", cfg.Size())
	}
}

func TestCFG_CreateBlock(t *testing.T) {
	cfg := NewCFG("test")

	block1 := cfg.CreateBlock("block1")
	if block1.Label != "block1" {
		t.Errorf("Block label should be 'block1', got %s", block1.Label)
	}
	if cfg.Blocks[block1.ID] != block1 {
		t.Error("Block should be added to CFG blocks map")
	}

	block2 := cfg.CreateBlock("")
	if block2.Label != "" {
		t.Error("Block with empty label should have empty label")
	}

	// Verify IDs are unique
	if block1.ID == block2.ID {
		t.Error("Blocks should have unique IDs")
	}
}

func TestCFG_AddBlock(t *testing.T) {
	cfg := NewCFG("test")

	// Add nil block - should be ignored
	cfg.AddBlock(nil)
	initialSize := cfg.Size()

	// Add existing block
	block := NewBasicBlock("external_block")
	cfg.AddBlock(block)
	if cfg.Blocks["external_block"] != block {
		t.Error("Block should be added to CFG")
	}
	if cfg.Size() != initialSize+1 {
		t.Error("CFG size should increase by 1")
	}
}

func TestCFG_RemoveBlock(t *testing.T) {
	cfg := NewCFG("test")
	block := cfg.CreateBlock("removable")

	// Connect blocks
	cfg.ConnectBlocks(cfg.Entry, block, EdgeNormal)
	cfg.ConnectBlocks(block, cfg.Exit, EdgeNormal)

	initialSize := cfg.Size()
	cfg.RemoveBlock(block)

	if cfg.Size() != initialSize-1 {
		t.Error("CFG size should decrease after removal")
	}
	if cfg.Blocks[block.ID] != nil {
		t.Error("Block should be removed from blocks map")
	}

	// Should not remove entry or exit
	cfg.RemoveBlock(cfg.Entry)
	if cfg.Entry == nil || cfg.Blocks[cfg.Entry.ID] == nil {
		t.Error("Entry block should not be removed")
	}

	cfg.RemoveBlock(cfg.Exit)
	if cfg.Exit == nil || cfg.Blocks[cfg.Exit.ID] == nil {
		t.Error("Exit block should not be removed")
	}

	// Should handle nil
	cfg.RemoveBlock(nil) // Should not panic
}

func TestCFG_ConnectBlocks(t *testing.T) {
	cfg := NewCFG("test")
	block := cfg.CreateBlock("middle")

	// Connect entry to middle
	edge1 := cfg.ConnectBlocks(cfg.Entry, block, EdgeNormal)
	if edge1 == nil {
		t.Error("ConnectBlocks should return edge")
	}
	if len(cfg.Entry.Successors) != 1 {
		t.Error("Entry should have one successor")
	}

	// Connect middle to exit
	edge2 := cfg.ConnectBlocks(block, cfg.Exit, EdgeReturn)
	if edge2.Type != EdgeReturn {
		t.Errorf("Edge type should be EdgeReturn, got %s", edge2.Type)
	}

	// Connect with nil should return nil
	edge3 := cfg.ConnectBlocks(nil, block, EdgeNormal)
	if edge3 != nil {
		t.Error("ConnectBlocks with nil from should return nil")
	}

	edge4 := cfg.ConnectBlocks(block, nil, EdgeNormal)
	if edge4 != nil {
		t.Error("ConnectBlocks with nil to should return nil")
	}
}

func TestCFG_GetBlock(t *testing.T) {
	cfg := NewCFG("test")
	block := cfg.CreateBlock("test_block")

	retrieved := cfg.GetBlock(block.ID)
	if retrieved != block {
		t.Error("GetBlock should return the block")
	}

	nonExistent := cfg.GetBlock("nonexistent")
	if nonExistent != nil {
		t.Error("GetBlock should return nil for nonexistent block")
	}
}

func TestCFG_Size(t *testing.T) {
	cfg := NewCFG("test")
	if cfg.Size() != 2 {
		t.Errorf("New CFG should have 2 blocks, got %d", cfg.Size())
	}

	cfg.CreateBlock("block1")
	if cfg.Size() != 3 {
		t.Errorf("CFG should have 3 blocks after creating one, got %d", cfg.Size())
	}

	cfg.CreateBlock("block2")
	cfg.CreateBlock("block3")
	if cfg.Size() != 5 {
		t.Errorf("CFG should have 5 blocks, got %d", cfg.Size())
	}
}

func TestCFG_String(t *testing.T) {
	cfg := NewCFG("myFunction")
	result := cfg.String()
	if result != "CFG(myFunction): 2 blocks" {
		t.Errorf("CFG String() incorrect: %s", result)
	}
}

// Test visitor for CFG traversal
type testVisitor struct {
	visitedBlocks []string
	visitedEdges  []string
	stopAtBlock   string
	stopAtEdge    bool
}

func (v *testVisitor) VisitBlock(block *BasicBlock) bool {
	v.visitedBlocks = append(v.visitedBlocks, block.ID)
	return block.ID != v.stopAtBlock
}

func (v *testVisitor) VisitEdge(edge *Edge) bool {
	v.visitedEdges = append(v.visitedEdges, edge.From.ID+"->"+edge.To.ID)
	return !v.stopAtEdge
}

func TestCFG_Walk(t *testing.T) {
	cfg := NewCFG("test")
	block1 := cfg.CreateBlock("block1")
	block2 := cfg.CreateBlock("block2")

	cfg.ConnectBlocks(cfg.Entry, block1, EdgeNormal)
	cfg.ConnectBlocks(block1, block2, EdgeNormal)
	cfg.ConnectBlocks(block2, cfg.Exit, EdgeNormal)

	visitor := &testVisitor{}
	cfg.Walk(visitor)

	if len(visitor.visitedBlocks) != 4 {
		t.Errorf("Should visit 4 blocks, visited %d", len(visitor.visitedBlocks))
	}
	if len(visitor.visitedEdges) != 3 {
		t.Errorf("Should visit 3 edges, visited %d", len(visitor.visitedEdges))
	}
}

func TestCFG_Walk_StopEarly(t *testing.T) {
	cfg := NewCFG("test")
	block1 := cfg.CreateBlock("block1")
	block2 := cfg.CreateBlock("block2")

	cfg.ConnectBlocks(cfg.Entry, block1, EdgeNormal)
	cfg.ConnectBlocks(block1, block2, EdgeNormal)
	cfg.ConnectBlocks(block2, cfg.Exit, EdgeNormal)

	visitor := &testVisitor{stopAtBlock: block1.ID}
	cfg.Walk(visitor)

	// Should stop at block1
	if len(visitor.visitedBlocks) > 2 {
		t.Errorf("Should stop early, visited %d blocks", len(visitor.visitedBlocks))
	}
}

func TestCFG_Walk_NilEntry(t *testing.T) {
	cfg := &CFG{
		Name:   "test",
		Blocks: make(map[string]*BasicBlock),
		Entry:  nil,
	}

	visitor := &testVisitor{}
	cfg.Walk(visitor) // Should not panic

	if len(visitor.visitedBlocks) != 0 {
		t.Error("Should not visit any blocks with nil entry")
	}
}

func TestCFG_BreadthFirstWalk(t *testing.T) {
	cfg := NewCFG("test")
	block1 := cfg.CreateBlock("block1")
	block2 := cfg.CreateBlock("block2")

	cfg.ConnectBlocks(cfg.Entry, block1, EdgeCondTrue)
	cfg.ConnectBlocks(cfg.Entry, block2, EdgeCondFalse)
	cfg.ConnectBlocks(block1, cfg.Exit, EdgeNormal)
	cfg.ConnectBlocks(block2, cfg.Exit, EdgeNormal)

	visitor := &testVisitor{}
	cfg.BreadthFirstWalk(visitor)

	if len(visitor.visitedBlocks) != 4 {
		t.Errorf("Should visit 4 blocks, visited %d", len(visitor.visitedBlocks))
	}
	// Entry should be first
	if visitor.visitedBlocks[0] != cfg.Entry.ID {
		t.Error("Entry should be visited first in BFS")
	}
}

func TestCFG_BreadthFirstWalk_StopEarly(t *testing.T) {
	cfg := NewCFG("test")
	block1 := cfg.CreateBlock("block1")

	cfg.ConnectBlocks(cfg.Entry, block1, EdgeNormal)
	cfg.ConnectBlocks(block1, cfg.Exit, EdgeNormal)

	visitor := &testVisitor{stopAtBlock: cfg.Entry.ID}
	cfg.BreadthFirstWalk(visitor)

	// Should stop at entry
	if len(visitor.visitedBlocks) != 1 {
		t.Errorf("Should stop at entry, visited %d blocks", len(visitor.visitedBlocks))
	}
}

func TestCFG_BreadthFirstWalk_StopAtEdge(t *testing.T) {
	cfg := NewCFG("test")
	block1 := cfg.CreateBlock("block1")

	cfg.ConnectBlocks(cfg.Entry, block1, EdgeNormal)
	cfg.ConnectBlocks(block1, cfg.Exit, EdgeNormal)

	visitor := &testVisitor{stopAtEdge: true}
	cfg.BreadthFirstWalk(visitor)

	// Should visit entry and first edge then stop
	if len(visitor.visitedBlocks) != 1 {
		t.Errorf("Should stop after first edge, visited %d blocks", len(visitor.visitedBlocks))
	}
}

func TestCFG_BreadthFirstWalk_NilEntry(t *testing.T) {
	cfg := &CFG{
		Name:   "test",
		Blocks: make(map[string]*BasicBlock),
		Entry:  nil,
	}

	visitor := &testVisitor{}
	cfg.BreadthFirstWalk(visitor) // Should not panic

	if len(visitor.visitedBlocks) != 0 {
		t.Error("Should not visit any blocks with nil entry")
	}
}

func TestCFG_CyclicGraph(t *testing.T) {
	cfg := NewCFG("test")
	loopHeader := cfg.CreateBlock("loop_header")
	loopBody := cfg.CreateBlock("loop_body")

	cfg.ConnectBlocks(cfg.Entry, loopHeader, EdgeNormal)
	cfg.ConnectBlocks(loopHeader, loopBody, EdgeCondTrue)
	cfg.ConnectBlocks(loopHeader, cfg.Exit, EdgeCondFalse)
	cfg.ConnectBlocks(loopBody, loopHeader, EdgeLoop) // Back edge

	visitor := &testVisitor{}
	cfg.Walk(visitor)

	// Should handle cycle without infinite loop
	if len(visitor.visitedBlocks) != 4 {
		t.Errorf("Should visit 4 blocks even with cycle, visited %d", len(visitor.visitedBlocks))
	}
}
