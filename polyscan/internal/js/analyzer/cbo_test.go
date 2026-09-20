package analyzer

import (
	"slices"
	"testing"

	"github.com/ludo-technologies/polyscan/polyscan/internal/js/domain"
	"github.com/ludo-technologies/polyscan/polyscan/internal/js/parser"
)

func TestDefaultCBOAnalyzerConfig(t *testing.T) {
	config := DefaultCBOAnalyzerConfig()

	if config.IncludeBuiltins {
		t.Error("Expected IncludeBuiltins to be false by default")
	}
	if !config.IncludeTypeImports {
		t.Error("Expected IncludeTypeImports to be true by default")
	}
	if config.LowThreshold != 7 {
		t.Errorf("Expected LowThreshold to be 7, got %d", config.LowThreshold)
	}
	if config.MediumThreshold != 14 {
		t.Errorf("Expected MediumThreshold to be 14, got %d", config.MediumThreshold)
	}
}

func TestNewCBOAnalyzer(t *testing.T) {
	// Test with nil config
	analyzer := NewCBOAnalyzer(nil)
	if analyzer == nil {
		t.Fatal("Expected non-nil analyzer")
	}

	// Test with custom config
	config := &CBOAnalyzerConfig{
		IncludeBuiltins: true,
		LowThreshold:    5,
	}
	analyzer = NewCBOAnalyzer(config)
	if analyzer == nil {
		t.Fatal("Expected non-nil analyzer")
	}
}

func TestCBOImportDependencies(t *testing.T) {
	source := `
import React from 'react';
import { useState, useEffect } from 'react';
import lodash from 'lodash';
import { helper } from './utils';
`

	p := parser.NewParser()
	defer p.Close()

	ast, err := p.ParseString(source)
	if err != nil {
		t.Fatalf("Failed to parse: %v", err)
	}

	analyzer := NewCBOAnalyzer(DefaultCBOAnalyzerConfig())
	result, err := analyzer.AnalyzeFile(ast, "test.js")
	if err != nil {
		t.Fatalf("Failed to analyze: %v", err)
	}

	// Should have 3 unique import dependencies (react, lodash, utils)
	if result.Metrics.ImportDependencies != 3 {
		t.Errorf("Expected 3 import dependencies, got %d", result.Metrics.ImportDependencies)
	}

	// CouplingCount should equal import dependencies in this case
	if result.Metrics.CouplingCount != 3 {
		t.Errorf("Expected CouplingCount 3, got %d", result.Metrics.CouplingCount)
	}
}

func TestCBOInstantiationDependencies(t *testing.T) {
	source := `
import { UserService } from './user-service';
import { Logger } from './logger';

const userService = new UserService();
const logger = new Logger();
const anotherUser = new UserService();
`

	p := parser.NewParser()
	defer p.Close()

	ast, err := p.ParseString(source)
	if err != nil {
		t.Fatalf("Failed to parse: %v", err)
	}

	analyzer := NewCBOAnalyzer(DefaultCBOAnalyzerConfig())
	result, err := analyzer.AnalyzeFile(ast, "test.js")
	if err != nil {
		t.Fatalf("Failed to analyze: %v", err)
	}

	// Should have 2 unique instantiation dependencies (UserService, Logger)
	if result.Metrics.InstantiationDependencies != 2 {
		t.Errorf("Expected 2 instantiation dependencies, got %d", result.Metrics.InstantiationDependencies)
	}
}

func TestCBOBuiltinClassesNotCounted(t *testing.T) {
	source := `
const arr = new Array();
const date = new Date();
const map = new Map();
const promise = new Promise((resolve) => resolve());
`

	p := parser.NewParser()
	defer p.Close()

	ast, err := p.ParseString(source)
	if err != nil {
		t.Fatalf("Failed to parse: %v", err)
	}

	analyzer := NewCBOAnalyzer(DefaultCBOAnalyzerConfig())
	result, err := analyzer.AnalyzeFile(ast, "test.js")
	if err != nil {
		t.Fatalf("Failed to analyze: %v", err)
	}

	// Builtin classes should not be counted
	if result.Metrics.InstantiationDependencies != 0 {
		t.Errorf("Expected 0 instantiation dependencies for builtins, got %d", result.Metrics.InstantiationDependencies)
	}
}

