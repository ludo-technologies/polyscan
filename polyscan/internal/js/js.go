// Package js runs the jscan JavaScript/TypeScript analyses as one pipeline
// for polyscan analyze. The packages below it are the jscan implementation;
// this file is the entry point that configures, collects files and runs the
// analyses.
package js

import (
	"context"
	"fmt"
	"sync"

	"github.com/ludo-technologies/polyscan/polyscan/internal/js/analyzer"
	"github.com/ludo-technologies/polyscan/polyscan/internal/js/app"
	"github.com/ludo-technologies/polyscan/polyscan/internal/js/config"
	"github.com/ludo-technologies/polyscan/polyscan/internal/js/domain"
	"github.com/ludo-technologies/polyscan/polyscan/internal/js/service"
)

// Selection selects the analyses to run.
type Selection struct {
	Complexity bool
	DeadCode   bool
	Clones     bool
	CBO        bool
	Deps       bool
}

// AllAnalyses selects every analysis.
func AllAnalyses() Selection {
	return Selection{Complexity: true, DeadCode: true, Clones: true, CBO: true, Deps: true}
}

func (s Selection) count() int {
	count := 0
	for _, selected := range []bool{s.Complexity, s.DeadCode, s.Clones, s.CBO, s.Deps} {
		if selected {
			count++
		}
	}
	return count
}

// Result holds each analysis response next to its error. An analysis that
// was not selected leaves both nil; one that failed leaves the response nil
// and the error set, and the others still stand. Files is the run's own
// accounting of the files it covered and could not use, kept apart from the
// analyses so it is complete whichever of them ran.
type Result struct {
	Files         domain.AnalysisCoverage
	Complexity    *domain.ComplexityResponse
	ComplexityErr error
	DeadCode      *domain.DeadCodeResponse
	DeadCodeErr   error
	Clones        *domain.CloneResponse
	ClonesErr     error
	CBO           *domain.CBOResponse
	CBOErr        error
	Deps          *domain.DependencyGraphResponse
	DepsErr       error
}

// Failures returns the error of each analysis that failed, each naming its
// analysis.
func (r *Result) Failures() []error {
	var failures []error
	for _, failure := range []struct {
		name string
		err  error
	}{
		{"complexity", r.ComplexityErr},
		{"dead code", r.DeadCodeErr},
		{"clone", r.ClonesErr},
		{"CBO", r.CBOErr},
		{"dependency", r.DepsErr},
	} {
		if failure.err != nil {
			failures = append(failures, fmt.Errorf("JavaScript %s analysis error: %w", failure.name, failure.err))
		}
	}
	return failures
}

// Config returns the settings of the JavaScript/TypeScript analysis: jscan's
// defaults, with the complexity risk bands given and the exclude patterns
// added to jscan's own.
func Config(lowThreshold, mediumThreshold int, exclude []string) *config.Config {
	cfg := config.DefaultConfig()
	cfg.Complexity.LowThreshold = lowThreshold
	cfg.Complexity.MediumThreshold = mediumThreshold
	cfg.Analysis.ExcludePatterns = append(cfg.Analysis.ExcludePatterns, exclude...)
	return cfg
}

// ContainsFiles reports whether any JavaScript/TypeScript file exists under
// the paths, before any include or exclude pattern applies.
func ContainsFiles(paths []string) (bool, error) {
	files, err := app.NewFileHelper().CollectJSFiles(paths, true, nil, nil)
	if err != nil {
		return false, err
	}
	return len(files) > 0, nil
}

// CollectFiles collects the JavaScript/TypeScript files under each path,
// honoring the configuration's include and exclude patterns. Test files, as
// analyzer.IsTestFile names them, are left out unless includeTests is set.
func CollectFiles(paths []string, cfg *config.Config, includeTests bool) ([]string, error) {
	helper := app.NewFileHelper()
	var files []string
	for _, path := range paths {
		pathFiles, err := helper.CollectJSFiles([]string{path}, cfg.Analysis.Recursive, cfg.Analysis.IncludePatterns, cfg.Analysis.ExcludePatterns)
		if err != nil {
			return nil, fmt.Errorf("failed to collect files from %s: %w", path, err)
		}
		for _, file := range pathFiles {
			if includeTests || !analyzer.IsTestFile(file) {
				files = append(files, file)
			}
		}
	}
	return files, nil
}

