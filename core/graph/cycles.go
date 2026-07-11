package graph

// CycleResult holds the result of cycle detection via Tarjan's SCC algorithm.
type CycleResult struct {
	// Cycles contains all strongly connected components with more than one node.
	Cycles [][]string
	// HasCycles is true if any cycle was found.
	HasCycles bool
	// AffectedNodes contains all nodes that participate in at least one cycle.
	AffectedNodes map[string]bool
}

// CycleDetector finds strongly connected components using Tarjan's algorithm.
type CycleDetector struct {
	index    int
	stack    []string
	onStack  map[string]bool
	indices  map[string]int
	lowlinks map[string]int
	result   *CycleResult
}

// NewCycleDetector creates a new CycleDetector.
func NewCycleDetector() *CycleDetector {
	return &CycleDetector{}
}

// DetectCycles finds all cycles (SCCs with size > 1) in the directed graph.
func (d *CycleDetector) DetectCycles(g DirectedGraph) *CycleResult {
	d.index = 0
	d.stack = nil
	d.onStack = make(map[string]bool)
	d.indices = make(map[string]int)
	d.lowlinks = make(map[string]int)
	d.result = &CycleResult{
		AffectedNodes: make(map[string]bool),
	}

	for _, nodeID := range g.NodeIDs() {
		if _, visited := d.indices[nodeID]; !visited {
			d.strongConnect(g, nodeID)
		}
	}

	d.result.HasCycles = len(d.result.Cycles) > 0
	return d.result
}

func (d *CycleDetector) strongConnect(g DirectedGraph, v string) {
	d.indices[v] = d.index
	d.lowlinks[v] = d.index
	d.index++
	d.stack = append(d.stack, v)
	d.onStack[v] = true

	for _, w := range g.Successors(v) {
		if _, visited := d.indices[w]; !visited {
			d.strongConnect(g, w)
			if d.lowlinks[w] < d.lowlinks[v] {
				d.lowlinks[v] = d.lowlinks[w]
			}
		} else if d.onStack[w] {
			if d.indices[w] < d.lowlinks[v] {
				d.lowlinks[v] = d.indices[w]
			}
		}
	}

	if d.lowlinks[v] == d.indices[v] {
		var scc []string
		for {
			w := d.stack[len(d.stack)-1]
			d.stack = d.stack[:len(d.stack)-1]
			d.onStack[w] = false
			scc = append(scc, w)
			if w == v {
				break
			}
		}
		// Only record SCCs with more than one node (actual cycles).
		if len(scc) > 1 {
			d.result.Cycles = append(d.result.Cycles, scc)
			for _, node := range scc {
				d.result.AffectedNodes[node] = true
			}
		}
	}
}
