# CLI Reference

polyscan exposes three commands. `analyze` and `check` accept one or more paths, and every path may be a file or a directory.

| Command | Purpose | Fails the build? |
| --- | --- | --- |
| [`analyze`](analyze.md) | Run the full analysis and produce a report | No |
| [`check`](check.md) | Check the code against thresholds for CI | Yes |
| [`version`](version.md) | Print version information | No |

```console
$ polyscan --help
polyscan is a static analyzer that measures code quality across languages.
It currently analyzes cyclomatic complexity and code clones for Go, Rust and C++.

Usage:
  polyscan [command]

Available Commands:
  analyze     Analyze source files
  check       Check code quality against thresholds for CI
  completion  Generate the autocompletion script for the specified shell
  help        Help about any command
  version     Print version information

Flags:
  -h, --help      help for polyscan
  -v, --version   version for polyscan

Use "polyscan [command] --help" for more information about a command.
```

To fail a pipeline on the results, use [`check`](check.md). The [CI/CD page](../integrations/ci-cd.md) shows complete jobs. If you are looking for jscan's `deps` or `init` commands, see [migrating from jscan](../getting-started/migrating-from-jscan.md).

## Which files polyscan reads

`analyze` walks the directory tree and collects files by extension, dispatching each to its language:

| Language | Extensions |
| --- | --- |
| JavaScript | `.js`, `.jsx`, `.mjs`, `.cjs` |
| TypeScript | `.ts`, `.tsx`, `.mts`, `.cts` |
| Go | `.go` |
| Rust | `.rs` |
| C++ | `.cpp`, `.cc`, `.cxx`, `.hpp`, `.hh`, `.hxx`, `.h`, `.ipp`, `.inl` |

Header files, `.h` included, are analyzed as C++.

The JavaScript/TypeScript collection additionally honors the project's [configuration file](../configuration/index.md) and the `.gitignore` at the root of the analyzed path, exactly as jscan did. The other languages are collected by extension alone.