func TestCBORiskLevelLow(t *testing.T) {
	source := `
import { helper } from './utils';
import React from 'react';
`

	p := parser.NewParser()
	defer p.Close()

	ast, err := p.ParseString(source)
	if err != nil {
		t.Fatalf("Failed to parse: %v", err)
	}

	analyzer := NewCBOAnalyzer(DefaultCBOAnalyzerConfig())
	result, err := analyzer.AnalyzeFile(ast, "test.js")
	if err != nil {
		t.Fatalf("Failed to analyze: %v", err)
	}

	// With only 2 dependencies, risk should be low
	if result.RiskLevel != domain.RiskLevelLow {
		t.Errorf("Expected risk level Low, got %v", result.RiskLevel)
	}
}

func TestCBORiskLevelMedium(t *testing.T) {
	source := `
import a from 'a';
import b from 'b';
import c from 'c';
import d from 'd';
import e from 'e';
import f from 'f';
import g from 'g';
import h from 'h';
import i from 'i';
import j from 'j';
`

	p := parser.NewParser()
	defer p.Close()

	ast, err := p.ParseString(source)
	if err != nil {
		t.Fatalf("Failed to parse: %v", err)
	}

	analyzer := NewCBOAnalyzer(DefaultCBOAnalyzerConfig())
	result, err := analyzer.AnalyzeFile(ast, "test.js")
	if err != nil {
		t.Fatalf("Failed to analyze: %v", err)
	}

	// With 10 dependencies (> 7, <= 14), risk should be medium
	if result.RiskLevel != domain.RiskLevelMedium {
		t.Errorf("Expected risk level Medium, got %v (CBO: %d)", result.RiskLevel, result.Metrics.CouplingCount)
	}
}

func TestCBORiskLevelHigh(t *testing.T) {
	source := `
import a from 'a';
import b from 'b';
import c from 'c';
import d from 'd';
import e from 'e';
import f from 'f';
import g from 'g';
import h from 'h';
import i from 'i';
import j from 'j';
import k from 'k';
import l from 'l';
import m from 'm';
import n from 'n';
import o from 'o';
`

	p := parser.NewParser()
	defer p.Close()

	ast, err := p.ParseString(source)
	if err != nil {
		t.Fatalf("Failed to parse: %v", err)
	}

	analyzer := NewCBOAnalyzer(DefaultCBOAnalyzerConfig())
	result, err := analyzer.AnalyzeFile(ast, "test.js")
	if err != nil {
		t.Fatalf("Failed to analyze: %v", err)
	}

	// With 15 dependencies (> 14), risk should be high
	if result.RiskLevel != domain.RiskLevelHigh {
		t.Errorf("Expected risk level High, got %v (CBO: %d)", result.RiskLevel, result.Metrics.CouplingCount)
	}
}

func TestCBOExcludeBuiltins(t *testing.T) {
	source := `
import fs from 'fs';
import path from 'path';
import React from 'react';
`

	p := parser.NewParser()
	defer p.Close()

	ast, err := p.ParseString(source)
	if err != nil {
		t.Fatalf("Failed to parse: %v", err)
	}

	// With builtins excluded (default)
	analyzer := NewCBOAnalyzer(DefaultCBOAnalyzerConfig())
	result, err := analyzer.AnalyzeFile(ast, "test.js")
	if err != nil {
		t.Fatalf("Failed to analyze: %v", err)
	}

	// Only 'react' should be counted (fs and path are builtins)
	if result.Metrics.ImportDependencies != 1 {
		t.Errorf("Expected 1 import dependency (excluding builtins), got %d", result.Metrics.ImportDependencies)
	}
}

