package analysis

import (
	"fmt"
	"maps"
	"path/filepath"
	"slices"
	"sort"
	"strings"

	"github.com/ludo-technologies/polyscan/core/cbo"
	"github.com/ludo-technologies/polyscan/core/domain"
	"github.com/ludo-technologies/polyscan/polyscan/internal/engine"
	"github.com/ludo-technologies/polyscan/polyscan/internal/godeps"
	"github.com/ludo-technologies/polyscan/polyscan/internal/javadeps"
	"github.com/ludo-technologies/polyscan/polyscan/internal/lang/golang"
	"github.com/ludo-technologies/polyscan/polyscan/internal/lang/java"
	"github.com/ludo-technologies/polyscan/polyscan/internal/lang/rust"
	"github.com/ludo-technologies/polyscan/polyscan/internal/rustdeps"
)

// CoupledClass is the coupling result for one type.
type CoupledClass struct {
	Name     string `json:"name"`
	FilePath string `json:"file_path"`
	Language string `json:"language"`
	// StartLine and EndLine span the type's declaration.
	StartLine int `json:"start_line"`
	EndLine   int `json:"end_line"`
	// CBO is the number of distinct types of the analyzed tree that the
	// type's declaration and methods refer to. Types outside the tree, such
	// as the standard library's, do not count.
	CBO              int      `json:"cbo"`
	DependentClasses []string `json:"dependent_classes"`
	// Inheritance counts the references that embed or implement another
	// type; TypeHint every other reference.
	Inheritance int              `json:"inheritance"`
	TypeHint    int              `json:"type_hint"`
	IsAbstract  bool             `json:"is_abstract"`
	RiskLevel   domain.RiskLevel `json:"risk_level"`
}

// Coupling is the CBO analysis of a set of paths.
type Coupling struct {
	// Classes lists every type coupled to at least one other, sorted by
	// descending CBO, then by location.
	Classes []CoupledClass `json:"classes"`
	// TotalClasses includes uncoupled types omitted from Classes.
	TotalClasses  int      `json:"total_classes"`
	FilesAnalyzed int      `json:"files_analyzed"`
	Warnings      []string `json:"warnings,omitempty"`
}

// couplingFile is what the coupling analysis keeps of one file.
type couplingFile struct {
	language *engine.Language
	display  string
	// location identifies the unit a type belongs to: the directory when
	// the language spreads a type over one, otherwise the file.
	location string
	// importPath, pkg and imports are set for Go files inside a module.
	importPath string
	pkg        string
	imports    []godeps.Import
	// uses maps each name a Rust file imports to the paths it comes from,
	// and crate is the crate the file belongs to, nil outside of one.
	uses  map[string][]string
	crate *rustdeps.Crate
	// java is the package and the imports of a Java file, and root its
	// source root: the directory its package path starts in, such as src
	// for src/com/example/A.java of package com.example, or the file's own
	// directory when the path does not end in the package.
	java  *javadeps.File
	root  string
	types []engine.Type
}

// coupledType accumulates one type of the tree before it is measured.
type coupledType struct {
	key      string
	name     string
	language *engine.Language
	file     string
	start    int
	end      int
	declared bool
	abstract bool
	// root is the source root of the file that declares a Java type.
	root string
	refs []siteReference
}

// siteReference is a reference together with the file and the type that
// make it, which decide how the name resolves.
type siteReference struct {
	file  *couplingFile
	scope string
	ref   engine.Reference
}

// aliasTarget is the type a Go type alias names. importPath is empty when
// the target is a name of the same package; qualifier is the package
// identifier as written on a qualified target, used when displaying it.
type aliasTarget struct {
	name       string
	importPath string
	qualifier  string
}

// couplingBuilder collects the types of every file and resolves their
// references once the whole tree is known.
type couplingBuilder struct {
	files    []*couplingFile
	resolver *godeps.Resolver
	noModule map[string]bool
	crates   *rustdeps.Resolver
	// rustCrates and rustPackages hold the library and package names of
	// the Rust crates of the tree, and rustModules their modules: the
	// directories and file stems on the path of a Rust file and the inline
	// modules that enclose a type.
	rustCrates   map[string]bool
	rustPackages map[string]bool
	rustModules  map[string]bool
	// javaDuplicates maps a Java source root to the canonical names its
	// files refer to that the tree declares more than once and not exactly
	// once under that root. They are reported once per root.
	javaDuplicates map[string]map[string]bool
	warnings       []string
}

