package graph

import "sort"

// DirectedGraph provides read-only access to a directed graph.
// pyscn's DependencyGraph (map[string]*ModuleNode) and
// jscan's domain.DependencyGraph (method-based) can both implement this.
type DirectedGraph interface {
	// NodeIDs returns all node identifiers in the graph.
	NodeIDs() []string
	// Successors returns the IDs of nodes that this node has edges to.
	Successors(nodeID string) []string
	// Predecessors returns the IDs of nodes that have edges to this node.
	Predecessors(nodeID string) []string
	// NodeCount returns the number of nodes in the graph.
	NodeCount() int
	// HasNode returns true if the node exists in the graph.
	HasNode(nodeID string) bool
}

// MapGraph is a simple directed graph implementation backed by maps.
// Useful for testing and as a default implementation.
type MapGraph struct {
	nodes map[string]bool
	fwd   map[string]map[string]bool // from -> set of to
	rev   map[string]map[string]bool // to -> set of from
}

// NewMapGraph creates a new empty MapGraph.
func NewMapGraph() *MapGraph {
	return &MapGraph{
		nodes: make(map[string]bool),
		fwd:   make(map[string]map[string]bool),
		rev:   make(map[string]map[string]bool),
	}
}

// AddNode adds a node to the graph. No-op if already present.
func (g *MapGraph) AddNode(id string) {
	g.nodes[id] = true
}

// AddEdge adds a directed edge from → to. Both nodes are created if absent.
func (g *MapGraph) AddEdge(from, to string) {
	g.nodes[from] = true
	g.nodes[to] = true
	if g.fwd[from] == nil {
		g.fwd[from] = make(map[string]bool)
	}
	g.fwd[from][to] = true
	if g.rev[to] == nil {
		g.rev[to] = make(map[string]bool)
	}
	g.rev[to][from] = true
}

// NodeIDs returns all node identifiers sorted lexicographically.
func (g *MapGraph) NodeIDs() []string {
	ids := make([]string, 0, len(g.nodes))
	for id := range g.nodes {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	return ids
}

// Successors returns the IDs of nodes reachable by one edge from nodeID.
func (g *MapGraph) Successors(nodeID string) []string {
	s := g.fwd[nodeID]
	if len(s) == 0 {
		return nil
	}
	out := make([]string, 0, len(s))
	for id := range s {
		out = append(out, id)
	}
	sort.Strings(out)
	return out
}

// Predecessors returns the IDs of nodes that have an edge to nodeID.
func (g *MapGraph) Predecessors(nodeID string) []string {
	s := g.rev[nodeID]
	if len(s) == 0 {
		return nil
	}
	out := make([]string, 0, len(s))
	for id := range s {
		out = append(out, id)
	}
	sort.Strings(out)
	return out
}

// NodeCount returns the number of nodes.
func (g *MapGraph) NodeCount() int {
	return len(g.nodes)
}

// HasNode returns true if the node exists.
func (g *MapGraph) HasNode(nodeID string) bool {
	return g.nodes[nodeID]
}
