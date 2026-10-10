# polyscan check

Checks the code against fixed thresholds and sets the exit code from the result. This is the command for a CI quality gate.

```bash
polyscan check [path...]
```

With no path, `check` analyzes the current directory.

## Exit codes

| Code | Meaning |
| --- | --- |
| `0` | No quality issues were found |
| `1` | Quality issues were found. Each issue is printed |
| `2` | The analysis failed, so the result is not a verdict about the code. An invalid flag, an invalid configuration file, a missing path, a path with no supported source files, and a file that cannot be read or parsed all exit 2 |

A pipeline can treat 1 and 2 differently. Code 1 means the code needs work. Code 2 means the job itself needs attention, and it is never a pass.

## What fails the check

| Analysis | Fails when | Languages |
| --- | --- | --- |
| `complexity` | A function has cyclomatic complexity above `--max-complexity` (default 10) | All |
| `deadcode` | Critical dead code is found, such as code after a `return` | JavaScript/TypeScript |
| `deps` | There are more circular dependency cycles than `--max-cycles` (default 0). Every cycle then counts as one issue | Go, JavaScript/TypeScript |
| `clone` | Never. Clones are listed for information only | All |

`check` runs `complexity`, `deadcode` and `deps` by default. Clone detection runs only when `--select` names `clone`.

Only critical dead code fails the check. Unused imports and exports are warnings, and they never fail it, because a library's exports are usually imported outside the analyzed directory.

A file that cannot be read or parsed fails the check with exit code 2, because the check cannot say anything about that file. `--allow-parse-errors` skips files with a syntax error and lets the check pass on the rest. A file that cannot be read still fails. Go, Rust and C++ files are parsed with error recovery, so a syntax error in one of them leaves out only the functions that contain it and does not fail the check. The [analyze page](analyze.md#language-coverage) explains this.

## Synopsis

```bash
polyscan check                                  # Check the current directory
polyscan check src/                             # Check one directory
polyscan check --select complexity src/         # Complexity only
polyscan check --max-complexity 15 src/         # Raise the complexity limit
polyscan check --max-cycles 3 .                 # Allow up to 3 dependency cycles
polyscan check --select complexity,clone .      # Also list clones
polyscan check --allow-dead-code .              # Report dead code without failing
polyscan check --allow-parse-errors .           # Skip unparsable files without failing
polyscan check --quiet .                        # Print only warnings unless issues are found
polyscan check -c ci.polyscan.toml .            # Read the thresholds from a named file
```

## Flags

| Flag | Short | Default | Description |
| --- | --- | --- | --- |
| `--select` | `-s` | `complexity,deadcode,deps` | Comma-separated list of analyses to run. Accepts `complexity`, `deadcode`, `clone` and `deps` |
| `--max-complexity` | | `10` | Maximum allowed cyclomatic complexity of a function |
| `--max-cycles` | | `0` | Maximum allowed number of circular dependency cycles |
| `--allow-dead-code` | | `false` | Print critical dead code without failing the check |
| `--allow-circular-deps` | | `false` | Print circular dependencies without failing the check |
| `--allow-parse-errors` | | `false` | Skip files with a syntax error without failing the check. A file that cannot be read still fails |
| `--quiet` | `-q` | `false` | Print nothing unless issues are found, apart from warnings. Clones and allowed findings are not printed |
| `--config` | `-c` | | The `.polyscan.toml` file to read. Skips discovery. The file must exist |
| `--exclude` | | | Files and directories to leave out, as on [`analyze`](analyze.md) |
| `--include-tests` | | `false` | Analyze test files and test code, as on [`analyze`](analyze.md) |

## Output

`check` writes everything to standard error. Each finding is one line that starts with the file path and line number:

```console
$ polyscan check src/
Running quality check (complexity, deadcode, deps)...
src/server.go:24: Server.Handle is too complex (14 > 10)
src/util.js:3: Code after return statement is unreachable
circular dependency: src/a.js -> src/b.js
Error: found 3 quality issue(s)
```

## Configuration

`check` finds `.polyscan.toml` the same way as `analyze`: the nearest file in the first path or a directory above it, or the file named with `--config`. The `[check]` section sets the default of each threshold flag: `max_complexity`, `max_cycles`, `allow_dead_code`, `allow_circular_deps` and `allow_parse_errors`. A flag you give on the command line takes precedence. The `complexity` and `analysis` sections apply as well. See the [reference](../configuration/reference.md#check).

```toml title=".polyscan.toml"
[check]
max_complexity = 15
max_cycles = 2
```

Any configuration error exits with code 2, because the check cannot give a verdict without its settings.
