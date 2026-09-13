package analysis

import (
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/ludo-technologies/polyscan/core/domain"
)

func TestAnalyzeCouplingGo(t *testing.T) {
	dir := writeFiles(t, map[string]string{
		"go.mod": "module example.com/app\n",
		// Server depends on its own package's Handler and Config, on
		// model.User and store.Store through imports, one of them aliased,
		// and on nothing from the standard library or other modules.
		"server.go": `package app

import (
	"context"
	"sync"

	"example.com/app/model"
	st "example.com/app/store"
	"github.com/other/lib"
)

type Server struct {
	mu      sync.Mutex
	users   []model.User
	store   *st.Store
	handler Handler
	client  lib.Client
}

type Handler func(context.Context) error

type Config struct{ Name string }

func (s *Server) Start(cfg Config) error { return nil }
`,
		// A method in another file of the package still belongs to Server.
		"log.go": `package app

func (s *Server) Log(l Logger) {}

type Logger interface{ Log(string) }
`,
		// A type in a test file is a fixture, and a type nothing refers to
		// couples to nothing.
		"server_test.go": `package app

type fake struct{ s *Server }
`,
		"model/user.go": `package model

type User struct{ ID int }
`,
		// store declares its own Server, which must not be confused with
		// app's, and depends on model.User.
		"store/store.go": `package store

import "example.com/app/model"

type Store struct{ users map[int]model.User }

type Server struct{ s Store }
`,
	})

	report, err := Analyze([]string{dir}, Options{CBO: true, IncludeTests: true}, nil)
	if err != nil {
		t.Fatalf("Analyze: %v", err)
	}
	if report.Coupling == nil {
		t.Fatal("coupling is nil for a Go tree")
	}
	if report.Cohesion != nil || report.Complexity != nil {
		t.Error("only coupling was selected")
	}
	if report.Coupling.FilesAnalyzed != 4 || len(report.Coupling.Warnings) != 0 {
		t.Errorf("files = %d, warnings = %v; want 4 files and no warnings", report.Coupling.FilesAnalyzed, report.Coupling.Warnings)
	}
	relToDir := func(p string) string {
		abs := p
		if !filepath.IsAbs(p) {
			a, err := filepath.Abs(p)
			if err != nil {
				return p
			}
			abs = a
		}
		rel, err := filepath.Rel(dir, abs)
		if err != nil {
			return strings.TrimPrefix(p, dir+"/")
		}
		return rel
	}
	got := map[string][]string{}
	for _, class := range report.Coupling.Classes {
		got[relToDir(class.FilePath)+":"+class.Name] = class.DependentClasses
	}
	want := map[string][]string{
		"server.go:Server":      {"Config", "Handler", "Logger", "model.User", "st.Store"},
		"store/store.go:Store":  {"model.User"},
		"store/store.go:Server": {"Store"},
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("dependencies = %v, want %v", got, want)
	}
	server := report.Coupling.Classes[0]
	if server.Name != "Server" || server.Language != "Go" || server.CBO != 5 || server.RiskLevel != domain.RiskLevelMedium ||
		server.StartLine != 12 || server.EndLine != 18 || server.TypeHint != 5 {
		t.Errorf("Server = %+v, want CBO 5 (medium) at server.go:12-18", server)
	}
}

func TestAnalyzeCouplingGoTypeAlias(t *testing.T) {
	dir := writeFiles(t, map[string]string{
		"go.mod": "module example.com/app\n",
		// Methods on an alias receiver belong to the aliased struct, as
		// do a field and parameter typed by the alias.
		"alias.go": `package app

type Base struct{ n int }
type Alias = Base

type Dep struct{}
type Other struct{}

func (a *Alias) Use(d *Dep) {}
func (a *Alias) Also(o Other) {}
`,
		"holder.go": `package app

type Holder struct{ a *Alias }

func (h *Holder) Take(x Alias) {}
`,
		// A method in another file of the package still belongs to Base.
		"more.go": `package app

func (a *Alias) Extra(d *Dep) {}
`,
		"lib/parser.go": `package lib

type Parser struct{ n int }
type Flags = Parser
`,
		"wrap.go": `package app

import "example.com/app/lib"

type Local = lib.Flags

type Wrap struct{ fs *Local }
`,
	})

	report, err := Analyze([]string{dir}, Options{CBO: true}, nil)
	if err != nil {
		t.Fatalf("Analyze: %v", err)
	}
	got := map[string][]string{}
	for _, class := range report.Coupling.Classes {
		got[class.Name] = class.DependentClasses
	}
	want := map[string][]string{
		"Base":   {"Dep", "Other"},
		"Holder": {"Base"},
		"Wrap":   {"lib.Parser"},
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("dependencies = %v, want %v", got, want)
	}
	for _, class := range report.Coupling.Classes {
		if class.Name == "Alias" || class.Name == "Local" || class.Name == "Flags" {
			t.Errorf("unexpected class %s; aliases are not types of their own", class.Name)
		}
	}
	var base CoupledClass
	for _, class := range report.Coupling.Classes {
		if class.Name == "Base" {
			base = class
			break
		}
	}
	if base.CBO != 2 || base.TypeHint != 2 {
		t.Errorf("Base = %+v, want coupling_count 2", base)
	}
}

