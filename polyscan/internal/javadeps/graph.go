package javadeps

import (
	"fmt"
	"maps"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"github.com/ludo-technologies/polyscan/polyscan/internal/js/domain"
)

// pkg accumulates the files of one package.
type pkg struct {
	// dirs counts the files of the package per directory.
	dirs map[string]int
	// types are the names of the top-level types the package declares.
	types map[string]bool
	// publicTypes counts the public top-level types and publicAbstract the
	// ones among them that are abstract.
	publicTypes    int
	publicAbstract int
	// imports counts, per package of the tree, the files that import from
	// it.
	imports map[string]int
}

// abstractness is Martin's A for a Java package: the share of its public
// top-level types that are interfaces, annotation types or abstract classes.
// A package with no public type is concrete.
func (p *pkg) abstractness() float64 {
	if p.publicTypes == 0 {
		return 0
	}
	return float64(p.publicAbstract) / float64(p.publicTypes)
}

// dir is the directory that holds most of the files of the package, the
// first in lexical order among equals.
func (p *pkg) dir() string {
	best := ""
	for _, dir := range slices.Sorted(maps.Keys(p.dirs)) {
		if best == "" || p.dirs[dir] > p.dirs[best] {
			best = dir
		}
	}
	return best
}

// sourceFile is a file whose imports are resolved once every package of the
// tree is known.
type sourceFile struct {
	pkg     string
	imports []Import
}

// Build builds the package graph of files, which must be Java sources. A
// file belongs to the package its package declaration names, wherever the
// file lies, so the files of one package in several directories make one
// node, placed in the directory that holds most of them. A file without a package declaration belongs to the unnamed package,
// which no other package can import from, and is left out with one warning
// per directory. module-info.java declares a module, not a package, and is
// left out too. A file that cannot be read is left out and reported in the
// warnings.
//
// Edges join packages of the tree only: an import whose name leads to no
// type the tree declares, such as one of the JDK or of a library, adds none.
// Each pair of packages is joined by one edge whose weight is the number of
// files that import from the target.
func Build(files []string) (*domain.DependencyGraph, []string) {
	var warnings []string
	warn := func(format string, args ...any) {
		warnings = append(warnings, fmt.Sprintf(format, args...))
	}

	packages := map[string]*pkg{}
	var sources []sourceFile
	unnamed := map[string]bool{}
	for _, path := range slices.Sorted(slices.Values(files)) {
		if filepath.Base(path) == "module-info.java" {
			continue
		}
		content, err := os.ReadFile(path)
		if err != nil {
			warn("%s: %v", path, err)
			continue
		}
		file, err := ParseFile(content)
		if err != nil {
			warn("%s: %v; excluded from the dependency graph", path, err)
			continue
		}
		dir := filepath.Dir(path)
		if file.Package == "" {
			if !unnamed[dir] {
				unnamed[dir] = true
				warn("%s: Java files without a package declaration are left out of the dependency graph", dir)
			}
			continue
		}
		p, ok := packages[file.Package]
		if !ok {
			p = &pkg{dirs: map[string]int{}, types: map[string]bool{}, imports: map[string]int{}}
			packages[file.Package] = p
		}
		p.dirs[dir]++
		for _, declared := range file.Types {
			p.types[declared.Name] = true
			if declared.Public {
				p.publicTypes++
				if declared.Abstract {
					p.publicAbstract++
				}
			}
		}
		sources = append(sources, sourceFile{pkg: file.Package, imports: file.Imports})
	}

	for _, source := range sources {
		seen := map[string]bool{}
		for _, imported := range source.imports {
			target, ok := importedPackage(imported, packages)
			if ok && target != source.pkg && !seen[target] {
				seen[target] = true
				packages[source.pkg].imports[target]++
			}
		}
	}

	graph := domain.NewDependencyGraph()
	for name, p := range packages {
		graph.AddNode(&domain.ModuleNode{
			ID:           name,
			Name:         name,
			FilePath:     p.dir(),
			ModuleType:   domain.ModuleTypePackage,
			Abstractness: p.abstractness(),
			Exports:      []string{},
		})
	}
	for _, from := range graph.NodeIDs() {
		imports := packages[from].imports
		for _, to := range slices.Sorted(maps.Keys(imports)) {
			graph.AddEdge(&domain.DependencyEdge{
				From:     from,
				To:       to,
				EdgeType: domain.EdgeTypeImport,
				Weight:   imports[to],
			})
		}
	}
	graph.UpdateNodeFlags()
	return graph, warnings
}

// importedPackage returns the package of the tree an import draws on. An
// import on demand may name that package itself. Otherwise the name leads to
// a type: a package followed by one of its top-level types and, for a member
// type or a static member, the names after it, as com.example.Outer.Inner
// leads to Outer of com.example. A prefix of the name counts only when the
// segment after it is a top-level type the package declares, so an import
// from a library package that shares a prefix with a package of the tree,
// such as com.example.plugin.Plugin beside com.example, adds no edge.
func importedPackage(imported Import, packages map[string]*pkg) (string, bool) {
	if imported.OnDemand && packages[imported.Name] != nil {
		return imported.Name, true
	}
	segments := strings.Split(imported.Name, ".")
	for i := len(segments) - 1; i > 0; i-- {
		name := strings.Join(segments[:i], ".")
		if p := packages[name]; p != nil && p.types[segments[i]] {
			return name, true
		}
	}
	return "", false
}
