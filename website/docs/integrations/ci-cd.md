# CI/CD Integration

`polyscan check` is the quality gate. It exits 0 when it finds no issues, 1 when it finds quality issues, and 2 when the analysis fails, so a broken job never passes as a clean one.

```bash
polyscan check src/
```

By default, `check` fails on a function with complexity above 10, on critical dead code, and on any circular dependency. The [check page](../cli/check.md) lists the flags that change these thresholds.

`analyze` always exits 0 when the analysis itself ran. To gate on the health score or on any other field of the report, use its JSON output with `jq -e`, which exits 1 when the expression is false or null:

```bash
polyscan analyze --format json src/ > report.json
jq -e '.summary.health_score >= 75' report.json
```

## Before you write the job

Two things will otherwise waste your time.

**Gate on critical dead code, not all of it.** The warning that an exported function is not imported by another analyzed file fires constantly in libraries, whose exports are consumed outside the analyzed directory. `check` fails only on critical dead code for this reason. A JSON gate should start with `critical_dead_code == 0`.

**A gate is only as good as the file set it runs on.** Check the `Analyzing N files...` count against reality once, so that you know the gate covers your source tree. jscan versions up to 0.9.0 matched the exclude patterns `out` and `dist` against any part of a path and skipped `src/routes/`, `src/layout/`, and `src/checkout/`, which made a passing gate meaningless; polyscan matches whole names only, and see the [configuration reference](../configuration/reference.md#analysisexclude_patterns) for the current rules.

## GitHub Actions

### A quality gate

```yaml title=".github/workflows/quality.yml"
name: Code Quality

on:
  pull_request:
  push:
    branches: [main]

jobs:
  polyscan:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4

      - uses: actions/setup-node@v4
        with:
          node-version: '20'

      - name: Quality gate
        run: npx polyscan check src/
```

Pin the polyscan version (`npx polyscan@X.Y.Z`) once you rely on the gate: it keeps a new release from turning a green pipeline red without a change to your code.

### Publishing the HTML report

The report is a single self-contained file, which makes it a good artifact. Note that `analyze` never fails on findings, so this job reports without gating.

```yaml title=".github/workflows/quality.yml"
      - name: Full analysis
        run: npx polyscan analyze --no-open --output polyscan-report.html src/

      - uses: actions/upload-artifact@v4
        if: always()
        with:
          name: polyscan-report
          path: polyscan-report.html
          retention-days: 30
```

`--no-open` matters here. Without it polyscan tries to launch a browser, which does nothing useful on a runner.

### Posting the score on a pull request

```yaml
      - name: Analyze
        id: polyscan
        run: |
          npx polyscan analyze --format json src/ 2>/dev/null > report.json
          {
            echo "score=$(jq '.summary.health_score' report.json)"
            echo "grade=$(jq -r '.summary.grade' report.json)"
          } >> "$GITHUB_OUTPUT"

      - name: Comment
        if: github.event_name == 'pull_request'
        uses: actions/github-script@v7
        with:
          script: |
            await github.rest.issues.createComment({
              issue_number: context.issue.number,
              owner: context.repo.owner,
              repo: context.repo.repo,
              body: `**polyscan**: ${{ steps.polyscan.outputs.score }}/100 (grade ${{ steps.polyscan.outputs.grade }})`
            })
```

The `2>/dev/null` keeps the score summary out of the JSON file.

### Failing below a grade

```yaml
      - name: Require grade B or better
        run: |
          score=$(npx polyscan analyze --format json src/ 2>/dev/null | jq '.summary.health_score')
          echo "Health score: $score"
          if [ "$score" -lt 75 ]; then
            echo "::error::Health score $score is below the grade B threshold of 75"
            exit 1
          fi
```

Always run the same `--select` when you gate on the score, and remember that the score is computed over the dimensions that ran: a narrower selection is scored against a smaller budget, so scores from different selections are not comparable. See [the health score page](../output/health-score.md#selecting-fewer-analyses-changes-the-budget).

### Analyzing a monorepo

```yaml
jobs:
  polyscan:
    runs-on: ubuntu-latest
    strategy:
      fail-fast: false
      matrix:
        package: [core, web, api]
    steps:
      - uses: actions/checkout@v4
      - uses: actions/setup-node@v4
        with:
          node-version: '20'
      - run: npx polyscan check packages/${{ matrix.package }}/src
```

`fail-fast: false` means one failing package does not cancel the others, so a single run tells you about all of them.

## GitLab CI

```yaml title=".gitlab-ci.yml"
quality:
  stage: test
  image: node:20
  script:
    - npx polyscan check src/
    - npx polyscan analyze --no-open --output polyscan-report.html src/
  artifacts:
    when: always
    paths:
      - polyscan-report.html
    expire_in: 30 days
```

`when: always` publishes the report even when the gate fails, which is exactly when you want to read it.

Note the ordering. The gate runs first and stops the job on failure, so put the HTML run after it and rely on `when: always` to still collect the artifact. On a large repository, narrow the gate with `--select` if the double analysis costs too much time.

## Pre-commit hook

A gate on the whole project is too slow for a commit hook. Check only the staged files, and only their complexity:

```bash title=".git/hooks/pre-commit"
#!/usr/bin/env bash
set -euo pipefail

files=$(git diff --cached --name-only --diff-filter=ACM \
        | grep -E '\.(js|jsx|mjs|cjs|ts|tsx|mts|cts|go|rs|cpp|cc|cxx|hpp|hh|hxx|h)$' || true)

[ -z "$files" ] && exit 0

# shellcheck disable=SC2086
npx polyscan check --quiet --select complexity --max-complexity 20 $files
```

Make it executable with `chmod +x .git/hooks/pre-commit`.

Restricting `--select` to `complexity` is deliberate. Dead code detection on a handful of staged files reports almost every export as unused, because the importers are not in the file list.

## Docker

There is no official image. Install polyscan into whichever image you already use:

```dockerfile
FROM node:20-alpine
RUN npm install -g polyscan
WORKDIR /src
ENTRYPOINT ["polyscan"]
```

```bash
docker build -t polyscan-ci .
docker run --rm -v "$PWD:/src" polyscan-ci analyze --format text src/
```

## Tracking the score over time

Appending one line per commit gives you a trend without any extra infrastructure:

```bash
#!/usr/bin/env bash
set -euo pipefail

json=$(polyscan analyze --format json src/ 2>/dev/null)
printf '%s\t%s\t%s\t%s\n' \
  "$(git rev-parse --short HEAD)" \
  "$(date -u +%Y-%m-%dT%H:%M:%SZ)" \
  "$(jq '.summary.health_score' <<<"$json")" \
  "$(jq -r '.summary.grade' <<<"$json")" \
  >> quality-history.tsv
```

Two rules keep the series meaningful. Pin the polyscan version, since scoring changes between releases would show up as a change in your code quality. Keep the same `--select` and the same configuration file, for the same reason.

## Recommended progression

Adopting the strictest settings on day one means a permanently red pipeline that people learn to ignore. A workable sequence is:

1. **Report only.** Run `polyscan analyze` and publish the artifact. Nothing fails. Let the team look at it for a couple of weeks.
2. **Gate on complexity alone** with `polyscan check --select complexity --max-complexity N`, at a threshold `N` your codebase already passes.
3. **Lower the threshold** by five whenever the pipeline has been comfortably green for a while.
4. **Add the cycle check** once the circular imports are fixed, by adding `deps` to `--select`.
5. **Add `deadcode`** to `--select` once the unreachable code is removed. The selection is then the default, and `polyscan check` needs no `--select`.

## See also

- [`polyscan check`](../cli/check.md) for every flag of the gate
- [JSON schema](../output/json-schema.md) for every field a JSON gate can read
- [Configuration examples](../configuration/examples.md) for files to commit alongside these jobs
