package lcom

import (
	"reflect"
	"testing"

	"github.com/ludo-technologies/polyscan/core/domain"
)

func TestComputeLCOM4_EmptyMethods(t *testing.T) {
	result := ComputeLCOM4(nil, DefaultConfig())
	if result.LCOM4 != 0 {
		t.Errorf("LCOM4 = %d, want 0", result.LCOM4)
	}
	if result.RiskLevel != domain.RiskLevelLow {
		t.Errorf("RiskLevel = %q, want %q", result.RiskLevel, domain.RiskLevelLow)
	}
	if result.TotalMethods != 0 {
		t.Errorf("TotalMethods = %d, want 0", result.TotalMethods)
	}
	if len(result.MethodGroups) != 0 {
		t.Errorf("MethodGroups = %v, want empty", result.MethodGroups)
	}
}

func TestComputeLCOM4_SingleMethod(t *testing.T) {
	methods := []MethodAccess{
		{MethodName: "GetName", InstanceVars: map[string]bool{"name": true}},
	}
	result := ComputeLCOM4(methods, DefaultConfig())
	if result.LCOM4 != 1 {
		t.Errorf("LCOM4 = %d, want 1", result.LCOM4)
	}
	if result.RiskLevel != domain.RiskLevelLow {
		t.Errorf("RiskLevel = %q, want %q", result.RiskLevel, domain.RiskLevelLow)
	}
	if result.TotalMethods != 1 {
		t.Errorf("TotalMethods = %d, want 1", result.TotalMethods)
	}
	wantGroups := [][]string{{"GetName"}}
	if !reflect.DeepEqual(result.MethodGroups, wantGroups) {
		t.Errorf("MethodGroups = %v, want %v", result.MethodGroups, wantGroups)
	}
}

func TestComputeLCOM4_TwoMethodsShareVariable(t *testing.T) {
	methods := []MethodAccess{
		{MethodName: "GetName", InstanceVars: map[string]bool{"name": true}},
		{MethodName: "SetName", InstanceVars: map[string]bool{"name": true}},
	}
	result := ComputeLCOM4(methods, DefaultConfig())
	if result.LCOM4 != 1 {
		t.Errorf("LCOM4 = %d, want 1", result.LCOM4)
	}
	if result.RiskLevel != domain.RiskLevelLow {
		t.Errorf("RiskLevel = %q, want %q", result.RiskLevel, domain.RiskLevelLow)
	}
	wantGroups := [][]string{{"GetName", "SetName"}}
	if !reflect.DeepEqual(result.MethodGroups, wantGroups) {
		t.Errorf("MethodGroups = %v, want %v", result.MethodGroups, wantGroups)
	}
}

func TestComputeLCOM4_TwoIndependentMethods(t *testing.T) {
	methods := []MethodAccess{
		{MethodName: "GetName", InstanceVars: map[string]bool{"name": true}},
		{MethodName: "GetAge", InstanceVars: map[string]bool{"age": true}},
	}
	result := ComputeLCOM4(methods, DefaultConfig())
	if result.LCOM4 != 2 {
		t.Errorf("LCOM4 = %d, want 2", result.LCOM4)
	}
	// LCOM4=2 is at the low threshold (<=2), so still low
	if result.RiskLevel != domain.RiskLevelLow {
		t.Errorf("RiskLevel = %q, want %q", result.RiskLevel, domain.RiskLevelLow)
	}
	if len(result.MethodGroups) != 2 {
		t.Errorf("len(MethodGroups) = %d, want 2", len(result.MethodGroups))
	}
}

func TestComputeLCOM4_ThreeMethodsTwoSharingOneIndependent(t *testing.T) {
	methods := []MethodAccess{
		{MethodName: "GetName", InstanceVars: map[string]bool{"name": true}},
		{MethodName: "SetName", InstanceVars: map[string]bool{"name": true}},
		{MethodName: "GetAge", InstanceVars: map[string]bool{"age": true}},
	}
	result := ComputeLCOM4(methods, DefaultConfig())
	if result.LCOM4 != 2 {
		t.Errorf("LCOM4 = %d, want 2", result.LCOM4)
	}
	if result.TotalMethods != 3 {
		t.Errorf("TotalMethods = %d, want 3", result.TotalMethods)
	}
}

func TestComputeLCOM4_MultipleComponents(t *testing.T) {
	methods := []MethodAccess{
		{MethodName: "GetName", InstanceVars: map[string]bool{"name": true}},
		{MethodName: "GetAge", InstanceVars: map[string]bool{"age": true}},
		{MethodName: "GetEmail", InstanceVars: map[string]bool{"email": true}},
		{MethodName: "SetName", InstanceVars: map[string]bool{"name": true}},
	}
	result := ComputeLCOM4(methods, DefaultConfig())
	// GetName+SetName share "name" -> 1 component
	// GetAge alone -> 1 component
	// GetEmail alone -> 1 component
	// Total: 3 components
	if result.LCOM4 != 3 {
		t.Errorf("LCOM4 = %d, want 3", result.LCOM4)
	}
	if result.RiskLevel != domain.RiskLevelMedium {
		t.Errorf("RiskLevel = %q, want %q", result.RiskLevel, domain.RiskLevelMedium)
	}
}

