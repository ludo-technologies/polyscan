package cfg

import "testing"

// testClassifier is a StatementClassifier for testing.
type testClassifier struct{}

func (tc *testClassifier) IsReturn(stmt any) bool   { s, ok := stmt.(string); return ok && s == "return" }
func (tc *testClassifier) IsBreak(stmt any) bool     { s, ok := stmt.(string); return ok && s == "break" }
func (tc *testClassifier) IsContinue(stmt any) bool  { s, ok := stmt.(string); return ok && s == "continue" }
func (tc *testClassifier) IsThrow(stmt any) bool     { s, ok := stmt.(string); return ok && s == "throw" }

func TestReachabilityLinearCFG(t *testing.T) {
	c := NewCFG("linear")
	b1 := c.CreateBlock("b1")
	b2 := c.CreateBlock("b2")
	c.ConnectBlocks(c.Entry, b1, EdgeNormal)
	c.ConnectBlocks(b1, b2, EdgeNormal)
	c.ConnectBlocks(b2, c.Exit, EdgeNormal)

	result := AnalyzeReachability(c, nil)

	// entry + exit + b1 + b2 = 4 reachable
	if result.ReachableCount != 4 {
		t.Fatalf("expected 4 reachable, got %d", result.ReachableCount)
	}
	if result.UnreachableCount != 0 {
		t.Fatalf("expected 0 unreachable, got %d", result.UnreachableCount)
	}
}

func TestReachabilityBranching(t *testing.T) {
	c := NewCFG("branch")
	bTrue := c.CreateBlock("if_true")
	bFalse := c.CreateBlock("if_false")
	bJoin := c.CreateBlock("join")
	c.ConnectBlocks(c.Entry, bTrue, EdgeCondTrue)
	c.ConnectBlocks(c.Entry, bFalse, EdgeCondFalse)
	c.ConnectBlocks(bTrue, bJoin, EdgeNormal)
	c.ConnectBlocks(bFalse, bJoin, EdgeNormal)
	c.ConnectBlocks(bJoin, c.Exit, EdgeNormal)

	result := AnalyzeReachability(c, nil)

	if result.ReachableCount != 5 {
		t.Fatalf("expected 5 reachable, got %d", result.ReachableCount)
	}
	if result.UnreachableCount != 0 {
		t.Fatalf("expected 0 unreachable, got %d", result.UnreachableCount)
	}
}

func TestReachabilityUnreachableBlock(t *testing.T) {
	c := NewCFG("unreachable")
	b1 := c.CreateBlock("b1")
	orphan := c.CreateBlock("orphan") // not connected
	_ = orphan
	c.ConnectBlocks(c.Entry, b1, EdgeNormal)
	c.ConnectBlocks(b1, c.Exit, EdgeNormal)

	result := AnalyzeReachability(c, nil)

	if result.UnreachableCount != 1 {
		t.Fatalf("expected 1 unreachable, got %d", result.UnreachableCount)
	}
	if result.Reachable[orphan.ID] {
		t.Fatal("orphan should not be reachable")
	}
}

func TestReachabilityNilClassifier(t *testing.T) {
	c := NewCFG("nil_classifier")
	b1 := c.CreateBlock("b1")
	b1.AddStatement("return")
	b2 := c.CreateBlock("b2")
	c.ConnectBlocks(c.Entry, b1, EdgeNormal)
	c.ConnectBlocks(b1, b2, EdgeNormal)
	c.ConnectBlocks(b2, c.Exit, EdgeNormal)

	// Without classifier, all connected blocks are reachable.
	result := AnalyzeReachability(c, nil)
	if result.ReachableCount != 4 {
		t.Fatalf("expected 4 reachable with nil classifier, got %d", result.ReachableCount)
	}
}

func TestReachabilityWithClassifier(t *testing.T) {
	c := NewCFG("with_classifier")
	b1 := c.CreateBlock("b1")
	b1.AddStatement("return")
	b2 := c.CreateBlock("b2")
	c.ConnectBlocks(c.Entry, b1, EdgeNormal)
	c.ConnectBlocks(b1, b2, EdgeNormal)
	c.ConnectBlocks(b2, c.Exit, EdgeNormal)

	// With classifier, b2 should be unreachable (b1 has return).
	result := AnalyzeReachability(c, &testClassifier{})

	if result.Reachable[b2.ID] {
		t.Fatal("b2 should be unreachable after return block")
	}
	// entry + b1 reachable, exit and b2 unreachable
	if result.ReachableCount != 2 {
		t.Fatalf("expected 2 reachable, got %d", result.ReachableCount)
	}
}

func TestReachabilityMultiplePathsAroundReturn(t *testing.T) {
	c := NewCFG("multi_path")
	bRet := c.CreateBlock("return_block")
	bRet.AddStatement("return")
	bNorm := c.CreateBlock("normal_block")
	bJoin := c.CreateBlock("join")

	c.ConnectBlocks(c.Entry, bRet, EdgeCondTrue)
	c.ConnectBlocks(c.Entry, bNorm, EdgeCondFalse)
	c.ConnectBlocks(bRet, bJoin, EdgeNormal)
	c.ConnectBlocks(bNorm, bJoin, EdgeNormal)
	c.ConnectBlocks(bJoin, c.Exit, EdgeNormal)

	result := AnalyzeReachability(c, &testClassifier{})

	// bJoin is reachable via bNorm path (normal block doesn't terminate)
	if !result.Reachable[bJoin.ID] {
		t.Fatal("join should be reachable via alternative path")
	}
}

func TestReachabilityNilCFG(t *testing.T) {
	result := AnalyzeReachability(nil, nil)
	if result.ReachableCount != 0 {
		t.Fatalf("expected 0 reachable for nil CFG, got %d", result.ReachableCount)
	}
}

func TestReachabilityEmptyCFG(t *testing.T) {
	c := NewCFG("empty")
	// Just entry and exit, connected
	c.ConnectBlocks(c.Entry, c.Exit, EdgeNormal)

	result := AnalyzeReachability(c, &testClassifier{})
	if result.ReachableCount != 2 {
		t.Fatalf("expected 2 reachable (entry+exit), got %d", result.ReachableCount)
	}
}
