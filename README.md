# codescan-core

Shared Go module that provides language-agnostic code analysis algorithms for [pyscn](https://github.com/ludo-technologies/pyscn) (Python analyzer) and [jscan](https://github.com/ludo-technologies/jscan) (JavaScript/TypeScript analyzer).

## Packages

| Package | Description |
|---------|-------------|
| `apted/` | APTED tree edit distance algorithm with configurable cost models and normalization |
| `cfg/` | Control Flow Graph data structures, reachability analysis, McCabe cyclomatic complexity, dead code detection |
| `clone/` | AST feature extraction and 5 clone grouping strategies (Connected, KCore, StarMedoid, CompleteLinkage, Centroid) |
| `graph/` | Directed graph abstraction, Tarjan SCC cycle detection, Robert Martin coupling metrics |
| `lsh/` | Locality-Sensitive Hashing index and MinHash signatures |
| `util/` | Common utilities (browser opener) |

## Install

```
go get github.com/ludo-technologies/codescan-core@v0.1.0
```

## Design

Language-specific behavior is injected via interfaces and callbacks:

- **`graph.DirectedGraph`** — pyscn's `map[string]*ModuleNode` and jscan's `domain.DependencyGraph` both implement this
- **`cfg.StatementClassifier`** — abstracts return/break/continue/throw detection across languages
- **`cfg.ComplexityContributor`** — allows language-specific complexity additions (e.g. jscan's logical operators)
- **`graph.CouplingConfig.AbstractnessFunc`** — injects language-specific abstractness calculation
- **`clone.GroupingStrategy[T]`** — generic grouping with Go 1.24 generics for type-safe usage
- **`apted.CostModel`** — language-specific tree edit cost models (PythonCostModel, JavaScriptCostModel stay in each project)

## Requirements

Go 1.24+

## License

MIT