func TestCBOIncludeBuiltins(t *testing.T) {
	source := `
import fs from 'fs';
import path from 'path';
import React from 'react';
`

	p := parser.NewParser()
	defer p.Close()

	ast, err := p.ParseString(source)
	if err != nil {
		t.Fatalf("Failed to parse: %v", err)
	}

	// With builtins included
	config := &CBOAnalyzerConfig{
		IncludeBuiltins:    true,
		IncludeTypeImports: true,
		LowThreshold:       3,
		MediumThreshold:    7,
	}
	analyzer := NewCBOAnalyzer(config)
	result, err := analyzer.AnalyzeFile(ast, "test.js")
	if err != nil {
		t.Fatalf("Failed to analyze: %v", err)
	}

	// All 3 should be counted
	if result.Metrics.ImportDependencies != 3 {
		t.Errorf("Expected 3 import dependencies (including builtins), got %d", result.Metrics.ImportDependencies)
	}
}

func TestCBOEmptyFile(t *testing.T) {
	source := ``

	p := parser.NewParser()
	defer p.Close()

	ast, err := p.ParseString(source)
	if err != nil {
		t.Fatalf("Failed to parse: %v", err)
	}

	analyzer := NewCBOAnalyzer(DefaultCBOAnalyzerConfig())
	result, err := analyzer.AnalyzeFile(ast, "test.js")
	if err != nil {
		t.Fatalf("Failed to analyze: %v", err)
	}

	if result.Metrics.CouplingCount != 0 {
		t.Errorf("Expected CouplingCount 0 for empty file, got %d", result.Metrics.CouplingCount)
	}
	if result.RiskLevel != domain.RiskLevelLow {
		t.Errorf("Expected risk level Low for empty file, got %v", result.RiskLevel)
	}
}

func TestCBONilAST(t *testing.T) {
	analyzer := NewCBOAnalyzer(DefaultCBOAnalyzerConfig())
	result, err := analyzer.AnalyzeFile(nil, "test.js")
	if err != nil {
		t.Fatalf("Failed to analyze: %v", err)
	}

	if result.FilePath != "test.js" {
		t.Errorf("Expected file path 'test.js', got %q", result.FilePath)
	}
	if result.Metrics.CouplingCount != 0 {
		t.Errorf("Expected CouplingCount 0 for nil AST, got %d", result.Metrics.CouplingCount)
	}
}

func TestCBODependentClassesSorted(t *testing.T) {
	source := `
import z from 'z-package';
import a from 'a-package';
import m from 'm-package';
`

	p := parser.NewParser()
	defer p.Close()

	ast, err := p.ParseString(source)
	if err != nil {
		t.Fatalf("Failed to parse: %v", err)
	}

	analyzer := NewCBOAnalyzer(DefaultCBOAnalyzerConfig())
	result, err := analyzer.AnalyzeFile(ast, "test.js")
	if err != nil {
		t.Fatalf("Failed to analyze: %v", err)
	}

	// DependentClasses should be sorted alphabetically
	classes := result.Metrics.DependentClasses
	if len(classes) >= 2 {
		for i := 1; i < len(classes); i++ {
			if classes[i-1] > classes[i] {
				t.Errorf("DependentClasses not sorted: %v", classes)
				break
			}
		}
	}
}

func TestExtractModuleName(t *testing.T) {
	testCases := []struct {
		input    string
		expected string
	}{
		{"/path/to/file.js", "file"},
		{"/path/to/file.ts", "file"},
		{"file.js", "file"},
		{"path/file.tsx", "file"},
		{"index", "index"},
	}

	for _, tc := range testCases {
		t.Run(tc.input, func(t *testing.T) {
			result := extractModuleName(tc.input)
			if result != tc.expected {
				t.Errorf("extractModuleName(%q) = %q, want %q", tc.input, result, tc.expected)
			}
		})
	}
}

