package domain

import (
	"errors"
	"testing"
)

// Error tests

func TestDomainError_Error(t *testing.T) {
	// Without cause
	err := DomainError{
		Code:    "TEST_ERROR",
		Message: "Test message",
	}
	expected := "[TEST_ERROR] Test message"
	if err.Error() != expected {
		t.Errorf("Expected '%s', got '%s'", expected, err.Error())
	}

	// With cause
	cause := errors.New("underlying error")
	errWithCause := DomainError{
		Code:    "TEST_ERROR",
		Message: "Test message",
		Cause:   cause,
	}
	expectedWithCause := "[TEST_ERROR] Test message: underlying error"
	if errWithCause.Error() != expectedWithCause {
		t.Errorf("Expected '%s', got '%s'", expectedWithCause, errWithCause.Error())
	}
}

func TestDomainError_Unwrap(t *testing.T) {
	cause := errors.New("underlying error")
	err := DomainError{
		Code:    "TEST_ERROR",
		Message: "Test message",
		Cause:   cause,
	}

	unwrapped := err.Unwrap()
	if unwrapped != cause {
		t.Error("Unwrap should return the cause")
	}

	// Without cause
	errNoCause := DomainError{
		Code:    "TEST_ERROR",
		Message: "Test message",
	}
	if errNoCause.Unwrap() != nil {
		t.Error("Unwrap should return nil when no cause")
	}
}

func TestNewDomainError(t *testing.T) {
	cause := errors.New("cause")
	err := NewDomainError("CODE", "message", cause)

	domainErr, ok := err.(DomainError)
	if !ok {
		t.Fatal("Should return DomainError type")
	}
	if domainErr.Code != "CODE" {
		t.Errorf("Expected code 'CODE', got '%s'", domainErr.Code)
	}
	if domainErr.Message != "message" {
		t.Errorf("Expected message 'message', got '%s'", domainErr.Message)
	}
	if domainErr.Cause != cause {
		t.Error("Cause should be set")
	}
}

func TestNewInvalidInputError(t *testing.T) {
	cause := errors.New("invalid")
	err := NewInvalidInputError("bad input", cause)

	domainErr := err.(DomainError)
	if domainErr.Code != ErrCodeInvalidInput {
		t.Errorf("Expected code '%s', got '%s'", ErrCodeInvalidInput, domainErr.Code)
	}
}

func TestNewFileNotFoundError(t *testing.T) {
	err := NewFileNotFoundError("/path/to/file", nil)

	domainErr := err.(DomainError)
	if domainErr.Code != ErrCodeFileNotFound {
		t.Errorf("Expected code '%s', got '%s'", ErrCodeFileNotFound, domainErr.Code)
	}
	if domainErr.Message != "file not found: /path/to/file" {
		t.Errorf("Unexpected message: %s", domainErr.Message)
	}
}

func TestNewParseError(t *testing.T) {
	cause := errors.New("syntax error")
	err := NewParseError("test.js", cause)

	domainErr := err.(DomainError)
	if domainErr.Code != ErrCodeParseError {
		t.Errorf("Expected code '%s', got '%s'", ErrCodeParseError, domainErr.Code)
	}
}

func TestNewAnalysisError(t *testing.T) {
	err := NewAnalysisError("analysis failed", nil)

	domainErr := err.(DomainError)
	if domainErr.Code != ErrCodeAnalysisError {
		t.Errorf("Expected code '%s', got '%s'", ErrCodeAnalysisError, domainErr.Code)
	}
}

func TestNewConfigError(t *testing.T) {
	err := NewConfigError("invalid config", nil)

	domainErr := err.(DomainError)
	if domainErr.Code != ErrCodeConfigError {
		t.Errorf("Expected code '%s', got '%s'", ErrCodeConfigError, domainErr.Code)
	}
}

func TestNewOutputError(t *testing.T) {
	err := NewOutputError("write failed", nil)

	domainErr := err.(DomainError)
	if domainErr.Code != ErrCodeOutputError {
		t.Errorf("Expected code '%s', got '%s'", ErrCodeOutputError, domainErr.Code)
	}
}

