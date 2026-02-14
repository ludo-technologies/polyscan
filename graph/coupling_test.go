package graph

import (
	"math"
	"testing"
)

func TestCouplingEmptyGraph(t *testing.T) {
	g := NewMapGraph()
	result := ComputeCouplingMetrics(g, CouplingConfig{})

	if len(result) != 0 {
		t.Fatalf("expected 0 metrics, got %d", len(result))
	}
}

func TestCouplingIsolatedNode(t *testing.T) {
	g := NewMapGraph()
	g.AddNode("a")

	result := ComputeCouplingMetrics(g, CouplingConfig{})

	m := result["a"]
	if m == nil {
		t.Fatal("expected metrics for node a")
	}
	if m.Ca != 0 || m.Ce != 0 {
		t.Fatalf("expected Ca=0 Ce=0, got Ca=%d Ce=%d", m.Ca, m.Ce)
	}
	if m.Instability != 0.0 {
		t.Fatalf("expected Instability=0, got %f", m.Instability)
	}
	// Distance = |0 + 0 - 1| = 1
	if m.Distance != 1.0 {
		t.Fatalf("expected Distance=1.0, got %f", m.Distance)
	}
}

func TestCouplingSimpleEdge(t *testing.T) {
	g := NewMapGraph()
	g.AddEdge("a", "b")

	result := ComputeCouplingMetrics(g, CouplingConfig{})

	ma := result["a"]
	if ma.Ca != 0 || ma.Ce != 1 {
		t.Fatalf("a: expected Ca=0 Ce=1, got Ca=%d Ce=%d", ma.Ca, ma.Ce)
	}
	if ma.Instability != 1.0 {
		t.Fatalf("a: expected Instability=1.0, got %f", ma.Instability)
	}

	mb := result["b"]
	if mb.Ca != 1 || mb.Ce != 0 {
		t.Fatalf("b: expected Ca=1 Ce=0, got Ca=%d Ce=%d", mb.Ca, mb.Ce)
	}
	if mb.Instability != 0.0 {
		t.Fatalf("b: expected Instability=0.0, got %f", mb.Instability)
	}
}

func TestCouplingWithAbstractness(t *testing.T) {
	g := NewMapGraph()
	g.AddEdge("a", "b")
	g.AddEdge("c", "b")

	config := CouplingConfig{
		AbstractnessFunc: func(nodeID string) float64 {
			if nodeID == "b" {
				return 0.8
			}
			return 0.0
		},
	}

	result := ComputeCouplingMetrics(g, config)

	mb := result["b"]
	if mb.Ca != 2 || mb.Ce != 0 {
		t.Fatalf("b: expected Ca=2 Ce=0, got Ca=%d Ce=%d", mb.Ca, mb.Ce)
	}
	if mb.Instability != 0.0 {
		t.Fatalf("b: expected Instability=0.0, got %f", mb.Instability)
	}
	if mb.Abstractness != 0.8 {
		t.Fatalf("b: expected Abstractness=0.8, got %f", mb.Abstractness)
	}
	// Distance = |0.8 + 0.0 - 1| = 0.2
	if math.Abs(mb.Distance-0.2) > 1e-9 {
		t.Fatalf("b: expected Distance=0.2, got %f", mb.Distance)
	}
}

func TestCouplingMainSequence(t *testing.T) {
	// On the main sequence: A + I = 1
	g := NewMapGraph()
	g.AddEdge("a", "b")
	g.AddEdge("b", "c")

	config := CouplingConfig{
		AbstractnessFunc: func(nodeID string) float64 {
			// b has Ca=1 Ce=1, so I=0.5. For main sequence: A=0.5
			if nodeID == "b" {
				return 0.5
			}
			return 0.0
		},
	}

	result := ComputeCouplingMetrics(g, config)
	mb := result["b"]

	if math.Abs(mb.Instability-0.5) > 1e-9 {
		t.Fatalf("b: expected I=0.5, got %f", mb.Instability)
	}
	if math.Abs(mb.Distance) > 1e-9 {
		t.Fatalf("b: expected Distance≈0 (on main sequence), got %f", mb.Distance)
	}
}

func TestCouplingDiamond(t *testing.T) {
	g := NewMapGraph()
	g.AddEdge("a", "b")
	g.AddEdge("a", "c")
	g.AddEdge("b", "d")
	g.AddEdge("c", "d")

	result := ComputeCouplingMetrics(g, CouplingConfig{})

	// a: Ca=0, Ce=2
	if result["a"].Ca != 0 || result["a"].Ce != 2 {
		t.Fatalf("a: expected Ca=0 Ce=2, got Ca=%d Ce=%d", result["a"].Ca, result["a"].Ce)
	}

	// d: Ca=2, Ce=0
	if result["d"].Ca != 2 || result["d"].Ce != 0 {
		t.Fatalf("d: expected Ca=2 Ce=0, got Ca=%d Ce=%d", result["d"].Ca, result["d"].Ce)
	}

	// b: Ca=1, Ce=1, I=0.5
	if result["b"].Ca != 1 || result["b"].Ce != 1 {
		t.Fatalf("b: expected Ca=1 Ce=1, got Ca=%d Ce=%d", result["b"].Ca, result["b"].Ce)
	}
	if math.Abs(result["b"].Instability-0.5) > 1e-9 {
		t.Fatalf("b: expected I=0.5, got %f", result["b"].Instability)
	}
}

func TestCouplingNilAbstractnessFunc(t *testing.T) {
	g := NewMapGraph()
	g.AddEdge("a", "b")

	result := ComputeCouplingMetrics(g, CouplingConfig{})

	if result["a"].Abstractness != 0.0 {
		t.Fatalf("expected default Abstractness=0.0, got %f", result["a"].Abstractness)
	}
}
