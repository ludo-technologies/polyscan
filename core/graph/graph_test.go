package graph

import (
	"testing"
)

func TestNewMapGraph(t *testing.T) {
	g := NewMapGraph()
	if g.NodeCount() != 0 {
		t.Fatalf("expected 0 nodes, got %d", g.NodeCount())
	}
	if ids := g.NodeIDs(); len(ids) != 0 {
		t.Fatalf("expected empty NodeIDs, got %v", ids)
	}
}

func TestAddNode(t *testing.T) {
	g := NewMapGraph()
	g.AddNode("a")
	g.AddNode("b")
	g.AddNode("a") // duplicate

	if g.NodeCount() != 2 {
		t.Fatalf("expected 2 nodes, got %d", g.NodeCount())
	}
	if !g.HasNode("a") || !g.HasNode("b") {
		t.Fatal("expected nodes a and b to exist")
	}
	if g.HasNode("c") {
		t.Fatal("expected node c to not exist")
	}
}

func TestAddEdge(t *testing.T) {
	g := NewMapGraph()
	g.AddEdge("a", "b")
	g.AddEdge("a", "c")
	g.AddEdge("b", "c")

	if g.NodeCount() != 3 {
		t.Fatalf("expected 3 nodes, got %d", g.NodeCount())
	}

	succ := g.Successors("a")
	if len(succ) != 2 || succ[0] != "b" || succ[1] != "c" {
		t.Fatalf("expected successors [b c], got %v", succ)
	}

	pred := g.Predecessors("c")
	if len(pred) != 2 || pred[0] != "a" || pred[1] != "b" {
		t.Fatalf("expected predecessors [a b], got %v", pred)
	}
}

func TestNodeIDsSorted(t *testing.T) {
	g := NewMapGraph()
	g.AddNode("z")
	g.AddNode("a")
	g.AddNode("m")

	ids := g.NodeIDs()
	if len(ids) != 3 || ids[0] != "a" || ids[1] != "m" || ids[2] != "z" {
		t.Fatalf("expected sorted [a m z], got %v", ids)
	}
}

func TestSuccessorsPredecessorsEmpty(t *testing.T) {
	g := NewMapGraph()
	g.AddNode("a")

	if succ := g.Successors("a"); succ != nil {
		t.Fatalf("expected nil successors, got %v", succ)
	}
	if pred := g.Predecessors("a"); pred != nil {
		t.Fatalf("expected nil predecessors, got %v", pred)
	}
	if succ := g.Successors("nonexistent"); succ != nil {
		t.Fatalf("expected nil for nonexistent node, got %v", succ)
	}
}

func TestDuplicateEdge(t *testing.T) {
	g := NewMapGraph()
	g.AddEdge("a", "b")
	g.AddEdge("a", "b") // duplicate

	succ := g.Successors("a")
	if len(succ) != 1 {
		t.Fatalf("expected 1 successor after duplicate edge, got %d", len(succ))
	}
}

func TestSelfLoop(t *testing.T) {
	g := NewMapGraph()
	g.AddEdge("a", "a")

	if g.NodeCount() != 1 {
		t.Fatalf("expected 1 node, got %d", g.NodeCount())
	}
	succ := g.Successors("a")
	if len(succ) != 1 || succ[0] != "a" {
		t.Fatalf("expected self-loop successor [a], got %v", succ)
	}
	pred := g.Predecessors("a")
	if len(pred) != 1 || pred[0] != "a" {
		t.Fatalf("expected self-loop predecessor [a], got %v", pred)
	}
}

func TestDirectedGraphInterface(t *testing.T) {
	// Verify MapGraph satisfies DirectedGraph.
	var _ DirectedGraph = NewMapGraph()
}

func TestComplexGraph(t *testing.T) {
	g := NewMapGraph()
	// Diamond: a -> b, a -> c, b -> d, c -> d
	g.AddEdge("a", "b")
	g.AddEdge("a", "c")
	g.AddEdge("b", "d")
	g.AddEdge("c", "d")

	if g.NodeCount() != 4 {
		t.Fatalf("expected 4 nodes, got %d", g.NodeCount())
	}

	// a: 2 successors, 0 predecessors
	if len(g.Successors("a")) != 2 {
		t.Fatalf("expected 2 successors for a, got %d", len(g.Successors("a")))
	}
	if pred := g.Predecessors("a"); pred != nil {
		t.Fatalf("expected nil predecessors for a, got %v", pred)
	}

	// d: 0 successors, 2 predecessors
	if succ := g.Successors("d"); succ != nil {
		t.Fatalf("expected nil successors for d, got %v", succ)
	}
	if len(g.Predecessors("d")) != 2 {
		t.Fatalf("expected 2 predecessors for d, got %d", len(g.Predecessors("d")))
	}
}

func TestAddEdgeCreatesNodes(t *testing.T) {
	g := NewMapGraph()
	g.AddEdge("x", "y")

	if !g.HasNode("x") || !g.HasNode("y") {
		t.Fatal("AddEdge should create both nodes")
	}
}
