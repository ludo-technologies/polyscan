---
name: cli-analysis
description: Run the jscan command-line tool for JavaScript/TypeScript code quality analysis - CI/CD quality gates, HTML/JSON reports, full analysis runs, and project configuration. Use when user wants a CI check, a shareable report file, or to configure jscan for a project.
---

# JavaScript/TypeScript Code Quality Analysis with the jscan CLI

Use the `jscan` command-line tool when the task needs report files, CI integration, or project configuration.

No install needed: `npx jscan@latest <command>` (or `npm install -g jscan` for a permanent install). The examples below use a bare `jscan` for readability — prefix with `npx jscan@latest` when jscan isn't installed.

## Commands

| Command | Purpose |
|---------|---------|
| `jscan analyze <path>` | Comprehensive analysis: complexity, dead code, clones, coupling (CBO), dependencies |
| `jscan check <path>` | Fast pass/fail quality gate for CI with configurable thresholds |
| `jscan deps <path>` | Dependency graph analysis and visualization (text/JSON/DOT) |
| `jscan init` | Generate a `jscan.config.json` config file |

## analyze — full analysis and reports

```bash
jscan analyze src/                          # HTML report (jscan-report.html) + opens browser
jscan analyze --no-open src/                # HTML report without opening a browser
jscan analyze --json src/                   # JSON results to stdout
jscan analyze --text src/                   # human-readable text to stdout
jscan analyze --select complexity,deadcode src/   # only specific analyses
jscan analyze -o report.html src/           # custom report path
```

Key flags:

- `--json` / `--text`: write results to stdout instead of an HTML report
- `--select`: `complexity,deadcode,clone,cbo,deps` (all run by default)
- `--no-open`: don't auto-open the HTML report in a browser (use in scripts)
- `-o <path>`: HTML report path (default: `jscan-report.html` in the current directory)
- `-c <path>`: config file

The HTML report path is printed on completion. With `--json`, the machine-readable results go to stdout and the health-score summary goes to stderr, so stdout stays parseable.

## check — CI quality gate

```bash
jscan check src/                                     # complexity + deadcode + deps
jscan check --select complexity --max-complexity 10 src/
jscan check --select deps src/                       # fail on circular dependencies
jscan check --allow-dead-code src/
jscan check --json src/                              # machine-readable result
```

Exit codes: `0` no issues, `1` quality issues found, `2` analysis failed (invalid input, missing files). Default gates: complexity > 10 fails, any dead code finding fails (relax with `--allow-dead-code`), dependency cycles fail (relax with `--allow-circular-deps` or `--max-cycles <n>`).

`--select` accepts: `complexity`, `deadcode`, `deps`.

## Configuration

Run `jscan init` to scaffold `jscan.config.json` (`.jscanrc.json` also works) when a project wants persistent settings. The config file supplies complexity thresholds, exclude patterns, and the default for `check --max-complexity`; command-line flags always win over config values. Output format and `--select` are flag-only — the config file does not change them.

## Reporting Results

Summarize the health score and grade, list the specific functions/files behind each failing category, and suggest fixes. For CI setup, recommend `jscan check` in the pipeline and `jscan analyze` HTML reports for periodic deep reviews.
