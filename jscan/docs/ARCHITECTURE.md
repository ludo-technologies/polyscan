# Architecture

## Overview

jscan uses a layered architecture inspired by **Clean Architecture**. Core analysis logic stays isolated from CLI/output concerns, while command handlers can call application use cases or services directly for pragmatic orchestration.

## Layer Diagram

```text
┌──────────────────────────────────────────────┐
│                  CLI (cmd/)                  │
│       cobra commands, arg parsing, I/O       │
├──────────────────────────────────────────────┤
│              Application (app/)              │
│      reusable use cases / file orchestration │
├──────────────────────────────────────────────┤
│               Service (service/)             │
│   analysis orchestration, formatting, output │
├──────────────────────────────────────────────┤
│              Internal (internal/)            │
│    parser, analyzers, config, reporter       │
├──────────────────────────────────────────────┤
│               Domain (domain/)               │
│      pure models and service interfaces       │
└──────────────────────────────────────────────┘
```

**Typical runtime flows:**

- `cmd -> service -> internal -> domain`
- `cmd -> app -> service -> internal -> domain`

All layers depend on `domain` for shared types; `domain` depends on nothing.

## Layer Descriptions

### cmd/jscan -- CLI Interface

Entry point using [cobra](https://github.com/spf13/cobra). Handles command-line argument parsing, flag configuration, and output rendering. Commands include:

- `analyze` - Run full project analysis
- `check` - Run health checks against thresholds
- `deps` - Analyze module dependencies
- `init` - Initialize a jscan configuration file

For performance-sensitive commands, CLI handlers may orchestrate services directly.

### app -- Application Use Cases

Provides reusable orchestration/use-case logic that can be used by CLI handlers and tests. Examples:

- `analyze_usecase.go` - Full analysis pipeline
- `complexity_usecase.go` - Complexity-focused analysis
- `dead_code_usecase.go` - Dead code workflow delegating to `domain.DeadCodeService`

### service -- Service Layer

Business logic services that operate between the CLI and core analyzers:

- **complexity_service** - Orchestrates complexity analysis
- **dead_code_service** - Orchestrates dead code detection
- **dead_code_aggregate** - Cross-file dead code aggregation (unused imports/exports, orphan files)
- **clone_service** - Orchestrates clone detection
- **cbo_service** - Orchestrates coupling metrics
- **dependency_graph_service** - Orchestrates dependency graph construction
- **output_formatter** - Formats results as text, JSON, HTML, or CSV
- **dot_formatter** - Generates DOT graph output for dependency visualization
- **parallel_executor** - Manages concurrent file analysis
- **progress_manager** - Terminal progress bar rendering
- **config_loader** - Loads and validates jscan configuration
- **browser** - Opens HTML reports in the system browser

### internal/parser -- Tree-sitter Integration

Wraps [go-tree-sitter](https://github.com/smacker/go-tree-sitter) to parse JavaScript and TypeScript source files into concrete syntax trees (CSTs). Provides the foundation for all downstream analysis.

### internal/analyzer -- Core Analysis Engines

The heart of jscan. Contains all static analysis algorithms:

- **CFG construction** (`cfg.go`, `cfg_builder.go`) - Builds control flow graphs from parsed ASTs
- **Reachability analysis** (`reachability.go`) - Determines reachable/unreachable code paths from CFG
- **Cyclomatic complexity** (`complexity.go`) - McCabe cyclomatic complexity calculation
- **Dead code detection** (`dead_code.go`, `unused_code.go`) - Detects unreachable code, unused imports/exports, and orphan files
- **Clone detection** (`clone_detector.go`) - Identifies duplicate code using APTED tree edit distance combined with MinHash/LSH for candidate selection
  - `apted.go` / `apted_tree.go` / `apted_cost.go` - APTED tree edit distance algorithm
  - `minhash.go` - MinHash fingerprinting for approximate similarity
  - `lsh_index.go` - Locality-sensitive hashing index for fast candidate retrieval
  - `ast_features.go` - AST feature extraction for fingerprinting
  - `grouping_strategy.go` - Strategies for grouping code fragments
- **Module analysis** (`module_analyzer.go`) - ESM and CommonJS import/export resolution
- **Dependency graph** (`dependency_graph.go`) - Builds the full module dependency graph
- **CBO metrics** (`cbo.go`, `coupling_metrics.go`) - Coupling Between Objects measurement
- **Circular dependency detection** (`circular_detector.go`) - Finds circular dependencies using Tarjan's strongly connected components algorithm

### internal/reporter -- Output Formatting

Formats complexity analysis results for different output targets.

### internal/config -- Configuration Management

Reads and manages jscan configuration (thresholds, ignore patterns, output settings).

### domain -- Domain Models

Pure data structures with no external dependencies:

- `complexity.go` - Complexity measurement models
- `dead_code.go` - Dead code finding types
- `clone.go` - Clone detection result types
- `cbo.go` - Coupling metric types
- `dependency_graph.go` - Dependency graph types
- `module.go` - Module/import/export types
- `output.go` - Output configuration types
- `system_analysis.go` - Top-level analysis result types
- `errors.go` - Domain error types

## Design Decisions

### Why layered + pragmatic orchestration?

The layered split keeps analysis engines independent of CLI/output concerns, but still allows direct service orchestration from command handlers where it simplifies concurrency and UX behavior. This keeps critical analysis logic testable while reducing command-level duplication.

### Why tree-sitter?

tree-sitter provides fast, incremental, error-tolerant parsing. Unlike regex-based approaches, it produces a full concrete syntax tree, enabling accurate structural analysis. It handles malformed files gracefully, which is important when scanning real-world codebases.

### Why APTED + MinHash/LSH for clone detection?

Pure tree edit distance (APTED) is accurate but O(n^3) per pair comparison, making it impractical for large codebases. MinHash fingerprinting with LSH indexing provides O(1) approximate similarity lookups to narrow candidates before running the expensive APTED comparison. This two-phase approach balances accuracy with performance.

### Why Tarjan's algorithm for circular dependencies?

Tarjan's algorithm finds all strongly connected components in a directed graph in O(V+E) time. Each strongly connected component with more than one node represents a circular dependency. This is more efficient and complete than naive cycle detection approaches.
