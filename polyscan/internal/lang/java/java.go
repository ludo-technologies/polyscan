// Package java defines how polyscan analyzes Java.
package java

import (
	"github.com/ludo-technologies/polyscan/polyscan/internal/engine"
	tsjava "github.com/smacker/go-tree-sitter/java"
)

// Language is the Java definition. Lambdas are not extracted on their own:
// their decision points count toward the enclosing method, as Go's function
// literals do. The methods of an anonymous class are extracted like any
// other method. Initializer blocks and field initializers are not
// functions. In a class declared inside a method, an anonymous class
// included, their decision points count toward that method, as a lambda's
// do. Elsewhere they are not counted.
var Language = &engine.Language{
	Name:       "Java",
	Extensions: []string{".java"},
	Grammar:    tsjava.GetLanguage(),
	// The conventions of Maven and Gradle: test sources under src/test, and
	// the class names the Surefire and Failsafe plugins run. Bazel projects
	// keep tests under javatests, and Android projects keep instrumented
	// tests under src/androidTest.
	TestFiles: []string{
		"*Test.java", "*Tests.java", "*TestCase.java", "*IT.java",
		"test/", "tests/", "javatests/", "androidTest/",
	},
	// The method and constructor patterns of tree-sitter-java's
	// queries/tags.scm, restricted to members that have a body, so abstract,
	// interface and native methods are not counted as functions. A record's
	// compact constructor has a body and no parameter list.
	Definitions: `
(method_declaration name: (identifier) @name body: (block)) @definition.method
(constructor_declaration name: (identifier) @name) @definition.constructor
(compact_constructor_declaration name: (identifier) @name) @definition.constructor
`,
	// Classes, enums and records are types, whose methods are methods of the
	// type. An interface has no instance fields for its default methods to
	// share, so it only qualifies names, as a Rust trait does. An anonymous
	// class and the body of an enum constant are named after the type they
	// extend and the constant, so their methods are not read as methods of
	// the enclosing class.
	Scopes: `
(class_declaration name: (identifier) @receiver body: (class_body) @scope)
(enum_declaration name: (identifier) @receiver body: (enum_body) @scope)
(record_declaration name: (identifier) @receiver body: (class_body) @scope)
(interface_declaration name: (identifier) @module body: (interface_body) @scope)
(annotation_type_declaration name: (identifier) @module body: (annotation_type_body) @scope)
(enum_constant name: (identifier) @module body: (class_body) @scope)
(object_creation_expression type: [
  (type_identifier) @module
  (generic_type . (type_identifier) @module)
  (scoped_type_identifier (type_identifier) @module .)
  (generic_type . (scoped_type_identifier (type_identifier) @module .))
] (class_body) @scope)
`,
	// A switch arm counts when its label is a case: the default label has
	// no value, and it is the path the other arms branch away from. The
	// arms of the colon form are switch_block_statement_group nodes and the
	// arms of the arrow form are switch_rule nodes; both forms appear in
	// switch statements and switch expressions. A label holds one or more
	// values, as in case A, B, and counts once. A guard (case P p when ...)
	// is a further branch. A catch clause is the exception edge core/cfg
	// counts.
	Decisions: `
(if_statement) @branch
(guard) @branch
(ternary_expression) @ternary
(for_statement) @loop
(enhanced_for_statement) @loop
(while_statement) @loop
(do_statement) @loop
(switch_block_statement_group (switch_label . (_))) @case
(switch_rule (switch_label . (_))) @case
(catch_clause) @exception
(binary_expression operator: ["&&" "||"]) @logical_operator
`,
	// A lambda is not a definition, so a block body holds statements of the
	// method that contains it rather than of the expression the lambda sits
	// in, and an expression body is the value the lambda returns.
	Returns: `
(return_statement) @return
(lambda_expression body: (block) @closure)
(lambda_expression body: [
  (binary_expression)
  (parenthesized_expression)
  (unary_expression)
] @return)
`,
	// An if in the else arm of another if continues that if's chain. A
	// catch clause is part of the try statement that already opened a
	// level.
	Nesting: `
(if_statement) @nesting
(if_statement alternative: (if_statement) @continuation)
(switch_expression) @nesting
(for_statement) @nesting
(enhanced_for_statement) @nesting
(while_statement) @nesting
(do_statement) @nesting
(try_statement) @nesting
(try_with_resources_statement) @nesting
`,
	Clone: engine.CloneSpec{
		Identifiers: []string{
			"identifier", "type_identifier", "this", "super",
			"integral_type", "floating_point_type", "boolean_type", "void_type",
		},
		Literals: []string{
			"decimal_integer_literal", "hex_integer_literal", "octal_integer_literal",
			"binary_integer_literal", "decimal_floating_point_literal", "hex_floating_point_literal",
			"string_literal", "character_literal", "true", "false", "null_literal",
		},
		Patterns: []string{
			"if_statement", "for_statement", "enhanced_for_statement", "while_statement", "do_statement",
			"switch_expression", "switch_block_statement_group", "switch_rule", "try_statement",
			"try_with_resources_statement", "catch_clause", "throw_statement", "return_statement",
			"break_statement", "continue_statement", "yield_statement", "synchronized_statement",
			"method_invocation", "field_access", "array_access", "binary_expression", "unary_expression",
			"update_expression", "assignment_expression", "ternary_expression", "cast_expression",
			"instanceof_expression", "object_creation_expression", "array_creation_expression",
			"lambda_expression", "method_reference", "local_variable_declaration",
		},
		Structural: []string{
			"program", "method_declaration", "constructor_declaration", "compact_constructor_declaration",
			"lambda_expression", "class_declaration", "interface_declaration", "enum_declaration",
			"record_declaration", "annotation_type_declaration", "class_body",
		},
		ControlFlow: []string{
			"if_statement", "for_statement", "enhanced_for_statement", "while_statement", "do_statement",
			"switch_expression", "switch_block", "switch_block_statement_group", "switch_rule",
			"switch_label", "guard", "try_statement", "try_with_resources_statement", "catch_clause",
			"finally_clause", "throw_statement", "return_statement", "break_statement",
			"continue_statement", "yield_statement", "labeled_statement", "synchronized_statement",
		},
		Expressions: []string{
			"binary_expression", "unary_expression", "update_expression", "assignment_expression",
			"ternary_expression", "method_invocation", "argument_list", "field_access", "array_access",
			"cast_expression", "instanceof_expression", "object_creation_expression",
			"array_creation_expression", "array_initializer", "parenthesized_expression",
			"method_reference", "class_literal",
		},
		Related: [][2]string{
			{"for_statement", "enhanced_for_statement"},
			{"for_statement", "while_statement"},
			{"enhanced_for_statement", "while_statement"},
			{"while_statement", "do_statement"},
			{"if_statement", "ternary_expression"},
			{"if_statement", "switch_expression"},
			{"switch_block_statement_group", "switch_rule"},
			{"try_statement", "try_with_resources_statement"},
			{"binary_expression", "unary_expression"},
			{"assignment_expression", "update_expression"},
			{"class_declaration", "record_declaration"},
		},
	},
}
