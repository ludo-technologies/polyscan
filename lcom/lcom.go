package lcom

import (
	"sort"

	"github.com/ludo-technologies/codescan-core/domain"
)

// MethodAccess describes a method and which instance variables it accesses.
type MethodAccess struct {
	MethodName   string
	InstanceVars map[string]bool
}

// Result holds the LCOM4 analysis result for a class.
type Result struct {
	ClassName         string
	FilePath          string
	StartLine         int
	EndLine           int
	LCOM4             int
	TotalMethods      int
	ExcludedMethods   int
	InstanceVariables []string
	MethodGroups      [][]string
	RiskLevel         domain.RiskLevel
}

// Config holds configuration for LCOM analysis.
type Config struct {
	LowThreshold    int
	MediumThreshold int
}

// DefaultConfig returns the default LCOM configuration.
func DefaultConfig() Config {
	return Config{
		LowThreshold:    domain.DefaultLCOMLowThreshold,
		MediumThreshold: domain.DefaultLCOMMediumThreshold,
	}
}

// ComputeLCOM4 computes the LCOM4 metric using the Union-Find algorithm.
// LCOM4 counts the number of connected components among methods,
// where two methods are connected if they share at least one instance variable.
func ComputeLCOM4(methods []MethodAccess, config Config) *Result {
	result := &Result{
		MethodGroups: [][]string{},
	}

	if len(methods) == 0 {
		result.LCOM4 = 0
		result.RiskLevel = AssessRisk(0, config)
		return result
	}

	// Sort method names for deterministic output
	sortedMethods := make([]MethodAccess, len(methods))
	copy(sortedMethods, methods)
	sort.Slice(sortedMethods, func(i, j int) bool {
		return sortedMethods[i].MethodName < sortedMethods[j].MethodName
	})

	result.TotalMethods = len(sortedMethods)

	// Collect all instance variables
	allVars := make(map[string]bool)
	for _, m := range sortedMethods {
		for v := range m.InstanceVars {
			allVars[v] = true
		}
	}
	varList := make([]string, 0, len(allVars))
	for v := range allVars {
		varList = append(varList, v)
	}
	sort.Strings(varList)
	result.InstanceVariables = varList

	// If only one method, LCOM4 = 1
	if len(sortedMethods) == 1 {
		result.LCOM4 = 1
		result.MethodGroups = [][]string{{sortedMethods[0].MethodName}}
		result.RiskLevel = AssessRisk(1, config)
		return result
	}

	// Union-Find
	n := len(sortedMethods)
	parent := make([]int, n)
	rank := make([]int, n)
	for i := range parent {
		parent[i] = i
	}

	var find func(int) int
	find = func(x int) int {
		if parent[x] != x {
			parent[x] = find(parent[x]) // path compression
		}
		return parent[x]
	}

	union := func(x, y int) {
		rx, ry := find(x), find(y)
		if rx == ry {
			return
		}
		if rank[rx] < rank[ry] {
			parent[rx] = ry
		} else if rank[rx] > rank[ry] {
			parent[ry] = rx
		} else {
			parent[ry] = rx
			rank[rx]++
		}
	}

	// Build variable -> method indices mapping
	varToMethods := make(map[string][]int)
	for i, m := range sortedMethods {
		for v := range m.InstanceVars {
			varToMethods[v] = append(varToMethods[v], i)
		}
	}

	// Union methods that share variables
	for _, indices := range varToMethods {
		for k := 1; k < len(indices); k++ {
			union(indices[0], indices[k])
		}
	}

	// Count connected components and build method groups
	components := make(map[int][]int) // root -> list of method indices
	for i := range sortedMethods {
		root := find(i)
		components[root] = append(components[root], i)
	}

	// Sort component roots for deterministic output
	roots := make([]int, 0, len(components))
	for r := range components {
		roots = append(roots, r)
	}
	sort.Ints(roots)

	result.LCOM4 = len(components)
	result.MethodGroups = make([][]string, 0, len(components))
	for _, root := range roots {
		indices := components[root]
		group := make([]string, len(indices))
		for i, idx := range indices {
			group[i] = sortedMethods[idx].MethodName
		}
		sort.Strings(group)
		result.MethodGroups = append(result.MethodGroups, group)
	}

	result.RiskLevel = AssessRisk(result.LCOM4, config)
	return result
}

// AssessRisk determines the risk level based on LCOM4 and config thresholds.
func AssessRisk(lcom4 int, config Config) domain.RiskLevel {
	if lcom4 <= config.LowThreshold {
		return domain.RiskLevelLow
	}
	if lcom4 <= config.MediumThreshold {
		return domain.RiskLevelMedium
	}
	return domain.RiskLevelHigh
}
