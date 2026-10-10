package main

import (
	"errors"
	"fmt"
	"io"
	"slices"
	"strings"

	"github.com/ludo-technologies/polyscan/polyscan/internal/config"
	jsdomain "github.com/ludo-technologies/polyscan/polyscan/internal/js/domain"
	"github.com/spf13/cobra"
)

// Exit codes of polyscan check. CI pipelines branch on them: a quality issue
// is a verdict about the code, while an analysis error means the verdict
// itself cannot be trusted.
const (
	// exitCodeQualityIssues means the analysis ran and found problems. Every
	// other command also exits with it on any error.
	exitCodeQualityIssues = 1
	// exitCodeAnalysisError means the analysis could not be completed over
	// the requested paths, so the result must not be read as a pass.
	exitCodeAnalysisError = 2
)

// analysisError marks an error as an analysis failure rather than a quality
// verdict, so main maps it to exitCodeAnalysisError.
type analysisError struct {
	err error
}

func (e *analysisError) Error() string { return e.err.Error() }

func (e *analysisError) Unwrap() error { return e.err }

// exitCodeFor maps a command error to the process exit code.
func exitCodeFor(err error) int {
	var analysisErr *analysisError
	if errors.As(err, &analysisErr) {
		return exitCodeAnalysisError
	}
	return exitCodeQualityIssues
}

// checkAnalyses are the analyses --select accepts. Clone detection lists clones
// without failing the check; cbo and lcom have no threshold to check.
var checkAnalyses = []string{selectComplexity, selectDeadCode, selectClone, selectDeps}

type checkOptions struct {
	selected []string
	quiet    bool
	files    fileFlags
	// limits holds the threshold flags until load lays them over the
	// configuration's, and the merged thresholds after.
	limits config.Check
}

func checkCmd() *cobra.Command {
	var opts checkOptions

	cmd := &cobra.Command{
		Use:   "check [path...]",
		Short: "Check code quality against thresholds for CI",
		Long: `Check code quality against fixed thresholds, for CI pipelines.

The check fails on:
  - a function with cyclomatic complexity above --max-complexity (default 10)
  - critical dead code, such as code after a return (JavaScript/TypeScript only)
  - more circular dependency cycles than --max-cycles (default 0)
  - a file that cannot be read or parsed (Go, Rust and C++ files parse with
    recovery: a syntax error leaves out only the functions containing it)

Complexity, dead code and dependency analysis run by default. Clone detection
runs only when selected with --select, and clones never fail the check. With no
path, the current directory is checked.

The thresholds and the files to check are read from the [check] and
[analysis] sections of the nearest ` + config.FileName + ` in the first path or a
directory above it, or of the file given with --config. A flag takes
precedence over the file, and --exclude adds to its patterns.

Exit codes:
  0  No issues found
  1  Quality issues found
  2  Analysis failed (invalid flag, missing path, unparsable file, etc.)

Examples:
  polyscan check                               # Check the current directory
  polyscan check --select complexity src/      # Complexity only
  polyscan check --max-cycles 3 .              # Allow up to 3 dependency cycles
  polyscan check --allow-dead-code .           # Report dead code without failing
  polyscan check --allow-parse-errors .        # Skip unparsable files without failing
  polyscan check --quiet .                     # Print only warnings unless issues are found`,
		Args: cobra.ArbitraryArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) == 0 {
				args = []string{"."}
			}
			cfg, err := opts.load(cmd, args[0])
			if err != nil {
				return &analysisError{err: err}
			}
			issues, err := opts.run(args, cfg, cmd.ErrOrStderr())
			if err != nil {
				return &analysisError{err: err}
			}
			if issues > 0 {
				return fmt.Errorf("found %d quality issue(s)", issues)
			}
			return nil
		},
	}
	// An unusable invocation is not a verdict about the code: without this,
	// an unknown flag would exit 1 and read as a quality failure.
	cmd.SetFlagErrorFunc(func(_ *cobra.Command, err error) error {
		return &analysisError{err: err}
	})

	cmd.Flags().StringSliceVarP(&opts.selected, "select", "s",
		[]string{selectComplexity, selectDeadCode, selectDeps},
		"Analyses to run (comma-separated): "+strings.Join(checkAnalyses, ","))
	defaults := config.Default().Check
	cmd.Flags().IntVar(&opts.limits.MaxComplexity, "max-complexity", defaults.MaxComplexity, "Maximum allowed cyclomatic complexity of a function")
	cmd.Flags().IntVar(&opts.limits.MaxCycles, "max-cycles", defaults.MaxCycles, "Maximum allowed circular dependency cycles")
	cmd.Flags().BoolVar(&opts.limits.AllowDeadCode, "allow-dead-code", defaults.AllowDeadCode, "Report dead code without failing")
	cmd.Flags().BoolVar(&opts.limits.AllowCircularDeps, "allow-circular-deps", defaults.AllowCircularDeps, "Report circular dependencies without failing")
	cmd.Flags().BoolVar(&opts.limits.AllowParseErrors, "allow-parse-errors", defaults.AllowParseErrors, "Skip unparsable files without failing; unreadable files still fail")
	cmd.Flags().BoolVarP(&opts.quiet, "quiet", "q", false, "Print nothing but warnings unless issues are found")
	opts.files.add(cmd)
	return cmd
}

