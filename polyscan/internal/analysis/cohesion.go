package analysis

import (
	"sort"
	"strings"

	"github.com/ludo-technologies/polyscan/core/domain"
	"github.com/ludo-technologies/polyscan/core/lcom"
	"github.com/ludo-technologies/polyscan/polyscan/internal/engine"
)

// Class is the cohesion result for one type.
type Class struct {
	Name     string `json:"name"`
	FilePath string `json:"file_path"`
	Language string `json:"language"`
	// StartLine and EndLine span the type's methods in FilePath, or the
	// type's declaration when the class is attributed to a declaring file
	// that holds none of the measured methods. A type whose methods are
	// spread over several files without a single declaration is placed in
	// the first of them.
	StartLine int `json:"start_line"`
	EndLine   int `json:"end_line"`
	// LCOM4 is the number of connected components among the methods, where
	// two methods are connected when they share a field or one calls the
	// other.
	LCOM4 int `json:"lcom4"`
	// TotalMethods counts every method of the type; ExcludedMethods the
	// ones without a receiver parameter, which cannot reach instance state
	// and stay out of the graph.
	TotalMethods      int              `json:"total_methods"`
	ExcludedMethods   int              `json:"excluded_methods"`
	InstanceVariables []string         `json:"instance_variables"`
	MethodGroups      [][]string       `json:"method_groups"`
	RiskLevel         domain.RiskLevel `json:"risk_level"`
}

// CohesionSummary aggregates every analyzed type.
type CohesionSummary struct {
	TotalClasses      int     `json:"total_classes"`
	AverageLCOM       float64 `json:"average_lcom"`
	MaxLCOM           int     `json:"max_lcom"`
	MinLCOM           int     `json:"min_lcom"`
	LowRiskClasses    int     `json:"low_risk_classes"`
	MediumRiskClasses int     `json:"medium_risk_classes"`
	HighRiskClasses   int     `json:"high_risk_classes"`
}

// Cohesion is the LCOM4 analysis of a set of paths.
type Cohesion struct {
	// Classes is sorted by descending LCOM4, then by location.
	Classes []Class         `json:"classes"`
	Summary CohesionSummary `json:"summary"`
}

// classMethods collects the methods of one type before they are measured.
type classMethods struct {
	name     string
	language *engine.Language
	methods  []classMethod
}

type classMethod struct {
	engine.Function
	file string
}

// typeDeclaration is the file and line span declaring one type.
type typeDeclaration struct {
	file       string
	start, end int
}

// cohesionBuilder groups methods by type. A type is identified by its
// language, its receiver name and the file that declares its methods, or
// the directory when the language lets a type's methods span one.
type cohesionBuilder struct {
	classes map[string]*classMethods
	keys    []string
	// index tracks the type declarations of the tree; see declIndex.
	index *declIndex[*typeDeclaration]
}

func newCohesionBuilder() *cohesionBuilder {
	return &cohesionBuilder{classes: map[string]*classMethods{}, index: newDeclIndex[*typeDeclaration]()}
}

func (b *cohesionBuilder) add(language *engine.Language, display string, fn engine.Function) {
	if fn.Receiver == "" {
		return
	}
	key := scopedKey(language, locationOf(language, display), fn.Receiver)
	class, ok := b.classes[key]
	if !ok {
		class = &classMethods{name: fn.Receiver, language: language}
		b.classes[key] = class
		b.keys = append(b.keys, key)
	}
	class.methods = append(class.methods, classMethod{Function: fn, file: display})
}

// setDeclarations records the file and line span declaring each type from
// the engine result.
func (b *cohesionBuilder) setDeclarations(language *engine.Language, display string, result *engine.Result) {
	location := locationOf(language, display)
	for _, t := range result.Types {
		if t.Declared && !t.IsTest {
			b.index.add(language, location, t.Name, &typeDeclaration{file: display, start: t.StartLine, end: t.EndLine})
		}
	}
}

