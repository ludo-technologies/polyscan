# Configuration Examples

Complete `.polyscan.toml` files for common project shapes. Each file is valid as written and applies to every language in the project: Go, Rust, C++ and JavaScript/TypeScript.

Remember three rules while reading these:

- `analysis.exclude` adds to the built-in skips and never replaces them. Dependency directories such as `node_modules` and `vendor`, build output such as `dist` and `build`, and minified files are already skipped, so you only list what is specific to your project. The [reference](reference.md#built-in-skips) has the full list.
- An entry without a slash matches a whole file or directory name anywhere on the path, so `fixtures` skips every directory named `fixtures` and leaves `src/fixtures_loader.ts` alone. An entry with a slash matches a path relative to the analyzed directory. The [reference](reference.md#analysisexclude) explains the matching rules in full.
- A flag on the command line takes precedence over the file.

## Starting point for any project

The smallest file worth writing. It sets complexity thresholds a little more forgiving than the built-in defaults of 9 and 19.

```toml title=".polyscan.toml"
[complexity]
low_threshold = 10
medium_threshold = 20
```

Run it against your source directory rather than the repository root, so that configuration files and scripts stay out of the analysis without needing a pattern for them:

```bash
polyscan analyze src/
```

## React or Next.js application

Next.js projects keep generated output in `.next`, which is already in the built-in list for JavaScript/TypeScript, and often have a `src/app` or `src/pages` tree full of route files.

```toml title=".polyscan.toml"
[complexity]
low_threshold = 12
medium_threshold = 24

[analysis]
exclude = ["storybook-static", "src/generated/**"]
```

The thresholds are raised because component code accumulates conditional rendering, which counts toward cyclomatic complexity without being genuinely hard to read. To hide the trivial components so that the report is about the parts worth looking at, pass `--min-complexity 3` to `polyscan analyze`.

Next.js reserves several export names that nothing in your code imports. polyscan recognizes them and does not report them as unused, but only inside App Router convention files, meaning a file under a path containing `/app/` and named `page`, `layout`, `template`, `loading`, `error`, `not-found`, `default`, or `route`. In those files the default export is exempt, along with `metadata`, `generateMetadata`, `viewport`, `generateViewport`, `generateStaticParams`, `dynamic`, `dynamicParams`, `revalidate`, `fetchCache`, `runtime`, `preferredRegion`, and `maxDuration`. In `route` files the HTTP verb exports such as `GET` and `POST` are exempt as well.

## Node.js backend service

Backend code is a better fit for stricter thresholds. The `[check]` section sets the limits of `polyscan check`, so a CI job can run the command without repeating them as flags.

```toml title=".polyscan.toml"
[complexity]
low_threshold = 8
medium_threshold = 15

[check]
max_complexity = 20
```

With this file in place, the following command fails when a function has a complexity above 20:

```bash
polyscan check --select complexity src/
```

## Library or published package

A library's public exports are consumed by other repositories, so polyscan will always report them as unused. A gate has to ignore the warning-level findings, which makes the configuration file itself fairly plain.

```toml title=".polyscan.toml"
[complexity]
low_threshold = 8
medium_threshold = 16
```

```bash
# check fails only on critical dead code; the unused-export warnings are expected here
polyscan check --select deadcode src/
```

You still get value from the full dead code analysis in the report, where the critical findings, which are genuinely unreachable statements, are worth acting on even though the warnings are not.

## Go, Rust or C++ project

The same file works for the other languages. This example leaves out generated code and a directory of test fixtures, and it analyzes test code too.

```toml title=".polyscan.toml"
[analysis]
exclude = ["testdata", "internal/gen/**", "*.pb.go"]
include_tests = true

[check]
max_complexity = 15
max_cycles = 0
```

`vendor`, `target`, `build`, `dist`, `third_party` and every directory whose name starts with a dot are already skipped for these languages.

## Monorepo

There is no workspace-aware mode. Run polyscan once per package, and give each package its own file so that thresholds can differ between a strict core library and a looser internal tool.

```text
repo/
├── .polyscan.toml          ← used by packages without their own
└── packages/
    ├── core/
    │   ├── .polyscan.toml  ← stricter
    │   └── src/
    └── web/
        ├── .polyscan.toml  ← looser
        └── src/
```

Because discovery walks upward from the analyzed path, `polyscan analyze packages/core/src` finds `packages/core/.polyscan.toml` first and uses the repository root file only when the package has none. The nearest file wins, and files are never merged. A package file therefore has to repeat any setting from the root file that the package still needs.

```bash
# Analyze each package separately
for pkg in packages/*/; do
  echo "== $pkg"
  polyscan analyze --format json "$pkg/src" 2>/dev/null \
    | jq -e '.summary.health_score >= 75' || exit 1
done
```

Analyzing packages separately has one consequence worth understanding. The unused-export check can only see the files in the current run, so anything `packages/web` imports from `packages/core` is reported as an unused export while `core` is analyzed alone. Run `polyscan analyze packages/` to see the whole picture, and the per-package runs to gate each package. The run on `packages/` finds the root file, not the package files.

## Legacy codebase you are improving gradually

When the current state is far from where you want it, set thresholds you can actually pass today and tighten them over time.

```toml title=".polyscan.toml"
[complexity]
low_threshold = 20
medium_threshold = 40

[analysis]
exclude = ["legacy/generated"]

[check]
max_complexity = 40
```

Pass `--min-complexity 15` to `polyscan analyze` to keep the report focused on the worst functions rather than producing thousands of lines nobody reads. Ratchet the thresholds down every time the numbers improve comfortably, and the report will pull the codebase in the right direction without ever blocking work.

Note that `legacy/generated` contains a slash, so it is matched against the path rather than against a single name. It skips that directory and everything under it, and it matches nothing else.

## A different file for one job

`--config` names a file explicitly and skips discovery, so a CI job can use stricter settings than the file in the repository. Any file name works.

```toml title="ci.polyscan.toml"
[complexity]
low_threshold = 8
medium_threshold = 15

[check]
max_complexity = 12
```

```bash
polyscan check --config ci.polyscan.toml src/
```

## See also

- [Configuration reference](reference.md) for every key and its validation rules
- [CI/CD integration](../integrations/ci-cd.md) for using these files in a pipeline
