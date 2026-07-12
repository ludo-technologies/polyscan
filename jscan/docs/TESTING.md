# Testing Guide

## Running Tests

```bash
# Run all tests with verbose output
make test

# Run short tests only (skips long-running tests)
make test-short

# Run tests directly with go test
go test ./...

# Run tests for a specific package
go test ./internal/analyzer/...
go test ./service/...
go test ./app/...
```

## Running Benchmarks

```bash
# Run all benchmarks with memory allocation stats
make bench

# Run benchmarks for a specific package
go test -bench=. -benchmem ./internal/analyzer/...
```

## Code Coverage

```bash
# Generate an HTML coverage report
make coverage

# This produces:
#   coverage.out  - raw coverage profile
#   coverage.html - HTML report (open in browser)
```

## Test Data

Test fixtures live under the `testdata/` directory:

```
testdata/
└── javascript/
    └── simple/     # Simple JavaScript files for basic test scenarios
```

Test files use these fixtures to parse real JavaScript source code through tree-sitter, ensuring analysis results are validated against actual language constructs rather than synthetic inputs.

## Writing New Tests

Follow standard Go testing conventions:

- Test files are named `*_test.go` and placed alongside the code they test
- Use `testing.T` for unit tests and `testing.B` for benchmarks
- Use `testdata/` for fixture files -- Go tooling automatically excludes this directory from builds
- Use `t.Helper()` in test helper functions for accurate error line reporting
- Use `t.Skip()` or `testing.Short()` to conditionally skip long-running tests

### Example Test Structure

```go
func TestComplexity_SimpleFunction(t *testing.T) {
    // Arrange: load fixture
    src, err := os.ReadFile("testdata/javascript/simple/example.js")
    if err != nil {
        t.Fatal(err)
    }

    // Act: run analysis
    result, err := analyzer.CalculateComplexity(src)
    if err != nil {
        t.Fatal(err)
    }

    // Assert: verify results
    if result.Cyclomatic != 3 {
        t.Errorf("expected cyclomatic complexity 3, got %d", result.Cyclomatic)
    }
}
```

### Test Utilities

Shared test helpers are available in `internal/testutil/` for common setup and assertion patterns.