func newCouplingBuilder() *couplingBuilder {
	return &couplingBuilder{
		resolver:       godeps.NewResolver(),
		noModule:       map[string]bool{},
		crates:         rustdeps.NewResolver(),
		rustCrates:     map[string]bool{},
		rustPackages:   map[string]bool{},
		rustModules:    map[string]bool{},
		javaDuplicates: map[string]map[string]bool{},
	}
}

// add records the types of one file. A Go file also contributes its package
// clause and imports, through which qualified references resolve; a Go file
// with no go.mod above it keeps its unqualified references only, with one
// warning per directory. A Rust file contributes its use declarations, its
// crate and its modules, which tell its references to types of other crates
// apart. A Java file contributes its package and its imports.
func (b *couplingBuilder) add(language *engine.Language, display, path string, content []byte, result *engine.Result) error {
	file := &couplingFile{language: language, display: display, location: display}
	if language.TypeSpansDirectory {
		file.location = filepath.Dir(display)
	}
	for _, t := range result.Types {
		if !t.IsTest {
			file.types = append(file.types, t)
		}
	}
	absolute, err := filepath.Abs(path)
	if err != nil {
		return err
	}
	dir := filepath.Dir(absolute)
	switch language {
	case golang.Language:
		info, err := godeps.ParseFile(content)
		if err != nil {
			b.warnings = append(b.warnings, fmt.Sprintf("%s: %v; its types are absent from the coupling analysis", display, err))
			return nil
		}
		importPath, noModule, err := b.resolver.ImportPath(dir)
		if err != nil {
			return err
		}
		if noModule && !b.noModule[dir] {
			b.noModule[dir] = true
			b.warnings = append(b.warnings, fmt.Sprintf("%s: no go.mod above it; references to types of other packages are not resolved", file.location))
		}
		file.importPath, file.pkg, file.imports = importPath, info.Package, info.Imports
	case rust.Language:
		uses, err := rustdeps.ParseUses(content)
		if err != nil {
			return err
		}
		file.uses = uses
		if file.crate, err = b.crates.Crate(dir); err != nil {
			return err
		}
		if file.crate != nil {
			b.rustCrates[file.crate.Name] = true
			b.rustPackages[file.crate.Package] = true
		}
		modules := strings.Split(filepath.ToSlash(strings.TrimSuffix(display, ".rs")), "/")
		for _, t := range file.types {
			scopes := strings.Split(t.Name, language.Separator())
			modules = append(modules, scopes[:len(scopes)-1]...)
		}
		// An absolute or relative display path also yields the empty
		// segment and dots, which no Rust path into the tree starts with.
		for _, module := range modules {
			if module != "" && module != "." && module != ".." {
				b.rustModules[module] = true
			}
		}
	case java.Language:
		if file.java, err = javadeps.ParseFile(content); err != nil {
			return err
		}
		file.root = filepath.Dir(display)
		if file.java.Package != "" {
			packageDir := filepath.Join(strings.Split(file.java.Package, ".")...)
			if file.root == packageDir {
				file.root = "."
			} else {
				file.root = strings.TrimSuffix(file.root, string(filepath.Separator)+packageDir)
			}
		}
	}
	b.files = append(b.files, file)
	return nil
}