func TestNormalizeModuleName(t *testing.T) {
	testCases := []struct {
		input    string
		expected string
	}{
		{"./utils", "utils"},
		{"../lib/helper", "lib"},
		{"react", "react"},
		{"@types/node", "@types/node"},
		{"lodash/debounce", "lodash"},
		{"@company/shared-lib/utils", "@company/shared-lib"},
	}

	for _, tc := range testCases {
		t.Run(tc.input, func(t *testing.T) {
			result := normalizeModuleName(tc.input)
			if result != tc.expected {
				t.Errorf("normalizeModuleName(%q) = %q, want %q", tc.input, result, tc.expected)
			}
		})
	}
}

func TestIsBuiltinClass(t *testing.T) {
	builtins := []string{"Array", "Object", "Map", "Set", "Promise", "Date", "Error"}
	for _, name := range builtins {
		if !isBuiltinClass(name) {
			t.Errorf("Expected %q to be a builtin class", name)
		}
	}

	nonBuiltins := []string{"MyClass", "UserService", "CustomError", "AppState"}
	for _, name := range nonBuiltins {
		if isBuiltinClass(name) {
			t.Errorf("Expected %q to NOT be a builtin class", name)
		}
	}
}

func TestIsPrimitiveType(t *testing.T) {
	primitives := []string{"string", "number", "boolean", "void", "null", "undefined", "any", "unknown"}
	for _, name := range primitives {
		if !isPrimitiveType(name) {
			t.Errorf("Expected %q to be a primitive type", name)
		}
	}

	nonPrimitives := []string{"MyType", "UserInterface", "CustomClass"}
	for _, name := range nonPrimitives {
		if isPrimitiveType(name) {
			t.Errorf("Expected %q to NOT be a primitive type", name)
		}
	}
}

func TestCalculateRiskLevel(t *testing.T) {
	config := DefaultCBOAnalyzerConfig()
	analyzer := NewCBOAnalyzer(config)

	testCases := []struct {
		cbo      int
		expected domain.RiskLevel
	}{
		{0, domain.RiskLevelLow},
		{1, domain.RiskLevelLow},
		{7, domain.RiskLevelLow},
		{8, domain.RiskLevelMedium},
		{10, domain.RiskLevelMedium},
		{14, domain.RiskLevelMedium},
		{15, domain.RiskLevelHigh},
		{20, domain.RiskLevelHigh},
		{100, domain.RiskLevelHigh},
	}

	for _, tc := range testCases {
		result := analyzer.calculateRiskLevel(tc.cbo)
		if result != tc.expected {
			t.Errorf("calculateRiskLevel(%d) = %v, want %v", tc.cbo, result, tc.expected)
		}
	}
}

func TestCBOAttributeAccessDependencies(t *testing.T) {
	source := `
import userService from './user-service';
import { logger as log } from './logger';
import * as services from './services';

function doSomething() {
    userService.getUser();
    userService.updateUser();
    log.log('message');
    services.nested.run();
}
`

	p := parser.NewParser()
	defer p.Close()

	ast, err := p.ParseString(source)
	if err != nil {
		t.Fatalf("Failed to parse: %v", err)
	}

	analyzer := NewCBOAnalyzer(DefaultCBOAnalyzerConfig())
	result, err := analyzer.AnalyzeFile(ast, "test.js")
	if err != nil {
		t.Fatalf("Failed to analyze: %v", err)
	}

	if result.Metrics.AttributeAccessDependencies != 3 {
		t.Errorf("Expected 3 attribute access dependencies, got %d", result.Metrics.AttributeAccessDependencies)
	}
	if result.Metrics.CouplingCount != 3 || !slices.Equal(result.Metrics.DependentClasses, []string{"logger", "services", "user-service"}) {
		t.Errorf("Expected only the three imported modules, got %+v", result.Metrics)
	}
}

