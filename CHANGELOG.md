# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

## [0.7.0] - 2026-06-12

### Changed

- Overhaul clone detection accuracy (synced with pyscn): APTED correctness fixes (key-root ordering, forest-distance subtree cost, max-based similarity normalization), Type-1/Type-2 gating on textual and normalized-AST-hash similarity, recalibrated thresholds (0.85/0.75/0.70/0.65), Type-3 disabled by default, exact complete-linkage clustering, and MinLines/MinNodes defaults raised to 10/20
- Switch the duplication score metric to clone-group density (groups per 1000 lines, 0-10% penalty scale)
- Soften CBO coupling score calibration and use architecture compliance directly as the architecture score
- Recalibrate coupling zone classification (Zone of Pain / Zone of Uselessness predicates, instability thresholds 0.2/0.8 to 0.3/0.7)
- Exclude dynamic `import()` edges from circular dependency detection; load-time cycles are still reported when a static edge exists
- Report module-scope code as `<module>` instead of `__main__`

### Fixed

- Share clones per fragment so overlapping clone groups merge correctly
- Reject overlapping same-file fragment pairs and remove strict-subset clone group members
- Merge contiguous same-reason dead code findings and skip empty-statement-only blocks

### Performance

- Optimize cross-file dead code analysis with a shared import graph
- Speed up clone detection with a Jaccard pre-filter, cached fragment features, and an LSH candidate cap

## [0.6.2] - 2026-02-19

### Fixed

- Reduce dead code false positives for Next.js and TypeScript imports

## [0.6.1] - 2026-02-15

### Fixed

- Set complexity function locations (start line, column, end line) from AST nodes
- Tidy go.mod dependencies

## [0.6.0] - 2026-02-15

### Fixed

- Stop counting nested functions' operators in parent complexity
- Resolve npx "command not found" by removing bin field from platform packages
- Add files field to main npm package to reduce package size

## [0.5.0] - 2026-02-15

### Changed

- Redesign README to match pyscn style (centered header, Quick Start, collapsible install)
- Add algorithm details to Features section
- Add demo video link

## [0.4.0] - 2026-02-15

### Changed

- Refactor dead code aggregation into dedicated service layer and align architecture docs

### Fixed

- Improve progress bar UX and speed up clone analysis (#46)

## [0.3.0] - 2026-02-14

### Changed

- Unify JSON output keys to snake_case and rename `detect_after_raise` to `detect_after_throw`

### Fixed

- Apply config `max_complexity` when CLI flag is not explicitly set
- Stop auto-discovering extensionless `.jscanrc`
- Wire config loading and harden CI workflows

## [0.2.2] - 2026-02-12

### Changed

- Switch npm distribution to per-platform packages (esbuild-style) for faster installation

## [0.2.1] - 2026-02-12

### Fixed

- Fix version not embedded in release binaries via ldflags

## [0.2.0] - 2026-02-12

### Added

- CLI analysis summary and modernize README ([#38](https://github.com/ludo-technologies/jscan/pull/38))
- Detect orphan files and unused exported functions ([#37](https://github.com/ludo-technologies/jscan/pull/37))
- Detect unused imports and exports in dead code analysis ([#36](https://github.com/ludo-technologies/jscan/pull/36))

### Fixed

- Improve dependency score accuracy ([#41](https://github.com/ludo-technologies/jscan/pull/41))
- Detect nested functions in BuildAll via recursive AST walk ([#40](https://github.com/ludo-technologies/jscan/pull/40))
- Resolve extensionless imports in dependency graph builder ([#39](https://github.com/ludo-technologies/jscan/pull/39))
- Resolve golangci-lint errors across codebase
- Adjust health score thresholds and add score to text output ([#35](https://github.com/ludo-technologies/jscan/pull/35))

## [0.1.1] - 2026-02-02

### Fixed

- Extract binary to temp dir to avoid overwriting bin/jscan script

## [0.1.0] - 2026-02-02

### Added

- JSON output format
- HTML output format with Lighthouse-style scoring
- Dead Code Service layer
- Application layer with Use Cases
- APTED (Tree Edit Distance) algorithm for clone detection
- MinHash and LSH Index for clone detection
- Clone Detector with Type 1-4 support
- Clone Grouping Strategies
- Module Analyzer for JS/TS import/export analysis
- CBO (Coupling Between Objects) metrics
- Dependency Graph with cycle detection
- DOT format for dependency visualization
- `check` command for CI/CD integration
- `init` command for config file generation
- Progress manager for long-running analysis tasks
- Parallel executor for concurrent task execution
- Default exclude patterns for common directories
- npm package distribution

### Changed

- Default output format to HTML for analyze command

### Fixed

- Clone loss bug and improved determinism in grouping strategies
- Various build and distribution fixes

## [0.1.0-alpha] - 2025-11-27

### Added

- Initial implementation with complexity analysis and dead code detection
- tree-sitter based JavaScript/TypeScript parsing
- CLI with analyze command
- Configuration file support (jscan.config.json)

[Unreleased]: https://github.com/ludo-technologies/jscan/compare/v0.6.1...HEAD
[0.6.1]: https://github.com/ludo-technologies/jscan/compare/v0.6.0...v0.6.1
[0.6.0]: https://github.com/ludo-technologies/jscan/compare/v0.5.0...v0.6.0
[0.5.0]: https://github.com/ludo-technologies/jscan/compare/v0.4.0...v0.5.0
[0.4.0]: https://github.com/ludo-technologies/jscan/compare/v0.3.0...v0.4.0
[0.3.0]: https://github.com/ludo-technologies/jscan/compare/v0.2.2...v0.3.0
[0.2.2]: https://github.com/ludo-technologies/jscan/compare/v0.2.1...v0.2.2
[0.2.1]: https://github.com/ludo-technologies/jscan/compare/v0.2.0...v0.2.1
[0.2.0]: https://github.com/ludo-technologies/jscan/compare/v0.1.1...v0.2.0
[0.1.1]: https://github.com/ludo-technologies/jscan/compare/v0.1.0...v0.1.1
[0.1.0]: https://github.com/ludo-technologies/jscan/compare/v0.1.0-alpha...v0.1.0
[0.1.0-alpha]: https://github.com/ludo-technologies/jscan/releases/tag/v0.1.0-alpha
