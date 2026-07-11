package domain

import "fmt"

// RiskLevel represents the risk level of a metric.
type RiskLevel string

const (
	RiskLevelLow    RiskLevel = "low"
	RiskLevelMedium RiskLevel = "medium"
	RiskLevelHigh   RiskLevel = "high"
)

// OutputFormat represents an output format type.
type OutputFormat string

const (
	OutputFormatText OutputFormat = "text"
	OutputFormatJSON OutputFormat = "json"
	OutputFormatYAML OutputFormat = "yaml"
	OutputFormatCSV  OutputFormat = "csv"
	OutputFormatHTML OutputFormat = "html"
	OutputFormatDOT  OutputFormat = "dot"
)

// SortCriteria represents a sorting criterion for results.
type SortCriteria string

const (
	SortByComplexity SortCriteria = "complexity"
	SortByName       SortCriteria = "name"
	SortByRisk       SortCriteria = "risk"
	SortBySimilarity SortCriteria = "similarity"
	SortBySize       SortCriteria = "size"
	SortByLocation   SortCriteria = "location"
	SortByCoupling   SortCriteria = "coupling"
	SortByCohesion   SortCriteria = "cohesion"
)

// CloneType represents the type of code clone (Type-1 through Type-4).
type CloneType int

const (
	Type1Clone CloneType = 1 // Exact clones (identical except whitespace/comments)
	Type2Clone CloneType = 2 // Renamed/parameterized clones
	Type3Clone CloneType = 3 // Near-miss clones (statements added/removed)
	Type4Clone CloneType = 4 // Semantic clones (different syntax, same behavior)
)

// String returns the string representation of a CloneType.
func (ct CloneType) String() string {
	switch ct {
	case Type1Clone:
		return "Type-1"
	case Type2Clone:
		return "Type-2"
	case Type3Clone:
		return "Type-3"
	case Type4Clone:
		return "Type-4"
	default:
		return fmt.Sprintf("Unknown(%d)", int(ct))
	}
}

// CloneTypeNames maps clone types to their short names.
var CloneTypeNames = map[CloneType]string{
	Type1Clone: "Exact",
	Type2Clone: "Renamed",
	Type3Clone: "Near-miss",
	Type4Clone: "Semantic",
}

// CloneTypeDescriptions maps clone types to their descriptions.
var CloneTypeDescriptions = map[CloneType]string{
	Type1Clone: "Identical code fragments except for whitespace and comments",
	Type2Clone: "Structurally identical with renamed identifiers or changed literals",
	Type3Clone: "Near-miss clones with added, removed, or modified statements",
	Type4Clone: "Semantically similar code with different syntactic structure",
}