// load loads the configuration for a check of target and lays the threshold
// flags given on the command line over its [check] section.
func (o *checkOptions) load(cmd *cobra.Command, target string) (*config.Config, error) {
	cfg, err := o.files.load(cmd, target)
	if err != nil {
		return nil, err
	}
	override(cmd, "max-complexity", &cfg.Check.MaxComplexity, o.limits.MaxComplexity)
	override(cmd, "max-cycles", &cfg.Check.MaxCycles, o.limits.MaxCycles)
	override(cmd, "allow-dead-code", &cfg.Check.AllowDeadCode, o.limits.AllowDeadCode)
	override(cmd, "allow-circular-deps", &cfg.Check.AllowCircularDeps, o.limits.AllowCircularDeps)
	override(cmd, "allow-parse-errors", &cfg.Check.AllowParseErrors, o.limits.AllowParseErrors)
	o.limits = cfg.Check
	return cfg, nil
}

// run analyzes paths and writes each finding to w. It returns the number of
// quality issues, or an error when the analysis could not give a verdict.
// Lines that inform without failing the check, such as clones and allowed
// findings, are left out in quiet mode.
func (o checkOptions) run(paths []string, cfg *config.Config, w io.Writer) (int, error) {
	if o.limits.MaxComplexity < 1 {
		return 0, fmt.Errorf("--max-complexity must be at least 1")
	}
	if o.limits.MaxCycles < 0 {
		return 0, fmt.Errorf("--max-cycles must not be negative")
	}
	for _, name := range o.selected {
		if !slices.Contains(checkAnalyses, name) {
			return 0, fmt.Errorf("invalid analysis %q, must be one of: %s", name, strings.Join(checkAnalyses, ", "))
		}
	}
	options, selection, err := parseSelection(o.selected)
	if err != nil {
		return 0, err
	}

	info := w
	if o.quiet {
		info = io.Discard
	}
	fmt.Fprintf(info, "Running quality check (%s)...\n", strings.Join(o.selected, ", "))

	results, failures, err := runAnalyses(paths, options, selection, cfg)
	if err != nil {
		return 0, err
	}
	for _, failure := range failures {
		fmt.Fprintln(w, failure)
	}
	if len(failures) > 0 {
		return 0, fmt.Errorf("%d analysis(es) failed", len(failures))
	}
	if err := o.checkDiagnostics(results.Files.Diagnostics, w, info); err != nil {
		return 0, err
	}

	issues := 0
	if selection.Complexity {
		issues += o.complexityIssues(results.Complexity, w)
	}
	if selection.DeadCode {
		issues += o.deadCodeIssues(results.DeadCode, w, info)
	}
	if selection.Clones {
		reportClones(results.Clone, info)
	}
	if selection.Deps {
		issues += o.cycleIssues(results.Deps, w, info)
	}
	if issues == 0 {
		fmt.Fprintln(info, "Code quality check passed")
	}
	return issues, nil
}

