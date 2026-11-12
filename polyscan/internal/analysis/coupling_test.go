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

	t.Chdir(dir)
	report, err := Analyze([]string{"."}, Options{CBO: true, IncludeTests: true}, nil)
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
	got := map[string][]string{}
	for _, class := range report.Coupling.Classes {
		got[class.FilePath+":"+class.Name] = class.DependentClasses
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

func TestAnalyzeCouplingRustExternalPaths(t *testing.T) {
	dir := writeFiles(t, map[string]string{
		"Cargo.toml": "[package]\nname = \"my-proxy\"\n",
		"src/config.rs": `pub struct ServerConfig { pub port: u16 }
`,
		// Another crate's ServerConfig, written with its path or imported
		// by a use, is not the tree's.
		"src/acceptor.rs": `use std::sync::Arc;

pub struct Acceptor { config: Arc<rustls::ServerConfig> }

impl Acceptor {
    pub fn port(&self) -> usize { Arc::strong_count(&self.config) }
}
`,
		"src/endpoint.rs": `use wtransport::ServerConfig;

pub struct Endpoint;

impl Endpoint {
    pub fn build() { let _ = ServerConfig::builder(); }
}
`,
		"src/listener.rs": `use quinn as q;

pub struct Listener;

impl Listener {
    pub fn new() { let _ = quinn::ServerConfig::with_crypto(); let _ = q::ServerConfig::default(); }
}
`,
		// The tree's ServerConfig is reached through crate, a module of
		// the tree, a use of either, a module alias or the crate's name.
		"src/server.rs": `use crate::config::ServerConfig as Settings;
use crate::config as cfg;

pub struct Qualified { config: crate::config::ServerConfig }
pub struct ByModule { config: config::ServerConfig }
pub struct Imported { config: Settings }
pub struct Aliased { config: cfg::ServerConfig }
`,
		"src/main.rs": `use my_proxy::config::ServerConfig;

struct App { config: ServerConfig }
struct Absolute { config: ::my_proxy::config::ServerConfig }
struct AbsoluteExternal { config: ::rustls::ServerConfig }
`,
		// Another crate of the tree reaches the library under the name its
		// Cargo.toml gives the dependency, and a dependency renamed from a
		// crate outside the tree stays outside.
		"app/Cargo.toml": `[package]
name = "app"

[dependencies]
settings = { package = "my-proxy", path = ".." }
tls = { package = "rustls", version = "0.23" }
`,
		"app/src/main.rs": `struct Server { config: settings::config::ServerConfig }
struct Secure { config: tls::ServerConfig }
`,
		// A name the file declares is that declaration even when a nested
		// scope imports another crate's type under the same name.
		"src/error.rs": `pub struct Error;

pub struct Failure { cause: Error }

mod io {
    use std::io::Error;
}
`,
		// Uses in two scopes import Value from paths to different types,
		// which the file's references cannot tell apart.
		"src/flags.rs": `pub struct Short;

impl Short {
    fn value() { use crate::config::ServerConfig as Value; let _ = Value::default(); }
}

pub struct Long;

impl Long {
    fn value() { use crate::error::Error as Value; let _ = Value::default(); }
}
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
		"Qualified": {"ServerConfig"},
		"ByModule":  {"ServerConfig"},
		"Imported":  {"ServerConfig"},
		"Aliased":   {"ServerConfig"},
		"App":       {"ServerConfig"},
		"Absolute":  {"ServerConfig"},
		"Server":    {"ServerConfig"},
		"Failure":   {"Error"},
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("dependencies = %v\nwant %v", got, want)
	}
	found := false
	for _, w := range report.Coupling.Warnings {
		found = found || strings.Contains(w, `ambiguous reference "Value": imported from 2 different paths`)
	}
	if !found {
		t.Errorf("warnings = %v, want one about the ambiguous import of Value", report.Coupling.Warnings)
	}
}

func TestAnalyzeCouplingRustMacroArguments(t *testing.T) {
	dir := writeFiles(t, map[string]string{
		"Cargo.toml": "[package]\nname = \"my-proxy\"\n",
		"src/lib.rs": `pub enum State { Up, Down }

pub struct Update { pub port: u16 }

pub struct Direct;

impl Direct {
    pub fn is_up(&self, s: &str) -> bool {
        let state = if s == "up" { State::Up } else { State::Down };
        let _batch = Update { port: 1 };
        matches!(state, State::Up)
    }
}

pub struct InMacro;

impl InMacro {
    pub fn is_up(&self, s: &str) -> bool {
        let _batch = vec![Update { port: 1 }];
        matches!(s.len(), 2) && matches!(Some(State::Up), Some(State::Up))
    }
}

pub struct Qualified;

impl Qualified {
    pub fn new() { let _ = vec![crate::config::ServerConfig { port: 1 }]; }
}

// A type of another crate named inside a macro is not the tree's type of
// the same name, and a name that is only an argument is not a type.
pub struct Elsewhere;

impl Elsewhere {
    pub fn check(k: Kind) -> bool {
        let _ = vec![rustls::ServerConfig { port: 1 }];
        let _ = format!("{}", ErrorKind);
        matches!(k, std::io::ErrorKind::NotFound)
    }
}
`,
		"src/config.rs": `pub struct ServerConfig { pub port: u16 }

pub struct ErrorKind;
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
		"Direct":    {"State", "Update"},
		"InMacro":   {"State", "Update"},
		"Qualified": {"ServerConfig"},
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("dependencies = %v\nwant %v", got, want)
	}
}

func TestAnalyzeCouplingJava(t *testing.T) {
	dir := writeFiles(t, map[string]string{
		// A type of the same package, a member type, a type imported on
		// demand and a static call count; java.util.List does not. A
		// member type is named after the type that declares it.
		"src/com/example/model/Order.java": `package com.example.model;

import java.util.List;
import com.example.util.*;

public class Order extends Entity implements Priced {
    private List<Line> lines;
    private Status status = Status.OPEN;
    private Money total;

    public Order() { Validator.check(this); }

    public Money price() { return total; }

    public enum Status { OPEN, CLOSED }

    public static class Builder {
        private final java.util.function.Supplier<Order> create = Order::new;
        Order build() { return create.get(); }
    }
}
`,
		"src/com/example/model/Entity.java": `package com.example.model;

public abstract class Entity { protected long id; }
`,
		"src/com/example/model/Priced.java": `package com.example.model;

public interface Priced { com.example.util.Money price(); }
`,
		// A type written with its package needs no import.
		"src/com/example/model/Line.java": `package com.example.model;

public class Line { private com.example.catalog.Product product; }
`,
		// A single import hides the type of the same name in the package.
		"src/com/example/model/Schedule.java": `package com.example.model;

import java.time.Duration;

public class Schedule { private Duration every; }
`,
		"src/com/example/model/Duration.java": `package com.example.model;

public class Duration {}
`,
		"src/com/example/util/Money.java": `package com.example.util;

public record Money(long cents) {}
`,
		"src/com/example/util/Validator.java": `package com.example.util;

public final class Validator { public static void check(Object o) {} }
`,
		// Order.Builder names the member type, not Order, and so does a
		// single import of a member type.
		"src/com/example/catalog/Product.java": `package com.example.catalog;

import com.example.model.Order;
import com.example.model.Order.Status;

public class Product {
    private Order.Builder pending;
    private Status shown;
}
`,
		// A class declared in a method is not the type of the tree it is
		// named like, and a static import of a method names no type.
		"src/com/example/app/Main.java": `package com.example.app;

import static com.example.util.Validator.check;
import com.example.model.*;

public class Main {
    public static void main(String[] args) {
        check(new Line());
        class Money { Duration due() { return null; } }
    }
}
`,
		// The types of the unnamed package are found by their names.
		"Tool.java": `public class Tool { private Helper helper; }
`,
		"Helper.java": `class Helper {}
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
		"Order":         {"Entity", "Line", "Money", "Order.Status", "Priced", "Validator"},
		"Order.Builder": {"Order"},
		"Priced":        {"Money"},
		"Line":          {"Product"},
		"Product":       {"Order.Builder", "Order.Status"},
		"Main":          {"Line"},
		"Tool":          {"Helper"},
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("dependencies = %v\nwant %v", got, want)
	}
	if report.Coupling.Warnings != nil {
		t.Errorf("warnings = %v, want none", report.Coupling.Warnings)
	}
	order := report.Coupling.Classes[0]
	if order.Name != "Order" || order.Inheritance != 2 || order.TypeHint != 4 || order.RiskLevel != domain.RiskLevelMedium {
		t.Errorf("Order = %+v, want CBO 6 with two inheritances", order)
	}
	for _, class := range report.Coupling.Classes {
		if class.IsAbstract != (class.Name == "Priced") {
			t.Errorf("%s: abstract = %v", class.Name, class.IsAbstract)
		}
	}
}

func TestAnalyzeCouplingJavaAmbiguousNames(t *testing.T) {
	dir := writeFiles(t, map[string]string{
		// Two source roots declare com.example.model.Entity. A file of
		// either root refers to its own copy, and a file of a third root
		// cannot tell them apart.
		"one/com/example/model/Entity.java": "package com.example.model;\n\npublic class Entity {}\n",
		"two/com/example/model/Entity.java": "package com.example.model;\n\npublic class Entity {}\n",
		"one/com/example/model/User.java":   "package com.example.model;\n\npublic class User extends Entity { Graph g; }\n",
		"two/com/example/model/Admin.java":  "package com.example.model;\n\npublic class Admin extends Entity {}\n",
		"three/com/example/app/App.java": `package com.example.app;

import com.example.model.Entity;
import com.example.model.User;

public class App { Entity current; User user; }
`,
		"one/com/example/model/Graph.java": `package com.example.model;

import com.example.a.*;
import com.example.b.*;

public class Graph { Node root; Edge edge; }
`,
		// Two imports on demand that bring in a Node each.
		"one/com/example/a/Node.java": "package com.example.a;\n\npublic class Node {}\n",
		"one/com/example/b/Node.java": "package com.example.b;\n\npublic class Node {}\n",
		"one/com/example/b/Edge.java": "package com.example.b;\n\npublic class Edge {}\n",
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
		"User":  {"Entity", "Graph"},
		"Admin": {"Entity"},
		"Graph": {"Edge"},
		"App":   {"User"},
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("dependencies = %v\nwant %v", got, want)
	}
	wantWarnings := []string{
		filepath.Join(dir, "one/com/example/model/Graph.java") + `: ambiguous reference "Node": imported as com.example.a.Node and com.example.b.Node, left unresolved`,
		filepath.Join(dir, "three") + ": references to 1 type that the tree declares more than once, and not once under this source root, are left unresolved, such as com.example.model.Entity",
	}
	if !reflect.DeepEqual(report.Coupling.Warnings, wantWarnings) {
		t.Errorf("warnings = %q\nwant %q", report.Coupling.Warnings, wantWarnings)
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