// build resolves every reference and measures the types. A reference counts
// when it names a type the tree declares: for Go an unqualified name is a
// type of the same package and a qualified one is looked up through the
// file's imports; for Rust a name is a type of the same file, or else of
// any file, unless the path it is written with or imported from by a use
// leads out of the tree, to a type of another crate; for Java a name is
// looked up as javac looks it up, through the enclosing types, the imports
// and the package of its file. A type whose methods a file holds without
// declaring it, as a Go type is across its package and a Rust type across
// impl blocks, joins its declaration; a Rust impl for a type no file
// declares is left out, and so is a Java class declared in a method.
func (b *couplingBuilder) build() *Coupling {
	types := map[string]*coupledType{}
	var order []string
	typeOf := func(file *couplingFile, name string) *coupledType {
		key := file.language.Name + "\x00" + file.location + "\x00" + name
		t, ok := types[key]
		if !ok {
			t = &coupledType{key: key, name: name, language: file.language}
			types[key] = t
			order = append(order, key)
		}
		return t
	}
	// byImportPath maps a Go package's import path and type name to the
	// declared type; byBare maps a language and bare type name to the
	// declared types of the whole tree, byFileBare to those of one file;
	// byCanonical maps the canonical name of a Java type, its package and
	// its name, to its declarations.
	byImportPath := map[string]*coupledType{}
	packageNames := map[string]string{}
	byBare := map[string][]*coupledType{}
	byFileBare := map[*couplingFile]map[string]*coupledType{}
	byCanonical := map[string][]*coupledType{}
	bareKey := func(language *engine.Language, name string) string {
		return language.Name + "\x00" + bareName(language, name)
	}

	for _, file := range b.files {
		if file.importPath != "" {
			packageNames[file.importPath] = file.pkg
		}
		byFileBare[file] = map[string]*coupledType{}
		for _, et := range file.types {
			t := typeOf(file, et.Name)
			if et.Declared && !t.declared {
				t.declared = true
				t.file, t.start, t.end = file.display, et.StartLine, et.EndLine
				t.abstract = et.Abstract
				if file.importPath != "" {
					byImportPath[file.importPath+"."+et.Name] = t
				}
				bare := bareKey(file.language, et.Name)
				byBare[bare] = append(byBare[bare], t)
				byFileBare[file][bareName(file.language, et.Name)] = t
				if file.java != nil {
					t.root = file.root
					canonical := qualifyJava(file.java.Package, et.Name)
					byCanonical[canonical] = append(byCanonical[canonical], t)
				}
			}
			for _, ref := range et.References {
				t.refs = append(t.refs, siteReference{file: file, scope: et.Name, ref: ref})
			}
		}
	}

	// A Go type alias is not a type of its own: methods on an alias
	// receiver and references typed by the alias belong to the aliased
	// named type. packageNames is complete here, so a qualified target
	// can be resolved through the alias file's imports.
	aliases := map[string]aliasTarget{}
	aliasByImport := map[string]aliasTarget{}
	for _, file := range b.files {
		for _, et := range file.types {
			if !et.Alias || et.AliasOf.Name == "" {
				continue
			}
			target := aliasTarget{name: et.AliasOf.Name, qualifier: et.AliasOf.Package}
			if et.AliasOf.Package != "" {
				path, ok := importPathOf(file, et.AliasOf.Package, packageNames)
				if !ok {
					continue
				}
				target.importPath = path
			}
			key := file.language.Name + "\x00" + file.location + "\x00" + et.Name
			aliases[key] = target
			if file.importPath != "" {
				aliasByImport[file.importPath+"."+et.Name] = target
			}
		}
	}
	b.mergeAliasMethods(types, aliases)

	// A Rust impl block in a file that does not declare its type belongs
	// to the declaration elsewhere when there is exactly one. An ambiguous
	// bare name with more than one declaration across the tree is left
	// unresolved with a warning.
	for _, key := range order {
		t := types[key]
		if t.declared || t.language != rust.Language {
			continue
		}
		candidates := byBare[bareKey(t.language, t.name)]
		switch {
		case len(candidates) == 1:
			candidates[0].refs = append(candidates[0].refs, t.refs...)
		case len(candidates) > 1:
			// t.file is only set for declared types; the file of an
			// undeclared block is the location in its key, which is the
			// display path for a language scoped per file.
			loc := strings.SplitN(key, "\x00", 3)[1]
			b.warnings = append(b.warnings, fmt.Sprintf("%s: ambiguous type name %q: undeclared impl block matches %d declarations across the tree, references left unresolved", loc, t.name, len(candidates)))
		}
	}

	resolve := func(site siteReference) (*coupledType, string) {
		file, ref := site.file, site.ref
		switch {
		case file.language.TypeSpansDirectory && ref.Package != "":
			target, name, ok := b.resolveQualified(file, ref, byImportPath, packageNames, aliasByImport)
			if !ok {
				return nil, ""
			}
			return target, ref.Package + "." + name
		case file.language.TypeSpansDirectory:
			if dest, display, ok := b.resolveAlias(file, ref.Name, types, aliases, aliasByImport, byImportPath); ok {
				return dest, display
			}
			target, ok := types[file.language.Name+"\x00"+file.location+"\x00"+ref.Name]
			if !ok || !target.declared {
				return nil, ""
			}
			return target, ref.Name
		case file.language == java.Language:
			return b.resolveJava(site, byCanonical)
		default:
			// A name the file declares is that declaration, whatever a use
			// in a nested scope imports under the same name.
			name := ref.Name
			if _, local := byFileBare[file][name]; !local || ref.Package != "" {
				var ok bool
				if name, ok = b.rustName(file, ref); !ok {
					return nil, ""
				}
			}
			if target, ok := byFileBare[file][name]; ok {
				return target, name
			}
			candidates := byBare[bareKey(file.language, name)]
			switch {
			case len(candidates) == 1:
				return candidates[0], name
			case len(candidates) > 1:
				b.warnings = append(b.warnings, fmt.Sprintf("%s: ambiguous reference %q: resolves to %d declarations across the tree, left unresolved", file.display, name, len(candidates)))
				return nil, ""
			}
			return nil, ""
		}
	}

	// ComputeCBO returns one result per class, in order.
	var classes []*cbo.ClassInfo
	var owners []*coupledType
	for _, key := range order {
		t := types[key]
		if !t.declared {
			continue
		}
		class := &cbo.ClassInfo{
			Name:       t.name,
			FilePath:   t.file,
			StartLine:  t.start,
			EndLine:    t.end,
			IsAbstract: t.abstract,
		}
		// The breakdown counts dependencies, so a type referred to twice
		// in the same way is one dependency.
		seen := map[cbo.ClassDependency]bool{}
		for _, site := range t.refs {
			target, display := resolve(site)
			if target == nil || target == t {
				continue
			}
			dep := cbo.ClassDependency{ClassName: display, Kind: cbo.DepTypeHint}
			if site.ref.Embedded {
				dep.Kind = cbo.DepInheritance
			}
			if !seen[dep] {
				seen[dep] = true
				class.Dependencies = append(class.Dependencies, dep)
			}
		}
		classes = append(classes, class)
		owners = append(owners, t)
	}

	for _, root := range slices.Sorted(maps.Keys(b.javaDuplicates)) {
		names := slices.Sorted(maps.Keys(b.javaDuplicates[root]))
		types := fmt.Sprintf("%d types", len(names))
		if len(names) == 1 {
			types = "1 type"
		}
		b.warnings = append(b.warnings, fmt.Sprintf("%s: references to %s that the tree declares more than once, and not once under this source root, are left unresolved, such as %s", root, types, names[0]))
	}

	coupling := &Coupling{Classes: []CoupledClass{}, TotalClasses: len(classes), FilesAnalyzed: len(b.files), Warnings: b.warnings}
	for i, result := range cbo.ComputeCBO(classes, cbo.DefaultConfig()) {
		if result.CouplingCount == 0 {
			continue
		}
		coupling.Classes = append(coupling.Classes, CoupledClass{
			Name:             result.ClassName,
			FilePath:         result.FilePath,
			Language:         owners[i].language.Name,
			StartLine:        result.StartLine,
			EndLine:          result.EndLine,
			CBO:              result.CouplingCount,
			DependentClasses: result.DependentClasses,
			Inheritance:      result.DependencyBreakdown[cbo.DepInheritance],
			TypeHint:         result.DependencyBreakdown[cbo.DepTypeHint],
			IsAbstract:       result.IsAbstract,
			RiskLevel:        result.RiskLevel,
		})
	}
	sort.SliceStable(coupling.Classes, func(i, j int) bool {
		a, b := coupling.Classes[i], coupling.Classes[j]
		if a.CBO != b.CBO {
			return a.CBO > b.CBO
		}
		if a.FilePath != b.FilePath {
			return a.FilePath < b.FilePath
		}
		return a.StartLine < b.StartLine
	})
	return coupling
}

