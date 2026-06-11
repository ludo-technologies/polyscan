package analyzer

import (
	"fmt"
	"sort"
	"strings"

	"github.com/ludo-technologies/jscan/domain"
)

// CircularDependencyDetector detects circular dependencies using Tarjan's algorithm
type CircularDependencyDetector struct {
	// Tarjan's algorithm state (reset on each detection)
	index    int
	stack    []string
	indices  map[string]int
	lowlinks map[string]int
	onStack  map[string]bool
	sccs     [][]string
}

// NewCircularDependencyDetector creates a new CircularDependencyDetector
func NewCircularDependencyDetector() *CircularDependencyDetector {
	return &CircularDependencyDetector{}
}

// isLoadTimeEdge reports whether an edge participates in module loading.
// Dynamic import() edges are evaluated only when the call executes, so they
// cannot form a load-time circular import and are excluded from cycle
// detection. A pair connected by both a static and a dynamic import is still
// a load-time dependency via its static edge. See pyscn issue #460.
func isLoadTimeEdge(edge *domain.DependencyEdge) bool {
	return edge != nil && edge.EdgeType != domain.EdgeTypeDynamic
}

// DetectCycles finds all cycles in the dependency graph using Tarjan's SCC algorithm
func (d *CircularDependencyDetector) DetectCycles(graph *domain.DependencyGraph) *domain.CircularDependencyAnalysis {
	if graph == nil || graph.NodeCount() == 0 {
		return &domain.CircularDependencyAnalysis{
			HasCircularDependencies: false,
			TotalCycles:             0,
			TotalModulesInCycles:    0,
			CircularDependencies:    []domain.CircularDependency{},
		}
	}

	// Find all strongly connected components using Tarjan's algorithm
	sccs := d.tarjanSCC(graph)

	// Filter to only SCCs with more than one node (actual cycles)
	var cycles []domain.CircularDependency
	var modulesInCycles = make(map[string]bool)
	var coreInfrastructure = make(map[string]int) // module -> count of cycles it's in

	for _, scc := range sccs {
		if len(scc) > 1 {
			// This is an actual cycle
			cycle := d.buildCycleInfo(scc, graph)
			cycles = append(cycles, cycle)

			for _, module := range scc {
				modulesInCycles[module] = true
				coreInfrastructure[module]++
			}
		}
	}

	// Find core infrastructure (modules in multiple cycles)
	var coreModules []string
	for module, count := range coreInfrastructure {
		if count > 1 {
			coreModules = append(coreModules, module)
		}
	}
	sort.Strings(coreModules)

	// Generate cycle breaking suggestions
	suggestions := d.suggestCycleBreaking(cycles, graph)

	return &domain.CircularDependencyAnalysis{
		HasCircularDependencies:  len(cycles) > 0,
		TotalCycles:              len(cycles),
		TotalModulesInCycles:     len(modulesInCycles),
		CircularDependencies:     cycles,
		CycleBreakingSuggestions: suggestions,
		CoreInfrastructure:       coreModules,
	}
}

// tarjanSCC implements Tarjan's strongly connected components algorithm
func (d *CircularDependencyDetector) tarjanSCC(graph *domain.DependencyGraph) [][]string {
	// Initialize state
	d.index = 0
	d.stack = make([]string, 0)
	d.indices = make(map[string]int)
	d.lowlinks = make(map[string]int)
	d.onStack = make(map[string]bool)
	d.sccs = make([][]string, 0)

	// Process all nodes
	nodeIDs := graph.GetAllNodeIDs()
	sort.Strings(nodeIDs) // For deterministic results

	for _, nodeID := range nodeIDs {
		if _, visited := d.indices[nodeID]; !visited {
			d.strongconnect(nodeID, graph)
		}
	}

	return d.sccs
}

// strongconnect is the recursive function for Tarjan's algorithm
func (d *CircularDependencyDetector) strongconnect(v string, graph *domain.DependencyGraph) {
	// Set the depth index for v to the smallest unused index
	d.indices[v] = d.index
	d.lowlinks[v] = d.index
	d.index++
	d.stack = append(d.stack, v)
	d.onStack[v] = true

	// Consider successors of v
	edges := graph.GetOutgoingEdges(v)
	for _, edge := range edges {
		// Lazy (dynamic import) edges do not run at module load time, so
		// they cannot form a load-time cycle. Skip them. See issue #460.
		if !isLoadTimeEdge(edge) {
			continue
		}
		w := edge.To
		// Skip external/unresolved nodes that aren't in the graph
		if graph.GetNode(w) == nil {
			continue
		}

		if _, visited := d.indices[w]; !visited {
			// Successor w has not yet been visited; recurse on it
			d.strongconnect(w, graph)
			d.lowlinks[v] = min(d.lowlinks[v], d.lowlinks[w])
		} else if d.onStack[w] {
			// Successor w is in stack and hence in the current SCC
			d.lowlinks[v] = min(d.lowlinks[v], d.indices[w])
		}
	}

	// If v is a root node, pop the stack and generate an SCC
	if d.lowlinks[v] == d.indices[v] {
		scc := make([]string, 0)
		for {
			w := d.stack[len(d.stack)-1]
			d.stack = d.stack[:len(d.stack)-1]
			d.onStack[w] = false
			scc = append(scc, w)
			if w == v {
				break
			}
		}
		// Sort for deterministic output
		sort.Strings(scc)
		d.sccs = append(d.sccs, scc)
	}
}

