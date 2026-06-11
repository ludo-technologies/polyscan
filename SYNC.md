# pyscn → jscan sync management

pyscn is the upstream (source of truth); changes to shared algorithms are ported to jscan manually or by an agent.
The `/sync-pyscn` command reads this file to run a sync.

- Upstream: https://github.com/ludo-technologies/pyscn (local clone: `../pyscn`)
- **Sync baseline SHA**: `f0457d7e5826aab91f879fc88b9b01858c8f78f6` (2025-11-27, v1.4.1)
  - The pyscn state that jscan's initial implementation was based on. All pyscn changes after this point are an **unported backlog**; the two tools are not in sync until the initial catch-up completes.
  - Update this value to pyscn's `origin/main` HEAD on each sync run.

## Initial catch-up (incomplete)

About 210 commits (71 of them classified as "sync") have accumulated on the synced files since the baseline.
The initial catch-up must be done by **state comparison**, not by replaying individual commits:

1. For each target file, compare pyscn's current `origin/main` content against jscan's current content and identify missing fixes and constant changes (use the commit log only to understand the intent of a change)
2. Split the work into one PR per area, ordered by churn:
   - **Clone detection** — **Done** (2026-06-11, compared against pyscn `73acc60`. PR: sync/catchup-clone) — covered `clone_detector.go`, `apted*.go`, the grouping strategies, `domain/clone.go`, `ast_features.go`, and newly ported `textual_similarity.go` / `syntactic_similarity.go`
   - **Scoring** — `domain/analyze.go` (16), `domain/system_analysis.go` (9), `domain/complexity.go` (9). Goal: identical grade computation in both tools
   - **Misc** — `dependency_graph.go`, `reachability.go`, `lsh_index.go`, `coupling_metrics.go`, and other low-commit files
3. Once all areas are done, delete this section and update the baseline SHA to the `origin/main` HEAD used during the catch-up

## Sync policy classification

| Classification | Meaning |
|---|---|
| **sync** | Language-independent algorithm. Port essentially all pyscn changes |
| **case-by-case** | Shared concept but needs language adaptation. Decide per change whether to port |
| **reference-only** | Language-specific. Never port code. Only report significant design-direction changes to a human |

## File mapping

### internal/analyzer

| pyscn | jscan | Classification | Notes |
|---|---|---|---|
| `internal/analyzer/apted.go` | `internal/analyzer/apted.go` | sync | Core APTED algorithm |
| `internal/analyzer/apted_tree.go` | `internal/analyzer/apted_tree.go` | sync | |
| `internal/analyzer/apted_cost.go` | `internal/analyzer/apted_cost.go` | reference-only | Cost models are language-specific (Python AST vs JS/TS AST) |
| `internal/analyzer/minhash.go` | `internal/analyzer/minhash.go` | sync | |
| `internal/analyzer/lsh_index.go` | `internal/analyzer/lsh_index.go` | sync | |
| `internal/analyzer/ast_features.go` | `internal/analyzer/ast_features.go` | case-by-case | Feature-extraction framework is shared; node-type handling is language-specific |
| `internal/analyzer/clone_detector.go` | `internal/analyzer/clone_detector.go` | case-by-case | Pipeline structure is shared |
| `internal/analyzer/grouping_strategy.go`<br>`internal/analyzer/connected_grouping.go`<br>`internal/analyzer/k_core_grouping.go`<br>`internal/analyzer/star_medoid_grouping.go`<br>`internal/analyzer/centroid_grouping.go`<br>`internal/analyzer/complete_linkage_grouping.go`<br>`internal/analyzer/complete_linkage_clusterer.go`<br>`internal/analyzer/complete_linkage_heap.go`<br>`internal/analyzer/group_dedup.go`<br>`internal/analyzer/grouping_mode.go` | `internal/analyzer/grouping_strategy.go` | sync | **Many-to-one**: pyscn splits one file per strategy, jscan keeps them in a single file |
| `internal/analyzer/cfg.go` | `internal/analyzer/cfg.go` | sync | BasicBlock/CFG data structures |
| `internal/analyzer/cfg_builder.go` | `internal/analyzer/cfg_builder.go` | reference-only | Control-flow semantics are language-specific (try/except/match vs try/catch/switch/hoisting) |
| `internal/analyzer/reachability.go` | `internal/analyzer/reachability.go` | sync | BFS reachability analysis |
| `internal/analyzer/complexity.go` | `internal/analyzer/complexity.go` | case-by-case | McCabe computation is shared; branch-node counting is language-specific (`??`, ternary, etc.) |
| `internal/analyzer/dead_code.go` | `internal/analyzer/dead_code.go` | case-by-case | CFG traversal is shared; JS adds hoisting handling |
| `internal/analyzer/cbo.go` | `internal/analyzer/cbo.go` | reference-only | AST traversal for dependency collection is language-specific |
| `internal/analyzer/coupling_metrics.go` | `internal/analyzer/coupling_metrics.go` | sync | Martin metrics (Ca/Ce/I/A) |
| `internal/analyzer/circular_detector.go` | `internal/analyzer/circular_detector.go` | sync | DFS cycle detection |
| `internal/analyzer/dependency_graph.go` | `internal/analyzer/dependency_graph.go` | case-by-case | Graph construction is shared; ModuleInfo contents are language-specific |
| `internal/analyzer/textual_similarity.go` | `internal/analyzer/textual_similarity.go` | case-by-case | Type-1 gate. Comment removal is language-specific (`#` vs `//` / `/* */`) |
| `internal/analyzer/syntactic_similarity.go` | `internal/analyzer/syntactic_similarity.go` | sync | Type-2 gate (Jaccard over normalized AST hashes). `jaccardSimilarity` lives here too |
| `internal/analyzer/module_analyzer.go` | `internal/analyzer/module_analyzer.go` | reference-only | Import resolution is language-specific (`__init__.py` vs ESM/CJS/Node builtins) |

