package config

import (
	"testing"
)

func TestComplexityConfig_AssessRiskLevel(t *testing.T) {
	config := &ComplexityConfig{
		LowThreshold:    5,
		MediumThreshold: 10,
	}

	tests := []struct {
		complexity int
		expected   string
	}{
		{1, "low"},
		{5, "low"},
		{6, "medium"},
		{10, "medium"},
		{11, "high"},
		{100, "high"},
	}

	for _, tc := range tests {
		result := config.AssessRiskLevel(tc.complexity)
		if result != tc.expected {
			t.Errorf("AssessRiskLevel(%d) = %s, expected %s", tc.complexity, result, tc.expected)
		}
	}
}

func TestDefaultConstants(t *testing.T) {
	// Verify constants have expected values
	if DefaultLowComplexityThreshold != 9 {
		t.Errorf("DefaultLowComplexityThreshold should be 9, got %d", DefaultLowComplexityThreshold)
	}
	if DefaultMediumComplexityThreshold != 19 {
		t.Errorf("DefaultMediumComplexityThreshold should be 19, got %d", DefaultMediumComplexityThreshold)
	}
	if DefaultMinComplexityFilter != 1 {
		t.Errorf("DefaultMinComplexityFilter should be 1, got %d", DefaultMinComplexityFilter)
	}
	if DefaultDeadCodeMinSeverity != "info" {
		t.Errorf("DefaultDeadCodeMinSeverity should be 'info', got '%s'", DefaultDeadCodeMinSeverity)
	}
	if DefaultDeadCodeSortBy != "severity" {
		t.Errorf("DefaultDeadCodeSortBy should be 'severity', got '%s'", DefaultDeadCodeSortBy)
	}
}

func TestAnalysisConfig_Defaults(t *testing.T) {
	config := DefaultConfig()

	// Check include patterns
	hasJsPattern := false
	for _, pattern := range config.Analysis.IncludePatterns {
		if pattern == "**/*.js" {
			hasJsPattern = true
			break
		}
	}
	if !hasJsPattern {
		t.Error("Include patterns should contain **/*.js")
	}

	// Check exclude patterns
	hasNodeModules := false
	for _, pattern := range config.Analysis.ExcludePatterns {
		if pattern == "node_modules" {
			hasNodeModules = true
			break
		}
	}
	if !hasNodeModules {
		t.Error("Exclude patterns should contain node_modules")
	}
}
