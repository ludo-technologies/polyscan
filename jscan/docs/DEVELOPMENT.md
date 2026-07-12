# Development Guide

## Prerequisites

- **Go 1.24+** - [Download](https://go.dev/dl/)
- **golangci-lint** - Required for linting (`make lint`)

## Getting Started

```bash
# Clone the repository
git clone https://github.com/ludo-technologies/polyscan.git
cd jscan

# Download dependencies
go mod download

# Build the binary
make build
```

## Makefile Targets

| Target | Description |
|---|---|
| `make build` | Build the `jscan` binary |
| `make test` | Run all tests with verbose output |
| `make test-short` | Run short tests only (skip long-running tests) |
| `make bench` | Run benchmarks with memory allocation stats |
| `make coverage` | Generate HTML coverage report (`coverage.html`) |
| `make lint` | Run `golangci-lint` |
| `make fmt` | Format all Go source files |
| `make clean` | Remove build artifacts, coverage files, and `dist/` |
| `make install` | Build and install the binary via `go install` |
| `make run` | Build and run against `testdata/javascript/simple/` |
| `make version` | Print version, commit, and build date |
| `make build-all` | Cross-compile for linux/amd64, linux/arm64, darwin/amd64, darwin/arm64, windows/amd64 |
| `make deps` | Download and verify module dependencies |
| `make tidy` | Tidy `go.mod` and `go.sum` |

## Project Structure

```
jscan/
├── cmd/jscan/          # CLI entry point (main.go, analyze.go, check.go, deps.go, init.go)
├── domain/             # Domain models (complexity, dead code, clone, coupling, output)
├── app/                # Application use cases
├── service/            # Service layer
├── internal/
│   ├── parser/         # tree-sitter JavaScript/TypeScript parser
│   ├── analyzer/       # Analysis engines (CFG, complexity, dead code, clones, deps)
│   ├── config/         # Configuration management
│   ├── reporter/       # Output formatting (text, JSON, HTML, CSV, DOT)
│   ├── constants/      # Shared constants
│   ├── testutil/       # Test utilities
│   └── version/        # Version information
├── npm/                # npm package wrapper
└── testdata/           # Test fixtures (javascript/)
```

## Build Details

The build injects version metadata via linker flags (`-ldflags`):

- `Version` - from `git describe --tags --always --dirty`
- `Commit` - from `git rev-parse --short HEAD`
- `Date` - build date
- `BuiltBy` - set to `make` when built via Makefile
