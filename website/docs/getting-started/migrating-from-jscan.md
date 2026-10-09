# Migrating from jscan

jscan, the JavaScript/TypeScript analyzer, has merged into polyscan. The same analysis — complexity, dead code, clone detection, coupling, dependencies, and the health score — now runs from the `polyscan` CLI, alongside Go, Rust and C++ support.

The [`jscan` npm package](https://www.npmjs.com/package/jscan) remains published as a thin wrapper that prints a deprecation notice and runs polyscan, so an existing `npx jscan analyze` keeps working while you migrate: the wrapper translates the `--json`, `--text` and `--html` shorthands to `--format`, and exits with a hint on `--config`. New setups should call `polyscan` directly.

## Command mapping

| jscan | polyscan |
| --- | --- |
| `npx jscan analyze src/` | `npx polyscan analyze src/` |
| `jscan analyze --json src/` | `polyscan analyze --format json src/` |
| `jscan analyze --text src/` | `polyscan analyze --format text src/` |
| `jscan analyze -o report.html src/` | `polyscan analyze -o report.html src/` |
| `jscan check src/` | `polyscan check src/`. See [the `check` command](#the-check-command) |
| `jscan deps src/` | `polyscan analyze --select deps src/` |
| `jscan init` | Retired — [write the file by hand](#the-init-command) |
| `jscan version` | `polyscan version` |

The default HTML report file is now `polyscan-report.html` rather than `jscan-report.html`.

## Flag changes on `analyze`

- The `--json`, `--text` and `--html` shorthands are gone; use `--format json|text|html`.
- The `--config` / `-c` flag is gone. The configuration file is discovered automatically, [searching upward from the analyzed path](../configuration/index.md#how-polyscan-finds-your-config-file).
- `--min-complexity` is now a flag on `analyze`; it limits which functions the report lists without changing any score.
- Everything else — `--select`, `--no-open`, `-o/--output` — is unchanged.

## The `check` command

`polyscan check` replaces `jscan check` and keeps its exit codes: 0 for no issues, 1 for quality issues, and 2 when the analysis fails. The differences are in the flags:

- The thresholds come from flags only. The configuration file's `complexity.max_complexity` key is not read, so pass `--max-complexity` instead. The `--config` flag is gone, as on `analyze`.
- `--allow-parse-errors` waives only syntax errors. A file that cannot be read still fails the check with exit code 2.
- The `--json` and `--verbose` flags are gone. `check` prints each finding as one line on standard error.
- `--select` also accepts `clone`, which lists clones without failing the check.

The [check page](../cli/check.md) lists every flag. The `jscan` npm wrapper does not forward `check`. It exits 2 with a hint to run `polyscan check`.

To gate on the health score, use the JSON output of `analyze`:

```bash
polyscan analyze --format json src/ > report.json
jq -e '.summary.health_score >= 75' report.json
```

## The `deps` command

Dependency analysis lives in `polyscan analyze --select deps`. The text output carries the summary and cycle list, and the `deps` section of the JSON output carries the graph, the per-module Martin metrics, and the coupling analysis:

```bash
polyscan analyze --format json --select deps src/ | jq '.deps.analysis'
```

The Graphviz DOT export (`jscan deps --dot`) was retired with the command. The nodes and edges are in `.deps.graph` of the JSON output if you want to render the graph yourself.

## The `init` command

`jscan init` was retired. polyscan still reads the same configuration file — `jscan.config.json` or any of the [other accepted names](../configuration/index.md#accepted-filenames) — it just no longer generates one. The [configuration examples](../configuration/examples.md) page has complete files to copy.

## Configuration is unchanged

Your existing `jscan.config.json` keeps working as-is for the JavaScript/TypeScript analysis: same discovery order, same keys, same validation, including the `JSCAN_CONFIG` environment variable. See the [configuration guide](../configuration/index.md).

## The npm package

Replace `jscan` with `polyscan` in `devDependencies` and scripts. Version numbers restart from polyscan's own line, so `jscan@0.10.0` is older code than the first polyscan releases despite the higher number; jscan's release history up to the merge is in the repository's [`jscan/CHANGELOG.md`](https://github.com/ludo-technologies/polyscan/blob/main/jscan/CHANGELOG.md).

If you install with Go, the path changed too:

```bash
go install github.com/ludo-technologies/polyscan/polyscan/cmd/polyscan@latest
```