// rustName returns the name a Rust reference has in the tree, which is the
// last segment of the path a use imports it from, or false when the
// reference names a type of another crate. A name without a path or a use
// is a type of the same module or of a glob import and keeps its name. The
// uses of every scope of a file count, so a name that scopes import from
// paths to different types, in the tree or not, is ambiguous and left
// unresolved with a warning.
func (b *couplingBuilder) rustName(file *couplingFile, ref engine.Reference) (string, bool) {
	if ref.Package != "" {
		return ref.Name, b.intoTree(file, ref.Package)
	}
	paths, ok := file.uses[ref.Name]
	if !ok {
		return ref.Name, true
	}
	// An empty name stands for a type of another crate.
	var names []string
	for _, path := range paths {
		name := ""
		if b.intoTree(file, path) {
			name = bareName(file.language, path)
		}
		if !slices.Contains(names, name) {
			names = append(names, name)
		}
	}
	if len(names) > 1 {
		b.warnings = append(b.warnings, fmt.Sprintf("%s: ambiguous reference %q: imported from %d different paths, left unresolved", file.display, ref.Name, len(names)))
		return "", false
	}
	return names[0], names[0] != ""
}

// intoTree reports whether a Rust path leads into the tree, either as
// written or through a use of the file that imports its first segment. Any
// other path, such as rustls::ServerConfig, std::sync::Arc, ::serde::Value
// or the T::Output of a generic parameter, leads elsewhere.
func (b *couplingBuilder) intoTree(file *couplingFile, path string) bool {
	if b.startsInTree(file, path) {
		return true
	}
	first, _, _ := strings.Cut(path, file.language.Separator())
	for _, imported := range file.uses[first] {
		if b.startsInTree(file, imported) {
			return true
		}
	}
	return false
}

