# Configuration

polyscan runs without any configuration. A configuration file lets you change the complexity thresholds, leave more files out, analyze test code, and set the limits of `polyscan check`. The same file configures every language: Go, Rust, C++ and JavaScript/TypeScript.

The file is named `.polyscan.toml` and is written in TOML. Write one by hand, or copy a [complete example](examples.md). The [reference](reference.md) documents every key.

```toml title=".polyscan.toml"
[analysis]
exclude = ["fixtures", "src/generated/**"]

[complexity]
low_threshold = 10
medium_threshold = 20

[check]
max_complexity = 15
```

Keys that the file leaves out keep their defaults, and a run with no file uses the defaults for everything.

## How polyscan finds your config file

polyscan starts at the first path you asked it to analyze, or at the directory of that path if it is a file. It checks that directory for a `.polyscan.toml`, then each parent directory in turn, up to the filesystem root. The first `.polyscan.toml` it finds is the one it reads.

Nothing else is searched. polyscan does not look in the current working directory, in `~/.config`, in your home directory, or in an environment variable.

Searching upward from the analyzed path rather than from the current directory means that analyzing `packages/api/src` from the repository root still picks up `packages/api/.polyscan.toml`.

### Naming the file with `--config`

`--config` (short form `-c`) on both `polyscan analyze` and `polyscan check` names the file to read and skips discovery:

```bash
polyscan analyze --config ci.polyscan.toml src/
polyscan check -c config/strict.toml src/
```

Any file name works, but the file must exist and must be TOML in the format described here.

## The file is strict

polyscan rejects a file that it cannot apply completely. Every key in the file takes effect, and a key that polyscan does not know is an error rather than a warning. This means that a misspelled key never silently does nothing.

An unknown key or section fails the run and names the key:

```console
$ polyscan analyze src/
Error: /work/app/.polyscan.toml: unknown key(s): analysis.exlude (see https://docs.codescan.dev/polyscan/configuration/)
```

A value of the wrong type, and a TOML syntax error, are reported with the position in the file in the form `path:line:column`. For example, `/work/app/.polyscan.toml:3:18:` starts the message for a problem on line 3, column 18.

A value that is the right type but out of range fails too, and the message names the key:

```console
$ polyscan analyze src/
Error: /work/app/.polyscan.toml: complexity.medium_threshold (5) must be greater than complexity.low_threshold (10)
```

The rules for each key are listed in the [reference](reference.md).

