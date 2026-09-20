# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Fixed

- TypeScript classes written as `abstract class` or as a `class` expression reach clone detection as class fragments. The AST builder had a case for `class_declaration` only, so the other two spellings fell through to the generic builder, kept their tree-sitter type names, and were never offered as fragments; the CFG took the module name for them too, since the generic node carries no `Name`. Before, two files holding the same abstract base class or the same class expression reported no clone pair, while the same code as a plain class did (#158)

## [0.4.0] - 2026-09-17

### Added

- The analyze JSON document carries a top-level `schema_version`, currently `1`, that changes only when a documented key is renamed, removed or changes type
- The analyze JSON document carries a top-level `diagnostics` array, one record per file the run could not read or parse, each with a `code` of `read_error` or `parse_error`, the file path and a message. The key is omitted when every file was analyzed. Before, a consumer had to parse the `errors` strings of the individual analyses to tell a read failure from a parse failure (#140)

### Changed

- Each module in `deps.analysis.module_metrics` no longer carries `lines_of_code`, `function_count`, `class_count`, `maintainability`, `technical_debt`, `transitive_dependencies` or `package`. The dependency analysis measured none of them, so each one was serialized as a fixed zero, an empty list or an empty string that a consumer could not tell apart from a real measurement. File sizes are reported in `module_quality`, and `deps.analysis.dependency_matrix` holds the full edge set to walk for a module's transitive dependencies (#144)
- The Go, Rust and C++ risk level is derived from the complexity with flat dispatch collapsed. A `switch` or `match` that holds no decision point beyond its own arms counts as a single decision point, because its arm count measures the width of a lookup table rather than branching logic; an arm that branches, loops, guards or holds a nested switch keeps the whole construct counted arm by arm, the uncounted arms included: the `default` of a Go or C++ switch and the last arm of a Rust match. The reported `complexity` is unchanged and stays comparable with gocyclo. Before, a Bubble Tea or tview key handler whose every arm delegates to a method reached medium or high risk on its arm count alone (#142)
- The Go, Rust and C++ risk level also collapses the short-circuit operators of one returned expression into a single decision point. The operands of `return a && b && c && d` decide the value the function hands back rather than the path it takes to its one exit, so they are not separate paths through it; the operators of a condition, of an assignment, and of a returned expression that holds a branch of another kind, such as a ternary or a closure that branches, stay counted one by one. Each value of a multi-value Go return is an expression of its own, and the statements of a closure or lambda body belong to the function that holds it rather than to the expression the closure sits in. A Rust function returns the tail expression of its body, and a closure its tail or its whole expression body, while the tail of an inner block is not a return. The reported `complexity` is unchanged and stays comparable with gocyclo. Before, a short validation predicate reached the complexity of a function with real branching on its operator count alone (#143)
- JavaScript/TypeScript coupling (CBO) reports the names a file's type annotations reference in `type_hint_dependencies`, and those names stay out of `coupling_count` and the risk level. TypeScript erases a type annotation at compile time, so it declares a dependency without exercising one. Before, the breakdown was always 0 because the annotations were never read

### Fixed

- Go and Rust cohesion (LCOM4) no longer counts a stub method as a group of its own. A method that touches no field, calls no sibling method and is called by none, such as a constant return an interface requires or a placeholder that only panics, is listed in `excluded_methods` instead. Before, a type implementing a four-method interface with one stateful method reported `lcom4: 4` (#139)
- A JavaScript/TypeScript selection that parses but declares no function, such as a directory of type declarations, is a normal empty complexity result. Before, the whole complexity section was dropped with "no functions found to analyze"
- Every collection in the analyze JSON document is present even when empty, as `[]` or `{}`. Before, `dead_code.files`, the coupling summary, `base_classes`, the dependency cycle and coupling lists, `longest_chains`, and each module's `direct_dependencies`, `dependents` and `public_interface` were `null` when there was nothing to list, and `public_interface` was never filled in; it now lists the module's exports. The never-populated `most_depended_upon_classes` key is gone from the coupling summary
- File paths in a report are spelled the way the analyzed path was given on the command line, for every language. Before, when the target resolved outside the working directory, Go, Rust and C++ findings carried an absolute path while JavaScript/TypeScript findings carried the relative one, so a single report held two spellings of the same file (#136)
- Go methods declared on a type-alias receiver (`type Alias = Base`; `func (a *Alias) M()`) are attributed to the aliased type in the coupling (CBO) analysis, and a field or parameter typed by an alias credits that type too. Before, the alias was not a declared type, the methods were dropped, and `Base` reported no coupling (#133)

## [0.3.3] - 2026-09-12

### Added

- `polyscan analyze --exclude` leaves files and directories out of every analysis, for every language. A pattern without a slash is a glob matched against the file name and against each directory on the path (`fixtures`); a pattern with a slash is matched against the path relative to the analyzed directory, with `**` for any number of segments (`src/generated/**`). The patterns follow the rules of `exclude_patterns` in `jscan.config.json`, and for JavaScript/TypeScript they are added to that file's own exclude patterns

### Changed

- Test files and test code are left out of every analysis by default, where before they were analyzed for complexity and, for JavaScript/TypeScript, for everything. Go `*_test.go`; Rust `#[test]` functions, `#[cfg(test)]` items, `tests.rs`, `*_tests.rs` and `tests/`; C++ `*_test.*`, `*_tests.*`, `test_*.*`, `*Test.*`, `test/` and `tests/`; JavaScript/TypeScript `*.test.*`, `*.spec.*` and `__tests__/`. `--include-tests` restores the previous behavior, with test code still out of clone detection, cohesion, coupling and dependency analysis

### Fixed

- A Rust type declared in one file with `impl` blocks in another was counted twice in the report overview, because the cohesion analysis filed it under the file of its first method while the coupling analysis filed it under the declaring file. The cohesion analysis now uses the declaring file too, so the two rows merge into one class (#131)
- A bare Rust type name that resolves to more than one declaration in the tree is left unresolved in the coupling analysis, with a warning. Before, a reference to such a name was assigned to whichever declaration was found first, while an `impl` block for it was dropped (#132)

## [0.3.2] - 2026-09-10

### Changed

- The `cbo` and `deps.analysis` sections of the JSON report now use snake_case keys, as every other section does and as pyscn's `cbo` and `system.dependency_analysis` output does. The domain structs behind them had no `json` tags, so a report contained `Classes` and `Instability` next to `total_files` and `health_score`. A consumer that decoded the PascalCase keys must switch to the snake_case ones (#128)

## [0.3.1] - 2026-09-10

### Fixed

- A relative JavaScript/TypeScript import written with the extension of the compiled output, as `Node16` and `NodeNext` module resolution require, now resolves to the TypeScript source. `import { x } from './foo.js'` reaches `foo.ts`, `foo.tsx` or `foo.d.ts`, `.jsx` reaches `.tsx`, and `.mjs` and `.cjs` reach `.mts` and `.cts`, with the same rewrite applied to index files and path aliases. Before, the specifier became an external module of the dependency graph, the target file was reported as an entry point, and its exports were reported as unused (#126)

## [0.3.0] - 2026-09-09

### Added

- Class coupling (CBO) for Go and Rust. `polyscan analyze` counts, for each type, the other types of the analyzed tree that its declaration and methods refer to, and scores the result in the Coupling dimension with the thresholds pyscn uses (low up to 3, medium up to 7). Only types the tree declares count: for Go an unqualified name is a type of the same package and `pkg.T` resolves through `go.mod` to a package of the tree, for Rust a bare name resolves to a declaration in the same file or elsewhere in the tree, so the standard library and other modules never count. An embedded field or interface and an implemented trait count as inheritance. Types coupled to nothing, test files and `#[cfg(test)]` code stay out. A Go tree without a `go.mod` keeps only its same-package references, with a warning. In the report a Go or Rust type can now have both a coupling and a cohesion row, and the class count in the overview counts it once
- Class cohesion (LCOM4) for Go and Rust. `polyscan analyze` measures, for each type, how many groups its methods fall into when two methods are connected by a shared field or a call between them, and scores the result in a Cohesion dimension with the thresholds pyscn uses (low up to 2, medium up to 5). A Go type is measured over every method of its package, since they may be spread across files; a Rust type over the `impl` blocks in a file. Methods without a receiver parameter, such as Rust associated functions and Go methods with an unnamed receiver, cannot touch instance state and are listed as excluded. Test files and `#[cfg(test)]` code stay out. `--select lcom` runs it alone
- Dependency analysis for Go. `polyscan analyze` builds the package import graph of a Go tree, resolving each import through the nearest `go.mod`, and reports the same instability, abstractness, main-sequence distance, depth and longest chains it reports for JavaScript/TypeScript, scored in the Dependencies dimension. Abstractness is the share of a package's exported type declarations that are interfaces. Test files, `vendor` and `testdata` directories, and imports of other modules stay out of the graph, and a tree without a `go.mod` leaves the dimension out with a warning rather than scoring it clean

### Changed

- A Rust `impl` names its type by the bare identifier, so the methods of `impl G<T>` read `G::m` instead of `G<T>::m`, and `impl G<i32>` and `impl G<String>` are blocks of one type in the cohesion and coupling analyses. An impl for a reference type such as `&Foo` is a block of `Foo`

### Fixed

- Go, Rust and C++ analysis no longer walks into version control, dependency and build output directories. A directory whose name starts with a dot, such as `.git`, and `node_modules`, `vendor`, `target`, `build`, `dist` and `third_party` are skipped; a path named on the command line is still analyzed whatever it is called. Before, a Rust project's `target` directory and a Go project's `vendor` directory were analyzed as if they were the project's own code
- `--output` is honored for JSON and text reports. `polyscan analyze --format json --output report.json` wrote the report to stdout and ignored the path; it now writes the file, and the default for JSON and text stays stdout (#125)
- The Go cohesion analysis sees a field behind a chained index. tree-sitter-go parses `t.m[k][k]` as a type instantiation, so a method that touched a field only through a nested map or slice index was not connected to the methods sharing that field (#117)
- The `package` flag of a JavaScript/TypeScript module survives into its dependency metrics, so package and non-package modules are told apart in the report again (#124)
- A runtime error no longer prints the command usage after the error message (#121)

## [0.2.2] - 2026-09-05

### Changed

- Clone pairs for Go, Rust and C++ are reported from similarity 0.80 instead of 0.70, and the duplication penalty saturates at a 60 percent fragment ratio instead of 30 percent. Pairs between 0.70 and 0.80 were mostly functions that share a shape rather than code, such as two output formatters' `switch` statements, and the 30 percent saturation scored cobra and polyscan at 25/100 and testify and afero at 0/100 on duplication. With the new settings those repositories score 70, 70, 50 and 30, and pyscn moves from 0 to 60 (#105)

## [0.2.1] - 2026-09-05

### Fixed

- Nesting depth is computed for Go, Rust and C++. The generic engine had no notion of nesting, so the directory rows for those languages always printed an average of 0.00 and a maximum of 0. Each language now declares a nesting query and the engine measures a function's depth the way the JavaScript analyzer does: the body is depth 0, an else-if continues its chain, and code in a separately extracted function is that function's own (#94)
- Parse errors are charged to the health score whichever analyses ran. Skipped files were counted only by the complexity analysis, so a run that left complexity out of `--select` scored unparsable files as clean. The dead-code rate is also divided by the files dead code analysis covered, so Go, Rust and C++ files in a mixed tree no longer dilute the JavaScript-only rate (#92)
- The report header shows the real project directory. A relative target such as `polyscan/` printed the project name twice because the common directory was the relative path itself; the analyzed files are resolved to absolute paths first
- The complexity penalty saturated once the weighted ratio of medium and high risk functions reached 5 percent, so a mostly clean codebase with a few complex functions scored 0/100 on complexity. It now saturates at 30 percent, matching the documented intent, and any function above the medium threshold costs at least one point so the score only reads 100 when no function is at risk (#96)

## [0.2.0] - 2026-09-03

### Added

- JavaScript and TypeScript analysis, moved in from jscan. `polyscan analyze` runs complexity, dead code, clone detection, class coupling (CBO) and module dependency analysis on `.js`, `.jsx`, `.ts` and `.tsx` files, reads an existing `jscan.config.json` (or any of the other accepted names) when present, and the `jscan` npm package became a deprecated wrapper around this CLI

### Changed

- One report, one health score across languages. `polyscan analyze` renders every language into the jscan-style report (health score, verdict, per-dimension cards, hotspot files) as HTML, JSON and text; the separate JavaScript report (`polyscan-report.js.html`, the `javascript` JSON key) is gone
- The health score is computed over the dimensions that ran: each enabled dimension is charged against its own maximum, and a dimension a language does not have (dead code, coupling and dependencies outside JavaScript/TypeScript) is left out of the score rather than scored as clean
- `--select` now covers every analysis (`complexity`, `deadcode`, `clone`, `cbo`, `deps`, default all) and applies across languages
- One JSON shape for every language, with `language` on every function and clone fragment

## [0.1.0] - 2026-08-28

The first release of the multi-language analyzer. One `polyscan` binary detects the language of each file by its extension and runs the same analyses on Go, Rust and C++.

### Added

- `polyscan analyze` with an HTML report, opened in the browser by default, and `--format json` and `--format text` for stdout. `--select complexity,clone` chooses the analyses, `--min-complexity` filters the listed functions, `--output` names the HTML file and `--no-open` skips the browser
- Cyclomatic complexity per function, following the `core/cfg` conventions: one point per two-way branch or loop, one per switch or match case with the default excluded, one per short-circuit operator, and one per exception edge (`?` in Rust, `catch` in C++). Closures and lambdas count toward the enclosing function. Go numbers match gocyclo
- Clone detection over functions of at least 10 lines of code and 20 syntax nodes, through `core/clone`: the APTED tree edit distance with a per-language cost model, Type-1 to Type-3 classification, connected grouping with the shared dedupe passes, and MinHash banding above 10,000 candidate pairs. Test code is analyzed for complexity but excluded from clone detection: `*_test.go`; Rust `#[test]` functions, `#[cfg(test)]` items, `tests.rs`, `*_tests.rs` and `tests/`; C++ `*_test.*`, `*_tests.*`, `test_*.*`, `*Test.*`, `test/` and `tests/`
- A generic tree-sitter engine. A language is a grammar plus declarative queries: function definitions, enclosing scopes, decision points, test code, and a clone spec naming identifiers, literals, structural patterns and cost tiers
- Go, Rust and C++ definitions. Headers, `.h` included, are analyzed as C++
- A file with a syntax error is analyzed without the functions that contain the error and reported as partial with a warning; a file that cannot be read is skipped and reported as an error
- Distribution through npm (`npx polyscan`), with native builds per platform, and `go install github.com/ludo-technologies/polyscan/polyscan/cmd/polyscan@latest`

### Known limitations

- C++ is parsed without the preprocessor: every `#if` branch is analyzed, macros are not expanded, and code whose syntax needs expansion is left out of the file's partial result
- Rust macro invocations are opaque token trees, and the bundled grammar predates Rust 2024 syntax such as `unsafe extern` blocks
- Type-4 (semantic) clones are not reported; polyscan runs no control-flow or data-flow analysis
