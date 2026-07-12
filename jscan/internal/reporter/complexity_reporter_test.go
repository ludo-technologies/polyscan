package reporter

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"

	"github.com/ludo-technologies/polyscan/jscan/internal/config"
	"gopkg.in/yaml.v3"
)

// mockComplexityResult implements ComplexityResult interface for testing
type mockComplexityResult struct {
	complexity   int
	functionName string
	riskLevel    string
	metrics      map[string]int
}

func (m *mockComplexityResult) GetComplexity() int {
	return m.complexity
}

func (m *mockComplexityResult) GetFunctionName() string {
	return m.functionName
}

func (m *mockComplexityResult) GetRiskLevel() string {
	return m.riskLevel
}

func (m *mockComplexityResult) GetDetailedMetrics() map[string]int {
	if m.metrics == nil {
		return map[string]int{}
	}
	return m.metrics
}

func newMockResult(name string, complexity int, risk string) *mockComplexityResult {
	return &mockComplexityResult{
		complexity:   complexity,
		functionName: name,
		riskLevel:    risk,
		metrics: map[string]int{
			"nodes":              complexity * 2,
			"edges":              complexity*2 + 1,
			"if_statements":      complexity / 2,
			"loop_statements":    1,
			"exception_handlers": 0,
			"switch_cases":       0,
		},
	}
}

func TestNewComplexityReporter_NilConfig(t *testing.T) {
	var buf bytes.Buffer
	_, err := NewComplexityReporter(nil, &buf)

	if err == nil {
		t.Error("NewComplexityReporter should return error for nil config")
	}
	if !strings.Contains(err.Error(), "configuration cannot be nil") {
		t.Errorf("Error should mention nil configuration, got: %v", err)
	}
}

func TestNewComplexityReporter_NilWriter(t *testing.T) {
	cfg := config.DefaultConfig()
	_, err := NewComplexityReporter(cfg, nil)

	if err == nil {
		t.Error("NewComplexityReporter should return error for nil writer")
	}
	if !strings.Contains(err.Error(), "writer cannot be nil") {
		t.Errorf("Error should mention nil writer, got: %v", err)
	}
}

func TestNewComplexityReporter_InvalidConfig(t *testing.T) {
	cfg := config.DefaultConfig()
	cfg.Complexity.LowThreshold = 0 // Invalid

	var buf bytes.Buffer
	_, err := NewComplexityReporter(cfg, &buf)

	if err == nil {
		t.Error("NewComplexityReporter should return error for invalid config")
	}
	if !strings.Contains(err.Error(), "invalid configuration") {
		t.Errorf("Error should mention invalid configuration, got: %v", err)
	}
}

func TestNewComplexityReporter_Valid(t *testing.T) {
	cfg := config.DefaultConfig()
	var buf bytes.Buffer

	reporter, err := NewComplexityReporter(cfg, &buf)

	if err != nil {
		t.Fatalf("NewComplexityReporter should not return error: %v", err)
	}
	if reporter == nil {
		t.Fatal("Reporter should not be nil")
	}
}

func TestComplexityReporter_GetWriter(t *testing.T) {
	cfg := config.DefaultConfig()
	var buf bytes.Buffer

	reporter, _ := NewComplexityReporter(cfg, &buf)
	writer := reporter.GetWriter()

	if writer != &buf {
		t.Error("GetWriter should return the writer passed to constructor")
	}
}

func TestComplexityReporter_GenerateReport_EmptyResults(t *testing.T) {
	cfg := config.DefaultConfig()
	var buf bytes.Buffer

	reporter, _ := NewComplexityReporter(cfg, &buf)
	report := reporter.GenerateReport([]ComplexityResult{}, 0)

	if report == nil {
		t.Fatal("Report should not be nil")
	}
	if len(report.Results) != 0 {
		t.Error("Results should be empty")
	}
	if report.Summary.TotalFunctions != 0 {
		t.Error("TotalFunctions should be 0 for empty results")
	}
	if report.Metadata.FilesAnalyzed != 0 {
		t.Error("FilesAnalyzed should be 0")
	}
}

