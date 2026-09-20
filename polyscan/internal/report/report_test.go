package report

import (
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	coredomain "github.com/ludo-technologies/polyscan/core/domain"
	"github.com/ludo-technologies/polyscan/polyscan/internal/analysis"
	"github.com/ludo-technologies/polyscan/polyscan/internal/clone"
	"github.com/ludo-technologies/polyscan/polyscan/internal/js"
	"github.com/ludo-technologies/polyscan/polyscan/internal/js/domain"
	"github.com/ludo-technologies/polyscan/polyscan/internal/js/service"
)

func TestCouplingScoreIncludesUncoupledTypes(t *testing.T) {
	for _, language := range []struct {
		name, file, prefix, dep, hub string
	}{
		{"Go", "main.go", "package main\n", "type Dep%d struct{}\n", "type Hub struct {\n%s}\n"},
		{"Rust", "main.rs", "", "struct Dep%d {}\n", "struct Hub {\n%s}\n"},
	} {
		t.Run(language.name, func(t *testing.T) {
			dir := t.TempDir()
			var source, fields strings.Builder
			source.WriteString(language.prefix)
			for i := 0; i < 12; i++ {
				fmt.Fprintf(&source, language.dep, i)
				switch language.name {
				case "Go":
					fmt.Fprintf(&fields, "d%d Dep%d\n", i, i)
				case "Rust":
					fmt.Fprintf(&fields, "d%d: Dep%d,\n", i, i)
				}
			}
			for _, withHub := range []bool{false, true} {
				if withHub {
					fmt.Fprintf(&source, language.hub, fields.String())
				}
				path := filepath.Join(dir, language.file)
				if err := os.WriteFile(path, []byte(source.String()), 0600); err != nil {
					t.Fatal(err)
				}
				raw, err := analysis.Analyze([]string{path}, analysis.Options{CBO: true}, nil)
				if err != nil {
					t.Fatal(err)
				}
				results, err := Combine(raw, nil)
				if err != nil {
					t.Fatal(err)
				}
				wantTotal, wantListed, wantScore := 12, 0, 100
				if withHub {
					wantTotal, wantListed, wantScore = 13, 1, 80
				}
				cbo := results.CBO
				if cbo == nil {
					t.Fatal("missing coupling response")
				}
				s := cbo.Summary
				if s.TotalClasses != wantTotal || s.ClassesAnalyzed != wantTotal || len(cbo.Classes) != wantListed || s.LowRiskClasses != 12 || s.HighRiskClasses != wantListed || s.CBODistribution["0"] != 12 || s.MinCBO != 0 {
					t.Fatalf("withHub=%v: summary=%+v, listed=%d", withHub, s, len(cbo.Classes))
				}
				if withHub && (s.AverageCBO != 12.0/13 || s.MaxCBO != 12 || s.CBODistribution["10+"] != 1) {
					t.Fatalf("incorrect Hub statistics: %+v", s)
				}
				summary := service.BuildAnalyzeSummary(results)
				if summary.CouplingScore != wantScore {
					t.Fatalf("coupling score = %d, want %d", summary.CouplingScore, wantScore)
				}
				// Showing the omitted zeros must not change the health score.
				visible := append([]domain.ClassCoupling{}, cbo.Classes...)
				for i := 0; i < 12; i++ {
					visible = append(visible, domain.ClassCoupling{Name: fmt.Sprintf("Dep%d", i), RiskLevel: domain.RiskLevelLow})
				}
				unfiltered := results
				unfiltered.CBO = &domain.CBOResponse{Classes: visible, Summary: service.SummarizeCoupling(visible, 1, 0)}
				if got := service.BuildAnalyzeSummary(unfiltered); got.HealthScore != summary.HealthScore || got.CouplingScore != summary.CouplingScore {
					t.Fatalf("presentation changed score: filtered=%+v, visible=%+v", summary, got)
				}
				// A mixed-language merge must retain the hidden population too.
				merged := mergeCoupling(cbo, &domain.CBOResponse{Classes: []domain.ClassCoupling{{Name: "module", RiskLevel: domain.RiskLevelLow}}, Summary: domain.CBOSummary{TotalClasses: 1, FilesAnalyzed: 1}})
				if merged.Summary.TotalClasses != wantTotal+1 || merged.Summary.CBODistribution["0"] != 13 || merged.Summary.LowRiskClasses != 13 {
					t.Fatalf("merged summary lost zeros: %+v", merged.Summary)
				}
			}
		})
	}
}

