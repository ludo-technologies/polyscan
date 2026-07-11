package domain

import "testing"

func TestCloneType_String(t *testing.T) {
	tests := []struct {
		ct   CloneType
		want string
	}{
		{Type1Clone, "Type-1"},
		{Type2Clone, "Type-2"},
		{Type3Clone, "Type-3"},
		{Type4Clone, "Type-4"},
		{CloneType(99), "Unknown(99)"},
	}
	for _, tt := range tests {
		got := tt.ct.String()
		if got != tt.want {
			t.Errorf("CloneType(%d).String() = %q, want %q", int(tt.ct), got, tt.want)
		}
	}
}

func TestCloneTypeNames(t *testing.T) {
	if len(CloneTypeNames) != 4 {
		t.Errorf("expected 4 clone type names, got %d", len(CloneTypeNames))
	}
	if CloneTypeNames[Type1Clone] != "Exact" {
		t.Errorf("Type1Clone name = %q, want %q", CloneTypeNames[Type1Clone], "Exact")
	}
}

func TestCloneTypeDescriptions(t *testing.T) {
	if len(CloneTypeDescriptions) != 4 {
		t.Errorf("expected 4 clone type descriptions, got %d", len(CloneTypeDescriptions))
	}
	for ct, desc := range CloneTypeDescriptions {
		if desc == "" {
			t.Errorf("CloneType %d has empty description", int(ct))
		}
	}
}

func TestRiskLevelValues(t *testing.T) {
	if RiskLevelLow != "low" {
		t.Errorf("RiskLevelLow = %q", RiskLevelLow)
	}
	if RiskLevelMedium != "medium" {
		t.Errorf("RiskLevelMedium = %q", RiskLevelMedium)
	}
	if RiskLevelHigh != "high" {
		t.Errorf("RiskLevelHigh = %q", RiskLevelHigh)
	}
}

func TestOutputFormatValues(t *testing.T) {
	formats := []OutputFormat{
		OutputFormatText, OutputFormatJSON, OutputFormatYAML,
		OutputFormatCSV, OutputFormatHTML, OutputFormatDOT,
	}
	for _, f := range formats {
		if f == "" {
			t.Errorf("empty output format")
		}
	}
}

func TestSortCriteriaValues(t *testing.T) {
	criteria := []SortCriteria{
		SortByComplexity, SortByName, SortByRisk, SortBySimilarity,
		SortBySize, SortByLocation, SortByCoupling, SortByCohesion,
	}
	for _, c := range criteria {
		if c == "" {
			t.Errorf("empty sort criteria")
		}
	}
}