func TestComplexityReporter_GenerateReport_WithResults(t *testing.T) {
	cfg := config.DefaultConfig()
	var buf bytes.Buffer

	reporter, _ := NewComplexityReporter(cfg, &buf)

	results := []ComplexityResult{
		newMockResult("lowFunc", 3, "low"),
		newMockResult("medFunc", 12, "medium"),
		newMockResult("highFunc", 25, "high"),
	}

	report := reporter.GenerateReport(results, 2)

	if report == nil {
		t.Fatal("Report should not be nil")
	}
	if len(report.Results) != 3 {
		t.Errorf("Should have 3 results, got %d", len(report.Results))
	}
	if report.Summary.TotalFunctions != 3 {
		t.Errorf("TotalFunctions should be 3, got %d", report.Summary.TotalFunctions)
	}
	if report.Metadata.FilesAnalyzed != 2 {
		t.Errorf("FilesAnalyzed should be 2, got %d", report.Metadata.FilesAnalyzed)
	}
}

func TestComplexityReporter_GenerateReport_SummaryStats(t *testing.T) {
	cfg := config.DefaultConfig()
	var buf bytes.Buffer

	reporter, _ := NewComplexityReporter(cfg, &buf)

	results := []ComplexityResult{
		newMockResult("func1", 5, "low"),
		newMockResult("func2", 10, "medium"),
		newMockResult("func3", 15, "medium"),
	}

	report := reporter.GenerateReport(results, 1)

	if report.Summary.MinComplexity != 5 {
		t.Errorf("MinComplexity should be 5, got %d", report.Summary.MinComplexity)
	}
	if report.Summary.MaxComplexity != 15 {
		t.Errorf("MaxComplexity should be 15, got %d", report.Summary.MaxComplexity)
	}
	expectedAvg := 10.0 // (5+10+15)/3
	if report.Summary.AverageComplexity != expectedAvg {
		t.Errorf("AverageComplexity should be %.2f, got %.2f", expectedAvg, report.Summary.AverageComplexity)
	}
}

func TestComplexityReporter_GenerateReport_RiskDistribution(t *testing.T) {
	cfg := config.DefaultConfig()
	var buf bytes.Buffer

	reporter, _ := NewComplexityReporter(cfg, &buf)

	results := []ComplexityResult{
		newMockResult("low1", 3, "low"),
		newMockResult("low2", 5, "low"),
		newMockResult("med1", 12, "medium"),
		newMockResult("high1", 25, "high"),
	}

	report := reporter.GenerateReport(results, 1)

	if report.Summary.RiskDistribution.Low != 2 {
		t.Errorf("Low risk count should be 2, got %d", report.Summary.RiskDistribution.Low)
	}
	if report.Summary.RiskDistribution.Medium != 1 {
		t.Errorf("Medium risk count should be 1, got %d", report.Summary.RiskDistribution.Medium)
	}
	if report.Summary.RiskDistribution.High != 1 {
		t.Errorf("High risk count should be 1, got %d", report.Summary.RiskDistribution.High)
	}
}

func TestComplexityReporter_GenerateReport_ComplexityDistribution(t *testing.T) {
	cfg := config.DefaultConfig()
	var buf bytes.Buffer

	reporter, _ := NewComplexityReporter(cfg, &buf)

	results := []ComplexityResult{
		newMockResult("f1", 1, "low"),
		newMockResult("f2", 3, "low"),
		newMockResult("f3", 7, "low"),
		newMockResult("f4", 15, "medium"),
		newMockResult("f5", 25, "high"),
	}

	report := reporter.GenerateReport(results, 1)

	dist := report.Summary.ComplexityDistribution
	if dist["1"] != 1 {
		t.Errorf("Complexity 1 count should be 1, got %d", dist["1"])
	}
	if dist["2-5"] != 1 {
		t.Errorf("Complexity 2-5 count should be 1, got %d", dist["2-5"])
	}
	if dist["6-10"] != 1 {
		t.Errorf("Complexity 6-10 count should be 1, got %d", dist["6-10"])
	}
	if dist["11-20"] != 1 {
		t.Errorf("Complexity 11-20 count should be 1, got %d", dist["11-20"])
	}
	if dist["21+"] != 1 {
		t.Errorf("Complexity 21+ count should be 1, got %d", dist["21+"])
	}
}

