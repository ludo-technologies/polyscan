// Package javadeps reads what the coupling analysis needs to resolve the type
// names of a Java file: the package it declares and the names it imports.
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

// File is the package declaration and the imports of one Java file.
type File struct {
	// Package is the name in the package declaration, or empty for a file
	// of the unnamed package.
	Package string
	// Imports are the import declarations, in source order.
	Imports []Import
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

// ParseFile reads the package declaration and the imports of a Java source.
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
		for _, capture := range match.Captures {
			name := strings.Join(strings.Fields(capture.Node.Content(source)), "")
			switch q.CaptureNameForId(capture.Index) {
			case "package":
				file.Package = name
			case "import":
				next := capture.Node.NextNamedSibling()
				file.Imports = append(file.Imports, Import{Name: name, OnDemand: next != nil && next.Type() == "asterisk"})
			}
		}
	})
	return file, nil
}
