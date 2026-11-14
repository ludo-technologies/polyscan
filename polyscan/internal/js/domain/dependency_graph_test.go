package domain

import (
	"reflect"
	"testing"
)

func TestMerge(t *testing.T) {
	graph := NewDependencyGraph()
	graph.AddNode(&ModuleNode{ID: "a"})
	other := NewDependencyGraph()
	other.AddNode(&ModuleNode{ID: "b"})
	other.AddNode(&ModuleNode{ID: "c"})
	other.AddEdge(&DependencyEdge{From: "b", To: "c"})

	if err := graph.Merge(other); err != nil {
		t.Fatal(err)
	}
	if got, want := graph.NodeIDs(), []string{"a", "b", "c"}; !reflect.DeepEqual(got, want) {
		t.Errorf("nodes = %v, want %v", got, want)
	}
	if got := graph.Predecessors("c"); !reflect.DeepEqual(got, []string{"b"}) {
		t.Errorf("predecessors of c = %v, want [b]", got)
	}

	// A node both graphs hold would lose the edges of one side.
	shared := NewDependencyGraph()
	shared.AddNode(&ModuleNode{ID: "a"})
	if err := graph.Merge(shared); err == nil {
		t.Error("merging a graph that holds node a again succeeded")
	}
}