func TestComplexityReporter_GenerateReport_Warnings(t *testing.T) {
	cfg := config.DefaultConfig()
	cfg.Complexity.MaxComplexity = 20
	var buf bytes.Buffer

	reporter, _ := NewComplexityReporter(cfg, &buf)

	results := []ComplexityResult{
		newMockResult("normalFunc", 10, "medium"),
		newMockResult("highFunc", 25, "high"),
		newMockResult("veryHighFunc", 30, "high"),
	}

	report := reporter.GenerateReport(results, 1)

	// Should have warnings for high complexity functions
	if len(report.Warnings) == 0 {
		t.Error("Should have warnings for high complexity functions")
	}

	// Check for max complexity exceeded warnings
	foundMaxExceeded := false
	foundHighComplexity := false
	for _, warning := range report.Warnings {
		if warning.Type == "max_complexity_exceeded" {
			foundMaxExceeded = true
		}
		if warning.Type == "high_complexity" {
			foundHighComplexity = true
		}
	}

	if !foundMaxExceeded {
		t.Error("Should have max_complexity_exceeded warning")
	}
	if !foundHighComplexity {
		t.Error("Should have high_complexity warning")
	}
}

func TestComplexityReporter_ReportComplexity_JSON(t *testing.T) {
	cfg := config.DefaultConfig()
	cfg.Output.Format = "json"
	var buf bytes.Buffer

	reporter, _ := NewComplexityReporter(cfg, &buf)

	results := []ComplexityResult{
		newMockResult("testFunc", 5, "low"),
	}

	err := reporter.ReportComplexity(results)
	if err != nil {
		t.Fatalf("ReportComplexity should not return error: %v", err)
	}

	// Verify output is valid JSON
	var report ComplexityReport
	if err := json.Unmarshal(buf.Bytes(), &report); err != nil {
		t.Fatalf("Output should be valid JSON: %v", err)
	}

	if len(report.Results) != 1 {
		t.Errorf("Should have 1 result, got %d", len(report.Results))
	}
	if report.Results[0].FunctionName != "testFunc" {
		t.Errorf("Function name should be 'testFunc', got '%s'", report.Results[0].FunctionName)
	}
}

func TestComplexityReporter_ReportComplexity_YAML(t *testing.T) {
	cfg := config.DefaultConfig()
	cfg.Output.Format = "yaml"
	var buf bytes.Buffer

	reporter, _ := NewComplexityReporter(cfg, &buf)

	results := []ComplexityResult{
		newMockResult("testFunc", 5, "low"),
	}

	err := reporter.ReportComplexity(results)
	if err != nil {
		t.Fatalf("ReportComplexity should not return error: %v", err)
	}

	// Verify output is valid YAML
	var report ComplexityReport
	if err := yaml.Unmarshal(buf.Bytes(), &report); err != nil {
		t.Fatalf("Output should be valid YAML: %v", err)
	}

	if len(report.Results) != 1 {
		t.Errorf("Should have 1 result, got %d", len(report.Results))
	}
}

func TestComplexityReporter_ReportComplexity_CSV(t *testing.T) {
	cfg := config.DefaultConfig()
	cfg.Output.Format = "csv"
	var buf bytes.Buffer

	reporter, _ := NewComplexityReporter(cfg, &buf)

	results := []ComplexityResult{
		newMockResult("func1", 5, "low"),
		newMockResult("func2", 15, "medium"),
	}

	err := reporter.ReportComplexity(results)
	if err != nil {
		t.Fatalf("ReportComplexity should not return error: %v", err)
	}

	output := buf.String()
	lines := strings.Split(strings.TrimSpace(output), "\n")

	// Should have header + 2 data rows
	if len(lines) != 3 {
		t.Errorf("Should have 3 lines (header + 2 rows), got %d", len(lines))
	}

	// Verify header
	if !strings.Contains(lines[0], "Function") {
		t.Error("CSV header should contain 'Function'")
	}
	if !strings.Contains(lines[0], "Complexity") {
		t.Error("CSV header should contain 'Complexity'")
	}

	// Verify data rows
	if !strings.Contains(lines[1], "func") {
		t.Error("Data row should contain function name")
	}
}

func TestComplexityReporter_ReportComplexity_Text(t *testing.T) {
	cfg := config.DefaultConfig()
	cfg.Output.Format = "text"
	var buf bytes.Buffer

	reporter, _ := NewComplexityReporter(cfg, &buf)

	results := []ComplexityResult{
		newMockResult("testFunc", 5, "low"),
	}

	err := reporter.ReportComplexity(results)
	if err != nil {
		t.Fatalf("ReportComplexity should not return error: %v", err)
	}

	output := buf.String()

	if !strings.Contains(output, "Complexity Analysis Report") {
		t.Error("Text output should contain report title")
	}
	if !strings.Contains(output, "Summary") {
		t.Error("Text output should contain summary section")
	}
	if !strings.Contains(output, "Total Functions") {
		t.Error("Text output should contain total functions count")
	}
	if !strings.Contains(output, "testFunc") {
		t.Error("Text output should contain function name")
	}
}

