package cfg

import "testing"

// testContributor adds extra complexity for testing.
type testContributor struct {
	extras map[string]int // blockID -> extra complexity
}

func (tc *testContributor) ExtraComplexity(block *BasicBlock) int {
	return tc.extras[block.ID]
}

func TestComplexityLinear(t *testing.T) {
	// entry -> b1 -> exit (no decisions)
	c := NewCFG("linear")
	b1 := c.CreateBlock("b1")
	c.ConnectBlocks(c.Entry, b1, EdgeNormal)
	c.ConnectBlocks(b1, c.Exit, EdgeNormal)

	result := ComputeComplexity(c, nil)

	if result.McCabe != 1 {
		t.Fatalf("expected McCabe=1, got %d", result.McCabe)
	}
	if result.DecisionPoints != 0 {
		t.Fatalf("expected 0 decision points, got %d", result.DecisionPoints)
	}
}

func TestComplexityIfElse(t *testing.T) {
	// entry -> [cond] -true-> b_true -> join -> exit
	//                  -false-> b_false -> join
	c := NewCFG("if_else")
	cond := c.CreateBlock("cond")
	bTrue := c.CreateBlock("true")
	bFalse := c.CreateBlock("false")
	join := c.CreateBlock("join")

	c.ConnectBlocks(c.Entry, cond, EdgeNormal)
	c.ConnectBlocks(cond, bTrue, EdgeCondTrue)
	c.ConnectBlocks(cond, bFalse, EdgeCondFalse)
	c.ConnectBlocks(bTrue, join, EdgeNormal)
	c.ConnectBlocks(bFalse, join, EdgeNormal)
	c.ConnectBlocks(join, c.Exit, EdgeNormal)

	result := ComputeComplexity(c, nil)

	if result.DecisionPoints != 1 {
		t.Fatalf("expected 1 decision point, got %d", result.DecisionPoints)
	}
	if result.McCabe != 2 {
		t.Fatalf("expected McCabe=2, got %d", result.McCabe)
	}
	if result.EdgeBreakdown[EdgeCondTrue] != 1 {
		t.Fatalf("expected 1 true edge, got %d", result.EdgeBreakdown[EdgeCondTrue])
	}
	if result.EdgeBreakdown[EdgeCondFalse] != 1 {
		t.Fatalf("expected 1 false edge, got %d", result.EdgeBreakdown[EdgeCondFalse])
	}
}

func TestComplexityNestedIf(t *testing.T) {
	// Two decision blocks: 2 decision points -> McCabe = 3
	c := NewCFG("nested_if")
	cond1 := c.CreateBlock("cond1")
	cond2 := c.CreateBlock("cond2")
	b1 := c.CreateBlock("b1")
	b2 := c.CreateBlock("b2")
	b3 := c.CreateBlock("b3")
	join := c.CreateBlock("join")

	c.ConnectBlocks(c.Entry, cond1, EdgeNormal)
	c.ConnectBlocks(cond1, cond2, EdgeCondTrue)
	c.ConnectBlocks(cond1, b1, EdgeCondFalse)
	c.ConnectBlocks(cond2, b2, EdgeCondTrue)
	c.ConnectBlocks(cond2, b3, EdgeCondFalse)
	c.ConnectBlocks(b1, join, EdgeNormal)
	c.ConnectBlocks(b2, join, EdgeNormal)
	c.ConnectBlocks(b3, join, EdgeNormal)
	c.ConnectBlocks(join, c.Exit, EdgeNormal)

	result := ComputeComplexity(c, nil)

	if result.DecisionPoints != 2 {
		t.Fatalf("expected 2 decision points, got %d", result.DecisionPoints)
	}
	if result.McCabe != 3 {
		t.Fatalf("expected McCabe=3, got %d", result.McCabe)
	}
}

func TestComplexityLoop(t *testing.T) {
	c := NewCFG("loop")
	loopBlock := c.CreateBlock("loop")
	body := c.CreateBlock("body")

	c.ConnectBlocks(c.Entry, loopBlock, EdgeNormal)
	c.ConnectBlocks(loopBlock, body, EdgeLoop)
	c.ConnectBlocks(body, loopBlock, EdgeNormal)
	c.ConnectBlocks(loopBlock, c.Exit, EdgeNormal)

	result := ComputeComplexity(c, nil)

	if result.DecisionPoints != 1 {
		t.Fatalf("expected 1 decision point (loop), got %d", result.DecisionPoints)
	}
	if result.McCabe != 2 {
		t.Fatalf("expected McCabe=2, got %d", result.McCabe)
	}
}

func TestComplexityException(t *testing.T) {
	c := NewCFG("exception")
	tryBlock := c.CreateBlock("try")
	handler := c.CreateBlock("handler")
	after := c.CreateBlock("after")

	c.ConnectBlocks(c.Entry, tryBlock, EdgeNormal)
	c.ConnectBlocks(tryBlock, handler, EdgeException)
	c.ConnectBlocks(tryBlock, after, EdgeNormal)
	c.ConnectBlocks(handler, after, EdgeNormal)
	c.ConnectBlocks(after, c.Exit, EdgeNormal)

	result := ComputeComplexity(c, nil)

	if result.DecisionPoints != 1 {
		t.Fatalf("expected 1 decision point (exception), got %d", result.DecisionPoints)
	}
	if result.EdgeBreakdown[EdgeException] != 1 {
		t.Fatalf("expected 1 exception edge, got %d", result.EdgeBreakdown[EdgeException])
	}
}

func TestComplexityWithContributor(t *testing.T) {
	c := NewCFG("contributor")
	b1 := c.CreateBlock("b1")
	c.ConnectBlocks(c.Entry, b1, EdgeNormal)
	c.ConnectBlocks(b1, c.Exit, EdgeNormal)

	contrib := &testContributor{
		extras: map[string]int{b1.ID: 3},
	}

	result := ComputeComplexity(c, contrib)

	if result.ExtraContributions != 3 {
		t.Fatalf("expected 3 extra contributions, got %d", result.ExtraContributions)
	}
	// McCabe = 0 (decision) + 3 (extra) + 1 = 4
	if result.McCabe != 4 {
		t.Fatalf("expected McCabe=4, got %d", result.McCabe)
	}
}

func TestComplexityNilCFG(t *testing.T) {
	result := ComputeComplexity(nil, nil)
	if result.McCabe != 1 {
		t.Fatalf("expected McCabe=1 for nil CFG, got %d", result.McCabe)
	}
}
