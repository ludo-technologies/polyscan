package analysis

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/ludo-technologies/polyscan/core/domain"
)

func writeFiles(t *testing.T, files map[string]string) string {
	t.Helper()
	dir := t.TempDir()
	for name, content := range files {
		path := filepath.Join(dir, name)
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	return dir
}

func TestAnalyzeCohesionGroupsGoMethodsByPackage(t *testing.T) {
	dir := writeFiles(t, map[string]string{
		// Server's methods span two files. Start/Stop share running,
		// Log calls a function-typed field that Stop also touches, and
		// Version touches nothing: three components in total.
		"server.go": `package p

type Server struct {
	running bool
	logf    func(string)
}

func (s *Server) Start() { s.running = true }
func (s *Server) Stop()  { s.running = false; s.logf("bye") }
func (Server) Version() string { return "1" }
`,
		"log.go": `package p

func (s *Server) Log(msg string) { s.logf(msg) }
func (s *Server) Name() string  { return "srv" }
`,
		// A type declared in a test file is a fixture and stays out.
		"server_test.go": `package p

type fake struct{ n int }

func (f *fake) A() { f.n++ }
func (f *fake) B() {}
`,
		// A second package with a type of the same name is another class.
		"other/server.go": `package other

type Server struct{ a int }

func (s *Server) A() { s.a++ }
func (s *Server) B() {}
`,
	})

	report, err := Analyze([]string{dir}, Options{LCOM: true}, nil)
	if err != nil {
		t.Fatalf("Analyze: %v", err)
	}
	if report.Cohesion == nil {
		t.Fatal("cohesion is nil for a Go tree")
	}
	if report.Complexity != nil || report.Clones != nil {
		t.Error("only cohesion was selected")
	}
	classes := report.Cohesion.Classes
	if len(classes) != 2 {
		t.Fatalf("got %d classes, want 2: %+v", len(classes), classes)
	}

	server := classes[0]
	if server.Name != "Server" || server.Language != "Go" || filepath.Base(server.FilePath) != "log.go" {
		t.Errorf("first class = %+v, want Server placed in log.go", server)
	}
	if server.LCOM4 != 2 || server.TotalMethods != 5 || server.ExcludedMethods != 1 {
		t.Errorf("Server: LCOM4 %d, methods %d, excluded %d; want 2, 5, 1", server.LCOM4, server.TotalMethods, server.ExcludedMethods)
	}
	wantGroups := [][]string{{"Log", "Start", "Stop"}, {"Name"}}
	if !reflect.DeepEqual(server.MethodGroups, wantGroups) {
		t.Errorf("Server groups = %v, want %v", server.MethodGroups, wantGroups)
	}
	if !reflect.DeepEqual(server.InstanceVariables, []string{"logf", "running"}) {
		t.Errorf("Server variables = %v", server.InstanceVariables)
	}
	if server.StartLine != 3 || server.EndLine != 4 {
		t.Errorf("Server spans %d-%d in log.go, want 3-4", server.StartLine, server.EndLine)
	}

	other := classes[1]
	if other.LCOM4 != 2 || filepath.Base(filepath.Dir(other.FilePath)) != "other" {
		t.Errorf("second class = %+v, want other.Server with LCOM4 2", other)
	}

	summary := report.Cohesion.Summary
	want := CohesionSummary{TotalClasses: 2, AverageLCOM: 2, MaxLCOM: 2, MinLCOM: 2, LowRiskClasses: 2}
	if summary != want {
		t.Errorf("summary = %+v, want %+v", summary, want)
	}
}

func TestAnalyzeCohesionRust(t *testing.T) {
	dir := writeFiles(t, map[string]string{
		"lib.rs": `
pub struct Counter { n: u32, log: Vec<String> }

impl Counter {
    pub fn new() -> Self { Counter { n: 0, log: vec![] } }
    pub fn inc(&mut self) { self.n += 1; self.record() }
    fn record(&mut self) { self.log.push(String::new()) }
    pub fn a(&self) -> u32 { 1 }
    pub fn b(&self) -> u32 { 2 }
    pub fn c(&self) -> u32 { 3 }
    pub fn d(&self) -> u32 { 4 }
    pub fn e(&self) -> u32 { 5 }
}

pub struct Builder;
impl Builder { pub fn new() -> Self { Builder } }

#[cfg(test)]
mod tests {
    struct Fixture;
    impl Fixture { fn x(&self) {} fn y(&self) {} }
}
`,
	})

	report, err := Analyze([]string{dir}, Options{LCOM: true}, nil)
	if err != nil {
		t.Fatalf("Analyze: %v", err)
	}
	classes := report.Cohesion.Classes
	if len(classes) != 1 {
		t.Fatalf("got %d classes, want only Counter (Builder has no method with a receiver): %+v", len(classes), classes)
	}
	counter := classes[0]
	if counter.Name != "Counter" || counter.LCOM4 != 6 || counter.ExcludedMethods != 1 || counter.RiskLevel != domain.RiskLevelHigh {
		t.Errorf("Counter = %+v, want LCOM4 6 (high) with new excluded", counter)
	}
}

func TestAnalyzeCohesionRustUsesDeclaringFile(t *testing.T) {
	// A Rust type whose declaration and impl blocks live in different
	// files should use the declaring file in both coupling and cohesion,
	// so countClasses counts it once.
	dir := writeFiles(t, map[string]string{
		"unit.rs": `pub struct Unit;
`,
		"types.rs": `pub struct Foo { n: u32, u: Unit }

impl Foo {
    pub fn new() -> Self { Foo { n: 0, u: Unit } }
}
`,
		"ops.rs": `use crate::types::Foo;

impl Foo {
    pub fn inc(&mut self) { self.n += 1; }
    pub fn get(&self) -> u32 { self.n }
}
`,
	})

	report, err := Analyze([]string{dir}, Options{LCOM: true, CBO: true}, nil)
	if err != nil {
		t.Fatalf("Analyze: %v", err)
	}
	if report.Cohesion == nil {
		t.Fatal("cohesion is nil for a Rust tree")
	}
	classes := report.Cohesion.Classes
	if len(classes) != 1 {
		t.Fatalf("got %d classes, want 1 (Foo): %+v", len(classes), classes)
	}
	foo := classes[0]
	if foo.Name != "Foo" || foo.Language != "Rust" {
		t.Errorf("class = %+v, want Rust Foo", foo)
	}
	// The declaring file is types.rs; without the fix it would be ops.rs.
	// Only the ops.rs methods are measured (new has no receiver and its
	// group is left out), and the line span is the declaration's.
	if filepath.Base(foo.FilePath) != "types.rs" {
		t.Errorf("FilePath = %q, want types.rs (the declaring file)", foo.FilePath)
	}
	if foo.TotalMethods != 2 {
		t.Errorf("TotalMethods = %d, want 2 (inc, get)", foo.TotalMethods)
	}
	if foo.StartLine != 1 || foo.EndLine != 1 {
		t.Errorf("span = %d-%d, want the declaration's 1-1 in types.rs, not the methods' lines in ops.rs", foo.StartLine, foo.EndLine)
	}

	// Coupling should also use the declaring file.
	if report.Coupling == nil {
		t.Fatal("coupling is nil for a Rust tree")
	}
	if len(report.Coupling.Classes) != 1 {
		t.Fatalf("coupling classes = %d, want 1", len(report.Coupling.Classes))
	}
	couplingFoo := report.Coupling.Classes[0]
	if filepath.Base(couplingFoo.FilePath) != "types.rs" {
		t.Errorf("coupling FilePath = %q, want types.rs", couplingFoo.FilePath)
	}

	// Both analyses must agree on the declaring file so that
	// countClasses merges them into one type instead of counting two.
	if couplingFoo.FilePath != foo.FilePath {
		t.Errorf("FilePath mismatch: coupling=%q, cohesion=%q, want both types.rs",
			couplingFoo.FilePath, foo.FilePath)
	}
}

func TestAnalyzeCohesionRustLCOMAloneUsesDeclaringFile(t *testing.T) {
	// The declaring file must be used even without the coupling analysis:
	// --select lcom alone collects the declarations itself.
	dir := writeFiles(t, map[string]string{
		"types.rs": `pub struct Foo { n: u32 }

impl Foo {
    pub fn new() -> Self { Foo { n: 0 } }
}
`,
		"ops.rs": `use crate::types::Foo;

impl Foo {
    pub fn inc(&mut self) { self.n += 1; }
    pub fn get(&self) -> u32 { self.n }
}
`,
	})

	report, err := Analyze([]string{dir}, Options{LCOM: true}, nil)
	if err != nil {
		t.Fatalf("Analyze: %v", err)
	}
	if report.Coupling != nil {
		t.Fatal("coupling was not selected but is present")
	}
	classes := report.Cohesion.Classes
	if len(classes) != 1 {
		t.Fatalf("got %d classes, want 1 (Foo): %+v", len(classes), classes)
	}
	foo := classes[0]
	if filepath.Base(foo.FilePath) != "types.rs" {
		t.Errorf("FilePath = %q, want types.rs (the declaring file)", foo.FilePath)
	}
	if foo.StartLine != 1 || foo.EndLine != 1 {
		t.Errorf("span = %d-%d, want the declaration's 1-1", foo.StartLine, foo.EndLine)
	}
}

func TestAnalyzeCohesionRustSameNamedTypesStayDistinct(t *testing.T) {
	// Two unrelated Foo types in different files must each keep their own
	// file: an ambiguous bare name is never attributed to the file seen
	// first.
	dir := writeFiles(t, map[string]string{
		"a.rs": `pub struct Foo { n: u32 }

impl Foo {
    pub fn inc(&mut self) { self.n += 1; }
    pub fn get(&self) -> u32 { self.n }
}
`,
		"b.rs": `pub struct Foo { s: String }

impl Foo {
    pub fn push(&mut self, v: String) { self.s.push_str(&v); }
    pub fn len(&self) -> usize { self.s.len() }
}
`,
	})

	report, err := Analyze([]string{dir}, Options{LCOM: true}, nil)
	if err != nil {
		t.Fatalf("Analyze: %v", err)
	}
	byFile := map[string]int{}
	for _, class := range report.Cohesion.Classes {
		if class.Language == "Rust" && class.Name == "Foo" {
			byFile[filepath.Base(class.FilePath)]++
		}
	}
	want := map[string]int{"a.rs": 1, "b.rs": 1}
	if !reflect.DeepEqual(byFile, want) {
		t.Errorf("Foo classes by file = %v, want %v", byFile, want)
	}
}

func TestAnalyzeCohesionAbsentWithoutSupportedLanguage(t *testing.T) {
	dir := writeFiles(t, map[string]string{"a.cpp": "struct S { int a; void m() { a = 1; } };\n"})
	report, err := Analyze([]string{dir}, Options{LCOM: true, Complexity: true}, nil)
	if err != nil {
		t.Fatalf("Analyze: %v", err)
	}
	if report.Cohesion != nil {
		t.Errorf("cohesion = %+v, want none for a C++ tree", report.Cohesion)
	}
	dir = writeFiles(t, map[string]string{"a.go": "package p\n\nfunc F() {}\n"})
	report, err = Analyze([]string{dir}, Options{LCOM: true}, nil)
	if err != nil {
		t.Fatalf("Analyze: %v", err)
	}
	if report.Cohesion == nil || report.Cohesion.Summary.TotalClasses != 0 {
		t.Errorf("cohesion = %+v, want an empty result for a Go tree without types", report.Cohesion)
	}
}