func TestComplexityReporter_ReportComplexity_EmptyResults_Text(t *testing.T) {
	cfg := config.DefaultConfig()
	cfg.Output.Format = "text"
	var buf bytes.Buffer

	reporter, _ := NewComplexityReporter(cfg, &buf)

	err := reporter.ReportComplexity([]ComplexityResult{})
	if err != nil {
		t.Fatalf("ReportComplexity should not return error: %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, "Complexity Analysis Report") {
		t.Error("Text format should contain report header even with empty results")
	}
	if !strings.Contains(output, "Total Functions: 0") {
		t.Error("Should show 0 total functions")
	}
}

func TestComplexityReporter_ReportComplexityWithFileCount(t *testing.T) {
	cfg := config.DefaultConfig()
	cfg.Output.Format = "json"
	var buf bytes.Buffer

	reporter, _ := NewComplexityReporter(cfg, &buf)

	results := []ComplexityResult{
		newMockResult("testFunc", 5, "low"),
	}

	err := reporter.ReportComplexityWithFileCount(results, 10)
	if err != nil {
		t.Fatalf("ReportComplexityWithFileCount should not return error: %v", err)
	}

	var report ComplexityReport
	if err := json.Unmarshal(buf.Bytes(), &report); err != nil {
		t.Fatalf("Output should be valid JSON: %v", err)
	}

	if report.Metadata.FilesAnalyzed != 10 {
		t.Errorf("FilesAnalyzed should be 10, got %d", report.Metadata.FilesAnalyzed)
	}
}

func TestComplexityReporter_filterAndSortResults_MinComplexity(t *testing.T) {
	cfg := config.DefaultConfig()
	cfg.Output.MinComplexity = 10
	var buf bytes.Buffer

	reporter, _ := NewComplexityReporter(cfg, &buf)

	results := []ComplexityResult{
		newMockResult("low", 5, "low"),
		newMockResult("medium", 15, "medium"),
		newMockResult("high", 25, "high"),
	}

	filtered := reporter.filterAndSortResults(results)

	if len(filtered) != 2 {
		t.Errorf("Should filter out functions below min complexity, got %d", len(filtered))
	}
}

func TestComplexityReporter_filterAndSortResults_SortByComplexity(t *testing.T) {
	cfg := config.DefaultConfig()
	cfg.Output.SortBy = "complexity"
	var buf bytes.Buffer

	reporter, _ := NewComplexityReporter(cfg, &buf)

	results := []ComplexityResult{
		newMockResult("low", 5, "low"),
		newMockResult("high", 25, "high"),
		newMockResult("medium", 15, "medium"),
	}

	sorted := reporter.filterAndSortResults(results)

	// Should be sorted descending by complexity
	if sorted[0].GetComplexity() != 25 {
		t.Errorf("First result should have complexity 25, got %d", sorted[0].GetComplexity())
	}
	if sorted[1].GetComplexity() != 15 {
		t.Errorf("Second result should have complexity 15, got %d", sorted[1].GetComplexity())
	}
	if sorted[2].GetComplexity() != 5 {
		t.Errorf("Third result should have complexity 5, got %d", sorted[2].GetComplexity())
	}
}

func TestComplexityReporter_filterAndSortResults_SortByRisk(t *testing.T) {
	cfg := config.DefaultConfig()
	cfg.Output.SortBy = "risk"
	var buf bytes.Buffer

	reporter, _ := NewComplexityReporter(cfg, &buf)

	results := []ComplexityResult{
		newMockResult("low", 5, "low"),
		newMockResult("high", 25, "high"),
		newMockResult("medium", 15, "medium"),
	}

	sorted := reporter.filterAndSortResults(results)

	// Should be sorted by risk level (high > medium > low)
	if sorted[0].GetRiskLevel() != "high" {
		t.Errorf("First result should have high risk, got %s", sorted[0].GetRiskLevel())
	}
	if sorted[1].GetRiskLevel() != "medium" {
		t.Errorf("Second result should have medium risk, got %s", sorted[1].GetRiskLevel())
	}
	if sorted[2].GetRiskLevel() != "low" {
		t.Errorf("Third result should have low risk, got %s", sorted[2].GetRiskLevel())
	}
}

func TestComplexityReporter_filterAndSortResults_SortByName(t *testing.T) {
	cfg := config.DefaultConfig()
	cfg.Output.SortBy = "name"
	var buf bytes.Buffer

	reporter, _ := NewComplexityReporter(cfg, &buf)

	results := []ComplexityResult{
		newMockResult("charlie", 15, "medium"),
		newMockResult("alpha", 5, "low"),
		newMockResult("beta", 25, "high"),
	}

	sorted := reporter.filterAndSortResults(results)

	// Should be sorted alphabetically by name
	if sorted[0].GetFunctionName() != "alpha" {
		t.Errorf("First result should be 'alpha', got %s", sorted[0].GetFunctionName())
	}
	if sorted[1].GetFunctionName() != "beta" {
		t.Errorf("Second result should be 'beta', got %s", sorted[1].GetFunctionName())
	}
	if sorted[2].GetFunctionName() != "charlie" {
		t.Errorf("Third result should be 'charlie', got %s", sorted[2].GetFunctionName())
	}
}

func TestComplexityReporter_compareRiskLevel(t *testing.T) {
	cfg := config.DefaultConfig()
	var buf bytes.Buffer

	reporter, _ := NewComplexityReporter(cfg, &buf)

	// high > medium
	if !reporter.compareRiskLevel("high", "medium") {
		t.Error("high should be greater than medium")
	}

	// medium > low
	if !reporter.compareRiskLevel("medium", "low") {
		t.Error("medium should be greater than low")
	}

	// high > low
	if !reporter.compareRiskLevel("high", "low") {
		t.Error("high should be greater than low")
	}

	// low is not > high
	if reporter.compareRiskLevel("low", "high") {
		t.Error("low should not be greater than high")
	}

	// same level
	if reporter.compareRiskLevel("medium", "medium") {
		t.Error("same levels should not return true")
	}
}

func TestComplexityReporter_getRiskColor(t *testing.T) {
	cfg := config.DefaultConfig()
	var buf bytes.Buffer

	reporter, _ := NewComplexityReporter(cfg, &buf)

	if reporter.getRiskColor("high") != "\033[31m" {
		t.Error("high risk should be red")
	}
	if reporter.getRiskColor("medium") != "\033[33m" {
		t.Error("medium risk should be yellow")
	}
	if reporter.getRiskColor("low") != "\033[32m" {
		t.Error("low risk should be green")
	}
	if reporter.getRiskColor("unknown") != "\033[0m" {
		t.Error("unknown risk should reset color")
	}
}

func TestComplexityReporter_TextOutput_WithDetails(t *testing.T) {
	cfg := config.DefaultConfig()
	cfg.Output.Format = "text"
	cfg.Output.ShowDetails = true
	var buf bytes.Buffer

	reporter, _ := NewComplexityReporter(cfg, &buf)

	results := []ComplexityResult{
		newMockResult("testFunc", 5, "low"),
	}

	err := reporter.ReportComplexity(results)
	if err != nil {
		t.Fatalf("ReportComplexity should not return error: %v", err)
	}

	output := buf.String()

	if !strings.Contains(output, "Nodes") {
		t.Error("Text output with details should contain 'Nodes'")
	}
	if !strings.Contains(output, "Edges") {
		t.Error("Text output with details should contain 'Edges'")
	}
}

func TestComplexityReporter_TextOutput_WithWarnings(t *testing.T) {
	cfg := config.DefaultConfig()
	cfg.Output.Format = "text"
	var buf bytes.Buffer

	reporter, _ := NewComplexityReporter(cfg, &buf)

	results := []ComplexityResult{
		newMockResult("highFunc", 25, "high"),
	}

	err := reporter.ReportComplexity(results)
	if err != nil {
		t.Fatalf("ReportComplexity should not return error: %v", err)
	}

	output := buf.String()

	if !strings.Contains(output, "Warnings") {
		t.Error("Text output should contain warnings section for high complexity")
	}
}

func TestFormatComplexityBrief_EmptyResults(t *testing.T) {
	result := FormatComplexityBrief([]ComplexityResult{})

	if result != "No functions analyzed" {
		t.Errorf("Empty results should return 'No functions analyzed', got '%s'", result)
	}
}

func TestFormatComplexityBrief_WithResults(t *testing.T) {
	results := []ComplexityResult{
		newMockResult("func1", 5, "low"),
		newMockResult("func2", 15, "medium"),
		newMockResult("func3", 25, "high"),
	}

	result := FormatComplexityBrief(results)

	if !strings.Contains(result, "3 functions analyzed") {
		t.Errorf("Should mention 3 functions analyzed, got '%s'", result)
	}
	if !strings.Contains(result, "Avg:") {
		t.Error("Should contain average complexity")
	}
	if !strings.Contains(result, "Max: 25") {
		t.Error("Should contain max complexity of 25")
	}
	if !strings.Contains(result, "High Risk: 1") {
		t.Error("Should contain high risk count of 1")
	}
}

func TestSerializableComplexityResult_Fields(t *testing.T) {
	result := SerializableComplexityResult{
		Complexity:        15,
		FunctionName:      "testFunc",
		RiskLevel:         "medium",
		Nodes:             30,
		Edges:             35,
		IfStatements:      5,
		LoopStatements:    2,
		ExceptionHandlers: 1,
		SwitchCases:       3,
	}

	if result.Complexity != 15 {
		t.Error("Complexity should be 15")
	}
	if result.FunctionName != "testFunc" {
		t.Error("FunctionName should be 'testFunc'")
	}
	if result.RiskLevel != "medium" {
		t.Error("RiskLevel should be 'medium'")
	}
}

func TestReportSummary_Fields(t *testing.T) {
	summary := ReportSummary{
		TotalFunctions:    10,
		AverageComplexity: 12.5,
		MaxComplexity:     25,
		MinComplexity:     3,
		RiskDistribution: RiskDistribution{
			Low:    5,
			Medium: 3,
			High:   2,
		},
		ComplexityDistribution: map[string]int{
			"1-5":  3,
			"6-10": 4,
		},
	}

	if summary.TotalFunctions != 10 {
		t.Error("TotalFunctions should be 10")
	}
	if summary.AverageComplexity != 12.5 {
		t.Error("AverageComplexity should be 12.5")
	}
	if summary.RiskDistribution.High != 2 {
		t.Error("High risk count should be 2")
	}
}

func TestReportWarning_Fields(t *testing.T) {
	warning := ReportWarning{
		Type:         "max_complexity_exceeded",
		Message:      "Function exceeds maximum complexity",
		FunctionName: "complexFunc",
		Complexity:   50,
	}

	if warning.Type != "max_complexity_exceeded" {
		t.Error("Type should be 'max_complexity_exceeded'")
	}
	if warning.Complexity != 50 {
		t.Error("Complexity should be 50")
	}
}

func TestComplexityReport_Serialization_JSON(t *testing.T) {
	report := &ComplexityReport{
		Summary: ReportSummary{
			TotalFunctions: 5,
		},
		Results: []SerializableComplexityResult{
			{FunctionName: "test", Complexity: 10},
		},
		Warnings: []ReportWarning{
			{Type: "warning", Message: "test warning"},
		},
	}

	data, err := json.Marshal(report)
	if err != nil {
		t.Fatalf("Should marshal to JSON: %v", err)
	}

	var unmarshaled ComplexityReport
	if err := json.Unmarshal(data, &unmarshaled); err != nil {
		t.Fatalf("Should unmarshal from JSON: %v", err)
	}

	if unmarshaled.Summary.TotalFunctions != 5 {
		t.Error("Unmarshaled TotalFunctions should be 5")
	}
}

func TestComplexityReport_Serialization_YAML(t *testing.T) {
	report := &ComplexityReport{
		Summary: ReportSummary{
			TotalFunctions: 5,
		},
		Results: []SerializableComplexityResult{
			{FunctionName: "test", Complexity: 10},
		},
	}

	data, err := yaml.Marshal(report)
	if err != nil {
		t.Fatalf("Should marshal to YAML: %v", err)
	}

	var unmarshaled ComplexityReport
	if err := yaml.Unmarshal(data, &unmarshaled); err != nil {
		t.Fatalf("Should unmarshal from YAML: %v", err)
	}

	if unmarshaled.Summary.TotalFunctions != 5 {
		t.Error("Unmarshaled TotalFunctions should be 5")
	}
}

func TestMockComplexityResult_GetDetailedMetrics_Nil(t *testing.T) {
	mock := &mockComplexityResult{
		complexity:   5,
		functionName: "test",
		riskLevel:    "low",
		metrics:      nil,
	}

	metrics := mock.GetDetailedMetrics()
	if metrics == nil {
		t.Error("GetDetailedMetrics should return empty map, not nil")
	}
	if len(metrics) != 0 {
		t.Error("GetDetailedMetrics should return empty map when metrics is nil")
	}
}