// checkDiagnostics reports each file the analysis could not use and fails
// unless every one is a parse error that --allow-parse-errors waives.
func (o checkOptions) checkDiagnostics(diagnostics []jsdomain.AnalysisDiagnostic, w, info io.Writer) error {
	blocking := 0
	for _, diagnostic := range diagnostics {
		if o.limits.AllowParseErrors && diagnostic.Code == jsdomain.DiagnosticCodeParse {
			fmt.Fprintln(info, diagnostic)
			continue
		}
		fmt.Fprintln(w, diagnostic)
		blocking++
	}
	if blocking == 0 {
		return nil
	}
	if o.limits.AllowParseErrors {
		return fmt.Errorf("%d file(s) could not be read", blocking)
	}
	return fmt.Errorf("%d file(s) could not be analyzed (use --allow-parse-errors to skip parse errors)", blocking)
}

// complexityIssues counts the functions above --max-complexity among every
// analyzed function, not only the ones the report filters leave visible.
func (o checkOptions) complexityIssues(complexity *jsdomain.ComplexityResponse, w io.Writer) int {
	if complexity == nil {
		return 0
	}
	issues := 0
	for _, fn := range complexity.AnalyzedFunctions {
		if fn.Metrics.Complexity > o.limits.MaxComplexity {
			issues++
			fmt.Fprintf(w, "%s:%d: %s is too complex (%d > %d)\n",
				fn.FilePath, fn.StartLine, fn.Name, fn.Metrics.Complexity, o.limits.MaxComplexity)
		}
	}
	return issues
}

// deadCodeIssues counts the critical dead code findings. Unused imports and
// exports are warnings at most and never fail the check.
func (o checkOptions) deadCodeIssues(deadCode *jsdomain.DeadCodeResponse, w, info io.Writer) int {
	if deadCode == nil {
		return 0
	}
	out := w
	if o.limits.AllowDeadCode {
		out = info
	}
	found := 0
	report := func(findings []jsdomain.DeadCodeFinding) {
		for _, finding := range findings {
			if finding.Severity.IsAtLeast(jsdomain.DeadCodeSeverityCritical) {
				found++
				fmt.Fprintf(out, "%s:%d: %s\n", finding.Location.FilePath, finding.Location.StartLine, finding.Description)
			}
		}
	}
	for _, file := range deadCode.Files {
		for _, fn := range file.Functions {
			report(fn.Findings)
		}
		report(file.FileLevelFindings)
	}
	if o.limits.AllowDeadCode {
		if found > 0 {
			fmt.Fprintf(info, "Found %d dead code issue(s) (allowed by --allow-dead-code)\n", found)
		}
		return 0
	}
	return found
}

func reportClones(clones *jsdomain.CloneResponse, info io.Writer) {
	if clones == nil || len(clones.ClonePairs) == 0 {
		return
	}
	for _, pair := range clones.ClonePairs {
		fmt.Fprintf(info, "%s:%d: clone of %s:%d (similarity: %.1f%%)\n",
			pair.Clone1.Location.FilePath, pair.Clone1.Location.StartLine,
			pair.Clone2.Location.FilePath, pair.Clone2.Location.StartLine,
			pair.Similarity*100)
	}
	fmt.Fprintf(info, "Found %d code clone(s) (informational)\n", len(clones.ClonePairs))
}

// cycleIssues counts the circular dependency cycles when there are more than
// --max-cycles of them. Each cycle counts as one issue.
func (o checkOptions) cycleIssues(deps *jsdomain.DependencyGraphResponse, w, info io.Writer) int {
	if deps == nil || deps.Analysis == nil || deps.Analysis.CircularDependencies == nil {
		return 0
	}
	cycles := deps.Analysis.CircularDependencies.CircularDependencies
	failing := len(cycles) > o.limits.MaxCycles && !o.limits.AllowCircularDeps
	out := info
	if failing {
		out = w
	}
	for _, cycle := range cycles {
		fmt.Fprintf(out, "circular dependency: %s\n", strings.Join(cycle.Modules, " -> "))
	}
	switch {
	case failing:
		return len(cycles)
	case len(cycles) > o.limits.MaxCycles:
		fmt.Fprintf(info, "Found %d circular dependency cycle(s) (allowed by --allow-circular-deps)\n", len(cycles))
	case len(cycles) > 0:
		fmt.Fprintf(info, "Found %d circular dependency cycle(s) (within --max-cycles %d)\n", len(cycles), o.limits.MaxCycles)
	}
	return 0
}
