// Package javadeps reads the package, the imports and the top-level types of
// Java files, which the coupling analysis resolves type names through, and
// builds the package import graph of a Java tree from them.
package javadeps

import (
	"context"
	"fmt"
	"strings"
	"sync"

	"github.com/ludo-technologies/polyscan/polyscan/internal/engine"
	"github.com/ludo-technologies/polyscan/polyscan/internal/lang/java"
	sitter "github.com/smacker/go-tree-sitter"
)

// File is the package declaration, the imports and the top-level types of
// one Java file.
type File struct {
	// Package is the name in the package declaration, or empty for a file
	// of the unnamed package.
	Package string
	// Imports are the import declarations, in source order.
	Imports []Import
	// Types are the top-level type declarations, in source order.
	Types []Type
}

// Type is a top-level class, interface, enum, record or annotation type.
type Type struct {
	Name string
	// Public reports the public modifier.
	Public bool
	// Abstract reports an interface, an annotation type or an abstract
	// class.
	Abstract bool
}

// Import is one import declaration, static or not.
type Import struct {
	// Name is the name after import, written without whitespace: a type
	// or a static member for a single import, such as java.util.List, and
	// the package or type whose members it imports for an import on
	// demand, such as java.util for import java.util.*.
	Name string
	// OnDemand reports an import that ends in .*.
	OnDemand bool
}

const query = `
(package_declaration [(identifier) (scoped_identifier)] @package)
(import_declaration [(identifier) (scoped_identifier)] @import)
(program [
  (class_declaration name: (identifier) @name)
  (interface_declaration name: (identifier) @name)
  (enum_declaration name: (identifier) @name)
  (record_declaration name: (identifier) @name)
  (annotation_type_declaration name: (identifier) @name)
] @type)
`

var (
	compileOnce sync.Once
	compiled    *sitter.Query
	compileErr  error
)

func compile() (*sitter.Query, error) {
	compileOnce.Do(func() {
		compiled, compileErr = sitter.NewQuery([]byte(query), java.Language.Grammar)
		if compileErr != nil {
			compileErr = fmt.Errorf("Java import query: %w", compileErr)
		}
	})
	return compiled, compileErr
}

// ParseFile reads the package declaration, the imports and the top-level
// types of a Java source.
func ParseFile(source []byte) (*File, error) {
	q, err := compile()
	if err != nil {
		return nil, err
	}
	parser := sitter.NewParser()
	defer parser.Close()
	parser.SetLanguage(java.Language.Grammar)
	tree, err := parser.ParseCtx(context.Background(), nil, source)
	if err != nil {
		return nil, err
	}
	defer tree.Close()

	file := &File{}
	engine.ForEachMatch(q, tree.RootNode(), source, func(match *sitter.QueryMatch) {
		var declared *sitter.Node
		var typeName string
		for _, capture := range match.Captures {
			name := strings.Join(strings.Fields(capture.Node.Content(source)), "")
			switch q.CaptureNameForId(capture.Index) {
			case "package":
				file.Package = name
			case "import":
				next := capture.Node.NextNamedSibling()
				file.Imports = append(file.Imports, Import{Name: name, OnDemand: next != nil && next.Type() == "asterisk"})
			case "type":
				declared = capture.Node
			case "name":
				typeName = name
			}
		}
		if declared != nil {
			file.Types = append(file.Types, topLevelType(declared, typeName))
		}
	})
	return file, nil
}

// topLevelType reads the modifiers of a type declaration. An interface or an
// annotation type is abstract whatever its modifiers say.
func topLevelType(declaration *sitter.Node, name string) Type {
	kind := declaration.Type()
	t := Type{Name: name, Abstract: kind == "interface_declaration" || kind == "annotation_type_declaration"}
	for i := 0; i < int(declaration.NamedChildCount()); i++ {
		modifiers := declaration.NamedChild(i)
		if modifiers.Type() != "modifiers" {
			continue
		}
		for j := 0; j < int(modifiers.ChildCount()); j++ {
			switch modifiers.Child(j).Type() {
			case "public":
				t.Public = true
			case "abstract":
				t.Abstract = true
			}
		}
	}
	return t
}