// Run executes the selected analyses over files in parallel and collects
// every response and error into one Result.
//
// With several analyses selected, every file is parsed once and the parse
// trees are shared across them. Nothing references the snapshot after the
// goroutines below finish, so the shared trees become collectable as soon as
// the analyses are done with them — clone detection drops its per-fragment
// AST references itself once fragments are converted for APTED. A single
// selected analysis has nobody to share with and gets a snapshot that loads
// each file inside its fan-out and releases it right after, which holds far
// fewer parse trees at once. Dependency analysis is the exception: graph
// construction needs every module's tree at once.
//
// Test files among files count for complexity and dead code only. Clone
// detection, CBO and dependency analysis run over the other files: test
// functions share a skeleton by convention, and a test's classes and imports
// describe the tests, not the modules under test.
func Run(ctx context.Context, files []string, cfg *config.Config, selected Selection) *Result {
	var sources []string
	for _, file := range files {
		if !analyzer.IsTestFile(file) {
			sources = append(sources, file)
		}
	}
	// The snapshot holds the files some selected analysis reads; a file
	// no analysis loads would leave the accounting incomplete.
	if !selected.Complexity && !selected.DeadCode {
		files = sources
	}
	var snapshot *service.ProjectSnapshot
	if selected.count() > 1 || selected.Deps {
		snapshot = service.BuildProjectSnapshot(ctx, files)
	} else {
		snapshot = service.NewProjectSnapshot(files)
	}
	sourceSnapshot := snapshot
	if len(sources) != len(files) {
		sourceSnapshot = snapshot.Subset(func(path string) bool { return !analyzer.IsTestFile(path) })
	}

	result := &Result{}
	var wg sync.WaitGroup
	var mu sync.Mutex

	if selected.Complexity {
		wg.Add(1)
		go func() {
			defer wg.Done()
			resp, err := runComplexity(ctx, snapshot, files, cfg)
			mu.Lock()
			result.Complexity, result.ComplexityErr = resp, err
			mu.Unlock()
		}()
	}

	if selected.DeadCode {
		wg.Add(1)
		go func() {
			defer wg.Done()
			resp, err := runDeadCode(ctx, snapshot, files, cfg)
			mu.Lock()
			result.DeadCode, result.DeadCodeErr = resp, err
			mu.Unlock()
		}()
	}

	if selected.Clones {
		wg.Add(1)
		go func() {
			defer wg.Done()
			resp, err := runClones(ctx, sourceSnapshot, sources)
			mu.Lock()
			result.Clones, result.ClonesErr = resp, err
			mu.Unlock()
		}()
	}

	if selected.CBO {
		wg.Add(1)
		go func() {
			defer wg.Done()
			resp, err := runCBO(ctx, sourceSnapshot, sources)
			mu.Lock()
			result.CBO, result.CBOErr = resp, err
			mu.Unlock()
		}()
	}

	if selected.Deps {
		wg.Add(1)
		go func() {
			defer wg.Done()
			resp, err := runDeps(ctx, sourceSnapshot, sources)
			mu.Lock()
			result.Deps, result.DepsErr = resp, err
			mu.Unlock()
		}()
	}

	wg.Wait()
	result.Files = snapshot.Coverage()
	return result
}

// runComplexity runs complexity analysis over the snapshot.
func runComplexity(ctx context.Context, snapshot *service.ProjectSnapshot, files []string, cfg *config.Config) (*domain.ComplexityResponse, error) {
	svc := service.NewComplexityService(&cfg.Complexity)

	req := domain.ComplexityRequest{
		Paths:           files,
		LowThreshold:    cfg.Complexity.LowThreshold,
		MediumThreshold: cfg.Complexity.MediumThreshold,
		MinComplexity:   cfg.Output.MinComplexity,
		SortBy:          domain.SortCriteria(cfg.Output.SortBy),
	}

	return svc.AnalyzeSnapshot(ctx, snapshot, req)
}

// runDeadCode runs dead code analysis over the snapshot.
func runDeadCode(ctx context.Context, snapshot *service.ProjectSnapshot, files []string, cfg *config.Config) (*domain.DeadCodeResponse, error) {
	return service.AnalyzeDeadCodeSnapshot(ctx, snapshot, DeadCodeRequest(files, cfg))
}

// DeadCodeRequest builds the dead code request.
func DeadCodeRequest(files []string, cfg *config.Config) domain.DeadCodeRequest {
	return domain.DeadCodeRequest{
		Paths:       files,
		MinSeverity: domain.DeadCodeSeverity(cfg.DeadCode.MinSeverity),
		SortBy:      domain.DeadCodeSortCriteria(cfg.DeadCode.SortBy),
	}
}

// runClones runs clone detection over the snapshot.
func runClones(ctx context.Context, snapshot *service.ProjectSnapshot, files []string) (*domain.CloneResponse, error) {
	svc := service.NewCloneServiceWithDefaults()

	req := domain.DefaultCloneRequest()
	req.Paths = files

	return svc.DetectClonesInSnapshot(ctx, snapshot, req)
}

// runCBO runs CBO analysis over the snapshot.
func runCBO(ctx context.Context, snapshot *service.ProjectSnapshot, files []string) (*domain.CBOResponse, error) {
	svc := service.NewCBOServiceWithDefaults()

	req := domain.CBORequest{
		Paths: files,
	}

	return svc.AnalyzeSnapshot(ctx, snapshot, req)
}

// runDeps runs dependency analysis over the snapshot.
func runDeps(ctx context.Context, snapshot *service.ProjectSnapshot, files []string) (*domain.DependencyGraphResponse, error) {
	svc := service.NewDependencyGraphServiceWithDefaults()

	req := domain.DependencyGraphRequest{
		Paths:        files,
		DetectCycles: domain.BoolPtr(true),
	}

	return svc.AnalyzeSnapshot(ctx, snapshot, req)
}
