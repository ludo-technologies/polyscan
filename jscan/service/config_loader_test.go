package service

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/ludo-technologies/polyscan/jscan/domain"
)

func TestNewConfigurationLoader(t *testing.T) {
	loader := NewConfigurationLoader()

	if loader == nil {
		t.Fatal("NewConfigurationLoader should not return nil")
	}
}

func TestConfigurationLoader_LoadConfig_NonExistent(t *testing.T) {
	loader := NewConfigurationLoader()

	_, err := loader.LoadConfig("/nonexistent/config.json")
	if err == nil {
		t.Error("LoadConfig should return error for nonexistent file")
	}
}

func TestConfigurationLoader_LoadConfig_InvalidJSON(t *testing.T) {
	tempDir := t.TempDir()
	configFile := filepath.Join(tempDir, "config.json")
	if err := os.WriteFile(configFile, []byte("invalid json"), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	loader := NewConfigurationLoader()

	_, err := loader.LoadConfig(configFile)
	if err == nil {
		t.Error("LoadConfig should return error for invalid JSON")
	}
}

func TestConfigurationLoader_LoadConfig_Valid(t *testing.T) {
	tempDir := t.TempDir()
	configFile := filepath.Join(tempDir, "config.json")
	content := `{
		"complexity": {
			"low_threshold": 5,
			"medium_threshold": 10
		},
		"output": {
			"format": "json",
			"show_details": true,
			"sort_by": "name",
			"min_complexity": 2
		},
		"analysis": {
			"recursive": true
		}
	}`
	if err := os.WriteFile(configFile, []byte(content), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	loader := NewConfigurationLoader()

	req, err := loader.LoadConfig(configFile)
	if err != nil {
		t.Fatalf("LoadConfig should not return error: %v", err)
	}

	if req == nil {
		t.Fatal("Request should not be nil")
	}

	if req.LowThreshold != 5 {
		t.Errorf("LowThreshold should be 5, got %d", req.LowThreshold)
	}
	if req.MediumThreshold != 10 {
		t.Errorf("MediumThreshold should be 10, got %d", req.MediumThreshold)
	}
	if req.OutputFormat != "json" {
		t.Errorf("OutputFormat should be 'json', got '%s'", req.OutputFormat)
	}
	if !req.ShowDetails {
		t.Error("ShowDetails should be true")
	}
	if req.SortBy != "name" {
		t.Errorf("SortBy should be 'name', got '%s'", req.SortBy)
	}
	if req.MinComplexity != 2 {
		t.Errorf("MinComplexity should be 2, got %d", req.MinComplexity)
	}
	if !req.Recursive {
		t.Error("Recursive should be true")
	}
}

func TestConfigurationLoader_LoadDefaultConfig(t *testing.T) {
	loader := NewConfigurationLoader()

	req := loader.LoadDefaultConfig()

	if req == nil {
		t.Fatal("LoadDefaultConfig should not return nil")
	}

	// Should return default configuration values
	if req.LowThreshold <= 0 {
		t.Error("LowThreshold should be positive")
	}
	if req.MediumThreshold <= req.LowThreshold {
		t.Error("MediumThreshold should be greater than LowThreshold")
	}
}

func TestConfigurationLoader_FindDefaultConfigFile_NotFound(t *testing.T) {
	// Change to temp directory with no config files
	tempDir := t.TempDir()
	origDir, _ := os.Getwd()
	defer func() { _ = os.Chdir(origDir) }()

	if err := os.Chdir(tempDir); err != nil {
		t.Fatalf("Failed to change directory: %v", err)
	}

	loader := NewConfigurationLoader()

	configFile := loader.FindDefaultConfigFile()

	if configFile != "" {
		t.Errorf("Should not find config file in empty directory, got '%s'", configFile)
	}
}

func TestConfigurationLoader_FindDefaultConfigFile_Found(t *testing.T) {
	tempDir := t.TempDir()
	configFile := filepath.Join(tempDir, "jscan.config.json")
	if err := os.WriteFile(configFile, []byte("{}"), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	origDir, _ := os.Getwd()
	defer func() { _ = os.Chdir(origDir) }()

	if err := os.Chdir(tempDir); err != nil {
		t.Fatalf("Failed to change directory: %v", err)
	}

	loader := NewConfigurationLoader()

	found := loader.FindDefaultConfigFile()

	if found != "jscan.config.json" {
		t.Errorf("Should find 'jscan.config.json', got '%s'", found)
	}
}

func TestConfigurationLoader_FindDefaultConfigFile_AlternativeNames(t *testing.T) {
	tempDir := t.TempDir()

	// Test .jscanrc.json
	configFile := filepath.Join(tempDir, ".jscanrc.json")
	if err := os.WriteFile(configFile, []byte("{}"), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	origDir, _ := os.Getwd()
	defer func() { _ = os.Chdir(origDir) }()

	if err := os.Chdir(tempDir); err != nil {
		t.Fatalf("Failed to change directory: %v", err)
	}

	loader := NewConfigurationLoader()

	found := loader.FindDefaultConfigFile()

	if found != ".jscanrc.json" {
		t.Errorf("Should find '.jscanrc.json', got '%s'", found)
	}
}

func TestConfigurationLoader_MergeConfig_Paths(t *testing.T) {
	loader := NewConfigurationLoader()

	base := &domain.ComplexityRequest{
		Paths: []string{"original.js"},
	}

	override := &domain.ComplexityRequest{
		Paths: []string{"new1.js", "new2.js"},
	}

	merged := loader.MergeConfig(base, override)

	if len(merged.Paths) != 2 {
		t.Errorf("Should have 2 paths, got %d", len(merged.Paths))
	}
	if merged.Paths[0] != "new1.js" {
		t.Error("First path should be 'new1.js'")
	}
}

func TestConfigurationLoader_MergeConfig_OutputFormat(t *testing.T) {
	loader := NewConfigurationLoader()

	base := &domain.ComplexityRequest{
		OutputFormat: "text",
	}

	override := &domain.ComplexityRequest{
		OutputFormat: "json",
	}

	merged := loader.MergeConfig(base, override)

	if merged.OutputFormat != "json" {
		t.Errorf("OutputFormat should be 'json', got '%s'", merged.OutputFormat)
	}
}

func TestConfigurationLoader_MergeConfig_ShowDetails(t *testing.T) {
	loader := NewConfigurationLoader()

	base := &domain.ComplexityRequest{
		ShowDetails: false,
	}

	override := &domain.ComplexityRequest{
		ShowDetails: true,
	}

	merged := loader.MergeConfig(base, override)

	if !merged.ShowDetails {
		t.Error("ShowDetails should be true")
	}
}

func TestConfigurationLoader_MergeConfig_MinComplexity(t *testing.T) {
	loader := NewConfigurationLoader()

	base := &domain.ComplexityRequest{
		MinComplexity: 1, // default
	}

	override := &domain.ComplexityRequest{
		MinComplexity: 5, // non-default
	}

	merged := loader.MergeConfig(base, override)

	if merged.MinComplexity != 5 {
		t.Errorf("MinComplexity should be 5, got %d", merged.MinComplexity)
	}
}

func TestConfigurationLoader_MergeConfig_MaxComplexity(t *testing.T) {
	loader := NewConfigurationLoader()

	base := &domain.ComplexityRequest{
		MaxComplexity: 0, // default (no limit)
	}

	override := &domain.ComplexityRequest{
		MaxComplexity: 50,
	}

	merged := loader.MergeConfig(base, override)

	if merged.MaxComplexity != 50 {
		t.Errorf("MaxComplexity should be 50, got %d", merged.MaxComplexity)
	}
}

func TestConfigurationLoader_MergeConfig_SortBy(t *testing.T) {
	loader := NewConfigurationLoader()

	base := &domain.ComplexityRequest{
		SortBy: domain.SortByComplexity, // default
	}

	override := &domain.ComplexityRequest{
		SortBy: domain.SortByName,
	}

	merged := loader.MergeConfig(base, override)

	if merged.SortBy != domain.SortByName {
		t.Errorf("SortBy should be 'name', got '%s'", merged.SortBy)
	}
}

func TestConfigurationLoader_MergeConfig_Thresholds(t *testing.T) {
	loader := NewConfigurationLoader()

	base := &domain.ComplexityRequest{
		LowThreshold:    9,  // default
		MediumThreshold: 19, // default
	}

	override := &domain.ComplexityRequest{
		LowThreshold:    5,
		MediumThreshold: 15,
	}

	merged := loader.MergeConfig(base, override)

	if merged.LowThreshold != 5 {
		t.Errorf("LowThreshold should be 5, got %d", merged.LowThreshold)
	}
	if merged.MediumThreshold != 15 {
		t.Errorf("MediumThreshold should be 15, got %d", merged.MediumThreshold)
	}
}

func TestConfigurationLoader_MergeConfig_ConfigPath(t *testing.T) {
	loader := NewConfigurationLoader()

	base := &domain.ComplexityRequest{
		ConfigPath: "",
	}

	override := &domain.ComplexityRequest{
		ConfigPath: "/path/to/config.json",
	}

	merged := loader.MergeConfig(base, override)

	if merged.ConfigPath != "/path/to/config.json" {
		t.Errorf("ConfigPath should be '/path/to/config.json', got '%s'", merged.ConfigPath)
	}
}

func TestConfigurationLoader_MergeConfig_PreserveBase(t *testing.T) {
	loader := NewConfigurationLoader()

	base := &domain.ComplexityRequest{
		LowThreshold:    9,
		MediumThreshold: 19,
		OutputFormat:    "text",
	}

	override := &domain.ComplexityRequest{
		// Empty - should preserve base values
	}

	merged := loader.MergeConfig(base, override)

	if merged.LowThreshold != 9 {
		t.Error("Should preserve base LowThreshold")
	}
	if merged.MediumThreshold != 19 {
		t.Error("Should preserve base MediumThreshold")
	}
	if merged.OutputFormat != "text" {
		t.Error("Should preserve base OutputFormat")
	}
}

func TestConfigurationLoader_ValidateConfig_Valid(t *testing.T) {
	loader := NewConfigurationLoader()

	req := &domain.ComplexityRequest{
		LowThreshold:    5,
		MediumThreshold: 10,
		MinComplexity:   1,
		MaxComplexity:   50,
		OutputFormat:    domain.OutputFormatJSON,
	}

	err := loader.ValidateConfig(req)
	if err != nil {
		t.Errorf("Valid config should not return error: %v", err)
	}
}

func TestConfigurationLoader_ValidateConfig_InvalidLowThreshold(t *testing.T) {
	loader := NewConfigurationLoader()

	req := &domain.ComplexityRequest{
		LowThreshold:    0, // Invalid
		MediumThreshold: 10,
		OutputFormat:    domain.OutputFormatText,
	}

	err := loader.ValidateConfig(req)
	if err == nil {
		t.Error("Should return error for LowThreshold <= 0")
	}
}

func TestConfigurationLoader_ValidateConfig_InvalidMediumThreshold(t *testing.T) {
	loader := NewConfigurationLoader()

	req := &domain.ComplexityRequest{
		LowThreshold:    10,
		MediumThreshold: 5, // Less than low
		OutputFormat:    domain.OutputFormatText,
	}

	err := loader.ValidateConfig(req)
	if err == nil {
		t.Error("Should return error when MediumThreshold <= LowThreshold")
	}
}

func TestConfigurationLoader_ValidateConfig_NegativeMinComplexity(t *testing.T) {
	loader := NewConfigurationLoader()

	req := &domain.ComplexityRequest{
		LowThreshold:    5,
		MediumThreshold: 10,
		MinComplexity:   -1, // Invalid
		OutputFormat:    domain.OutputFormatText,
	}

	err := loader.ValidateConfig(req)
	if err == nil {
		t.Error("Should return error for negative MinComplexity")
	}
}

func TestConfigurationLoader_ValidateConfig_NegativeMaxComplexity(t *testing.T) {
	loader := NewConfigurationLoader()

	req := &domain.ComplexityRequest{
		LowThreshold:    5,
		MediumThreshold: 10,
		MaxComplexity:   -1, // Invalid
		OutputFormat:    domain.OutputFormatText,
	}

	err := loader.ValidateConfig(req)
	if err == nil {
		t.Error("Should return error for negative MaxComplexity")
	}
}

func TestConfigurationLoader_ValidateConfig_MinGreaterThanMax(t *testing.T) {
	loader := NewConfigurationLoader()

	req := &domain.ComplexityRequest{
		LowThreshold:    5,
		MediumThreshold: 10,
		MinComplexity:   50,
		MaxComplexity:   25, // Less than min
		OutputFormat:    domain.OutputFormatText,
	}

	err := loader.ValidateConfig(req)
	if err == nil {
		t.Error("Should return error when MinComplexity > MaxComplexity")
	}
}

func TestConfigurationLoader_ValidateConfig_InvalidOutputFormat(t *testing.T) {
	loader := NewConfigurationLoader()

	req := &domain.ComplexityRequest{
		LowThreshold:    5,
		MediumThreshold: 10,
		OutputFormat:    "xml", // Invalid
	}

	err := loader.ValidateConfig(req)
	if err == nil {
		t.Error("Should return error for invalid output format")
	}
}

func TestConfigurationLoader_ValidateConfig_ValidFormats(t *testing.T) {
	loader := NewConfigurationLoader()

	validFormats := []domain.OutputFormat{
		domain.OutputFormatText,
		domain.OutputFormatJSON,
		domain.OutputFormatHTML,
		domain.OutputFormatCSV,
	}

	for _, format := range validFormats {
		req := &domain.ComplexityRequest{
			LowThreshold:    5,
			MediumThreshold: 10,
			OutputFormat:    format,
		}

		err := loader.ValidateConfig(req)
		if err != nil {
			t.Errorf("Format '%s' should be valid, got error: %v", format, err)
		}
	}
}

func TestConfigurationLoader_convertToComplexityRequest(t *testing.T) {
	loader := NewConfigurationLoader()

	// Use internal config from package
	// This tests the conversion logic
	req := loader.LoadDefaultConfig()

	// Paths should be empty (set by caller)
	if len(req.Paths) != 0 {
		t.Errorf("Paths should be empty, got %d", len(req.Paths))
	}

	// Should have sensible defaults
	if req.LowThreshold <= 0 {
		t.Error("LowThreshold should be positive")
	}
	if req.MediumThreshold <= 0 {
		t.Error("MediumThreshold should be positive")
	}
}
