# Contributing to polyscan

Thank you for your interest in polyscan. The most useful contribution is to run polyscan on a project you work on and tell us what it got wrong.

## Start by Using polyscan

```bash
npx polyscan analyze .
```

Run it on your own code and open the HTML report. Check whether the findings and the score match what you know about the project. A wrong finding, a score that seems too harsh or too lenient, and a file that polyscan could not parse are all worth reporting.

## Reporting a Wrong Result

Open an issue with the **Wrong result** template. Tell us which finding is wrong and why, and include a small code sample that reproduces it if you can. Reports from real projects improve polyscan more than anything else, because they cover code that our test fixtures do not.

## Pull Requests

We welcome pull requests, and the ones we can review best come from your own use of polyscan. If you hit a bug in your project and want to fix it, open an issue first and link it from your pull request.

Issues labeled `good first issue` or `help wanted` are open to anyone. Before you start on one, run polyscan on a project of your own, so that you can see how the change shows up in a real report.

The pull request template asks which project you ran polyscan on. The answer helps us review the change against real code.

## Development

The repository holds two Go modules:

- `core/` is the shared analysis library used by polyscan and [pyscn](https://github.com/ludo-technologies/pyscn).
- `polyscan/` is the polyscan CLI.

Requires Go 1.24+. CI runs these checks in each module:

```bash
cd polyscan   # or core
test -z "$(gofmt -l .)"
go vet ./...
go test -race ./...
```

All new behavior must come with tests. Prefer table-driven tests where they fit.

### Commit Messages

Follow [Conventional Commits](https://www.conventionalcommits.org/):

- `feat:` new feature
- `fix:` bug fix
- `refactor:` code change that neither fixes a bug nor adds a feature
- `docs:` documentation only
- `test:` adding or updating tests

### Constraints on core

- **No language-specific dependencies.** Language-specific behavior is injected through interfaces (`StatementClassifier`, `CostModel`, `ComplexityContributor`, and others).
- **No external dependencies.** The module has no third-party dependencies by design.
- **Breaking changes require a major version bump.** Both pyscn and polyscan pin core to specific versions.
- All exported types and functions must have godoc comments.