func genericReport() *analysis.Report {
	fragment1 := clone.Fragment{ID: 0, Name: "Sum", Language: "Go", FilePath: "a.go", StartLine: 1, EndLine: 10, LineCount: 8, NodeCount: 20, Content: "func Sum() {}"}
	fragment2 := clone.Fragment{ID: 1, Name: "Add", Language: "Go", FilePath: "b.go", StartLine: 5, EndLine: 14, LineCount: 8, NodeCount: 20}
	return &analysis.Report{
		Files: analysis.Files{Total: 3, Analyzed: 2, Skipped: 1},
		Complexity: &analysis.Complexity{
			Functions: []analysis.Function{
				{Name: "Sum", FilePath: "a.go", Language: "Go", StartLine: 1, EndLine: 10, Complexity: 12, NestingDepth: 3, RiskLevel: coredomain.RiskLevelMedium},
				{Name: "Add", FilePath: "b.go", Language: "Go", StartLine: 5, EndLine: 14, Complexity: 2, RiskLevel: coredomain.RiskLevelLow},
			},
			Summary: analysis.ComplexitySummary{
				TotalFunctions:      2,
				AverageComplexity:   7,
				MaxComplexity:       12,
				MinComplexity:       2,
				LowRiskFunctions:    1,
				MediumRiskFunctions: 1,
			},
		},
		Clones: &clone.Report{
			Pairs: []clone.Pair{{
				ID: 0, Type: coredomain.Type1Clone, Similarity: 1,
				Fragment1: fragment1, Fragment2: fragment2,
			}},
			Groups: []clone.Group{{
				ID: 0, Type: coredomain.Type1Clone, Similarity: 1,
				Fragments: []clone.Fragment{fragment1, fragment2},
			}},
			Statistics: clone.Statistics{
				TotalFragments: 4, TotalClones: 2, TotalClonePairs: 1, TotalCloneGroups: 1,
				ClonesByType: map[string]int{"Type-1": 1}, AverageSimilarity: 1,
				LinesAnalyzed: 140, FilesAnalyzed: 2,
			},
		},
		FileLines:   map[string]int{"a.go": 100, "b.go": 40},
		Diagnostics: []domain.AnalysisDiagnostic{{FilePath: "c.go", Code: domain.DiagnosticCodeRead, Message: "read error"}},
	}
}