// build measures every type that has at least one method with a receiver
// parameter. A type with none, such as one that only has constructors, has
// no cohesion to measure and is left out.
func (b *cohesionBuilder) build() *Cohesion {
	cohesion := &Cohesion{Classes: []Class{}}
	// Merge the per-file groups of an unambiguously-declared type before
	// measuring, so a type whose declaration and methods span files is
	// measured once and reported once under its declaring file. An
	// ambiguous name keeps one group per file, so unrelated same-named
	// types stay distinct.
	merged := map[string]*classMethods{}
	decls := map[string]*typeDeclaration{}
	var order []string
	for _, key := range b.keys {
		cm := b.classes[key]
		target := key
		var decl *typeDeclaration
		if !cm.language.TypeSpansDirectory {
			if d, ok := b.index.unique(cm.language, cm.name); ok {
				decl = d
				target = scopedKey(cm.language, d.file, cm.name)
			}
		}
		m, ok := merged[target]
		if !ok {
			m = &classMethods{name: cm.name, language: cm.language}
			merged[target] = m
			decls[target] = decl
			order = append(order, target)
		}
		m.methods = append(m.methods, cm.methods...)
	}
	for _, key := range order {
		class := merged[key].measure()
		// Attribute the merged class to the declaring file with the
		// declaration's line span when it differs from the first
		// method's file, so the cohesion result matches the coupling
		// result.
		if d := decls[key]; d != nil && d.file != class.FilePath {
			class.FilePath = d.file
			class.StartLine, class.EndLine = d.start, d.end
		}
		if class.TotalMethods > class.ExcludedMethods {
			cohesion.Classes = append(cohesion.Classes, class)
		}
	}
	sort.SliceStable(cohesion.Classes, func(i, j int) bool {
		a, b := cohesion.Classes[i], cohesion.Classes[j]
		if a.LCOM4 != b.LCOM4 {
			return a.LCOM4 > b.LCOM4
		}
		if a.FilePath != b.FilePath {
			return a.FilePath < b.FilePath
		}
		return a.StartLine < b.StartLine
	})
	cohesion.Summary = summarizeCohesion(cohesion.Classes)
	return cohesion
}

// measure computes the class's LCOM4. A call whose name is not a sibling
// method, such as a call of a function-typed field, is an access to that
// field.
func (c *classMethods) measure() Class {
	sort.SliceStable(c.methods, func(i, j int) bool {
		if c.methods[i].file != c.methods[j].file {
			return c.methods[i].file < c.methods[j].file
		}
		return c.methods[i].StartLine < c.methods[j].StartLine
	})
	prefix := c.name + c.language.Separator()
	names := map[string]bool{}
	for _, method := range c.methods {
		names[strings.TrimPrefix(method.Name, prefix)] = true
	}

	class := Class{
		Name:      c.name,
		FilePath:  c.methods[0].file,
		Language:  c.language.Name,
		StartLine: c.methods[0].StartLine,
	}
	var accesses []lcom.MethodAccess
	for _, method := range c.methods {
		class.TotalMethods++
		if method.file == class.FilePath {
			class.EndLine = max(class.EndLine, method.EndLine)
		}
		if !method.HasSelf {
			class.ExcludedMethods++
			continue
		}
		access := lcom.MethodAccess{
			MethodName:   strings.TrimPrefix(method.Name, prefix),
			InstanceVars: map[string]bool{},
			Calls:        map[string]bool{},
		}
		for field := range method.Fields {
			access.InstanceVars[field] = true
		}
		for call := range method.Calls {
			if names[call] {
				access.Calls[call] = true
			} else {
				access.InstanceVars[call] = true
			}
		}
		accesses = append(accesses, access)
	}

	result := lcom.ComputeLCOM4(accesses, lcom.DefaultConfig())
	class.LCOM4 = result.LCOM4
	class.InstanceVariables = result.InstanceVariables
	class.MethodGroups = result.MethodGroups
	class.RiskLevel = result.RiskLevel
	return class
}

func summarizeCohesion(classes []Class) CohesionSummary {
	summary := CohesionSummary{TotalClasses: len(classes)}
	if len(classes) == 0 {
		return summary
	}
	total := 0
	summary.MinLCOM = classes[0].LCOM4
	for _, class := range classes {
		total += class.LCOM4
		summary.MaxLCOM = max(summary.MaxLCOM, class.LCOM4)
		summary.MinLCOM = min(summary.MinLCOM, class.LCOM4)
		switch class.RiskLevel {
		case domain.RiskLevelLow:
			summary.LowRiskClasses++
		case domain.RiskLevelMedium:
			summary.MediumRiskClasses++
		case domain.RiskLevelHigh:
			summary.HighRiskClasses++
		}
	}
	summary.AverageLCOM = float64(total) / float64(len(classes))
	return summary
}
