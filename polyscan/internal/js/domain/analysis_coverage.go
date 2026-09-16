package domain

import "fmt"

// DiagnosticCode identifies why a discovered source file was not analyzed.
type DiagnosticCode string

const (
	// DiagnosticCodeRead identifies a source file that could not be read.
	DiagnosticCodeRead DiagnosticCode = "read_error"
	// DiagnosticCodeParse identifies source that could not be parsed.
	DiagnosticCodeParse DiagnosticCode = "parse_error"
)

// AnalysisDiagnostic records why a discovered source file was not analyzed.
type AnalysisDiagnostic struct {
	FilePath string         `json:"file_path" yaml:"file_path"`
	Code     DiagnosticCode `json:"code" yaml:"code"`
	Message  string         `json:"message" yaml:"message"`
}

// String renders the diagnostic the way the per-analysis error lists and the
// CLI summary report it.
func (d AnalysisDiagnostic) String() string {
	return fmt.Sprintf("[%s] %s: %s", d.FilePath, d.Code, d.Message)
}

// DiagnosticMessages renders each diagnostic with String.
func DiagnosticMessages(diagnostics []AnalysisDiagnostic) []string {
	messages := make([]string, 0, len(diagnostics))
	for _, diagnostic := range diagnostics {
		messages = append(messages, diagnostic.String())
	}
	return messages
}

// AnalysisCoverage counts the files a run covered, kept apart from any one
// analysis: every analysis silently excludes the files it cannot read or
// parse, so the health score charges them separately whichever dimensions
// ran, and Diagnostics says per skipped file why.
type AnalysisCoverage struct {
	TotalFiles    int
	AnalyzedFiles int
	SkippedFiles  int
	Diagnostics   []AnalysisDiagnostic
}

// Add folds another run's coverage into this one, as when the generic engine
// and the JavaScript pipeline each cover part of a tree.
func (c *AnalysisCoverage) Add(other AnalysisCoverage) {
	c.TotalFiles += other.TotalFiles
	c.AnalyzedFiles += other.AnalyzedFiles
	c.SkippedFiles += other.SkippedFiles
	c.Diagnostics = append(c.Diagnostics, other.Diagnostics...)
}