// startsInTree reports whether the first segment of a Rust path is crate,
// self, super, a module of the tree or a crate of the tree. A path that
// starts with :: names a crate, as in ::my_crate::Config.
func (b *couplingBuilder) startsInTree(file *couplingFile, path string) bool {
	separator := file.language.Separator()
	if rest, ok := strings.CutPrefix(path, separator); ok {
		first, _, _ := strings.Cut(rest, separator)
		return b.treeCrate(file, first)
	}
	first, _, _ := strings.Cut(path, separator)
	switch first {
	case "crate", "self", "super":
		return true
	}
	return b.rustModules[first] || b.treeCrate(file, first)
}

// treeCrate reports whether a Rust file uses name for a crate of the tree:
// the library name of one, or a dependency of the file's crate whose
// package is one, under its own name or renamed with package = "...".
func (b *couplingBuilder) treeCrate(file *couplingFile, name string) bool {
	if b.rustCrates[name] {
		return true
	}
	if file.crate == nil {
		return false
	}
	pkg, ok := file.crate.Dependencies[name]
	return ok && b.rustPackages[pkg]
}

// resolveJava finds the type of the tree a Java reference names. A simple
// name is the type javaType finds. A qualified name a.b.C is a member type
// of the type its first segment names, or else the type C of the package
// a.b, as in java.util.List. A name that leads to a type of the JDK or of a
// library resolves to nothing, as does a name of a variable or a package.
// A canonical name the tree declares more than once, as two copies of a
// source tree do, is the declaration under the source root of the file
// that refers to it, which javac compiles together with that file; a
// reference from another root is left unresolved, with one warning per
// root.
func (b *couplingBuilder) resolveJava(site siteReference, byCanonical map[string][]*coupledType) (*coupledType, string) {
	file, ref := site.file, site.ref
	first, rest := ref.Name, ""
	if ref.Package != "" {
		first, rest, _ = strings.Cut(strings.Join(strings.Fields(ref.Package), "")+"."+ref.Name, ".")
	}
	canonical, ok := b.javaType(file, site.scope, first, byCanonical)
	if !ok && rest != "" {
		canonical, ok = first, true
	}
	if !ok || canonical == "" {
		return nil, ""
	}
	if rest != "" {
		canonical += "." + rest
	}
	candidates := byCanonical[canonical]
	if len(candidates) > 1 {
		var local []*coupledType
		for _, candidate := range candidates {
			if candidate.root == file.root {
				local = append(local, candidate)
			}
		}
		if len(local) != 1 {
			if b.javaDuplicates[file.root] == nil {
				b.javaDuplicates[file.root] = map[string]bool{}
			}
			b.javaDuplicates[file.root][canonical] = true
			return nil, ""
		}
		candidates = local
	}
	if len(candidates) == 0 {
		return nil, ""
	}
	return candidates[0], candidates[0].name
}