// Method receivers name values, not necessarily external classes (issue 150).
func TestCBOLocalMethodReceiversNotCounted(t *testing.T) {
	cases := []struct {
		name   string
		source string
	}{
		{
			name: "module and function variables",
			source: `
const cache = new Map();
let plugin = {};
var legacy = {};
function run(input) {
  const w = new Widget();
  let local = {};
  cache.set('key', w.render());
  plugin.run();
  legacy.run();
  local.run();
  return input.load();
}`,
		},
		{
			name: "destructured and callback bindings",
			source: `
const { nested: { target }, nextReset = {} } = options;
const [first, ...rest] = values;
target.run(); nextReset.run(); first.run(); rest.map(x => x.run());
const run = ({ app }, [item], ...args) => {
  app.run(); item.run(); args.map(arg => arg.run());
};
function defaults(input = {}) { input.load(); }
`,
		},
		{
			name: "nested scopes and chained receivers",
			source: `
function run(input) {
  return () => input.service.load();
}
for (const item of items) { item.run(); }
try { run(); } catch (error) { error.toString(); }
class Local { static run() {} }
Local.run();
`,
		},
	}
	for _, ext := range []string{"js", "ts"} {
		for _, tc := range cases {
			t.Run(ext+"/"+tc.name, func(t *testing.T) {
				p := parser.NewParser()
				defer p.Close()
				filePath := "plugin." + ext
				source := "import { Widget } from './dep';\n" + tc.source
				ast, err := p.ParseFile(filePath, []byte(source))
				if err != nil {
					t.Fatalf("Failed to parse: %v", err)
				}
				result, err := NewCBOAnalyzer(nil).AnalyzeFile(ast, filePath)
				if err != nil {
					t.Fatalf("Failed to analyze: %v", err)
				}
				if result.Metrics.CouplingCount != 1 || !slices.Equal(result.Metrics.DependentClasses, []string{"dep"}) {
					t.Errorf("Expected only dep, got %+v", result.Metrics)
				}
				if result.Metrics.AttributeAccessDependencies != 0 {
					t.Errorf("Local calls added %d attribute dependencies", result.Metrics.AttributeAccessDependencies)
				}
			})
		}
	}
}

func TestCBOLocalMethodReceiversIssue150(t *testing.T) {
	const source = `
import { Widget } from './dep'
const cache = new Map<string, string>()
export function run(input: { load(): string }, label: string): string {
  const w = new Widget()
  cache.set(label, w.render())
  return input.load()
}`
	p := parser.NewTypeScriptParser()
	defer p.Close()
	ast, err := p.ParseFile("main.ts", []byte(source))
	if err != nil {
		t.Fatalf("Failed to parse: %v", err)
	}
	config := DefaultCBOAnalyzerConfig()
	config.LowThreshold = 1
	config.MediumThreshold = 2
	result, err := NewCBOAnalyzer(config).AnalyzeFile(ast, "main.ts")
	if err != nil {
		t.Fatalf("Failed to analyze: %v", err)
	}
	if result.Metrics.CouplingCount != 1 || !slices.Equal(result.Metrics.DependentClasses, []string{"dep"}) {
		t.Errorf("Expected only dep, got %+v", result.Metrics)
	}
	if result.Metrics.InstantiationDependencies != 1 || result.Metrics.AttributeAccessDependencies != 0 {
		t.Errorf("Expected one constructor dependency and no receiver dependencies, got %+v", result.Metrics)
	}
	if result.RiskLevel != domain.RiskLevelLow {
		t.Errorf("Local calls inflated risk to %v", result.RiskLevel)
	}
}

func TestCBOBuiltinObjectsNotCounted(t *testing.T) {
	source := `
function doSomething() {
    console.log('message');
    JSON.parse('{}');
    Math.random();
}
`

	p := parser.NewParser()
	defer p.Close()

	ast, err := p.ParseString(source)
	if err != nil {
		t.Fatalf("Failed to parse: %v", err)
	}

	analyzer := NewCBOAnalyzer(DefaultCBOAnalyzerConfig())
	result, err := analyzer.AnalyzeFile(ast, "test.js")
	if err != nil {
		t.Fatalf("Failed to analyze: %v", err)
	}

	// Builtin objects should not be counted as dependencies
	if result.Metrics.CouplingCount != 0 {
		t.Errorf("Expected 0 coupling count for builtin objects only, got %d", result.Metrics.CouplingCount)
	}
}