A configuration error ends `polyscan analyze` with exit code 1 and `polyscan check` with exit code 2, because a failed analysis is never a verdict about the code. See the [exit codes of `check`](../cli/check.md#exit-codes).

## Flags and the file

A flag given on the command line takes precedence over the file:

| Setting | Flag | How the flag combines with the file |
| --- | --- | --- |
| `analysis.exclude` | `--exclude` | The flag's patterns are added to the file's patterns, and both apply |
| `analysis.include_tests` | `--include-tests` | The flag replaces the file's value, but only when you give it |
| `check.max_complexity` | `--max-complexity` | The flag replaces the file's value when you give it |
| `check.max_cycles` | `--max-cycles` | The flag replaces the file's value when you give it |
| `check.allow_dead_code` | `--allow-dead-code` | The flag replaces the file's value when you give it |
| `check.allow_circular_deps` | `--allow-circular-deps` | The flag replaces the file's value when you give it |
| `check.allow_parse_errors` | `--allow-parse-errors` | The flag replaces the file's value when you give it |

The `[check]` section sets the defaults of the `polyscan check` flags. `polyscan analyze` reads that section but does not use it, so one file can serve both commands.

## Files that are skipped without configuration

Some files are left out before any pattern applies.

For Go, Rust and C++, polyscan skips any directory whose name starts with a dot, and directories named `node_modules`, `vendor`, `target`, `build`, `dist` and `third_party`. A path named on the command line is analyzed whatever it is called.

For JavaScript/TypeScript, polyscan skips a built-in list of dependency directories, build outputs, framework caches and minified files. The [reference](reference.md#built-in-skips) has the complete list.

The `exclude` key adds to these skips. It cannot remove one.

### The `.gitignore`

For JavaScript/TypeScript files, polyscan also reads the `.gitignore` in the directory you asked it to analyze and skips anything that file ignores. Two details are worth knowing:

- Only the `.gitignore` at the root of the analyzed path is read. Running `polyscan analyze src/` uses `src/.gitignore` and does not read the repository's top-level `.gitignore`. Running `polyscan analyze .` from the repository root does read it.
- Global and nested gitignore files are not consulted, and neither is `.git/info/exclude`.

Go, Rust and C++ files are not filtered by the `.gitignore`.

If a file you expected in the report is missing, check the built-in skips, your `.gitignore` and the patterns in `analysis.exclude`.

## Migrating from a jscan configuration file

polyscan no longer reads the configuration files that jscan read: `jscan.config.json`, `.jscanrc.json`, `jscan.yaml`, `jscan.yml`, `.jscan.toml`, `.jscan.yml`, `jscan.json` and `.jscan.json`. It also no longer reads the `JSCAN_CONFIG` environment variable.

Because settings that quietly stop applying would be hard to notice, polyscan fails the run instead. If discovery reaches a directory that has one of those files and no `.polyscan.toml`, the run stops with this error:

```console
$ polyscan analyze src/
Error: /work/app/jscan.config.json is no longer read: move its settings to .polyscan.toml in the same directory (see https://docs.codescan.dev/polyscan/configuration/)
```

A `.polyscan.toml` in a nearer directory is found first, so the old file is never reached and causes no error. Setting the `JSCAN_CONFIG` environment variable is also an error. Unset the variable and use `.polyscan.toml` or `--config`.

Files that belong to pyscn, such as `.pyscn.toml` and `pyproject.toml`, and the `PYSCN_CONFIG` variable are neither read nor an error. pyscn may analyze the Python code in the same repository, and those files are its own.

To migrate, create a `.polyscan.toml` in the directory that held the old file and carry over these settings:

| Old key in `jscan.config.json` | In `.polyscan.toml` |
| --- | --- |
| `complexity.low_threshold` | `complexity.low_threshold`, unchanged |
| `complexity.medium_threshold` | `complexity.medium_threshold`, unchanged |
| `analysis.exclude_patterns` | `analysis.exclude`. Only the entries beyond the built-in list are needed |
| `output.min_complexity` | No key. Use the `--min-complexity` flag of `polyscan analyze` |
| Every other key | Dropped. The key has no replacement |

For example, this `jscan.config.json`:

```json title="jscan.config.json"
{
  "complexity": { "low_threshold": 10, "medium_threshold": 20 },
  "analysis": {
    "exclude_patterns": ["node_modules", "dist", "coverage", "*.min.js", "legacy/generated"]
  },
  "output": { "min_complexity": 5 }
}
```

becomes this `.polyscan.toml`. Every exclude entry except `legacy/generated` is already in the built-in list for JavaScript/TypeScript, so only that entry remains:

```toml title=".polyscan.toml"
[complexity]
low_threshold = 10
medium_threshold = 20

[analysis]
exclude = ["legacy/generated"]
```

The `output.min_complexity` setting moves to the command line:

```bash
polyscan analyze --min-complexity 5 src/
```

The keys that jscan accepted and then ignored, such as `clones.*`, `output.format` and `system_analysis.*`, are gone. The keys that jscan applied only to JavaScript/TypeScript and that have no replacement are `analysis.include_patterns`, `analysis.recursive`, `output.sort_by`, `dead_code.min_severity`, `dead_code.sort_by` and `complexity.report_unchanged`.

## Next

- [Configuration reference](reference.md) documents every key, its type, and its default.
- [Configuration examples](examples.md) has complete files for several kinds of project.
