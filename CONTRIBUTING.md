# Contributing to codescan-core

## Getting Started

```bash
git clone https://github.com/ludo-technologies/codescan-core.git
cd codescan-core
go test ./...
```

Requires Go 1.24+.

## Development Workflow

1. Fork the repository
2. Create a feature branch (`git checkout -b feature/my-change`)
3. Make your changes
4. Run tests and vet:
   ```bash
   go test ./... -count=1
   go vet ./...
   ```
5. Commit with a descriptive message
6. Push and open a Pull Request

## Commit Messages

Follow [Conventional Commits](https://www.conventionalcommits.org/):

- `feat:` new feature
- `fix:` bug fix
- `refactor:` code change that neither fixes a bug nor adds a feature
- `docs:` documentation only
- `test:` adding or updating tests

## Code Style

- Follow standard Go conventions (`gofmt`, `go vet`)
- Keep exported APIs minimal and well-documented with godoc
- All public types and functions must have godoc comments

## Testing

- All new functionality must include tests
- Tests must pass before merging: `go test ./... -count=1`
- Aim for table-driven tests where applicable

## Architecture Notes

codescan-core is a shared library used by [pyscn](https://github.com/ludo-technologies/pyscn) and [jscan](https://github.com/ludo-technologies/jscan). Key constraints:

- **No language-specific dependencies.** Language-specific behavior is injected via interfaces (`StatementClassifier`, `CostModel`, `ComplexityContributor`, etc.)
- **No external dependencies.** The module has zero third-party dependencies by design
- **Breaking changes require a major version bump.** Both pyscn and jscan pin to specific versions

## Reporting Issues

Use [GitHub Issues](https://github.com/ludo-technologies/codescan-core/issues). Include:

- Go version (`go version`)
- What you expected vs what happened
- Minimal reproduction steps
