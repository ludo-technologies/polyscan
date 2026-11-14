package javadeps

import (
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/ludo-technologies/polyscan/polyscan/internal/js/domain"
)

const fixture = "../../testdata/javadeps/src/main/java"

func writeFile(t *testing.T, path, content string) string {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	return path
}

// successors maps every node to the targets of its edges.
func successors(graph *domain.DependencyGraph) map[string][]string {
	edges := map[string][]string{}
	for _, from := range graph.NodeIDs() {
		edges[from] = graph.Successors(from)
	}
	return edges
}

func TestBuildFixture(t *testing.T) {
	files, err := filepath.Glob(filepath.Join(fixture, "com", "example", "*", "*.java"))
	if err != nil {
		t.Fatal(err)
	}
	graph, warnings := Build(files)
	if len(warnings) != 0 {
		t.Errorf("unexpected warnings: %v", warnings)
	}

	// app imports model on demand and service by type; Circle reaches
	// service through a static import and Service reaches model, which
	// closes a cycle javac accepts. java.util adds no edge.
	want := map[string][]string{
		"com.example.app":     {"com.example.model", "com.example.service"},
		"com.example.model":   {"com.example.service"},
		"com.example.service": {"com.example.model"},
	}
	if got := successors(graph); !reflect.DeepEqual(got, want) {
		t.Errorf("edges = %v, want %v", got, want)
	}
	for _, edge := range graph.GetOutgoingEdges("com.example.service") {
		if edge.Weight != 1 || edge.EdgeType != domain.EdgeTypeImport {
			t.Errorf("edge service -> %s: weight %d type %s, want one file importing", edge.To, edge.Weight, edge.EdgeType)
		}
	}

	app := graph.GetNode("com.example.app")
	model := graph.GetNode("com.example.model")
	if app.Name != "com.example.app" || !app.IsEntryPoint || app.IsLeaf || app.ModuleType != domain.ModuleTypePackage {
		t.Errorf("app = %+v, want an entry point package", app)
	}
	if model.Abstractness != 0.5 {
		t.Errorf("model abstractness = %v, want 0.5 (Shape among Shape and Circle)", model.Abstractness)
	}
	if want := filepath.Join(fixture, "com", "example", "model"); model.FilePath != want {
		t.Errorf("model file path = %q, want %q", model.FilePath, want)
	}
}

func TestBuildResolvesImportsToTypesOfTheTree(t *testing.T) {
	dir := t.TempDir()
	files := []string{
		writeFile(t, filepath.Join(dir, "core", "Core.java"), `package com.example;
public abstract class Core { public static class Nested {} }
`),
		writeFile(t, filepath.Join(dir, "core", "Helper.java"), "package com.example;\nclass Helper {}\n"),
		writeFile(t, filepath.Join(dir, "nested", "Nested.java"), `package com.example.nested;
import com.example.Core.Nested;
public interface Shape {}
`),
		writeFile(t, filepath.Join(dir, "statics", "Statics.java"), `package com.example.statics;
import static com.example.Core.*;
class Statics {}
`),
		writeFile(t, filepath.Join(dir, "library", "Library.java"), `package com.example.library;
import com.example.plugin.Plugin;
import com.example.Missing;
import com.example.library.Library;
import java.util.*;
class Library {}
`),
		writeFile(t, filepath.Join(dir, "split", "a", "A.java"), "package com.example.split;\nimport com.example.Core;\nclass A {}\n"),
		writeFile(t, filepath.Join(dir, "split", "b", "B.java"), "package com.example.split;\nimport com.example.*;\nclass B {}\n"),
		writeFile(t, filepath.Join(dir, "split", "b", "C.java"), "package com.example.split;\nclass C {}\n"),
	}
	graph, warnings := Build(files)
	if len(warnings) != 0 {
		t.Errorf("unexpected warnings: %v", warnings)
	}

	// A member type and a static import on demand both lead to Core. An
	// import from com.example.plugin, a library package sharing a prefix
	// with com.example, from a type com.example does not declare, from the
	// importing package itself and from the JDK adds no edge.
	want := map[string][]string{
		"com.example":         {},
		"com.example.library": {},
		"com.example.nested":  {"com.example"},
		"com.example.split":   {"com.example"},
		"com.example.statics": {"com.example"},
	}
	if got := successors(graph); !reflect.DeepEqual(got, want) {
		t.Errorf("edges = %v, want %v", got, want)
	}

	// One package in two directories is one node, placed in the directory
	// that holds most of its files, and its edge counts both importing
	// files.
	split := graph.GetNode("com.example.split")
	if want := filepath.Join(dir, "split", "b"); split.FilePath != want {
		t.Errorf("split file path = %q, want %q", split.FilePath, want)
	}
	if edges := graph.GetOutgoingEdges("com.example.split"); len(edges) != 1 || edges[0].Weight != 2 {
		t.Errorf("split edges = %+v, want one edge of weight 2", edges)
	}

	// Core is public and abstract; Helper is not public.
	if got := graph.GetNode("com.example").Abstractness; got != 1 {
		t.Errorf("com.example abstractness = %v, want 1", got)
	}
	if got := graph.GetNode("com.example.statics").Abstractness; got != 0 {
		t.Errorf("com.example.statics abstractness = %v, want 0 without a public type", got)
	}
}

func TestBuildLeavesOutFilesOfNoPackage(t *testing.T) {
	dir := t.TempDir()
	files := []string{
		writeFile(t, filepath.Join(dir, "scripts", "One.java"), "import com.example.Core;\nclass One {}\n"),
		writeFile(t, filepath.Join(dir, "scripts", "Two.java"), "class Two {}\n"),
		writeFile(t, filepath.Join(dir, "module-info.java"), "module com.example { requires java.sql; }\n"),
		writeFile(t, filepath.Join(dir, "core", "Core.java"), "package com.example;\npublic class Core {}\n"),
		filepath.Join(dir, "missing", "Missing.java"),
	}
	graph, warnings := Build(files)

	if want := []string{"com.example"}; !reflect.DeepEqual(graph.NodeIDs(), want) {
		t.Errorf("nodes = %v, want %v", graph.NodeIDs(), want)
	}
	// The files of the unnamed package warn once for their directory;
	// module-info.java does not warn.
	if len(warnings) != 2 {
		t.Fatalf("warnings = %v, want one for the unreadable file and one for the scripts directory", warnings)
	}
	if !strings.Contains(warnings[0], "Missing.java") {
		t.Errorf("warning[0] = %q, want the unreadable file", warnings[0])
	}
	if !strings.HasPrefix(warnings[1], filepath.Join(dir, "scripts")+":") || !strings.Contains(warnings[1], "without a package declaration") {
		t.Errorf("warning[1] = %q, want the scripts directory", warnings[1])
	}
}