func TestCBOCommonJSRequire(t *testing.T) {
	source := `
const fs = require('fs');
const lodash = require('lodash');
const utils = require('./utils');
fs.readFileSync('test.js');
lodash.map([], x => x);
utils.run();
`

	p := parser.NewParser()
	defer p.Close()

	ast, err := p.ParseString(source)
	if err != nil {
		t.Fatalf("Failed to parse: %v", err)
	}

	// Without builtins
	analyzer := NewCBOAnalyzer(DefaultCBOAnalyzerConfig())
	result, err := analyzer.AnalyzeFile(ast, "test.js")
	if err != nil {
		t.Fatalf("Failed to analyze: %v", err)
	}

	// Should have 2 dependencies (lodash and utils, excluding fs builtin)
	if result.Metrics.ImportDependencies != 2 {
		t.Errorf("Expected 2 import dependencies, got %d", result.Metrics.ImportDependencies)
	}
	if result.Metrics.CouplingCount != 2 || !slices.Equal(result.Metrics.DependentClasses, []string{"lodash", "utils"}) {
		t.Errorf("Expected only the two required modules, got %+v", result.Metrics)
	}
}

// TestCBOTypeHintsAreBreakdownOnly: a type annotation is a declaration, not
// a use. Annotated types show in the type_hint_dependencies breakdown but never
// count toward the coupling score, since TypeScript erases them at compile time.
func TestCBOTypeHintsAreBreakdownOnly(t *testing.T) {
	source := `
import { Cart } from './cart';
import { Logger } from './logger';

export function total(cart: Cart, logger?: Logger): number {
  return 0;
}
`
	p := parser.NewTypeScriptParser()
	defer p.Close()

	ast, err := p.ParseFile("test.ts", []byte(source))
	if err != nil {
		t.Fatalf("Failed to parse: %v", err)
	}

	config := DefaultCBOAnalyzerConfig()
	config.IncludeBuiltins = true
	result, err := NewCBOAnalyzer(config).AnalyzeFile(ast, "test.ts")
	if err != nil {
		t.Fatalf("Failed to analyze: %v", err)
	}

	if result.Metrics.TypeHintDependencies != 2 {
		t.Errorf("TypeHintDependencies = %d, want 2", result.Metrics.TypeHintDependencies)
	}
	if result.Metrics.CouplingCount != 2 || result.Metrics.ImportDependencies != 2 {
		t.Errorf("CouplingCount = %d with %d imports, want the 2 imported modules and nothing else",
			result.Metrics.CouplingCount, result.Metrics.ImportDependencies)
	}
	for _, dep := range result.Metrics.DependentClasses {
		if dep == "Cart" || dep == "Logger" {
			t.Errorf("annotation-only name %q counted as a dependency: %v", dep, result.Metrics.DependentClasses)
		}
	}
}

// TestCBOTypeHintsSkipDeclaredNames: property names of inline object types and
// parameter names of function types are declarations inside the annotation,
// not type references, and stay out of the type-hint breakdown.
func TestCBOTypeHintsSkipDeclaredNames(t *testing.T) {
	cases := []struct {
		name   string
		source string
		want   int
	}{
		{
			name: "primitive members only",
			source: `export function run(
  options: { label: string; count: number },
  callback: (value: string) => void
): void {}
`,
			want: 0,
		},
		{
			name: "type references inside members, generics and optional parameters",
			source: `export function run(
  options: { label: string; cb(x: Foo): Bar },
  m: Map<string, Baz>,
  n?: Qux
): void {}
`,
			want: 4,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			p := parser.NewTypeScriptParser()
			defer p.Close()
			ast, err := p.ParseFile("test.ts", []byte(tc.source))
			if err != nil {
				t.Fatalf("Failed to parse: %v", err)
			}
			result, err := NewCBOAnalyzer(DefaultCBOAnalyzerConfig()).AnalyzeFile(ast, "test.ts")
			if err != nil {
				t.Fatalf("Failed to analyze: %v", err)
			}
			if result.Metrics.TypeHintDependencies != tc.want || result.Metrics.CouplingCount != 0 {
				t.Errorf("TypeHintDependencies = %d, CouplingCount = %d, want %d and 0",
					result.Metrics.TypeHintDependencies, result.Metrics.CouplingCount, tc.want)
			}
		})
	}
}

