package graph

import (
	"sort"
	"testing"
)

func TestNoCycles(t *testing.T) {
	g := NewMapGraph()
	g.AddEdge("a", "b")
	g.AddEdge("b", "c")
	g.AddEdge("a", "c")

	d := NewCycleDetector()
	result := d.DetectCycles(g)

	if result.HasCycles {
		t.Fatal("expected no cycles in DAG")
	}
	if len(result.Cycles) != 0 {
		t.Fatalf("expected 0 cycles, got %d", len(result.Cycles))
	}
	if len(result.AffectedNodes) != 0 {
		t.Fatalf("expected 0 affected nodes, got %d", len(result.AffectedNodes))
	}
}

func TestSimpleCycle(t *testing.T) {
	g := NewMapGraph()
	g.AddEdge("a", "b")
	g.AddEdge("b", "c")
	g.AddEdge("c", "a")

	d := NewCycleDetector()
	result := d.DetectCycles(g)

	if !result.HasCycles {
		t.Fatal("expected cycles")
	}
	if len(result.Cycles) != 1 {
		t.Fatalf("expected 1 cycle, got %d", len(result.Cycles))
	}

	scc := result.Cycles[0]
	sort.Strings(scc)
	if len(scc) != 3 || scc[0] != "a" || scc[1] != "b" || scc[2] != "c" {
		t.Fatalf("expected cycle [a b c], got %v", scc)
	}

	for _, n := range []string{"a", "b", "c"} {
		if !result.AffectedNodes[n] {
			t.Fatalf("expected node %s to be affected", n)
		}
	}
}

func TestMultipleSCCs(t *testing.T) {
	g := NewMapGraph()
	// SCC 1: a <-> b
	g.AddEdge("a", "b")
	g.AddEdge("b", "a")
	// SCC 2: c <-> d
	g.AddEdge("c", "d")
	g.AddEdge("d", "c")
	// Bridge (no cycle): b -> c
	g.AddEdge("b", "c")

	d := NewCycleDetector()
	result := d.DetectCycles(g)

	if !result.HasCycles {
		t.Fatal("expected cycles")
	}
	if len(result.Cycles) != 2 {
		t.Fatalf("expected 2 cycles, got %d", len(result.Cycles))
	}

	// Sort cycles for deterministic comparison
	for i := range result.Cycles {
		sort.Strings(result.Cycles[i])
	}
	sort.Slice(result.Cycles, func(i, j int) bool {
		return result.Cycles[i][0] < result.Cycles[j][0]
	})

	if result.Cycles[0][0] != "a" || result.Cycles[0][1] != "b" {
		t.Fatalf("expected first SCC [a b], got %v", result.Cycles[0])
	}
	if result.Cycles[1][0] != "c" || result.Cycles[1][1] != "d" {
		t.Fatalf("expected second SCC [c d], got %v", result.Cycles[1])
	}
}

func TestSelfLoopNotDetected(t *testing.T) {
	// Tarjan SCC with size > 1 filter: self-loops are SCCs of size 1
	// and should NOT be reported as cycles.
	g := NewMapGraph()
	g.AddEdge("a", "a")

	d := NewCycleDetector()
	result := d.DetectCycles(g)

	if result.HasCycles {
		t.Fatal("self-loops should not be reported as cycles (SCC size 1)")
	}
}

func TestEmptyGraph(t *testing.T) {
	g := NewMapGraph()
	d := NewCycleDetector()
	result := d.DetectCycles(g)

	if result.HasCycles {
		t.Fatal("expected no cycles in empty graph")
	}
	if len(result.Cycles) != 0 {
		t.Fatalf("expected 0 cycles, got %d", len(result.Cycles))
	}
}

func TestSingleNode(t *testing.T) {
	g := NewMapGraph()
	g.AddNode("a")

	d := NewCycleDetector()
	result := d.DetectCycles(g)

	if result.HasCycles {
		t.Fatal("expected no cycles for single node")
	}
}

func TestDisconnectedWithCycle(t *testing.T) {
	g := NewMapGraph()
	// Disconnected component 1: cycle a -> b -> a
	g.AddEdge("a", "b")
	g.AddEdge("b", "a")
	// Disconnected component 2: no cycle
	g.AddEdge("x", "y")

	d := NewCycleDetector()
	result := d.DetectCycles(g)

	if !result.HasCycles {
		t.Fatal("expected cycles")
	}
	if len(result.Cycles) != 1 {
		t.Fatalf("expected 1 cycle, got %d", len(result.Cycles))
	}
	if !result.AffectedNodes["a"] || !result.AffectedNodes["b"] {
		t.Fatal("expected a and b to be affected")
	}
	if result.AffectedNodes["x"] || result.AffectedNodes["y"] {
		t.Fatal("x and y should not be affected")
	}
}

func TestLargeSCC(t *testing.T) {
	g := NewMapGraph()
	// Create a cycle: 1 -> 2 -> 3 -> 4 -> 5 -> 1
	g.AddEdge("1", "2")
	g.AddEdge("2", "3")
	g.AddEdge("3", "4")
	g.AddEdge("4", "5")
	g.AddEdge("5", "1")

	d := NewCycleDetector()
	result := d.DetectCycles(g)

	if !result.HasCycles {
		t.Fatal("expected cycle")
	}
	if len(result.Cycles) != 1 {
		t.Fatalf("expected 1 cycle, got %d", len(result.Cycles))
	}
	if len(result.Cycles[0]) != 5 {
		t.Fatalf("expected cycle of size 5, got %d", len(result.Cycles[0]))
	}
}

func TestDetectorReuse(t *testing.T) {
	d := NewCycleDetector()

	// First use: graph with cycle
	g1 := NewMapGraph()
	g1.AddEdge("a", "b")
	g1.AddEdge("b", "a")
	r1 := d.DetectCycles(g1)
	if !r1.HasCycles {
		t.Fatal("expected cycle in g1")
	}

	// Second use: graph without cycle (state should be reset)
	g2 := NewMapGraph()
	g2.AddEdge("x", "y")
	r2 := d.DetectCycles(g2)
	if r2.HasCycles {
		t.Fatal("expected no cycle in g2")
	}
}
