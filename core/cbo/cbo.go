package cbo

import (
	"path/filepath"
	"sort"
	"strings"

	"github.com/ludo-technologies/polyscan/core/domain"
)

// DependencyKind represents the type of dependency between classes.
type DependencyKind int

const (
	DepInheritance     DependencyKind = iota // Class inheritance
	DepTypeHint                              // Type hint/annotation
	DepInstantiation                         // Object instantiation
	DepAttributeAccess                       // Attribute access on another class
	DepImport                                // Import dependency
)

// String returns the string representation of a DependencyKind.
func (k DependencyKind) String() string {
	switch k {
	case DepInheritance:
		return "inheritance"
	case DepTypeHint:
		return "type_hint"
	case DepInstantiation:
		return "instantiation"
	case DepAttributeAccess:
		return "attribute_access"
	case DepImport:
		return "import"
	default:
		return "unknown"
	}
}

// ClassDependency represents a single dependency from one class to another.
type ClassDependency struct {
	ClassName string
	Kind      DependencyKind
}

// ClassInfo holds the information about a class needed for CBO computation.
// Language-specific analyzers populate this from their AST.
type ClassInfo struct {
	Name         string
	FilePath     string
	StartLine    int
	EndLine      int
	Dependencies []ClassDependency
	IsAbstract   bool
	BaseClasses  []string
	Methods      []string
	Attributes   []string
}

// Result holds the CBO analysis result for a single class.
type Result struct {
	ClassName           string
	FilePath            string
	StartLine           int
	EndLine             int
	CouplingCount       int
	DependencyBreakdown map[DependencyKind]int
	DependentClasses    []string
	RiskLevel           domain.RiskLevel
	IsAbstract          bool
}

// Config holds configuration for CBO analysis.
type Config struct {
	LowThreshold    int
	MediumThreshold int
	ExcludePatterns []string
}

// DefaultConfig returns the default CBO configuration.
func DefaultConfig() Config {
	return Config{
		LowThreshold:    domain.DefaultCBOLowThreshold,
		MediumThreshold: domain.DefaultCBOMediumThreshold,
	}
}

// ComputeCBO computes Coupling Between Objects for each class.
func ComputeCBO(classes []*ClassInfo, config Config) []*Result {
	results := make([]*Result, 0, len(classes))

	for _, class := range classes {
		if class == nil {
			continue
		}
		if len(config.ExcludePatterns) > 0 && MatchesPattern(class.Name, config.ExcludePatterns) {
			continue
		}

		// Count unique dependent classes and breakdown by kind
		uniqueClasses := make(map[string]bool)
		breakdown := make(map[DependencyKind]int)

		for _, dep := range class.Dependencies {
			if dep.ClassName != class.Name { // exclude self-references
				uniqueClasses[dep.ClassName] = true
				breakdown[dep.Kind]++
			}
		}

		// Sort dependent class names for deterministic output
		depClasses := make([]string, 0, len(uniqueClasses))
		for name := range uniqueClasses {
			depClasses = append(depClasses, name)
		}
		sort.Strings(depClasses)

		cbo := len(uniqueClasses)

		results = append(results, &Result{
			ClassName:           class.Name,
			FilePath:            class.FilePath,
			StartLine:           class.StartLine,
			EndLine:             class.EndLine,
			CouplingCount:       cbo,
			DependencyBreakdown: breakdown,
			DependentClasses:    depClasses,
			RiskLevel:           AssessRisk(cbo, config),
			IsAbstract:          class.IsAbstract,
		})
	}

	return results
}

// AssessRisk determines the risk level based on the CBO count and config thresholds.
func AssessRisk(cbo int, config Config) domain.RiskLevel {
	if cbo <= config.LowThreshold {
		return domain.RiskLevelLow
	}
	if cbo <= config.MediumThreshold {
		return domain.RiskLevelMedium
	}
	return domain.RiskLevelHigh
}

// MatchesPattern checks if a name matches any of the given wildcard patterns.
// Patterns support '*' as a wildcard using filepath.Match semantics.
func MatchesPattern(name string, patterns []string) bool {
	for _, pattern := range patterns {
		// Try matching the full name
		if matched, err := filepath.Match(pattern, name); err == nil && matched {
			return true
		}
		// Also try case-insensitive prefix match for simple patterns
		if !strings.Contains(pattern, "*") && !strings.Contains(pattern, "?") {
			if strings.EqualFold(name, pattern) {
				return true
			}
		}
	}
	return false
}