func TestNewUnsupportedFormatError(t *testing.T) {
	err := NewUnsupportedFormatError("xml")

	domainErr := err.(DomainError)
	if domainErr.Code != ErrCodeUnsupportedFormat {
		t.Errorf("Expected code '%s', got '%s'", ErrCodeUnsupportedFormat, domainErr.Code)
	}
	if domainErr.Message != "unsupported format: xml" {
		t.Errorf("Unexpected message: %s", domainErr.Message)
	}
}

func TestNewValidationError(t *testing.T) {
	err := NewValidationError("validation failed")

	domainErr := err.(DomainError)
	if domainErr.Code != ErrCodeInvalidInput {
		t.Errorf("Expected code '%s', got '%s'", ErrCodeInvalidInput, domainErr.Code)
	}
}

// Output format tests

func TestOutputFormat_Constants(t *testing.T) {
	formats := map[OutputFormat]string{
		OutputFormatText: "text",
		OutputFormatJSON: "json",
		OutputFormatYAML: "yaml",
		OutputFormatCSV:  "csv",
		OutputFormatHTML: "html",
		OutputFormatDOT:  "dot",
	}

	for format, expected := range formats {
		if string(format) != expected {
			t.Errorf("OutputFormat %s should equal '%s'", format, expected)
		}
	}
}

// Sort criteria tests

func TestSortCriteria_Constants(t *testing.T) {
	criteria := map[SortCriteria]string{
		SortByComplexity: "complexity",
		SortByName:       "name",
		SortByRisk:       "risk",
		SortBySimilarity: "similarity",
		SortBySize:       "size",
		SortByLocation:   "location",
		SortByCoupling:   "coupling",
	}

	for c, expected := range criteria {
		if string(c) != expected {
			t.Errorf("SortCriteria %s should equal '%s'", c, expected)
		}
	}
}

// Risk level tests

func TestRiskLevel_Constants(t *testing.T) {
	levels := map[RiskLevel]string{
		RiskLevelLow:    "low",
		RiskLevelMedium: "medium",
		RiskLevelHigh:   "high",
	}

	for level, expected := range levels {
		if string(level) != expected {
			t.Errorf("RiskLevel %s should equal '%s'", level, expected)
		}
	}
}

// Dead code severity tests

func TestDeadCodeSeverity_Constants(t *testing.T) {
	severities := map[DeadCodeSeverity]string{
		DeadCodeSeverityCritical: "critical",
		DeadCodeSeverityWarning:  "warning",
		DeadCodeSeverityInfo:     "info",
	}

	for severity, expected := range severities {
		if string(severity) != expected {
			t.Errorf("DeadCodeSeverity %s should equal '%s'", severity, expected)
		}
	}
}

// Dead code sort criteria tests

func TestDeadCodeSortCriteria_Constants(t *testing.T) {
	criteria := map[DeadCodeSortCriteria]string{
		DeadCodeSortBySeverity: "severity",
		DeadCodeSortByLine:     "line",
		DeadCodeSortByFile:     "file",
		DeadCodeSortByFunction: "function",
	}

	for c, expected := range criteria {
		if string(c) != expected {
			t.Errorf("DeadCodeSortCriteria %s should equal '%s'", c, expected)
		}
	}
}

// Complexity request tests

func TestComplexityRequest_Fields(t *testing.T) {
	req := ComplexityRequest{
		Paths:           []string{"/path/to/src"},
		OutputFormat:    OutputFormatJSON,
		MinComplexity:   5,
		MaxComplexity:   50,
		SortBy:          SortByComplexity,
		LowThreshold:    5,
		MediumThreshold: 10,
		Recursive:       true,
		IncludePatterns: []string{"*.js"},
		ExcludePatterns: []string{"node_modules"},
	}

	if len(req.Paths) != 1 {
		t.Error("Paths should have 1 element")
	}
	if req.OutputFormat != OutputFormatJSON {
		t.Error("OutputFormat should be JSON")
	}
	if req.MinComplexity != 5 {
		t.Error("MinComplexity should be 5")
	}
	if req.Recursive != true {
		t.Error("Recursive should be true")
	}
}

