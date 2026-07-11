# polyscan

Multi-language code quality analysis, built as a monorepo around a shared algorithmic core.

## Layout

| Directory | Description |
|-----------|-------------|
| [`core/`](core/) | Language-agnostic analysis algorithms (APTED tree edit distance, LSH/MinHash, CFG analysis, clone detection, coupling/cohesion metrics) as a standalone Go module |

Language analyzers planned to move into or start life in this monorepo:

- **jscan** (JavaScript/TypeScript) — currently at [ludo-technologies/jscan](https://github.com/ludo-technologies/jscan), migrating here
- **goscan** (Go) — planned

[pyscn](https://github.com/ludo-technologies/pyscn) (Python) remains an independent repository and consumes `core/` as a Go module dependency.

## Versioning

Each module is tagged with a directory prefix, e.g. `core/v0.2.0`.

## License

MIT