// buildCycleInfo creates a CircularDependency from an SCC
func (d *CircularDependencyDetector) buildCycleInfo(scc []string, graph *domain.DependencyGraph) domain.CircularDependency {
	// Find the cycle path (edges forming the cycle)
	var paths []domain.DependencyPath
	sccSet := make(map[string]bool)
	for _, module := range scc {
		sccSet[module] = true
	}

	// Find edges within the SCC
	for _, from := range scc {
		edges := graph.GetOutgoingEdges(from)
		for _, edge := range edges {
			if !isLoadTimeEdge(edge) {
				continue // lazy edges are not load-time dependencies (#460)
			}
			if sccSet[edge.To] {
				paths = append(paths, domain.DependencyPath{
					From:   from,
					To:     edge.To,
					Path:   []string{from, edge.To},
					Length: 1,
				})
			}
		}
	}

	// Calculate severity based on cycle size
	severity := d.calculateCycleSeverity(scc)

	// Generate description
	description := d.generateCycleDescription(scc)

	return domain.CircularDependency{
		Modules:      scc,
		Dependencies: paths,
		Severity:     severity,
		Size:         len(scc),
		Description:  description,
	}
}

// calculateCycleSeverity determines the severity of a cycle based on its size
func (d *CircularDependencyDetector) calculateCycleSeverity(scc []string) domain.CycleSeverity {
	size := len(scc)
	switch {
	case size <= 2:
		return domain.CycleSeverityLow
	case size <= 4:
		return domain.CycleSeverityMedium
	case size <= 6:
		return domain.CycleSeverityHigh
	default:
		return domain.CycleSeverityCritical
	}
}

// generateCycleDescription creates a human-readable description of the cycle
func (d *CircularDependencyDetector) generateCycleDescription(scc []string) string {
	if len(scc) == 0 {
		return ""
	}

	// Sort for consistent output
	sorted := make([]string, len(scc))
	copy(sorted, scc)
	sort.Strings(sorted)

	// Create a simple cycle representation
	var parts []string
	for _, module := range sorted {
		// Extract just the filename for readability
		parts = append(parts, getModuleBaseName(module))
	}

	return fmt.Sprintf("Circular dependency involving %d modules: %s", len(scc), strings.Join(parts, " <-> "))
}

// suggestCycleBreaking generates suggestions for breaking cycles
func (d *CircularDependencyDetector) suggestCycleBreaking(cycles []domain.CircularDependency, graph *domain.DependencyGraph) []string {
	var suggestions []string

	for _, cycle := range cycles {
		if len(cycle.Modules) == 0 {
			continue
		}

		// Find the best edge to break
		bestEdge := d.findBestEdgeToBreak(cycle, graph)
		if bestEdge != nil {
			suggestion := fmt.Sprintf(
				"Consider removing or inverting the dependency from '%s' to '%s' to break the cycle",
				getModuleBaseName(bestEdge.From),
				getModuleBaseName(bestEdge.To),
			)
			suggestions = append(suggestions, suggestion)
		}

		// Suggest interface extraction for larger cycles
		if len(cycle.Modules) >= 3 {
			suggestion := fmt.Sprintf(
				"Consider extracting interfaces for modules: %s",
				strings.Join(getModuleBaseNames(cycle.Modules), ", "),
			)
			suggestions = append(suggestions, suggestion)
		}
	}

	return suggestions
}

// findBestEdgeToBreak finds the edge with the lowest weight (least used) to break a cycle
func (d *CircularDependencyDetector) findBestEdgeToBreak(cycle domain.CircularDependency, graph *domain.DependencyGraph) *domain.DependencyEdge {
	var bestEdge *domain.DependencyEdge
	minWeight := int(^uint(0) >> 1) // Max int

	sccSet := make(map[string]bool)
	for _, module := range cycle.Modules {
		sccSet[module] = true
	}

	for _, module := range cycle.Modules {
		edges := graph.GetOutgoingEdges(module)
		for _, edge := range edges {
			if !isLoadTimeEdge(edge) {
				continue // breaking a lazy edge would not break the load-time cycle
			}
			if sccSet[edge.To] && edge.Weight < minWeight {
				minWeight = edge.Weight
				bestEdge = edge
			}
		}
	}

	return bestEdge
}

// getModuleBaseName extracts a readable module name from a path
func getModuleBaseName(path string) string {
	// Get the last component of the path
	parts := strings.Split(path, "/")
	if len(parts) > 0 {
		return parts[len(parts)-1]
	}
	return path
}

// getModuleBaseNames extracts readable names from multiple paths
func getModuleBaseNames(paths []string) []string {
	names := make([]string, len(paths))
	for i, path := range paths {
		names[i] = getModuleBaseName(path)
	}
	return names
}

// FindCyclePath finds a path from one module to another within a cycle
func (d *CircularDependencyDetector) FindCyclePath(from, to string, graph *domain.DependencyGraph) []string {
	if graph == nil {
		return nil
	}

	visited := make(map[string]bool)
	path := []string{}

	var dfs func(current string) bool
	dfs = func(current string) bool {
		if current == to {
			path = append(path, current)
			return true
		}

		if visited[current] {
			return false
		}
		visited[current] = true
		path = append(path, current)

		edges := graph.GetOutgoingEdges(current)
		for _, edge := range edges {
			if !isLoadTimeEdge(edge) {
				continue // lazy edges are not load-time dependencies (#460)
			}
			if dfs(edge.To) {
				return true
			}
		}

		// Backtrack
		path = path[:len(path)-1]
		return false
	}

	if dfs(from) {
		return path
	}
	return nil
}