// Complexity metrics tests

func TestComplexityMetrics_Fields(t *testing.T) {
	metrics := ComplexityMetrics{
		Complexity:        10,
		Nodes:             20,
		Edges:             25,
		NestingDepth:      3,
		IfStatements:      5,
		LoopStatements:    2,
		ExceptionHandlers: 1,
		SwitchCases:       3,
	}

	if metrics.Complexity != 10 {
		t.Errorf("Complexity should be 10, got %d", metrics.Complexity)
	}
	if metrics.Nodes != 20 {
		t.Errorf("Nodes should be 20, got %d", metrics.Nodes)
	}
}

// Function complexity tests

func TestFunctionComplexity_Fields(t *testing.T) {
	fc := FunctionComplexity{
		Name:        "testFunc",
		FilePath:    "/src/test.js",
		StartLine:   10,
		StartColumn: 1,
		EndLine:     20,
		Metrics: ComplexityMetrics{
			Complexity: 5,
		},
		RiskLevel: RiskLevelLow,
	}

	if fc.Name != "testFunc" {
		t.Errorf("Name should be 'testFunc', got '%s'", fc.Name)
	}
	if fc.RiskLevel != RiskLevelLow {
		t.Errorf("RiskLevel should be 'low', got '%s'", fc.RiskLevel)
	}
}

// Complexity summary tests

func TestComplexitySummary_Fields(t *testing.T) {
	summary := ComplexitySummary{
		TotalFunctions:         100,
		AverageComplexity:      5.5,
		MaxComplexity:          25,
		MinComplexity:          1,
		FilesAnalyzed:          10,
		LowRiskFunctions:       80,
		MediumRiskFunctions:    15,
		HighRiskFunctions:      5,
		ComplexityDistribution: map[string]int{"1-5": 50, "6-10": 30},
	}

	if summary.TotalFunctions != 100 {
		t.Errorf("TotalFunctions should be 100, got %d", summary.TotalFunctions)
	}
	if summary.AverageComplexity != 5.5 {
		t.Errorf("AverageComplexity should be 5.5, got %f", summary.AverageComplexity)
	}
}

// Dead code location tests

func TestDeadCodeLocation_Fields(t *testing.T) {
	loc := DeadCodeLocation{
		FilePath:    "/src/test.js",
		StartLine:   10,
		EndLine:     15,
		StartColumn: 1,
		EndColumn:   10,
	}

	if loc.FilePath != "/src/test.js" {
		t.Errorf("FilePath should be '/src/test.js', got '%s'", loc.FilePath)
	}
	if loc.StartLine != 10 {
		t.Errorf("StartLine should be 10, got %d", loc.StartLine)
	}
}

// Dead code finding tests

func TestDeadCodeFinding_Fields(t *testing.T) {
	finding := DeadCodeFinding{
		Location: DeadCodeLocation{
			FilePath:  "/src/test.js",
			StartLine: 10,
		},
		FunctionName: "myFunc",
		Code:         "console.log('unreachable');",
		Reason:       "unreachable_after_return",
		Severity:     DeadCodeSeverityCritical,
		Description:  "Code after return statement",
		Context:      []string{"return 42;", "console.log('unreachable');"},
		BlockID:      "block_1",
	}

	if finding.FunctionName != "myFunc" {
		t.Errorf("FunctionName should be 'myFunc', got '%s'", finding.FunctionName)
	}
	if finding.Severity != DeadCodeSeverityCritical {
		t.Errorf("Severity should be 'critical', got '%s'", finding.Severity)
	}
}

// Function dead code tests