func TestCombineGenericOnly(t *testing.T) {
	responses, err := Combine(genericReport(), nil)
	if err != nil {
		t.Fatal(err)
	}

	complexity := responses.Complexity
	if complexity == nil {
		t.Fatal("no complexity response")
	}
	if len(complexity.Functions) != 2 || complexity.Functions[0].Name != "Sum" {
		t.Fatalf("functions = %+v", complexity.Functions)
	}
	first := complexity.Functions[0]
	if first.Language != "Go" || first.Metrics.Complexity != 12 || first.Metrics.NestingDepth != 3 || first.RiskLevel != domain.RiskLevelMedium {
		t.Errorf("converted function = %+v", first)
	}
	summary := complexity.Summary
	if summary.TotalFunctions != 2 || summary.TotalFiles != 3 || summary.FilesAnalyzed != 2 || summary.SkippedFiles != 1 {
		t.Errorf("summary = %+v", summary)
	}
	if summary.ComplexityDistribution[12] != 1 || summary.ComplexityDistribution[2] != 1 {
		t.Errorf("distribution = %v", summary.ComplexityDistribution)
	}
	if rollup := complexity.ModuleRollups["a.go"]; rollup.LinesOfCode != 100 || rollup.AnalyzedFunctionCount != 1 || rollup.MaxComplexity != 12 {
		t.Errorf("rollup = %+v", rollup)
	}
	if len(complexity.Errors) != 1 {
		t.Errorf("errors = %v", complexity.Errors)
	}

	clones := responses.Clone
	if clones == nil {
		t.Fatal("no clone response")
	}
	if len(clones.ClonePairs) != 1 || len(clones.CloneGroups) != 1 || len(clones.Clones) != 2 {
		t.Fatalf("clones = %d pairs, %d groups, %d fragments",
			len(clones.ClonePairs), len(clones.CloneGroups), len(clones.Clones))
	}
	pair := clones.ClonePairs[0]
	if pair.Clone1.Language != "Go" || pair.Clone1.Location.FilePath != "a.go" || pair.Type != domain.Type1Clone {
		t.Errorf("pair = %+v", pair)
	}
	// A fragment keeps one identity across the pairs and groups it appears in.
	if pair.Clone1 != clones.CloneGroups[0].Clones[0] {
		t.Error("pair and group must share fragment instances")
	}
	stats := clones.Statistics
	if stats.TotalFragments != 4 || stats.TotalClones != 2 || stats.LinesAnalyzed != 140 || stats.FilesAnalyzed != 2 {
		t.Errorf("statistics = %+v", stats)
	}
	if responses.DeadCode != nil || responses.CBO != nil || responses.Deps != nil {
		t.Error("the generic engine has no dead code, CBO or dependency analysis")
	}
	want := domain.AnalysisCoverage{TotalFiles: 3, AnalyzedFiles: 2, SkippedFiles: 1,
		Diagnostics: []domain.AnalysisDiagnostic{{FilePath: "c.go", Code: domain.DiagnosticCodeRead, Message: "read error"}}}
	if !reflect.DeepEqual(responses.Files, want) {
		t.Errorf("files = %+v", responses.Files)
	}
}

func javascriptResult() *js.Result {
	fragment := &domain.Clone{
		ID: 0, Type: domain.Type2Clone, Language: "JavaScript",
		Location: &domain.CloneLocation{FilePath: "app.js", StartLine: 1, EndLine: 12},
	}
	other := &domain.Clone{
		ID: 1, Type: domain.Type2Clone, Language: "JavaScript",
		Location: &domain.CloneLocation{FilePath: "app.js", StartLine: 20, EndLine: 31},
	}
	return &js.Result{
		Files: domain.AnalysisCoverage{TotalFiles: 1, AnalyzedFiles: 1},
		Complexity: &domain.ComplexityResponse{
			Functions: []domain.FunctionComplexity{{
				Name: "handler", FilePath: "app.js", Language: "JavaScript", StartLine: 1, EndLine: 30,
				Metrics: domain.ComplexityMetrics{Complexity: 5}, RiskLevel: domain.RiskLevelLow,
			}},
			ModuleRollups: map[string]domain.ModuleComplexityMetrics{
				"app.js": {LinesOfCode: 30, AnalyzedFunctionCount: 1, AverageComplexity: 5, MaxComplexity: 5},
			},
			Summary: domain.ComplexitySummary{
				TotalFunctions: 1, FunctionsParsed: 1, AverageComplexity: 5,
				MaxComplexity: 5, MinComplexity: 5, FilesAnalyzed: 1, TotalFiles: 1,
				LowRiskFunctions: 1, ComplexityDistribution: map[int]int{5: 1},
			},
			Config: map[string]interface{}{"low_threshold": 9, "medium_threshold": 19},
		},
		DeadCode: &domain.DeadCodeResponse{},
		Clones: &domain.CloneResponse{
			Clones: []*domain.Clone{fragment, other},
			ClonePairs: []*domain.ClonePair{{
				ID: 0, Clone1: fragment, Clone2: other, Similarity: 0.9, Type: domain.Type2Clone,
			}},
			Statistics: &domain.CloneStatistics{
				TotalFragments: 3, TotalClones: 2, TotalClonePairs: 1,
				ClonesByType: map[string]int{"Type-2": 1}, AverageSimilarity: 0.9,
				LinesAnalyzed: 60, FilesAnalyzed: 1,
			},
			Success: true,
		},
	}
}