// javaType returns the canonical name of the type a simple Java name
// denotes where scope, the type making the reference, uses it. The name is
// looked up in the order javac follows: a member type of scope or of a
// type that encloses it, a single import, a type of the same package and an
// import on demand. A single import names the type even when it lies
// outside the tree. ok is false when no type of that name is in scope, so
// the name may be a package or a variable; the name is empty when imports
// on demand bring in more than one type of the tree under the name, which
// is ambiguous and warned about.
func (b *couplingBuilder) javaType(file *couplingFile, scope, name string, byCanonical map[string][]*coupledType) (string, bool) {
	pkg := file.java.Package
	for enclosing := scope; enclosing != ""; {
		if canonical := qualifyJava(pkg, enclosing+"."+name); len(byCanonical[canonical]) > 0 {
			return canonical, true
		}
		i := strings.LastIndex(enclosing, ".")
		enclosing = enclosing[:max(i, 0)]
	}
	var imported, onDemand []string
	for _, imp := range file.java.Imports {
		switch {
		case !imp.OnDemand && bareName(file.language, imp.Name) == name && !slices.Contains(imported, imp.Name):
			imported = append(imported, imp.Name)
		case imp.OnDemand && len(byCanonical[imp.Name+"."+name]) > 0 && !slices.Contains(onDemand, imp.Name+"."+name):
			onDemand = append(onDemand, imp.Name+"."+name)
		}
	}
	if len(imported) > 0 {
		return b.oneJavaType(file, name, imported), true
	}
	if canonical := qualifyJava(pkg, name); len(byCanonical[canonical]) > 0 {
		return canonical, true
	}
	if len(onDemand) > 0 {
		return b.oneJavaType(file, name, onDemand), true
	}
	return "", false
}

// oneJavaType returns the only canonical name a Java name can denote, or
// warns that its imports make it ambiguous and returns the empty name.
func (b *couplingBuilder) oneJavaType(file *couplingFile, name string, candidates []string) string {
	if len(candidates) == 1 {
		return candidates[0]
	}
	b.warnings = append(b.warnings, fmt.Sprintf("%s: ambiguous reference %q: imported as %s, left unresolved", file.display, name, strings.Join(candidates, " and ")))
	return ""
}

// qualifyJava returns the canonical name of a type of a Java package, which
// is the type's own name in the unnamed package.
func qualifyJava(pkg, name string) string {
	if pkg == "" {
		return name
	}
	return pkg + "." + name
}

// resolveQualified finds the type a Go reference pkg.T names: the file's
// import whose name is pkg, explicit or the package's own, must lead to a
// package of the tree that declares T, following type aliases in that
// package. The returned name is the declared type, which may differ from
// ref.Name when T is an alias.
func (b *couplingBuilder) resolveQualified(file *couplingFile, ref engine.Reference, byImportPath map[string]*coupledType, packageNames map[string]string, aliasByImport map[string]aliasTarget) (*coupledType, string, bool) {
	path, ok := importPathOf(file, ref.Package, packageNames)
	if !ok {
		return nil, "", false
	}
	target, name, ok := lookupInPackage(path, ref.Name, byImportPath, aliasByImport)
	return target, name, ok
}