// TestCBOInstantiationOnImportedClassCountsOnce: `new X()` on an imported
// binding must add only the module name to DependentClasses, not both the
// module and the raw constructor identifier, for every import form.
// (issue 151)
func TestCBOInstantiationOnImportedClassCountsOnce(t *testing.T) {
	cases := []struct {
		name       string
		source     string
		wantModule string
	}{
		{
			name: "default import",
			source: `
import Elysia from 'elysia'

export function run() {
  return new Elysia()
}
`,
			wantModule: "elysia",
		},
		{
			name: "named import with alias",
			source: `
import { Widget as W } from './dep'

export function run(): string {
  return new W().render()
}
`,
			wantModule: "dep",
		},
		{
			name: "namespace import",
			source: `
import * as dep from './dep'

export function run(): string {
  return new dep.Widget().render()
}
`,
			wantModule: "dep",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			p := parser.NewTypeScriptParser()
			defer p.Close()

			ast, err := p.ParseFile("test.ts", []byte(tc.source))
			if err != nil {
				t.Fatalf("Failed to parse: %v", err)
			}

			result, err := NewCBOAnalyzer(DefaultCBOAnalyzerConfig()).AnalyzeFile(ast, "test.ts")
			if err != nil {
				t.Fatalf("Failed to analyze: %v", err)
			}

			if result.Metrics.CouplingCount != 1 {
				t.Errorf("CouplingCount = %d, want 1", result.Metrics.CouplingCount)
			}
			if want := []string{tc.wantModule}; !slices.Equal(result.Metrics.DependentClasses, want) {
				t.Errorf("DependentClasses = %v, want %v", result.Metrics.DependentClasses, want)
			}
			// The constructor resolves to the import's module, so the
			// instantiation breakdown holds that module and not the raw name.
			if result.Metrics.InstantiationDependencies != 1 {
				t.Errorf("InstantiationDependencies = %d, want 1", result.Metrics.InstantiationDependencies)
			}
		})
	}
}

// TestCBOInstantiationOnBuiltinImportRespectsIncludeBuiltins: resolving a
// constructor to its module must not smuggle a builtin back in when builtins
// are excluded, but the same code counts the builtin when they are included.
func TestCBOInstantiationOnBuiltinImportRespectsIncludeBuiltins(t *testing.T) {
	const source = `
import { EventEmitter } from 'node:events'

export function run() {
  return new EventEmitter()
}
`

	cases := []struct {
		name            string
		includeBuiltins bool
		want            int
	}{
		{name: "builtins excluded", includeBuiltins: false, want: 0},
		{name: "builtins included", includeBuiltins: true, want: 1},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			p := parser.NewTypeScriptParser()
			defer p.Close()

			ast, err := p.ParseFile("test.ts", []byte(source))
			if err != nil {
				t.Fatalf("Failed to parse: %v", err)
			}

			config := DefaultCBOAnalyzerConfig()
			config.IncludeBuiltins = tc.includeBuiltins
			result, err := NewCBOAnalyzer(config).AnalyzeFile(ast, "test.ts")
			if err != nil {
				t.Fatalf("Failed to analyze: %v", err)
			}

			if result.Metrics.CouplingCount != tc.want {
				t.Errorf("CouplingCount = %d, want %d (DependentClasses: %v)",
					result.Metrics.CouplingCount, tc.want, result.Metrics.DependentClasses)
			}
		})
	}
}