func TestCombineMergesLanguages(t *testing.T) {
	responses, err := Combine(genericReport(), javascriptResult())
	if err != nil {
		t.Fatal(err)
	}

	complexity := responses.Complexity
	if len(complexity.Functions) != 3 {
		t.Fatalf("functions = %+v", complexity.Functions)
	}
	// Merged functions stay ranked by complexity across languages.
	names := []string{complexity.Functions[0].Name, complexity.Functions[1].Name, complexity.Functions[2].Name}
	if names[0] != "Sum" || names[1] != "handler" || names[2] != "Add" {
		t.Errorf("ranked functions = %v", names)
	}
	summary := complexity.Summary
	if summary.TotalFunctions != 3 || summary.TotalFiles != 4 || summary.FilesAnalyzed != 3 || summary.SkippedFiles != 1 {
		t.Errorf("summary = %+v", summary)
	}
	if responses.Files.TotalFiles != 4 || responses.Files.AnalyzedFiles != 3 || responses.Files.SkippedFiles != 1 {
		t.Errorf("files = %+v, want both languages' files counted", responses.Files)
	}
	if want := (7.0*2 + 5.0) / 3; summary.AverageComplexity != want {
		t.Errorf("AverageComplexity = %v, want %v", summary.AverageComplexity, want)
	}
	if summary.MaxComplexity != 12 || summary.MinComplexity != 2 {
		t.Errorf("max/min = %d/%d", summary.MaxComplexity, summary.MinComplexity)
	}
	if summary.ComplexityDistribution[5] != 1 || summary.ComplexityDistribution[12] != 1 {
		t.Errorf("distribution = %v", summary.ComplexityDistribution)
	}
	if len(complexity.ModuleRollups) != 3 {
		t.Errorf("rollups = %+v", complexity.ModuleRollups)
	}

	clones := responses.Clone
	if len(clones.ClonePairs) != 2 || len(clones.Clones) != 4 {
		t.Fatalf("clones = %d pairs, %d fragments", len(clones.ClonePairs), len(clones.Clones))
	}
	// Fragment IDs stay unique after the JavaScript response is rebased, and
	// pairs are re-ranked (highest similarity first) and renumbered.
	seen := map[int]bool{}
	for _, fragment := range clones.Clones {
		if seen[fragment.ID] {
			t.Errorf("duplicate fragment ID %d", fragment.ID)
		}
		seen[fragment.ID] = true
	}
	if clones.ClonePairs[0].Similarity != 1 || clones.ClonePairs[0].ID != 0 || clones.ClonePairs[1].ID != 1 {
		t.Errorf("pair ranking = %+v", clones.ClonePairs)
	}
	stats := clones.Statistics
	if stats.TotalFragments != 7 || stats.TotalClones != 4 || stats.TotalClonePairs != 2 ||
		stats.LinesAnalyzed != 200 || stats.FilesAnalyzed != 3 {
		t.Errorf("statistics = %+v", stats)
	}
	if want := (1.0 + 0.9) / 2; stats.AverageSimilarity != want {
		t.Errorf("AverageSimilarity = %v, want %v", stats.AverageSimilarity, want)
	}
	if stats.ClonesByType["Type-1"] != 1 || stats.ClonesByType["Type-2"] != 1 {
		t.Errorf("clones by type = %v", stats.ClonesByType)
	}

	if responses.DeadCode == nil {
		t.Error("the JavaScript dead code response must pass through")
	}
}

func TestCombineJavaScriptOnly(t *testing.T) {
	javascript := javascriptResult()
	responses, err := Combine(nil, javascript)
	if err != nil {
		t.Fatal(err)
	}
	if responses.Complexity != javascript.Complexity || responses.Clone != javascript.Clones {
		t.Error("a JavaScript-only run must pass its responses through unchanged")
	}
}

