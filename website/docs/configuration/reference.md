# Configuration Reference

Every key that `.polyscan.toml` accepts, with its type, its default, and its rules. The file configures every language: Go, Rust, C++ and JavaScript/TypeScript. These are all the keys. A key or section that is not listed here is an error. The [configuration guide](index.md) explains how polyscan finds the file and how flags combine with it.

A key that the file leaves out keeps its default.

---

## `analysis`

These keys choose which files every analysis covers.

### `analysis.exclude`

array of strings &nbsp;&middot;&nbsp; default `[]` &nbsp;&middot;&nbsp; flag `--exclude`

Further files and directories to leave out, for every language and every analysis. The patterns are added to the [built-in skips](#built-in-skips) and never replace them.

```toml
[analysis]
exclude = ["fixtures", "src/generated/**"]
```

The syntax is the same as the syntax of the `--exclude` flag:

- A glob **without a slash** matches a file name or a directory name anywhere on the path. `fixtures` skips every directory named `fixtures` at any depth, along with everything inside it. It leaves `src/fixtures_loader.ts` alone, because no name in that path is exactly `fixtures`. Glob characters apply to a single name, so `*.min.js` matches file names and `__*__` matches a directory named `__tests__`.
- A glob **with a slash** matches a path relative to the analyzed directory, where `**` stands for any number of path segments. `src/generated` skips that directory and everything under it. `src/generated/**` skips every file below it. `**/dist/**` skips every file below any directory named `dist`.

A pattern is matched against whole names, never against part of one. Patterns are matched relative to the path you pass to polyscan, so the directories above it are never considered.

When you also pass `--exclude` on the command line, both sets of patterns apply.

### `analysis.include_tests`

boolean &nbsp;&middot;&nbsp; default `false` &nbsp;&middot;&nbsp; flag `--include-tests`

Analyze test files and test code too. By default they are left out of every analysis. The output of `polyscan analyze --help` lists what counts as test code in each language.

```toml
[analysis]
include_tests = true
```

The `--include-tests` flag replaces this value only when you give it. A run without the flag uses the value from the file.

---

## `complexity`

Sets the risk bands of cyclomatic complexity. They apply to every language, and the HTML report's complexity histogram follows them.

### `complexity.low_threshold`

integer &nbsp;&middot;&nbsp; default `9`

The highest complexity of the low risk band, inclusive. A function at or below this value is low risk.

Must be at least 1.

### `complexity.medium_threshold`

integer &nbsp;&middot;&nbsp; default `19`

The highest complexity of the medium risk band, inclusive. A function above it is high risk.

Must be greater than `low_threshold`.

```toml
[complexity]
low_threshold = 10
medium_threshold = 20
```

The `config` object in the [JSON output](../output/json-schema.md) reports the thresholds a run used.

---

## `check`

Sets the thresholds of [`polyscan check`](../cli/check.md). Each key is the default of the flag with the same name, and the flag takes precedence when you give it. `polyscan analyze` reads this section but does not use it.

### `check.max_complexity`

integer &nbsp;&middot;&nbsp; default `10` &nbsp;&middot;&nbsp; flag `--max-complexity`

The highest cyclomatic complexity a function may have before the check fails.

Must be at least 1.

### `check.max_cycles`

integer &nbsp;&middot;&nbsp; default `0` &nbsp;&middot;&nbsp; flag `--max-cycles`

The number of circular dependency cycles the check allows.

Must be at least 0.

### `check.allow_dead_code`

boolean &nbsp;&middot;&nbsp; default `false` &nbsp;&middot;&nbsp; flag `--allow-dead-code`

Print critical dead code without failing the check.

### `check.allow_circular_deps`

boolean &nbsp;&middot;&nbsp; default `false` &nbsp;&middot;&nbsp; flag `--allow-circular-deps`

Print circular dependencies without failing the check.

### `check.allow_parse_errors`

boolean &nbsp;&middot;&nbsp; default `false` &nbsp;&middot;&nbsp; flag `--allow-parse-errors`

Skip files with a syntax error without failing the check. A file that cannot be read still fails.

```toml
[check]
max_complexity = 15
max_cycles = 2
allow_dead_code = true
```

---

## Built-in skips

Some files are left out before `analysis.exclude` applies. These skips are fixed behavior and not keys. The `exclude` key adds to them and cannot remove any of them.

### Go, Rust and C++

polyscan skips any directory whose name starts with a dot, and directories named `node_modules`, `vendor`, `target`, `build`, `dist` and `third_party`. A path named on the command line is analyzed whatever it is called.

### JavaScript/TypeScript

polyscan skips the files and directories that match this list:

```text
node_modules
bower_components
jspm_packages
vendor
assets
overrides
third_party
third-party
extern
external
dist
build
out
.output
.next
.nuxt
.vercel
.cache
.turbo
coverage
.git
*.min.js
*.min.mjs
*.min.cjs
*.bundle.js
*.map
```

The entries follow the matching rules of a glob without a slash, described under [`analysis.exclude`](#analysisexclude). An entry is matched against whole names, never against part of one, and matching ignores case, so `*.min.js` also skips `Vendor.MIN.JS`. `dist` skips every directory named `dist` and leaves `src/utils/distance.ts` alone.

Patterns are matched relative to the path you pass to polyscan. A project stored at `/home/me/build/myapp` is analyzed normally even though `build` is on the list. A file you name directly on the command line is matched on its own name alone, so `polyscan analyze src/dist/bundle.js` analyzes that file.

polyscan also reads the `.gitignore` at the root of the analyzed path and skips what it ignores. See [the `.gitignore` section](index.md#the-gitignore) of the guide.

---

## Keys that no longer exist

Earlier polyscan releases read a jscan-era configuration file. Its keys are not part of `.polyscan.toml`, and putting one in the file is an error. The JavaScript/TypeScript-only keys that used to take effect have no replacement in the file:

| Old key | What to do instead |
| --- | --- |
| `analysis.include_patterns` | Nothing. Use `analysis.exclude` to leave files out |
| `analysis.recursive` | Nothing |
| `output.min_complexity` | Use the `--min-complexity` flag of `polyscan analyze` |
| `output.sort_by` | Nothing |
| `dead_code.min_severity` | Nothing |
| `dead_code.sort_by` | Nothing |
| `complexity.report_unchanged` | Nothing |

The keys that were parsed and then ignored, such as `clones.*`, `output.format` and `system_analysis.*`, are gone as well. See [migrating a jscan configuration file](index.md#migrating-from-a-jscan-configuration-file).
