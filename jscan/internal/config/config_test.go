package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDefaultConfig(t *testing.T) {
	config := DefaultConfig()

	if config == nil {
		t.Fatal("DefaultConfig should not return nil")
	}

	// Verify complexity defaults
	if config.Complexity.LowThreshold != DefaultLowComplexityThreshold {
		t.Errorf("Expected LowThreshold %d, got %d", DefaultLowComplexityThreshold, config.Complexity.LowThreshold)
	}
	if config.Complexity.MediumThreshold != DefaultMediumComplexityThreshold {
		t.Errorf("Expected MediumThreshold %d, got %d", DefaultMediumComplexityThreshold, config.Complexity.MediumThreshold)
	}
	if !config.Complexity.Enabled {
		t.Error("Complexity should be enabled by default")
	}
	if !config.Complexity.ReportUnchanged {
		t.Error("ReportUnchanged should be true by default")
	}

	// Verify dead code defaults
	if !config.DeadCode.Enabled {
		t.Error("DeadCode should be enabled by default")
	}
	if config.DeadCode.MinSeverity != DefaultDeadCodeMinSeverity {
		t.Errorf("Expected MinSeverity %s, got %s", DefaultDeadCodeMinSeverity, config.DeadCode.MinSeverity)
	}
	if config.DeadCode.ContextLines != DefaultDeadCodeContextLines {
		t.Errorf("Expected ContextLines %d, got %d", DefaultDeadCodeContextLines, config.DeadCode.ContextLines)
	}

	// Verify output defaults
	if config.Output.Format != "text" {
		t.Errorf("Expected Format 'text', got '%s'", config.Output.Format)
	}
	if config.Output.SortBy != "complexity" {
		t.Errorf("Expected SortBy 'complexity', got '%s'", config.Output.SortBy)
	}

	// Verify analysis defaults
	if !config.Analysis.Recursive {
		t.Error("Recursive should be true by default")
	}
	if len(config.Analysis.IncludePatterns) == 0 {
		t.Error("IncludePatterns should not be empty")
	}
	if len(config.Analysis.ExcludePatterns) == 0 {
		t.Error("ExcludePatterns should not be empty")
	}
}

func TestConfig_Validate_Valid(t *testing.T) {
	config := DefaultConfig()

	err := config.Validate()
	if err != nil {
		t.Errorf("Default config should be valid, got error: %v", err)
	}
}

func TestConfig_Validate_InvalidLowThreshold(t *testing.T) {
	config := DefaultConfig()
	config.Complexity.LowThreshold = 0

	err := config.Validate()
	if err == nil {
		t.Error("Expected error for LowThreshold < 1")
	}
}

func TestConfig_Validate_InvalidMediumThreshold(t *testing.T) {
	config := DefaultConfig()
	config.Complexity.MediumThreshold = config.Complexity.LowThreshold

	err := config.Validate()
	if err == nil {
		t.Error("Expected error for MediumThreshold <= LowThreshold")
	}
}

func TestConfig_Validate_InvalidMaxComplexity(t *testing.T) {
	config := DefaultConfig()
	config.Complexity.MaxComplexity = -1

	err := config.Validate()
	if err == nil {
		t.Error("Expected error for MaxComplexity < 0")
	}
}

func TestConfig_Validate_MaxComplexityTooLow(t *testing.T) {
	config := DefaultConfig()
	config.Complexity.MaxComplexity = config.Complexity.MediumThreshold

	err := config.Validate()
	if err == nil {
		t.Error("Expected error for MaxComplexity <= MediumThreshold")
	}
}

func TestConfig_Validate_InvalidOutputFormat(t *testing.T) {
	config := DefaultConfig()
	config.Output.Format = "xml"

	err := config.Validate()
	if err == nil {
		t.Error("Expected error for invalid output format")
	}
}

func TestConfig_Validate_InvalidSortBy(t *testing.T) {
	config := DefaultConfig()
	config.Output.SortBy = "invalid"

	err := config.Validate()
	if err == nil {
		t.Error("Expected error for invalid sort_by")
	}
}

func TestConfig_Validate_InvalidMinComplexity(t *testing.T) {
	config := DefaultConfig()
	config.Output.MinComplexity = 0

	err := config.Validate()
	if err == nil {
		t.Error("Expected error for MinComplexity < 1")
	}
}