func TestComputeLCOM4_TransitiveConnection(t *testing.T) {
	// A accesses x,y; B accesses y,z; C accesses z
	// A-B connected via y, B-C connected via z => all in one component
	methods := []MethodAccess{
		{MethodName: "A", InstanceVars: map[string]bool{"x": true, "y": true}},
		{MethodName: "B", InstanceVars: map[string]bool{"y": true, "z": true}},
		{MethodName: "C", InstanceVars: map[string]bool{"z": true}},
	}
	result := ComputeLCOM4(methods, DefaultConfig())
	if result.LCOM4 != 1 {
		t.Errorf("LCOM4 = %d, want 1", result.LCOM4)
	}
	wantGroups := [][]string{{"A", "B", "C"}}
	if !reflect.DeepEqual(result.MethodGroups, wantGroups) {
		t.Errorf("MethodGroups = %v, want %v", result.MethodGroups, wantGroups)
	}
}

func TestAssessRisk_BoundaryValues(t *testing.T) {
	config := DefaultConfig()
	tests := []struct {
		lcom4 int
		want  domain.RiskLevel
	}{
		{0, domain.RiskLevelLow},
		{1, domain.RiskLevelLow},
		{2, domain.RiskLevelLow},    // at low threshold
		{3, domain.RiskLevelMedium}, // just above low threshold
		{4, domain.RiskLevelMedium},
		{5, domain.RiskLevelMedium}, // at medium threshold
		{6, domain.RiskLevelHigh},   // just above medium threshold
		{10, domain.RiskLevelHigh},
	}
	for _, tt := range tests {
		got := AssessRisk(tt.lcom4, config)
		if got != tt.want {
			t.Errorf("AssessRisk(%d) = %q, want %q", tt.lcom4, got, tt.want)
		}
	}
}

func TestDefaultConfig(t *testing.T) {
	config := DefaultConfig()
	if config.LowThreshold != domain.DefaultLCOMLowThreshold {
		t.Errorf("LowThreshold = %d, want %d", config.LowThreshold, domain.DefaultLCOMLowThreshold)
	}
	if config.MediumThreshold != domain.DefaultLCOMMediumThreshold {
		t.Errorf("MediumThreshold = %d, want %d", config.MediumThreshold, domain.DefaultLCOMMediumThreshold)
	}
}

func TestComputeLCOM4_MethodGroupsSorted(t *testing.T) {
	// Supply methods in reverse order to verify sorting
	methods := []MethodAccess{
		{MethodName: "Zebra", InstanceVars: map[string]bool{"x": true}},
		{MethodName: "Alpha", InstanceVars: map[string]bool{"x": true}},
		{MethodName: "Middle", InstanceVars: map[string]bool{"x": true}},
	}
	result := ComputeLCOM4(methods, DefaultConfig())
	if result.LCOM4 != 1 {
		t.Errorf("LCOM4 = %d, want 1", result.LCOM4)
	}
	wantGroups := [][]string{{"Alpha", "Middle", "Zebra"}}
	if !reflect.DeepEqual(result.MethodGroups, wantGroups) {
		t.Errorf("MethodGroups = %v, want %v", result.MethodGroups, wantGroups)
	}
}

func TestComputeLCOM4_InstanceVariablesCollectedAndSorted(t *testing.T) {
	methods := []MethodAccess{
		{MethodName: "A", InstanceVars: map[string]bool{"z": true, "a": true}},
		{MethodName: "B", InstanceVars: map[string]bool{"m": true, "a": true}},
	}
	result := ComputeLCOM4(methods, DefaultConfig())
	wantVars := []string{"a", "m", "z"}
	if !reflect.DeepEqual(result.InstanceVariables, wantVars) {
		t.Errorf("InstanceVariables = %v, want %v", result.InstanceVariables, wantVars)
	}
}

func TestComputeLCOM4_MethodWithNoVars(t *testing.T) {
	methods := []MethodAccess{
		{MethodName: "DoSomething", InstanceVars: map[string]bool{}},
		{MethodName: "GetName", InstanceVars: map[string]bool{"name": true}},
	}
	result := ComputeLCOM4(methods, DefaultConfig())
	// DoSomething shares no vars with GetName -> 2 components
	if result.LCOM4 != 2 {
		t.Errorf("LCOM4 = %d, want 2", result.LCOM4)
	}
}

func TestAssessRisk_CustomConfig(t *testing.T) {
	config := Config{LowThreshold: 1, MediumThreshold: 3}
	tests := []struct {
		lcom4 int
		want  domain.RiskLevel
	}{
		{0, domain.RiskLevelLow},
		{1, domain.RiskLevelLow},
		{2, domain.RiskLevelMedium},
		{3, domain.RiskLevelMedium},
		{4, domain.RiskLevelHigh},
	}
	for _, tt := range tests {
		got := AssessRisk(tt.lcom4, config)
		if got != tt.want {
			t.Errorf("AssessRisk(%d, custom) = %q, want %q", tt.lcom4, got, tt.want)
		}
	}
}