func TestCombineSelectionWithoutComplexity(t *testing.T) {
	generic := genericReport()
	generic.Complexity = nil
	responses, err := Combine(generic, nil)
	if err != nil {
		t.Fatal(err)
	}
	if responses.Complexity != nil {
		t.Error("no complexity analysis ran, so there must be no complexity response")
	}
	if responses.Clone == nil {
		t.Error("the clone response must survive without complexity")
	}
	if responses.Files.SkippedFiles != 1 {
		t.Errorf("files = %+v, want the skipped file charged without complexity", responses.Files)
	}
}

// TestCombineJavaScriptSkippedFilesWithoutComplexity covers #92 on the
// JavaScript side: a dead-code-only run still carries its unparsable files.
func TestCombineJavaScriptSkippedFilesWithoutComplexity(t *testing.T) {
	javascript := &js.Result{
		Files: domain.AnalysisCoverage{TotalFiles: 2, AnalyzedFiles: 1, SkippedFiles: 1,
			Diagnostics: []domain.AnalysisDiagnostic{{FilePath: "broken.js", Code: domain.DiagnosticCodeParse, Message: "syntax error at line 1"}}},
		DeadCode: &domain.DeadCodeResponse{},
	}
	responses, err := Combine(nil, javascript)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(responses.Files, javascript.Files) {
		t.Errorf("files = %+v, want %+v", responses.Files, javascript.Files)
	}
}

func depsResponse(ids ...string) *domain.DependencyGraphResponse {
	graph := domain.NewDependencyGraph()
	for _, id := range ids {
		graph.AddNode(&domain.ModuleNode{ID: id, Name: id, FilePath: id})
	}
	for i := 1; i < len(ids); i++ {
		graph.AddEdge(&domain.DependencyEdge{From: ids[i-1], To: ids[i], EdgeType: domain.EdgeTypeImport})
	}
	graph.UpdateNodeFlags()
	return &domain.DependencyGraphResponse{
		Graph:       graph,
		Analysis:    &domain.DependencyAnalysisResult{TotalModules: len(ids), MaxDepth: len(ids) - 1},
		Warnings:    []string{ids[0] + " warning"},
		GeneratedAt: "now",
		Version:     "test",
	}
}

func TestCombineGoDependencies(t *testing.T) {
	generic := &analysis.Report{Deps: depsResponse("example.com/m/app", "example.com/m/lib")}

	results, err := Combine(generic, nil)
	if err != nil {
		t.Fatal(err)
	}
	if results.Deps != generic.Deps {
		t.Error("a Go-only run should pass its dependency response through")
	}

	results, err = Combine(generic, &js.Result{Deps: depsResponse("src/a.js", "src/b.js", "src/c.js")})
	if err != nil {
		t.Fatal(err)
	}
	deps := results.Deps
	if deps.Graph.NodeCount() != 5 || deps.Graph.EdgeCount() != 3 {
		t.Errorf("merged graph has %d nodes and %d edges, want 5 and 3", deps.Graph.NodeCount(), deps.Graph.EdgeCount())
	}
	if deps.Analysis.TotalModules != 5 || deps.Analysis.MaxDepth != 2 {
		t.Errorf("merged analysis = modules %d depth %d, want 5 and 2 (re-analyzed over the union)", deps.Analysis.TotalModules, deps.Analysis.MaxDepth)
	}
	if want := []string{"example.com/m/app", "src/a.js"}; !reflect.DeepEqual(deps.Analysis.RootModules, want) {
		t.Errorf("roots = %v, want %v", deps.Analysis.RootModules, want)
	}
	if want := []string{"example.com/m/app warning", "src/a.js warning"}; !reflect.DeepEqual(deps.Warnings, want) {
		t.Errorf("warnings = %v, want %v", deps.Warnings, want)
	}
}

