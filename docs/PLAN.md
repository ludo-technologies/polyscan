# polyscan core Integration Plan

## Overview

`polyscan core` is a shared Go module that extracts language-agnostic code analysis algorithms from [pyscn](https://github.com/ludo-technologies/pyscn) (Python analyzer) and [jscan](https://github.com/ludo-technologies/jscan) (JavaScript/TypeScript analyzer).

**Module path:** `github.com/ludo-technologies/polyscan/core`

## Architecture

```
polyscan core/
├── apted/       # APTED tree edit distance algorithm
├── cfg/         # Control Flow Graph data structures + reachability, complexity, dead code
├── clone/       # AST feature extraction + clone grouping strategies (5 algorithms)
├── graph/       # Directed graph abstraction, Tarjan SCC, Robert Martin coupling metrics
├── lsh/         # Locality-Sensitive Hashing + MinHash
├── util/        # Browser opener, common utilities
└── docs/        # Documentation
```

## Extraction Tiers

### Tier 1: Direct Extraction (This PR)

Pure algorithms with no language-specific dependencies.

| Package | Source Files | Similarity | Description |
|---------|-------------|------------|-------------|
| `apted/` | `apted.go`, `apted_tree.go`, `apted_cost.go` | 95%/60%/50% | Tree edit distance. TreeNode + utilities shared; TreeConverter stays in each project. CostModel interface + DefaultCostModel + WeightedCostModel shared; PythonCostModel/JavaScriptCostModel stay in each project. |
| `lsh/` | `lsh_index.go`, `minhash.go` | 92%/95% | LSH index and MinHash signatures. Byte-for-byte identical algorithms. |
| `cfg/` | `cfg.go` | 98% | CFG data structures (EdgeType, Edge, BasicBlock, CFG, visitors). Decoupled from `parser.Node` by using `any`. |
| `clone/` | `ast_features.go` | 95% | AST feature extraction operating on generic TreeNode. |
| `util/` | `browser.go` | 100% | Browser opener. Identical in both projects. |

**Key design decisions for Tier 1:**
- `TreeNode.OriginalNode` changed from `*parser.Node` to `any` to remove parser dependency
- `BasicBlock.Statements` changed from `[]*parser.Node` to `[]any` for the same reason
- `CFG.FunctionNode` changed from `*parser.Node` to `any`
- APTED `ComputeSimilarity` normalization made configurable via `NormalizationMode` (pyscn uses `max(size1,size2)`, jscan uses `size1+size2`)
- `IsBoilerplateLabel` helper not included (Python-specific); referenced in pyscn's PythonCostModel only

### Tier 2: Interface Abstraction (Done)

Generic interfaces abstract over language-specific types.

| Package | Source Files | Description | Status |
|---------|-------------|-------------|--------|
| `graph/` | `graph.go`, `cycles.go`, `coupling.go` | `DirectedGraph` interface + `MapGraph`, Tarjan SCC cycle detection, Robert Martin coupling metrics (Ca/Ce/Instability/Abstractness/Distance) | Done |
| `clone/` | `grouping.go` | 5 generic grouping strategies using Go 1.24 generics: Connected (Union-Find), KCore, StarMedoid, CompleteLinkage (Bron-Kerbosch), Centroid (BFS expansion). `GroupableItem`/`ItemPair[T]`/`ItemGroup[T]`/`GroupingStrategy[T]` interfaces. | Done |
| `cfg/` | `reachability.go`, `complexity.go`, `deadcode.go` | Reachability analysis via DFS with `StatementClassifier`, McCabe cyclomatic complexity with `ComplexityContributor`, dead code detection with severity levels. | Done |

**Key design decisions for Tier 2:**
- `DirectedGraph` interface allows pyscn (`map[string]*ModuleNode`) and jscan (`domain.DependencyGraph`) to implement without modification
- `CouplingConfig.AbstractnessFunc` callback injects language-specific abstractness calculation
- `StatementClassifier` interface abstracts return/break/continue/throw detection across languages
- `ComplexityContributor` interface lets jscan add logical operator complexity while pyscn passes nil
- Clone grouping uses Go 1.24 generics (`GroupingStrategy[T GroupableItem]`) for type-safe usage without interface method call overhead

**Remaining Tier 2 items (future):**

| Module | Similarity | Required Abstraction |
|--------|------------|---------------------|
| Domain models (`system_analysis.go`, `errors.go`, `cbo.go`, `complexity.go`) | 80-99% | Rename language-specific references (e.g., `CollectPythonFiles` -> `CollectSourceFiles`) |
| Service orchestration (`*_service.go`) | 65-80% | Generic analysis orchestrator with pluggable parser |
| MCP infrastructure | N/A (jscan has none) | Shared `mcpkit` with argument parsing, output modes |

### Tier 3: Language-Specific (Not Shared)

| Module | Reason |
|--------|--------|
| `cfg_builder.go` | Python statements vs JS statements |
| `cbo.go` (analyzer) | Class-based vs module-based coupling |
| `dependency_graph.go` | Different module systems |
| `module_analyzer.go` | Language-specific import resolution |
| `dfa.go` | Python-specific variable semantics |
| `internal/parser/` | tree-sitter-python vs tree-sitter-javascript |

## Migration Strategy

### Phase 1: Publish polyscan core v0.1.0
1. Extract Tier 1 code into polyscan core
2. Write tests for all extracted packages
3. Tag v0.1.0

### Phase 2: Migrate pyscn
1. `go get github.com/ludo-technologies/polyscan/core@v0.1.0`
2. Replace internal packages with polyscan core imports
3. Keep `PythonCostModel`, `TreeConverter`, `cfg_builder` locally
4. Verify all existing tests pass

### Phase 3: Migrate jscan
1. Same as Phase 2 but for jscan
2. Add MCP support using shared `mcpkit` (Tier 2)

### Phase 4: Tier 2 extraction
1. Define generic interfaces for grouping strategies
2. Extract domain models with language-neutral naming
3. Extract service orchestration patterns

## Versioning

- Follow semver strictly
- Breaking changes in polyscan core require major version bump
- pyscn and jscan pin to specific minor versions
- Coordinate releases across all three repos

## Impact on Contributors

- pyscn's public API and CLI remain unchanged
- Internal import paths change (e.g., `pyscn/internal/analyzer` -> `polyscan core/apted`)
- PRs touching shared algorithms should go to polyscan core first
- Language-specific PRs continue to go to pyscn/jscan directly