func TestAnalyzeCouplingGoWithoutModule(t *testing.T) {
	dir := writeFiles(t, map[string]string{
		"a.go": `package p

import "example.com/x/model"

type A struct{ b B; u model.User }
type B struct{}
`,
	})
	report, err := Analyze([]string{dir}, Options{CBO: true}, nil)
	if err != nil {
		t.Fatalf("Analyze: %v", err)
	}
	classes := report.Coupling.Classes
	if len(classes) != 1 || classes[0].Name != "A" || !reflect.DeepEqual(classes[0].DependentClasses, []string{"B"}) {
		t.Errorf("classes = %+v, want A depending on B only", classes)
	}
	if len(report.Coupling.Warnings) != 1 || !strings.Contains(report.Coupling.Warnings[0], "no go.mod") {
		t.Errorf("warnings = %v, want one about the missing go.mod", report.Coupling.Warnings)
	}
}

func TestAnalyzeCouplingRust(t *testing.T) {
	dir := writeFiles(t, map[string]string{
		"types.rs": `
pub struct Foo { bar: Bar, items: Vec<Baz> }
pub struct Bar;
pub trait Shape { fn area(&self) -> Unit; }
pub struct Unit;
`,
		// The impl blocks of Foo live in another file and add Shape and
		// Baz; a Go type named like a Rust one is a different language's.
		"ops.rs": `
impl Shape for Foo { fn area(&self) -> Unit { Unit } }
impl Foo { pub fn baz(&self) -> Baz { Baz::new() } }
pub struct Baz;
impl Baz { pub fn new() -> Self { Baz } }
`,
		"tests.rs": `
struct Fixture { f: Foo }
`,
		"other.go": `package other

type Unit struct{ foo Foo }
`,
	})
	report, err := Analyze([]string{dir}, Options{CBO: true, IncludeTests: true}, nil)
	if err != nil {
		t.Fatalf("Analyze: %v", err)
	}
	got := map[string][]string{}
	for _, class := range report.Coupling.Classes {
		got[class.Language+":"+class.Name] = class.DependentClasses
	}
	want := map[string][]string{
		"Rust:Foo":   {"Bar", "Baz", "Shape", "Unit"},
		"Rust:Shape": {"Unit"},
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("dependencies = %v, want %v", got, want)
	}
	foo := report.Coupling.Classes[0]
	if foo.Name != "Foo" || filepath.Base(foo.FilePath) != "types.rs" || foo.Inheritance != 1 || foo.TypeHint != 3 || foo.RiskLevel != domain.RiskLevelMedium {
		t.Errorf("Foo = %+v, want CBO 4 with one inheritance at types.rs", foo)
	}
	shape := report.Coupling.Classes[1]
	if shape.Name != "Shape" || !shape.IsAbstract {
		t.Errorf("Shape = %+v, want an abstract class", shape)
	}
}

func TestAnalyzeCouplingRustAmbiguousTypeName(t *testing.T) {
	// Two files declare Point and a third holds an impl for it. The
	// ambiguous name must not silently merge the impl into one declaration
	// or resolve references arbitrarily; both sides warn and leave it out.
	dir := writeFiles(t, map[string]string{
		"unit.rs": `pub struct Unit(pub f64);
`,
		"helper.rs": `pub struct Helper;
`,
		"circle.rs": `pub struct Point { x: f64, y: f64, u: Unit }

impl Point {
    pub fn distance(&self, other: &Point) -> f64 { (self.x - other.x).abs() + self.u.0 }
}
`,
		"vector.rs": `pub struct Point { x: f64, y: f64, u: Unit }

impl Point {
    pub fn add(&self, other: &Point) -> Point { Point { x: self.x + other.x, y: self.y + other.y, u: Unit(self.u.0) } }
}
`,
		// ops.rs holds an impl for Point but declares nothing itself. With
		// two declarations of Point in the tree, the impl block must be
		// left unresolved: its references (Helper, Unit) must reach
		// neither declaration.
		"ops.rs": `use crate::circle::Point;

impl Point {
    pub fn origin(h: &Helper) -> Point { let _ = h; Point { x: 0.0, y: 0.0, u: Unit(0.0) } }
}
`,
	})
	report, err := Analyze([]string{dir}, Options{CBO: true}, nil)
	if err != nil {
		t.Fatalf("Analyze: %v", err)
	}
	// Both declarations appear as separate classes, each depending only on
	// Unit: the references of the orphaned impl block (Helper) reach
	// neither.
	byFile := map[string][]string{}
	for _, class := range report.Coupling.Classes {
		if class.Language == "Rust" && class.Name == "Point" {
			byFile[filepath.Base(class.FilePath)] = class.DependentClasses
		}
	}
	want := map[string][]string{
		"circle.rs": {"Unit"},
		"vector.rs": {"Unit"},
	}
	if !reflect.DeepEqual(byFile, want) {
		t.Errorf("Point dependencies = %v, want %v", byFile, want)
	}
	// The ambiguous impl must produce a warning, not a silent drop.
	found := false
	for _, w := range report.Coupling.Warnings {
		if strings.Contains(w, "ambiguous") && strings.Contains(w, "undeclared impl block") {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("warnings = %v, want one about the ambiguous undeclared impl block", report.Coupling.Warnings)
	}
}

func TestAnalyzeCouplingAbsentWithoutSupportedLanguage(t *testing.T) {
	dir := writeFiles(t, map[string]string{"a.cpp": "struct S { int a; };\n"})
	report, err := Analyze([]string{dir}, Options{CBO: true}, nil)
	if err != nil {
		t.Fatalf("Analyze: %v", err)
	}
	if report.Coupling != nil {
		t.Errorf("coupling = %+v, want none for a C++ tree", report.Coupling)
	}
}