### domain (scoring, type definitions)

| pyscn | jscan | Classification | Notes |
|---|---|---|---|
| `domain/analyze.go` | `domain/analyze.go` | sync | Health-score computation and penalty constants. **Grade computation must match across both tools** |
| `domain/system_analysis.go` | `domain/system_analysis.go` | sync | Same as above |
| `domain/complexity.go` | `domain/complexity.go` | case-by-case | Keep thresholds and risk-level definitions aligned |
| `domain/clone.go` | `domain/clone.go` | case-by-case | Keep similarity thresholds and clone-type definitions aligned |
| `domain/cbo.go` | `domain/cbo.go` | case-by-case | |
| `domain/dead_code.go` | `domain/dead_code.go` | case-by-case | |
| `domain/output.go` | `domain/output.go` | case-by-case | Mind JSON output schema compatibility |
| `domain/errors.go` | `domain/errors.go` | case-by-case | |

### jscan-specific (no upstream)

- `internal/analyzer/unused_code.go` — cross-file dead code detection (includes the Next.js App Router exception conventions)

### pyscn-specific, unported features (future porting candidates)

Out of sync scope, but reference points when considering porting features to jscan:

- Cognitive complexity: `cognitive_complexity.go`, `nesting_depth.go`, `raw_metrics.go`
- LCOM4: `lcom.go`, `domain/lcom.go`
- DFA (unused-variable detection): `dfa.go`, `dfa_builder.go`
- DI anti-pattern detection: `di_antipattern_detector.go` plus `di_*.go`, `*_detector.go`, `framework_patterns.go`
- Split similarity-analysis structure (remaining): `structural_similarity.go`, `semantic_similarity.go`, `similarity_analyzer.go`, `clone_classifier` (multi-dimensional classification) — `textual_similarity.go` and `syntactic_similarity.go` are already ported (see the file mapping)
- Re-export resolution: `reexport_resolver.go`
- Improvement suggestions: `domain/suggestion.go`
- MCP server: `mcp/`, `cmd/pyscn-mcp/`

## Pending changes

(Record changes here that a sync run decided not to port this time.)

Skipped during the clone detection catch-up (2026-06-11):

- **Docstring skipping** (`SkipDocstrings` and the related fields in `apted_tree.go` / `clone_detector.go` / `domain/clone.go`) — Python-specific. JS/TS ASTs have no docstring statements
- **Boilerplate cost model** (`NewPythonCostModelWithBoilerplateConfig`, `ReduceBoilerplateSimilarity` / `BoilerplateMultiplier`) — depends on Python framework patterns such as dataclass/Pydantic (`framework_patterns.go` is an unported feature)
- **Multi-dimensional classifier and DFA** (`CloneClassifier`, `EnableMultiDimensionalAnalysis` / `EnableSemanticAnalysis` / `EnableDFA`) — depend on pyscn-specific unported features (semantic/structural similarity, DFA)
- **Type-4 CFG gating** (`87babcc`, `9efebd4`) — changes inside `semantic_similarity.go` (unported). Port together with the semantic analysis
- **LSH int IDs and `WithMaxCandidates`** (`LSHMaxCandidates`) — requires the `lsh_index.go` API change. Port during the "Misc" area catch-up
- **`CloneConfigurationLoader.MergeConfig`** — config-loader-layer change. jscan's loader implementation differs, so out of scope
- **star_medoid graph optimization** (`buildSimilarityGraph` / `mostSimilarMedoid`) — performance-only change, structurally different from jscan's StarMedoid implementation (iterative reassignment over domain.Clone). The `averageGroupSimilarity` change to count only existing pairs was ported
- **pyscn's removal of the content-less `ExtractFragments`** — jscan tests still use it, so both variants were kept

## Sync history

| Date | pyscn SHA | Summary |
|---|---|---|
| 2026-06-11 | `f0457d7` | Set the baseline to the state jscan's initial implementation was based on (v1.4.1). The ~210 commits since then are the unported backlog targeted by the initial catch-up |
| 2026-06-11 | `73acc60` | Initial catch-up: clone detection area done. APTED correctness fixes (ascending key roots, forest-distance subtreeCost, max(size) normalization), bounded large-tree approximation (same-shape distance, label/shape profiles), Jaccard pre-filter, Type-1 textual-match gate and Type-2 syntactic gate (ported textual/syntactic similarity), threshold recalibration (0.85/0.75/0.70/0.65), Type-3 disabled by default, overlapping-range pair rejection (isOverlappingLocation), strict-subset group member removal (group_dedup), complete_linkage rewritten as agglomerative clustering, MinLines/MinNodes 10/20, LSH auto-enable by estimated pair count, introduced `parser.OrderedChildren` |