// importPathOf returns the import path the file binds to pkg, the name a
// qualified type is written with.
func importPathOf(file *couplingFile, pkg string, packageNames map[string]string) (string, bool) {
	for _, imported := range file.imports {
		name := imported.Name
		switch name {
		case "_", ".":
			continue
		case "":
			name = packageNames[imported.Path]
		}
		if name == pkg {
			return imported.Path, true
		}
	}
	return "", false
}

// mergeAliasMethods attributes the references of each same-package type
// alias, including methods on an alias receiver, to the aliased declared
// type. An alias of another package cannot hold methods.
func (b *couplingBuilder) mergeAliasMethods(types map[string]*coupledType, aliases map[string]aliasTarget) {
	for key, target := range aliases {
		src := types[key]
		if src == nil || target.importPath != "" {
			continue
		}
		parts := strings.SplitN(key, "\x00", 3)
		if len(parts) != 3 {
			continue
		}
		lang, loc := parts[0], parts[1]
		seen := map[string]bool{key: true}
		for {
			if target.importPath != "" {
				break
			}
			destKey := lang + "\x00" + loc + "\x00" + target.name
			if seen[destKey] {
				break
			}
			seen[destKey] = true
			if dest := types[destKey]; dest != nil && dest.declared {
				dest.refs = append(dest.refs, src.refs...)
				break
			}
			next, ok := aliases[destKey]
			if !ok {
				break
			}
			target = next
		}
	}
}

// resolveAlias follows a same-package alias name to the declared type it
// names. ok is false when name is not an alias.
func (b *couplingBuilder) resolveAlias(file *couplingFile, name string, types map[string]*coupledType, aliases map[string]aliasTarget, aliasByImport map[string]aliasTarget, byImportPath map[string]*coupledType) (*coupledType, string, bool) {
	target, ok := aliases[file.language.Name+"\x00"+file.location+"\x00"+name]
	if !ok {
		return nil, "", false
	}
	return followAlias(file, target, types, aliases, aliasByImport, byImportPath)
}

// followAlias walks an alias chain to a declared type. A same-package
// target is looked up in types; a qualified one through the imported
// package, following aliases there too.
func followAlias(file *couplingFile, target aliasTarget, types map[string]*coupledType, aliases map[string]aliasTarget, aliasByImport map[string]aliasTarget, byImportPath map[string]*coupledType) (*coupledType, string, bool) {
	seen := map[string]bool{}
	for {
		id := target.importPath + "\x00" + target.name
		if target.importPath == "" {
			id = file.location + "\x00" + target.name
		}
		if seen[id] {
			return nil, "", true
		}
		seen[id] = true
		if target.importPath == "" {
			key := file.language.Name + "\x00" + file.location + "\x00" + target.name
			if dest := types[key]; dest != nil && dest.declared {
				return dest, target.name, true
			}
			next, ok := aliases[key]
			if !ok {
				return nil, "", true
			}
			target = next
			continue
		}
		dest, name, ok := lookupInPackage(target.importPath, target.name, byImportPath, aliasByImport)
		if !ok {
			return nil, "", true
		}
		display := name
		if target.qualifier != "" {
			display = target.qualifier + "." + name
		}
		return dest, display, true
	}
}

// lookupInPackage finds the declared type name in importPath, following
// aliases of that package. The returned name is the declared type.
func lookupInPackage(importPath, name string, byImportPath map[string]*coupledType, aliasByImport map[string]aliasTarget) (*coupledType, string, bool) {
	seen := map[string]bool{}
	for {
		if seen[importPath+"."+name] {
			return nil, "", false
		}
		seen[importPath+"."+name] = true
		if t, ok := byImportPath[importPath+"."+name]; ok {
			return t, t.name, true
		}
		next, ok := aliasByImport[importPath+"."+name]
		if !ok {
			return nil, "", false
		}
		if next.importPath != "" {
			importPath = next.importPath
		}
		name = next.name
	}
}

// bareName strips the scope prefix from a type name.
func bareName(language *engine.Language, name string) string {
	if i := strings.LastIndex(name, language.Separator()); i >= 0 {
		return name[i+len(language.Separator()):]
	}
	return name
}