func TestConfig_Validate_EmptyIncludePatterns(t *testing.T) {
	config := DefaultConfig()
	config.Analysis.IncludePatterns = []string{}

	err := config.Validate()
	if err == nil {
		t.Error("Expected error for empty include patterns")
	}
}

func TestConfig_Validate_InvalidDeadCodeSeverity(t *testing.T) {
	config := DefaultConfig()
	config.DeadCode.MinSeverity = "invalid"

	err := config.Validate()
	if err == nil {
		t.Error("Expected error for invalid dead code severity")
	}
}

func TestConfig_Validate_InvalidContextLines(t *testing.T) {
	config := DefaultConfig()
	config.DeadCode.ContextLines = -1

	err := config.Validate()
	if err == nil {
		t.Error("Expected error for negative context lines")
	}

	config.DeadCode.ContextLines = 25
	err = config.Validate()
	if err == nil {
		t.Error("Expected error for context lines > 20")
	}
}

func TestConfig_Validate_InvalidDeadCodeSortBy(t *testing.T) {
	config := DefaultConfig()
	config.DeadCode.SortBy = "invalid"

	err := config.Validate()
	if err == nil {
		t.Error("Expected error for invalid dead code sort_by")
	}
}

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

func TestComplexityConfig_ShouldReport(t *testing.T) {
	// Enabled config
	enabledConfig := &ComplexityConfig{
		Enabled:         true,
		ReportUnchanged: true,
	}

	if !enabledConfig.ShouldReport(5) {
		t.Error("Should report complexity 5 when enabled")
	}
	if !enabledConfig.ShouldReport(1) {
		t.Error("Should report complexity 1 when ReportUnchanged is true")
	}

	// Disabled config
	disabledConfig := &ComplexityConfig{
		Enabled: false,
	}
	if disabledConfig.ShouldReport(5) {
		t.Error("Should not report when disabled")
	}

	// Report unchanged = false
	noUnchangedConfig := &ComplexityConfig{
		Enabled:         true,
		ReportUnchanged: false,
	}
	if noUnchangedConfig.ShouldReport(1) {
		t.Error("Should not report complexity 1 when ReportUnchanged is false")
	}
	if !noUnchangedConfig.ShouldReport(5) {
		t.Error("Should report complexity > 1 even when ReportUnchanged is false")
	}
}

func TestComplexityConfig_ExceedsMaxComplexity(t *testing.T) {
	// No limit
	noLimitConfig := &ComplexityConfig{
		MaxComplexity: 0,
	}
	if noLimitConfig.ExceedsMaxComplexity(100) {
		t.Error("Should not exceed when MaxComplexity is 0 (no limit)")
	}

	// With limit
	limitConfig := &ComplexityConfig{
		MaxComplexity: 20,
	}
	if limitConfig.ExceedsMaxComplexity(15) {
		t.Error("15 should not exceed max of 20")
	}
	if limitConfig.ExceedsMaxComplexity(20) {
		t.Error("20 should not exceed max of 20")
	}
	if !limitConfig.ExceedsMaxComplexity(25) {
		t.Error("25 should exceed max of 20")
	}
}

func TestDeadCodeConfig_ShouldDetectDeadCode(t *testing.T) {
	enabled := &DeadCodeConfig{Enabled: true}
	if !enabled.ShouldDetectDeadCode() {
		t.Error("Should detect when enabled")
	}

	disabled := &DeadCodeConfig{Enabled: false}
	if disabled.ShouldDetectDeadCode() {
		t.Error("Should not detect when disabled")
	}
}

func TestDeadCodeConfig_GetMinSeverityLevel(t *testing.T) {
	tests := []struct {
		severity string
		level    int
	}{
		{"info", 1},
		{"warning", 2},
		{"critical", 3},
		{"unknown", 2}, // Default to warning
	}

	for _, tc := range tests {
		config := &DeadCodeConfig{MinSeverity: tc.severity}
		result := config.GetMinSeverityLevel()
		if result != tc.level {
			t.Errorf("GetMinSeverityLevel(%s) = %d, expected %d", tc.severity, result, tc.level)
		}
	}
}

