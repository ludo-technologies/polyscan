// Package rustdeps reads what the coupling analysis needs to tell the types
// of a Rust tree from those of other crates: the names a file's use
// declarations import, and the name of the crate a file belongs to.
package rustdeps

import (
	"context"
	"fmt"
	"strings"
	"sync"

	"github.com/ludo-technologies/polyscan/polyscan/internal/engine"
	"github.com/ludo-technologies/polyscan/polyscan/internal/lang/rust"
	sitter "github.com/smacker/go-tree-sitter"
)

const query = `(use_declaration argument: (_) @argument)`

var (
	compileOnce sync.Once
	compiled    *sitter.Query
	compileErr  error
)

func compile() (*sitter.Query, error) {
	compileOnce.Do(func() {
		compiled, compileErr = sitter.NewQuery([]byte(query), rust.Language.Grammar)
		if compileErr != nil {
			compileErr = fmt.Errorf("Rust use query: %w", compileErr)
		}
	})
	return compiled, compileErr
}

// ParseUses maps each name the use declarations of a Rust source bring into
// scope to the paths it is imported from, written without whitespace: use
// a::b::{C, D as E} binds C to a::b::C and E to a::b::D, and use a::{self}
// binds a. A glob import binds no name, and neither does an import renamed
// to _. The declarations of every scope of the file count, so a name
// imported in two scopes keeps both paths.
func ParseUses(source []byte) (map[string][]string, error) {
	q, err := compile()
	if err != nil {
		return nil, err
	}
	parser := sitter.NewParser()
	defer parser.Close()
	parser.SetLanguage(rust.Language.Grammar)
	tree, err := parser.ParseCtx(context.Background(), nil, source)
	if err != nil {
		return nil, err
	}
	defer tree.Close()

	uses := map[string][]string{}
	engine.ForEachMatch(q, tree.RootNode(), source, func(match *sitter.QueryMatch) {
		for _, capture := range match.Captures {
			bind(capture.Node, "", source, uses)
		}
	})
	return uses, nil
}

// bind records the names one use tree imports below prefix.
func bind(node *sitter.Node, prefix string, source []byte, uses map[string][]string) {
	add := func(name, path string) {
		uses[name] = append(uses[name], path)
	}
	switch node.Type() {
	case "identifier", "scoped_identifier":
		path := join(prefix, text(node, source))
		add(lastSegment(path), path)
	case "self":
		if prefix != "" {
			add(lastSegment(prefix), prefix)
		}
	case "use_as_clause":
		if alias := text(node.ChildByFieldName("alias"), source); alias != "_" {
			add(alias, join(prefix, text(node.ChildByFieldName("path"), source)))
		}
	case "scoped_use_list":
		if path := node.ChildByFieldName("path"); path != nil {
			prefix = join(prefix, text(path, source))
		}
		bind(node.ChildByFieldName("list"), prefix, source, uses)
	case "use_list":
		for i := 0; i < int(node.NamedChildCount()); i++ {
			bind(node.NamedChild(i), prefix, source, uses)
		}
	}
}

func text(node *sitter.Node, source []byte) string {
	return strings.Join(strings.Fields(node.Content(source)), "")
}

func join(prefix, path string) string {
	if prefix == "" {
		return path
	}
	return prefix + "::" + path
}

func lastSegment(path string) string {
	if i := strings.LastIndex(path, "::"); i >= 0 {
		return path[i+len("::"):]
	}
	return path
}