func TestFunctionDeadCode_Fields(t *testing.T) {
	fdc := FunctionDeadCode{
		Name:           "testFunc",
		FilePath:       "/src/test.js",
		Findings:       []DeadCodeFinding{{Severity: DeadCodeSeverityCritical}},
		TotalBlocks:    10,
		DeadBlocks:     2,
		ReachableRatio: 0.8,
		CriticalCount:  1,
		WarningCount:   0,
		InfoCount:      0,
	}

	if fdc.Name != "testFunc" {
		t.Errorf("Name should be 'testFunc', got '%s'", fdc.Name)
	}
	if fdc.ReachableRatio != 0.8 {
		t.Errorf("ReachableRatio should be 0.8, got %f", fdc.ReachableRatio)
	}
	if len(fdc.Findings) != 1 {
		t.Errorf("Should have 1 finding, got %d", len(fdc.Findings))
	}
}

// File dead code tests

func TestFileDeadCode_Fields(t *testing.T) {
	fdc := FileDeadCode{
		FilePath:          "/src/test.js",
		Functions:         []FunctionDeadCode{},
		TotalFindings:     5,
		TotalFunctions:    10,
		AffectedFunctions: 3,
		DeadCodeRatio:     0.3,
	}

	if fdc.FilePath != "/src/test.js" {
		t.Errorf("FilePath should be '/src/test.js', got '%s'", fdc.FilePath)
	}
	if fdc.DeadCodeRatio != 0.3 {
		t.Errorf("DeadCodeRatio should be 0.3, got %f", fdc.DeadCodeRatio)
	}
}

// Dead code summary tests

func TestDeadCodeSummary_Fields(t *testing.T) {
	summary := DeadCodeSummary{
		TotalFiles:            10,
		TotalFunctions:        50,
		TotalFindings:         15,
		FilesWithDeadCode:     5,
		FunctionsWithDeadCode: 8,
		CriticalFindings:      5,
		WarningFindings:       7,
		InfoFindings:          3,
		FindingsByReason:      map[string]int{"after_return": 10, "unreachable_branch": 5},
		TotalBlocks:           500,
		DeadBlocks:            15,
		OverallDeadRatio:      0.03,
	}

	if summary.TotalFiles != 10 {
		t.Errorf("TotalFiles should be 10, got %d", summary.TotalFiles)
	}
	if summary.CriticalFindings != 5 {
		t.Errorf("CriticalFindings should be 5, got %d", summary.CriticalFindings)
	}
	if len(summary.FindingsByReason) != 2 {
		t.Errorf("FindingsByReason should have 2 entries, got %d", len(summary.FindingsByReason))
	}
}

// Dead code request tests

func TestDeadCodeRequest_Fields(t *testing.T) {
	showContext := true
	detectAfterReturn := true

	req := DeadCodeRequest{
		Paths:             []string{"/src"},
		OutputFormat:      OutputFormatJSON,
		ShowContext:       &showContext,
		ContextLines:      3,
		MinSeverity:       DeadCodeSeverityWarning,
		SortBy:            DeadCodeSortBySeverity,
		Recursive:         true,
		DetectAfterReturn: &detectAfterReturn,
	}

	if len(req.Paths) != 1 {
		t.Error("Paths should have 1 element")
	}
	if *req.ShowContext != true {
		t.Error("ShowContext should be true")
	}
	if req.MinSeverity != DeadCodeSeverityWarning {
		t.Errorf("MinSeverity should be 'warning', got '%s'", req.MinSeverity)
	}
}

// Error code constants tests

func TestErrorCodeConstants(t *testing.T) {
	codes := map[string]string{
		ErrCodeInvalidInput:      "INVALID_INPUT",
		ErrCodeFileNotFound:      "FILE_NOT_FOUND",
		ErrCodeParseError:        "PARSE_ERROR",
		ErrCodeAnalysisError:     "ANALYSIS_ERROR",
		ErrCodeConfigError:       "CONFIG_ERROR",
		ErrCodeOutputError:       "OUTPUT_ERROR",
		ErrCodeUnsupportedFormat: "UNSUPPORTED_FORMAT",
	}

	for code, expected := range codes {
		if code != expected {
			t.Errorf("Error code should be '%s', got '%s'", expected, code)
		}
	}
}
