# FAQ

## Usage

### Which languages does polyscan analyze?

JavaScript, TypeScript, Go, Rust and C++, detected by file extension in one run. Complexity and clone detection cover every language; dependency analysis covers Go and JavaScript/TypeScript; dead code and coupling (CBO) exist for JavaScript/TypeScript. Python has its own analyzer, [pyscn](https://github.com/ludo-technologies/pyscn), built on the same core.

### What happened to jscan?

It merged into polyscan: `polyscan analyze` runs the full jscan analysis on JavaScript/TypeScript files. The `jscan` npm package remains published as a deprecated wrapper that runs polyscan. The [migration page](getting-started/migrating-from-jscan.md) maps every old command and flag to its replacement.

### Why did polyscan analyze far fewer files than my project has?

Check your exclude patterns first. A pattern in `analysis.exclude` or in `--exclude` that has no slash matches a whole file or directory name, so `dist` skips a directory named `dist` and leaves `src/utils/distance.ts` alone. A pattern that names a directory you did not mean to exclude removes every file under it, and nothing in the output names the missing files. Compare the `Analyzing N files...` line against a `find` count to confirm. The [configuration reference](configuration/reference.md#analysisexclude) describes the matching rules.

A second possibility is the built-in skips. Go, Rust and C++ skip every directory whose name starts with a dot, and directories named `node_modules`, `vendor`, `target`, `build`, `dist` and `third_party`. JavaScript/TypeScript skips a longer built-in list. The [reference](configuration/reference.md#built-in-skips) lists both. A path named on the command line is analyzed whatever it is called.

A third possibility is your `.gitignore`. polyscan reads the one at the root of the analyzed directory and skips whatever JavaScript/TypeScript files it ignores. The `.gitignore` does not apply to Go, Rust and C++ files.

### Why is every one of my exports reported as unused?

The unused-export check compares imports against exports across the files in the current run only. It has no way to see importers outside that set.

This happens in two situations. Analyzing a subdirectory, such as `src/components/`, means the importers in the rest of `src/` were never read. Pass the whole source root instead. Analyzing a library means the importers are in other repositories entirely, and there is nothing to do about it except gate on `critical_dead_code` rather than the full count.

### Does polyscan need my project to compile?

No. It parses the source with tree-sitter and never invokes `tsc`, `go build`, `cargo` or a C++ compiler. It works on code that currently fails to compile, and it does not read `tsconfig.json`. One consequence for C++: files are parsed without the preprocessor, so code whose syntax only makes sense after macro expansion is reported as a partial file and the functions containing the error are left out.

### Does polyscan read `tsconfig.json` path aliases?

No. An import written `@/lib/util` becomes a module node of its own rather than resolving to `src/lib/util.ts`. This inflates module counts in the dependency analysis, drops real edges, and hides cycles that pass through an alias. Relative imports resolve correctly. See [the TypeScript guide](guides/typescript-projects.md#path-aliases-are-not-resolved).

### Does polyscan support Vue single-file components?

Not yet. `.vue` files are not collected, so the `<script>` block inside them is never analyzed. The `.ts` and `.js` files in a Vue or Nuxt project are analyzed normally.

### Can I run polyscan on a single file?

Yes. `analyze` accepts files as well as directories:

```bash
polyscan analyze --select complexity src/server/router.ts
```

Bear in mind that cross-file analyses cannot say anything useful about one file. Dead code detection will report its exports as unused, and the dependency graph will have one node.

### How do I fail CI on the results?

Run `polyscan check`. It exits 1 when it finds quality issues and 2 when the analysis fails:

```bash
polyscan check src/
```

`analyze` always exits 0 when the analysis ran. To gate on the health score, use the JSON output with `jq -e`:

```bash
polyscan analyze --format json src/ 2>/dev/null | jq -e '.summary.health_score >= 75'
```

The [CI/CD page](integrations/ci-cd.md) has complete jobs.

### Why did my score go up when I changed nothing?

Check whether the `--select` value changed, or whether the analyzed language mix changed. The score is computed over the dimensions that ran, so a narrower selection is scored against a smaller budget and the numbers are not comparable. Always use the same selection on the same tree when tracking the score over time. See [the health score page](output/health-score.md#selecting-fewer-analyses-changes-the-budget).

## Configuration

### Which config file does polyscan use?

`.polyscan.toml`, for every language. polyscan checks the directory of the first path you asked it to analyze, then each parent directory up to the filesystem root, and reads the first `.polyscan.toml` it finds. It searches nowhere else: not the current directory, not your home directory, and not an environment variable. `--config` names a file explicitly and skips the search. See [config discovery](configuration/index.md#how-polyscan-finds-your-config-file).

### Is there a config file for Go, Rust or C++?

Yes. The same `.polyscan.toml` configures Go, Rust, C++ and JavaScript/TypeScript. It sets the complexity thresholds, the files to leave out, whether test code is analyzed, and the limits of `polyscan check`. The [reference](configuration/reference.md) lists every key.

### polyscan fails with "unknown key". What is wrong?

The file contains a key or section that polyscan does not have, and the error names it. polyscan rejects unknown keys so that a misspelled key never silently does nothing. Compare the name in the message with the [reference](configuration/reference.md).

### polyscan fails with "is no longer read". What do I do?

Discovery reached a jscan configuration file, such as `jscan.config.json`, in a directory that has no `.polyscan.toml`. Move its settings to a `.polyscan.toml` in the same directory. The [migration steps](configuration/index.md#migrating-from-a-jscan-configuration-file) show which keys carry over.

### Does adding to `exclude` stop polyscan from skipping `node_modules`?

No. `analysis.exclude` and `--exclude` add to the built-in skips and never replace them.

## Results

### What counts toward cyclomatic complexity?

Each decision point adds one to a baseline of 1. For JavaScript/TypeScript that is: `if`, `else if`, each loop, each `catch`, each `case` label, each ternary, and each `&&`, `||`, or `??`. A `default` clause adds nothing, the same as `else`, and neither does optional chaining with `?.`. A function with no branching scores 1.

Go, Rust and C++ count the equivalent constructs of each language — including `select` cases in Go, `match` arms and `?` in Rust, and `catch` in C++ — and closures count toward their enclosing function, so the Go numbers match gocyclo. The [analyze reference](cli/analyze.md#complexity) has the per-language tables.

### The CBO section says "classes" but my code has none. Why?

The coupling analysis produces one entry per file, named after the module. The label "classes" comes from the shared metric definition and applies literally in pyscn, which analyzes Python. Read every count in that section as a per-module count.

### What is a good health score?

Grade A starts at 90 and grade B at 75. In practice, treat the category scores as more actionable than the total, because each category saturates at a different point and the total hides which one is costing you.

Each category saturates at a different point. The complexity penalty maxes out once the weighted ratio of problematic functions reaches 30 percent, so a handful of high complexity functions cost a few points rather than the whole category. In a very small codebase the same handful is a large share of the population, so the category score moves faster there.

### Why is `architecture_score` always 0?

Architecture validation is not implemented in polyscan. The field exists because the scoring code is shared with pyscn. Ignore both `architecture_score` and `arch_enabled` in JSON output.

### Why does polyscan flag code my linter does not?

They answer different questions. A linter checks style and known bug patterns statement by statement. polyscan builds a control flow graph and an import graph, which lets it find unreachable code, duplication across files, and circular imports. Neither replaces the other, and running both is normal.

## Performance

### Which analysis is slowest?

Clone detection, followed by dead code detection. The analyses run in parallel, so total time is set by the slowest rather than by their sum. Dropping clone detection is the single most effective speedup:

```bash
polyscan analyze --select complexity,deadcode,cbo,deps src/
```

### The progress bar sits near the end for a long time. Is it stuck?

No. The bar is driven by an elapsed-time estimate rather than by real progress. When a run takes longer than estimated, the bar advances slowly toward 99 percent rather than stopping, so that it does not look frozen. Large repositories often reach that state.

## Project

### How does polyscan relate to pyscn?

They are the same analyses for different languages: pyscn for Python, polyscan for everything else it supports. The language-independent algorithms live in a shared `core` module in the polyscan repository, which both depend on. That is why grade thresholds mean the same thing across the two.

### Where does polyscan's source live?

In the [polyscan monorepo](https://github.com/ludo-technologies/polyscan), under `polyscan/`. jscan's code moved into that module as its JavaScript/TypeScript backend; the standalone `jscan` repository was retired earlier.

### Is the JSON output stable?

Not yet. Fields are more likely to be added than removed, but treat the structure as subject to change between releases. Pin the polyscan version in anything that parses it.

### How do I report a bug?

Open an issue in the [polyscan repository](https://github.com/ludo-technologies/polyscan/issues). Including the polyscan version from `polyscan version`, the command you ran, and a small reproduction makes it much faster to act on.
