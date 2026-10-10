package config

// Default complexity thresholds based on McCabe complexity standards
const (
	// DefaultLowComplexityThreshold defines the upper bound for low complexity functions
	// Functions with complexity <= 9 are considered low risk and easy to maintain
	DefaultLowComplexityThreshold = 9

	// DefaultMediumComplexityThreshold defines the upper bound for medium complexity functions
	// Functions with complexity 10-19 are considered medium risk and may need refactoring
	DefaultMediumComplexityThreshold = 19

	// DefaultMinComplexityFilter defines the minimum complexity to report
	// Functions with complexity >= 1 will be included in reports
	DefaultMinComplexityFilter = 1
)

// DefaultDeadCodeMinSeverity defines the minimum severity level to report.
// Reporting from info up is what the analysis has always done, and the
// health score is calibrated against that full set of findings.
const DefaultDeadCodeMinSeverity = "info"

// DefaultDeadCodeSortBy defines the default sorting criteria
const DefaultDeadCodeSortBy = "severity"

// Config represents the main configuration structure
type Config struct {
	// Complexity holds complexity analysis configuration
	Complexity ComplexityConfig

	// DeadCode holds dead code detection configuration
	DeadCode DeadCodeConfig

	// Output holds output formatting configuration
	Output OutputConfig

	// Analysis holds general analysis configuration
	Analysis AnalysisConfig
}

// ComplexityConfig holds configuration for cyclomatic complexity analysis
type ComplexityConfig struct {
	// LowThreshold is the upper bound for low complexity (inclusive)
	LowThreshold int

	// MediumThreshold is the upper bound for medium complexity (inclusive)
	// Values above this are considered high complexity
	MediumThreshold int

	// ReportUnchanged controls whether to report functions with complexity = 1
	ReportUnchanged bool

	// MaxComplexity is the maximum allowed complexity before failing analysis
	// 0 means no limit
	MaxComplexity int
}

// OutputConfig holds configuration for output formatting
type OutputConfig struct {
	// SortBy specifies how to sort results: name, complexity, risk
	SortBy string

	// MinComplexity is the minimum complexity to report (filters low values)
	MinComplexity int
}

// DeadCodeConfig holds configuration for dead code detection
type DeadCodeConfig struct {
	// MinSeverity is the minimum severity level to report
	MinSeverity string

	// SortBy specifies how to sort results: severity, line, file, function
	SortBy string
}

// AnalysisConfig holds general analysis configuration
type AnalysisConfig struct {
	// IncludePatterns specifies file patterns to include
	IncludePatterns []string

	// ExcludePatterns specifies file patterns to exclude
	ExcludePatterns []string

	// Recursive controls whether to analyze directories recursively
	Recursive bool
}

// DefaultConfig returns the default configuration
func DefaultConfig() *Config {
	return &Config{
		Complexity: ComplexityConfig{
			LowThreshold:    DefaultLowComplexityThreshold,
			MediumThreshold: DefaultMediumComplexityThreshold,
			ReportUnchanged: true,
		},
		DeadCode: DeadCodeConfig{
			MinSeverity: DefaultDeadCodeMinSeverity,
			SortBy:      DefaultDeadCodeSortBy,
		},
		Output: OutputConfig{
			SortBy:        "complexity",
			MinComplexity: DefaultMinComplexityFilter,
		},
		Analysis: AnalysisConfig{
			IncludePatterns: []string{
				"**/*.js", "**/*.ts", "**/*.jsx", "**/*.tsx",
				"**/*.mjs", "**/*.cjs", "**/*.mts", "**/*.cts",
			},
			ExcludePatterns: []string{
				// Package managers and dependencies
				"node_modules",
				"bower_components",
				"jspm_packages",
				// Vendored / third-party code
				"vendor",
				"assets",
				"overrides",
				"third_party",
				"third-party",
				"extern",
				"external",
				// Build outputs
				"dist",
				"build",
				"out",
				".output",
				// Framework-specific
				".next",
				".nuxt",
				".vercel",
				// Cache directories
				".cache",
				".turbo",
				"coverage",
				// Version control
				".git",
				// Minified and bundled files
				"*.min.js",
				"*.min.mjs",
				"*.min.cjs",
				"*.bundle.js",
				// Source maps
				"*.map",
			},
			Recursive: true,
		},
	}
}

// AssessRiskLevel determines risk level based on complexity and thresholds
func (c *ComplexityConfig) AssessRiskLevel(complexity int) string {
	if complexity <= c.LowThreshold {
		return "low"
	} else if complexity <= c.MediumThreshold {
		return "medium"
	}
	return "high"
}
