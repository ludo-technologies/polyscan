package domain

import (
	"context"
	"io"
)

// CBORequest represents a request for CBO (Coupling Between Objects) analysis
type CBORequest struct {
	// Input files or directories to analyze
	Paths []string

	// Output configuration
	OutputFormat OutputFormat
	OutputWriter io.Writer
	OutputPath   string // Path to save output file (for HTML format)
	NoOpen       bool   // Don't auto-open HTML in browser
	ShowDetails  bool

	// Filtering and sorting
	MinCBO    int
	MaxCBO    int // 0 means no limit
	SortBy    SortCriteria
	ShowZeros *bool // Include classes with CBO = 0

	// CBO thresholds for risk assessment
	LowThreshold    int // Default: 3 (industry standard)
	MediumThreshold int // Default: 7 (industry standard)

	// Configuration
	ConfigPath string

	// Analysis options
	Recursive       *bool
	IncludePatterns []string
	ExcludePatterns []string

	// Analysis scope
	IncludeBuiltins *bool // Include dependencies on built-in types
	IncludeImports  *bool // Include imported modules in dependency count
}

// CBOMetrics represents detailed CBO metrics for a class
type CBOMetrics struct {
	// Core CBO metric - number of classes this class depends on
	CouplingCount int `json:"coupling_count" yaml:"coupling_count"`

	// Breakdown by dependency type
	InheritanceDependencies     int `json:"inheritance_dependencies" yaml:"inheritance_dependencies"`           // Base classes
	TypeHintDependencies        int `json:"type_hint_dependencies" yaml:"type_hint_dependencies"`               // Type annotations
	InstantiationDependencies   int `json:"instantiation_dependencies" yaml:"instantiation_dependencies"`       // Object creation
	AttributeAccessDependencies int `json:"attribute_access_dependencies" yaml:"attribute_access_dependencies"` // Method calls and attribute access
	ImportDependencies          int `json:"import_dependencies" yaml:"import_dependencies"`                     // Explicitly imported classes

	// Dependency details
	DependentClasses []string `json:"dependent_classes" yaml:"dependent_classes"` // List of class names this class depends on
}

// ClassCoupling represents CBO analysis result for a single class
type ClassCoupling struct {
	// Class identification
	Name     string `json:"name" yaml:"name"`
	FilePath string `json:"file_path" yaml:"file_path"`
	// Language names the language of a type the generic engine measured;
	// the JavaScript/TypeScript analysis, whose unit is the file, leaves it
	// empty.
	Language  string `json:"language,omitempty" yaml:"language,omitempty"`
	StartLine int    `json:"start_line" yaml:"start_line"`
	EndLine   int    `json:"end_line" yaml:"end_line"`

	// CBO metrics
	Metrics CBOMetrics `json:"metrics" yaml:"metrics"`

	// Risk assessment
	RiskLevel RiskLevel `json:"risk_level" yaml:"risk_level"`

	// Additional context
	IsAbstract  bool     `json:"is_abstract" yaml:"is_abstract"`
	BaseClasses []string `json:"base_classes" yaml:"base_classes"`
}

// CBOSummary represents aggregate CBO statistics
type CBOSummary struct {
	TotalClasses    int     `json:"total_classes" yaml:"total_classes"`
	AverageCBO      float64 `json:"average_cbo" yaml:"average_cbo"`
	MaxCBO          int     `json:"max_cbo" yaml:"max_cbo"`
	MinCBO          int     `json:"min_cbo" yaml:"min_cbo"`
	ClassesAnalyzed int     `json:"classes_analyzed" yaml:"classes_analyzed"`
	FilesAnalyzed   int     `json:"files_analyzed" yaml:"files_analyzed"`

	// Risk distribution
	LowRiskClasses    int `json:"low_risk_classes" yaml:"low_risk_classes"`
	MediumRiskClasses int `json:"medium_risk_classes" yaml:"medium_risk_classes"`
	HighRiskClasses   int `json:"high_risk_classes" yaml:"high_risk_classes"`

	// CBO distribution
	CBODistribution map[string]int `json:"cbo_distribution" yaml:"cbo_distribution"`

	// Most coupled classes (top 10)
	MostCoupledClasses []ClassCoupling `json:"most_coupled_classes" yaml:"most_coupled_classes"`

	// Classes with highest impact (most depended upon)
	MostDependedUponClasses []string `json:"most_depended_upon_classes" yaml:"most_depended_upon_classes"`
}

// CBOResponse represents the complete CBO analysis result
type CBOResponse struct {
	// Analysis results
	Classes []ClassCoupling `json:"classes" yaml:"classes"`
	Summary CBOSummary      `json:"summary" yaml:"summary"`

	// Warnings and issues
	Warnings []string `json:"warnings" yaml:"warnings"`
	Errors   []string `json:"errors" yaml:"errors"`

	// Metadata
	GeneratedAt string      `json:"generated_at" yaml:"generated_at"`
	Version     string      `json:"version" yaml:"version"`
	Config      interface{} `json:"config" yaml:"config"` // Configuration used for analysis
}

// CBOService defines the core business logic for CBO analysis
type CBOService interface {
	// Analyze performs CBO analysis on the given request
	Analyze(ctx context.Context, req CBORequest) (*CBOResponse, error)

	// AnalyzeFile analyzes a single JavaScript/TypeScript file
	AnalyzeFile(ctx context.Context, filePath string, req CBORequest) (*CBOResponse, error)
}

// CBOOutputFormatter defines the interface for formatting CBO analysis results
type CBOOutputFormatter interface {
	// Format formats the analysis response according to the specified format
	Format(response *CBOResponse, format OutputFormat) (string, error)

	// Write writes the formatted output to the writer
	Write(response *CBOResponse, format OutputFormat, writer io.Writer) error
}

// CBOAnalysisOptions provides configuration for CBO analysis behavior
type CBOAnalysisOptions struct {
	// Include system and built-in dependencies
	IncludeBuiltins bool

	// Maximum depth for dependency resolution
	MaxDependencyDepth int

	// Exclude patterns for class names
	ExcludeClassPatterns []string

	// Only analyze public classes (exclude private classes starting with _)
	PublicClassesOnly bool
}

// DefaultCBORequest returns a CBORequest with default values
func DefaultCBORequest() *CBORequest {
	return &CBORequest{
		OutputFormat:    OutputFormatText,
		ShowDetails:     false,
		MinCBO:          0,
		MaxCBO:          0,              // No limit
		SortBy:          SortByCoupling, // Sort by CBO value
		ShowZeros:       BoolPtr(false),
		LowThreshold:    3, // Industry standard: CBO <= 3 is low risk
		MediumThreshold: 7, // Industry standard: 3 < CBO <= 7 is medium risk
		Recursive:       BoolPtr(true),
		IncludeBuiltins: BoolPtr(false),
		IncludeImports:  BoolPtr(true),
		IncludePatterns: []string{"**/*.py"},
		ExcludePatterns: []string{},
	}
}