func TestLoadConfig_Default(t *testing.T) {
	// Load with empty path should return default
	config, err := LoadConfig("")
	if err != nil {
		t.Fatalf("LoadConfig with empty path failed: %v", err)
	}
	if config == nil {
		t.Fatal("Config should not be nil")
	}

	// Verify it matches default
	defaultCfg := DefaultConfig()
	if config.Complexity.LowThreshold != defaultCfg.Complexity.LowThreshold {
		t.Error("Loaded config should match default")
	}
}

func TestLoadConfig_NonExistent(t *testing.T) {
	_, err := LoadConfig("/nonexistent/path/config.yaml")
	if err == nil {
		t.Error("Expected error for non-existent config file")
	}
}

func TestSearchConfigInDirectory(t *testing.T) {
	// Create temp directory with config file
	tempDir, err := os.MkdirTemp("", "config_test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Create a config file
	configPath := filepath.Join(tempDir, "jscan.yaml")
	err = os.WriteFile(configPath, []byte("complexity:\n  low_threshold: 5"), 0644)
	if err != nil {
		t.Fatalf("Failed to write config file: %v", err)
	}

	// Search for config
	candidates := []string{"jscan.yaml", "jscan.yml"}
	result := searchConfigInDirectory(tempDir, candidates)

	if result != configPath {
		t.Errorf("Expected %s, got %s", configPath, result)
	}

	// Search in empty directory
	emptyDir, _ := os.MkdirTemp("", "empty_test")
	defer os.RemoveAll(emptyDir)

	result = searchConfigInDirectory(emptyDir, candidates)
	if result != "" {
		t.Error("Expected empty string for directory without config")
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
	if DefaultMaxComplexityLimit != 0 {
		t.Errorf("DefaultMaxComplexityLimit should be 0, got %d", DefaultMaxComplexityLimit)
	}
	if DefaultDeadCodeMinSeverity != "warning" {
		t.Errorf("DefaultDeadCodeMinSeverity should be 'warning', got '%s'", DefaultDeadCodeMinSeverity)
	}
	if DefaultDeadCodeContextLines != 3 {
		t.Errorf("DefaultDeadCodeContextLines should be 3, got %d", DefaultDeadCodeContextLines)
	}
	if DefaultDeadCodeSortBy != "severity" {
		t.Errorf("DefaultDeadCodeSortBy should be 'severity', got '%s'", DefaultDeadCodeSortBy)
	}
}

func TestConfig_ValidOutputFormats(t *testing.T) {
	config := DefaultConfig()
	validFormats := []string{"text", "json", "yaml", "csv", "html"}

	for _, format := range validFormats {
		config.Output.Format = format
		err := config.Validate()
		if err != nil {
			t.Errorf("Format '%s' should be valid, got error: %v", format, err)
		}
	}
}

func TestConfig_ValidSortOptions(t *testing.T) {
	config := DefaultConfig()
	validSortOptions := []string{"name", "complexity", "risk"}

	for _, sortBy := range validSortOptions {
		config.Output.SortBy = sortBy
		err := config.Validate()
		if err != nil {
			t.Errorf("SortBy '%s' should be valid, got error: %v", sortBy, err)
		}
	}
}

func TestConfig_ValidDeadCodeSeverities(t *testing.T) {
	config := DefaultConfig()
	validSeverities := []string{"critical", "warning", "info"}

	for _, severity := range validSeverities {
		config.DeadCode.MinSeverity = severity
		err := config.Validate()
		if err != nil {
			t.Errorf("Severity '%s' should be valid, got error: %v", severity, err)
		}
	}
}

func TestConfig_ValidDeadCodeSortBy(t *testing.T) {
	config := DefaultConfig()
	validSortOptions := []string{"severity", "line", "file", "function"}

	for _, sortBy := range validSortOptions {
		config.DeadCode.SortBy = sortBy
		err := config.Validate()
		if err != nil {
			t.Errorf("DeadCode SortBy '%s' should be valid, got error: %v", sortBy, err)
		}
	}
}

func TestLoadConfigWithTarget_EmptyPaths(t *testing.T) {
	// Both paths empty - should use defaults
	config, err := LoadConfigWithTarget("", "")
	if err != nil {
		t.Fatalf("LoadConfigWithTarget failed: %v", err)
	}
	if config == nil {
		t.Fatal("Config should not be nil")
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