func TestCombineCoupling(t *testing.T) {
	generic := &analysis.Report{Coupling: &analysis.Coupling{
		Classes: []analysis.CoupledClass{
			{Name: "Server", FilePath: "a.go", Language: "Go", StartLine: 3, EndLine: 9, CBO: 4, DependentClasses: []string{"Base", "Config", "model.User", "st.Store"}, Inheritance: 1, TypeHint: 3, RiskLevel: coredomain.RiskLevelMedium},
			{Name: "Config", FilePath: "b.go", Language: "Go", StartLine: 1, EndLine: 2, CBO: 1, DependentClasses: []string{"Base"}, TypeHint: 1, RiskLevel: coredomain.RiskLevelLow},
		},
		TotalClasses:  2,
		FilesAnalyzed: 2,
		Warnings:      []string{"x: no go.mod above it"},
	}}

	results, err := Combine(generic, nil)
	if err != nil {
		t.Fatal(err)
	}
	cbo := results.CBO
	if cbo == nil || len(cbo.Classes) != 2 {
		t.Fatalf("cbo = %+v, want two classes", cbo)
	}
	server := cbo.Classes[0]
	if server.Name != "Server" || server.Language != "Go" || server.Metrics.CouplingCount != 4 ||
		server.Metrics.InheritanceDependencies != 1 || server.Metrics.TypeHintDependencies != 3 ||
		server.RiskLevel != domain.RiskLevelMedium || !reflect.DeepEqual(server.Metrics.DependentClasses, []string{"Base", "Config", "model.User", "st.Store"}) {
		t.Errorf("Server = %+v", server)
	}
	summary := cbo.Summary
	if summary.TotalClasses != 2 || summary.FilesAnalyzed != 2 || summary.MaxCBO != 4 || summary.MinCBO != 1 ||
		summary.AverageCBO != 2.5 || summary.MediumRiskClasses != 1 || summary.LowRiskClasses != 1 ||
		len(summary.MostCoupledClasses) != 2 || summary.CBODistribution["4-7"] != 1 {
		t.Errorf("summary = %+v", summary)
	}
	if !reflect.DeepEqual(cbo.Warnings, []string{"x: no go.mod above it"}) {
		t.Errorf("warnings = %v", cbo.Warnings)
	}

	javascript := &js.Result{CBO: &domain.CBOResponse{
		Classes: []domain.ClassCoupling{{Name: "app", FilePath: "src/app.js", StartLine: 1, EndLine: 40,
			Metrics: domain.CBOMetrics{CouplingCount: 3, DependentClasses: []string{"react", "util", "zod"}}, RiskLevel: domain.RiskLevelLow}},
		Summary:     domain.CBOSummary{TotalClasses: 1, FilesAnalyzed: 1},
		Warnings:    []string{"js warning"},
		GeneratedAt: "then",
		Version:     "1",
		Config:      map[string]any{"low_threshold": 7},
	}}
	results, err = Combine(generic, javascript)
	if err != nil {
		t.Fatal(err)
	}
	cbo = results.CBO
	names := []string{cbo.Classes[0].Name, cbo.Classes[1].Name, cbo.Classes[2].Name}
	if !reflect.DeepEqual(names, []string{"Server", "app", "Config"}) {
		t.Errorf("merged ranking = %v, want Server, app, Config", names)
	}
	if cbo.Classes[1].Language != "" {
		t.Errorf("the JavaScript module carries no language, got %q", cbo.Classes[1].Language)
	}
	summary = cbo.Summary
	if summary.TotalClasses != 3 || summary.FilesAnalyzed != 3 || summary.LowRiskClasses != 2 || summary.MediumRiskClasses != 1 {
		t.Errorf("merged summary = %+v", summary)
	}
	if !reflect.DeepEqual(cbo.Warnings, []string{"x: no go.mod above it", "js warning"}) || cbo.GeneratedAt != "then" || cbo.Config.(map[string]any)["low_threshold"] != 7 {
		t.Errorf("merged metadata = warnings %v at %q with %v; the JavaScript configuration wins", cbo.Warnings, cbo.GeneratedAt, cbo.Config)
	}
}
