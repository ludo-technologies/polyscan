---
name: polyscan-fp-audit
description: Run polyscan against one JavaScript/TypeScript, Go, Rust or C++ repository and triage findings for false positives using sub-agents. Auto-files clear polyscan bugs as GitHub issues (with `auto-filed` label, dedup, rate limit) and accumulates cross-repo tuning patterns into draft files. Outputs a markdown report under `.polyscan/audit/results/`. Accepts a repo URL/path as argument, or auto-picks the next pending entry from `.polyscan/audit/queue.md`.
---

# polyscan FP Audit Skill

Run polyscan (built from this checkout) against a target repository, then triage findings to surface likely false positives.

All paths below are relative to the monorepo root (the directory holding `core/`, `polyscan/`, `jscan/`). The CLI lives in `polyscan/`; the audit workspace is `.polyscan/audit/` at the root (gitignored).

## Input

- `$ARGUMENTS` may be:
  - A GitHub URL: `https://github.com/owner/repo`
  - A local path to an already-cloned repo
  - Empty: pick the next `- [ ]` entry from `.polyscan/audit/queue.md`. If the queue is empty or missing, invoke the **`polyscan-repo-discovery`** skill first to populate it.

## Steps

### 1. Build polyscan (skip if recent)

Skip the build if `polyscan/polyscan` exists, was modified within the last 24h **and** is newer than the current `HEAD`:

```bash
if [ -x polyscan/polyscan ] \
   && [ "$(find polyscan/polyscan -mtime -1 -print 2>/dev/null)" ] \
   && [ polyscan/polyscan -nt .git/HEAD ]; then
  echo "skip build: polyscan binary is fresh"
else
  make -C polyscan build
fi
```

Verify `polyscan/polyscan --version` runs. If the build fails, stop and report.

### 2. Resolve the target

Slug = `<owner>-<repo>`, lowercase, non-alnum → `-`.

- **URL** (preferred — tarball download, no `.git/`):
  ```bash
  mkdir -p .polyscan/audit/repos/<slug>
  curl -fsSL "https://codeload.github.com/<owner>/<repo>/tar.gz/HEAD" \
    | tar -xz -C .polyscan/audit/repos/<slug> --strip-components=1

  # Record the audited commit for the report
  SHA=$(curl -fsSL -H "Accept: application/vnd.github.sha" \
    "https://api.github.com/repos/<owner>/<repo>/commits/HEAD")
  ```
  If the directory already exists from a prior run, `rm -rf` it first (tarballs can't be incrementally updated).
  For private repos, swap to `gh api repos/<owner>/<repo>/tarball/HEAD > /tmp/<slug>.tar.gz` and extract.
- **Local path**: use as-is, slug = basename. Don't download or delete.
- **From queue**: read the first `- [ ]` line, extract URL, then proceed as the URL case.

Record the repo's **primary language** (from the queue line, or by counting source files by extension) — the report and issue titles carry it.

### 3. Run polyscan

Create `TS=$(date +%Y%m%d_%H%M%S)` and `OUT=.polyscan/audit/results/<slug>/$TS/raw`.

```bash
mkdir -p "$OUT"
# `--format json` streams the full report to stdout; the health summary goes to stderr.
polyscan/polyscan analyze --format json "<repo-path>" > "$OUT/analyze.json" 2> "$OUT/analyze.stderr" || true
```

Don't abort on non-zero exit. Capture stderr; if `analyze.json` is empty or not valid JSON (`jq -e . "$OUT/analyze.json"`), stop and report. `analyze.stderr` also carries per-analysis warnings such as `JavaScript clone analysis error: ...` — those are findings in their own right (see step 5, `clear-bug`).

The single `analyze.json` holds every analysis under these top-level keys:

| Key | Languages | Per-finding path |
|---|---|---|
| `complexity` | all | `.complexity.functions[]` → `name`, `file_path`, `language`, `start_line`, `end_line`, `metrics.complexity`, `risk_level` |
| `clone` | all | `.clone.clone_pairs[]` → `clone1`/`clone2` (`language`, `location.{file_path,start_line,end_line}`, `content`, `line_count`), `similarity`, `type`; `.clone.clone_groups[]` |
| `dead_code` | JS/TS only | `.dead_code.files[].functions[].findings[]` → `location`, `reason`, `severity`, `description` |
| `cbo` | Go, Rust, JS/TS | `.cbo.classes[]` → `name`, `file_path`, `language` (Go, Rust; absent for JS/TS), `start_line`, `metrics.coupling_count`, `metrics.dependent_classes`, `risk_level` |
| `lcom` | Go, Rust | `.lcom.classes[]` → `name`, `file_path`, `language`, `start_line`, `metrics.lcom4`, `metrics.method_groups`, `risk_level` |
| `deps` | Go, JS/TS | `.deps.analysis`, `.deps.graph` (Go nodes are packages keyed by import path) |
| `summary` | — | `health_score`, `grade`, `total_loc`, `total_files`, `skipped_files`, per-dimension scores |

To narrow scope, add `--select complexity,deadcode,clone,cbo,lcom,deps`.

**Known structural zeros — never report these as bugs.** For Go/Rust/C++ the generic engine fills only `metrics.complexity` and `metrics.nesting_depth`; `nodes`, `edges`, `if_statements`, `loop_statements`, `exception_handlers` and `switch_cases` are always `0`. Clone fragments always have `hash: ""`, `complexity: 0` and `start_col`/`end_col` `0`. `cbo.classes[]` for JS lists one pseudo-class per file (module-level coupling), so a `name` equal to the file basename is expected. For Go and Rust a `cbo` entry is one type, and `dependent_classes` only ever names types declared in the analyzed tree: standard library, third-party and other-module types are absent by design, a type coupled to nothing is not listed, and a Go tree without `go.mod` lists same-package dependencies only (a warning says so). For Go `deps.analysis.circular_dependencies` is always empty (the compiler forbids import cycles), Go edges carry no `location`, and Go files outside any `go.mod`, `_test.go` files, and `vendor`/`testdata` directories are absent from the graph by design. A Go type in `lcom.classes[]` is measured over the methods of its whole package and placed in the first file that declares one, so `file_path` need not be the file that declares the type; `excluded_methods` counts methods without a receiver parameter, which is expected for Rust `new`-style associated functions, and stub methods that touch no field and take part in no sibling call, such as interface-mandated constant returns and `panic` placeholders. For Go, Rust and C++, `risk_level` is derived from the complexity with flat dispatch collapsed, so a `switch` or `match` that holds no decision point beyond its own arms counts once rather than once per arm; a key handler reported as `complexity: 14` with `risk_level: low` is that rule working, not a mismatch. The short-circuit operators of one returned expression collapse the same way, so a validation predicate reported as `complexity: 11` with `risk_level: low` is that rule working too.

### 4. Cluster findings

Read the JSON output. Group findings by `(analysis, language, rule_or_pattern)`. Examples:

- `complexity / go / cyclomatic >= 20`
- `clone / rust / type-2-similarity-high`
- `clone / cpp / cross-file-header-impl`
- `deadcode / ts / unreachable_after_return`
- `cbo / ts / coupling >= 10`
- `cbo / go / coupling >= 8`
- `lcom / go / lcom4 >= 6`

For each cluster, record the total count and pick **3–5 representative samples** (prefer variety: different files, different sizes, mix of risk levels). Also collect:

- `summary.skipped_files` and the `analyze.stderr` warnings — a skipped valid source file or a crashed analysis is its own cluster (`engine / <lang> / skipped-file`).
- **Score decomposition**: the per-dimension scores in `summary` and the penalty each implies.
  **The health score is not `100 - Σ penalties`.** `core/domain/scoring.go` exports
  `HealthScoreFromPenalties`, but the analyze path does not use it: `CalculateHealthScore`
  (`polyscan/internal/js/domain/analyze.go`) charges each *enabled* dimension's raw penalty against
  its own budget and calls `healthScoreFromPenaltyBudget`, which is a **ratio**:

  ```
  score = 100 - round(Σ penalty × 100 / Σ budget) - parseErrorPenalty
  ```

  Budgets are `MaxScoreBase` = 20 per dimension except dependencies, which is
  `MaxDependencyPenalty` = 16. To recover a dimension's raw penalty from its published score, invert
  `PenaltyToScore`: `penalty = (100 - score) / 5` for the 20-budget dimensions; for dependencies that
  gives the *normalized* penalty, so multiply by 16/20 for the raw one. A dimension's share of the
  loss is then `penalty / Σ budget × 100` points — the shares do **not** equal `100 - score`.

  `parseErrorPenalty` is subtracted afterwards and **unscaled**, and it has a floor:
  `MinParseErrorPenalty = 100 - GradeAThreshold + 1` = **11 points for even one unparsable file**
  (the intent being that a run with parse errors cannot grade A). On a large repo this routinely
  outweighs every real dimension, so check it before attributing a bad grade to code quality.

  Record the 2–3 contributors holding most of the loss, and reconcile your arithmetic against the
  reported `health_score` — if it does not land on the reported number exactly, your penalty
  derivation is wrong, not the tool's. Then sanity-check the grade against the findings: if it looks
  harsher or kinder than the findings justify, that gap is a lead for step 5b. A wrong *number* is as
  much a defect as a wrong *finding*, and triaging individual findings will never surface it.
- Findings whose `file_path` lies under `vendor/`, `third_party/`, `node_modules/`, `target/`, `build/`, `dist/`, `_deps/`, or matches generated-file markers. The generic (Go/Rust/C++) collector has **no directory exclusions**, so these are a known `tuning` pattern (`generic-scans-vendored-dirs`); count them, sample 1–2, and don't let them crowd out the clusters that matter.

### 5. Triage in parallel

Spawn one **`Explore` Agent per cluster**, all in a single message so they run concurrently.

If an agent stalls (no result after a few minutes, or a wrap-up request goes unanswered), `TaskStop`
it and triage that cluster yourself from the JSON. A cluster is a single analysis pass; it doesn't
need a sub-agent to be correct, and a stalled fan-out must not cost the audit its findings.

Brief each agent:

> "Triage polyscan findings of type **`<analysis> / <language> / <pattern>`** in repo `<repo-path>`.
>
> Findings to evaluate (JSON):
> ```json
> <cluster sample, including file path, line range, metric values, and clone content if present>
> ```
>
> For each finding:
> 1. Read the cited file/lines plus enough surrounding context (callers, enclosing type, tests, build files) to judge intent
> 2. Decide a verdict:
>    - `TP` — polyscan is correct, this is a real issue
>    - `FP` — polyscan is wrong. Language-specific FP patterns to check:
>      - **Go**: `// Code generated ... DO NOT EDIT.` files; table-driven tests; `if err != nil { return }` chains inflating complexity; `switch` over enum-like constants; `_test.go` clones that mirror a fixture; cgo shims
>      - **Rust**: `macro_rules!`/derive-expanded code; trait impls that must be repeated per type (`impl From<X> for Y`); exhaustive `match` on a large enum; `#[cfg(...)]` variants of the same function; `tests/` fixtures
>      - **C++**: the same declaration in a header and its definition in a `.cpp`; template specializations; `switch` dispatch on opcodes; `.h` files in `include/` that also exist in `src/`; generated protobuf/flatbuffers code
>      - **JS/TS**: `.d.ts` declarations; bundled/minified output that escaped exclusion; `switch` cases with `return` before `break` flagged as dead; overloads; generated GraphQL/OpenAPI clients; test snapshots
>      - **Any language**: two files that are the same intentional boilerplate (CLI entrypoints, example programs); a fixture directory of deliberately duplicated samples
>    - `unsure` — needs human judgment
> 3. Write 1–2 sentences explaining *why*, citing specific code constructs
> 4. Also classify `bug_class` for downstream auto-filing:
>    - `clear-bug` — structural anomaly in polyscan's output that any maintainer would call a bug. Examples: line range is `0-0` or `end_line < start_line`; `metrics.complexity` is negative, `0` for a non-empty function, or off by orders of magnitude; reported function/class name doesn't exist at that location; `language` field doesn't match the file extension; polyscan crashed on, or silently skipped, a valid source file (see `analyze.stderr` and `summary.skipped_files`); a clone pair where `clone1` and `clone2` are the same location; the same finding duplicated. **Narrow category — only pick this when you can point at the exact JSON field that is wrong.** The structural zeros listed in the brief are NOT bugs.
>    - `tuning` — polyscan works as designed but the heuristic misfires on a recognizable, generalizable pattern (e.g. generated code not excluded, header/impl duplicates counted as clones, error-handling chains counted as complexity). Most FPs land here.
>    - `none` — TP, or an FP too repo-specific to generalize
> 5. Assign a `pattern_slug` (kebab-case, generic — `go-generated-code-not-excluded`, `cpp-header-impl-clone`, not `foo-repo-bar-file`), and for `clear-bug` a priority:
>    - `P0` — crash, or output that is wrong for every file of a language
>    - `P1` — wrong metric or location on a common construct
>    - `P2` — wrong on an uncommon construct
>    - `P3` — cosmetic (naming, ordering)
>    and `good_first_issue: true` when the fix is plausibly local to one function.
>
> Return as JSON array:
> ```json
> [
>   {
>     "id": "<file:line>",
>     "verdict": "TP|FP|unsure",
>     "reason": "...",
>     "evidence": "<key code snippet or construct name>",
>     "bug_class": "clear-bug|tuning|none",
>     "pattern_slug": "<kebab-case>",
>     "priority": "P0|P1|P2|P3",
>     "good_first_issue": true
>   }
> ]
> ```
>
> Do not modify any files. Read-only triage."

### 5b. Score integrity (differential runs)

Step 5 asks "did polyscan flag the right *code*?" over one frozen `analyze.json`. This step asks the
orthogonal question — **"is the reported number itself right?"** — and it is the one step that
*re-runs* polyscan. Run it yourself, not in a read-only agent; a run takes well under a second.

Change **one knob at a time** and diff `summary`:

```bash
cd <repo-path-or-any-target>
for MC in 1 25; do
  polyscan/polyscan analyze --format json --min-complexity $MC . > /tmp/ps_$MC.json 2>/dev/null
  python3 -c "
import json; d=json.load(open('/tmp/ps_$MC.json')); s=d.get('summary',{})
print('$MC', s.get('health_score'), s.get('grade'), s.get('complexity_score'),
      (d.get('complexity') or {}).get('summary',{}).get('total_functions'))"
done
```

The invariant: **a presentation-only option must not move any score.** `--min-complexity` selects
what gets *listed*; it must leave `health_score`, the per-dimension scores and the summary
population counts untouched. (Verified holding as of this writing — `--min-complexity 1` vs `25`
both give health 75 / 24 functions on `polyscan/testdata`. Re-check it rather than assuming.)

Also run, and interpret carefully:

- **`--select`**: per-dimension scores must stay identical to the full run. `health_score` itself
  legitimately moves, and — because of the ratio in step 4 — it moves in **either** direction: a
  dimension that runs alone is judged at full weight instead of being averaged against healthier
  ones. Selecting *fewer* dimensions can therefore *lower* the score, which is the opposite of what
  "fewer penalties" suggests. On `rogerpadilla/uql` the full run scored 57/D while `--select clone`
  and `--select deps` each scored 39/F, and `--select complexity` scored 84/B — an 18-point spread
  either way from the same tree. None of that is a bug; it is documented at
  `polyscan/internal/js/service/doc.go` ("summaries built from different `--select` values are not
  comparable"). Do not file it. Do record which direction it moved for your target, since a CI gate
  using `--select` is graded on a different curve and usually a harsher one.
- **Population consistency across analyses**: two analyses over the same unit should agree on how
  many units exist. LCOM counting far more classes than CBO on the same tree is a lead, not a
  curiosity.
- **Known suspect — CBO filter-then-summarize.** `polyscan/internal/js/service/cbo_service.go:112-115`
  filters classes (`MinCBO`/`MaxCBO`/`ShowZeros`) and *then* calls `SummarizeCoupling(sortedClasses, …)`,
  whose `TotalClasses` (`:239`) becomes `summary.CBOClasses`
  (`polyscan/internal/js/service/output_formatter.go:332`) and thus the coupling-penalty denominator.
  This is the same shape as pyscn issue #785, where dropping zero-coupling classes from the
  denominator moved the health grade by two steps. **Checked on `rogerpadilla/uql` (2026-09-17) and
  it did not fire in the default `analyze` path**: `cbo.classes` held 280 entries against
  `analyzed_files` 280, with 47 of them at `coupling_count: 0` — the zero-coupling classes are
  present, so the denominator was complete, and `coupling_count == len(dependent_classes)` for all
  280. Cheap to re-check (two `jq` counts), so confirm it per target rather than assuming either way;
  what remains unexercised is the config path that sets `ShowZeros`/`MinCBO`, since there is no
  `--show-zeros` analyze flag. A focused Go test over `CBOServiceImpl` is still the way to settle it.

A score-integrity defect is `bug_class: "clear-bug"` — it is a structural anomaly in polyscan's output
with an exact wrong field — at `P0` if it affects every target, `P1` otherwise. Evidence is the
**before/after table** plus a minimal repro. These reproduce deterministically, so they verify
unusually well; don't discount them for lacking a per-finding `file:line`.

### 6. Auto-file `clear-bug` findings (with dedup + rate limit)

For each finding with `bug_class == "clear-bug"`:

1. **Dedup** against existing auto-filed issues:
   ```bash
   gh issue list -R ludo-technologies/polyscan -s all -L 50 \
     --label auto-filed --search "<pattern_slug>" --json number,title
   ```
   If any result mentions the same `pattern_slug`, **skip filing** — note "deduped against #<n>" in the report.

   Then run a second, wider pass over **all** issues, auto-filed or not, searching the construct
   rather than the slug — a maintainer's issue will not be named the way you named your pattern:

   ```bash
   gh issue list -R ludo-technologies/polyscan -s all -L 100 --json number,title,state \
     -q '.[] | select(.title|test("<keyword1>|<keyword2>";"i")) | "\(.number)\t\(.state)\t\(.title)"'
   ```

   The `auto-filed` label is the kill-switch for *rollback*, not a licence to ignore human issues.
   When an **open** manually-filed issue already covers the same root cause, do not open a second
   one: add your construct, repro and impact as a **comment** on it, and record that in the report
   under "Issues filed". Splitting one root cause across two issues is the expensive mistake this
   skill is meant to avoid. File separately only when the mechanism is genuinely different — say so
   in one line if it is a near miss. A **closed** issue that matches is a different signal: verify
   whether it actually regressed before filing anything, since the symptom may now have a new cause
   (on `rogerpadilla/uql`, dead-code findings looked exactly like the closed #126 and were in fact
   two unrelated live bugs).

2. **Verify claims against raw JSON.** Before drafting the issue body, for every specific metric value, field name, or count you're about to cite in "Actual Output", extract it directly from `raw/analyze.json` for that exact finding (via `jq`) and use that literal value — do not restate the triage sub-agent's prose `evidence` field if it paraphrases a number. If the value you were about to cite doesn't match what's actually in `raw/analyze.json`, the finding is not a real `clear-bug`: downgrade it to `unsure`/`none` and do not file.

3. **Rate limit** (skip if either is exceeded):
   - **Per-audit cap**: 2 auto-filed issues per audit run. If exceeded, file the rest as drafts only.
   - **Per-day cap**: 5 auto-filed issues per UTC day. Read `.polyscan/audit/issues.jsonl`, count entries where `ts` starts with today's date.

4. **File the issue**:
   ```bash
   gh issue create -R ludo-technologies/polyscan \
     --title "[BUG][auto] <lang>/<pattern_slug>: <one-line description>" \
     --label bug --label auto-filed --label "<priority>" \
     $( [ "<good_first_issue>" = "true" ] && printf -- '--label "good first issue"' ) \
     --body-file <draft-path>
   ```
   `<lang>` is `go`, `rust`, `cpp`, `js` or `ts`. `<priority>` is the `P0`/`P1`/`P2`/`P3` label from step 5. All labels are assumed to already exist in the repo — if `gh issue create` fails with "label not found", stop and surface the error rather than auto-creating.

   Body should include: bug description, repro steps (a minimal synthetic source file in the affected language if possible), expected behavior, actual output (cite the JSON snippet), polyscan version (`polyscan/polyscan --version`), priority rationale (one line, citing the rubric in step 5), and "Found via the FP-audit skill in repo `<owner/repo>@<sha>`". Keep it short; no test plan.

5. **Log it** by appending to `.polyscan/audit/issues.jsonl`:
   ```json
   {"ts": "2026-09-05T19:30:00Z", "slug": "<owner-repo>", "lang": "go", "pattern_slug": "...", "issue_url": "https://github.com/.../issues/N"}
   ```

### 6b. Append `tuning` findings to draft files

For each unique `pattern_slug` with `bug_class == "tuning"`:

1. Path: `.polyscan/audit/issues/draft-<pattern_slug>.md`
2. If the draft file doesn't exist, create it with this skeleton:
   ```markdown
   # [tuning draft] <pattern_slug>

   <one-paragraph explanation of the FP pattern, written generically; name the language(s) it applies to>

   ## Repos hitting this

   <!-- audit log appended below; one entry per repo -->
   ```
3. **Append** a new section per repo (don't duplicate if the repo+pattern already appears):
   ```markdown
   ### <owner/repo>@<sha-short> (<lang>) — audited 2026-09-05
   - **Findings**: `<file>:<line>` (<verdict>) — <reason>
   - **Evidence**: <snippet>
   ```
4. These drafts are **never auto-filed**. The user reviews them periodically (e.g., once 3+ repos hit the same pattern, they decide whether to file).

### 7. Aggregate the report

Write `.polyscan/audit/results/<slug>/$TS/report.md`:

```markdown
# polyscan FP Audit — <owner/repo>

- **Repo**: <url-or-path> (commit `<sha>`)
- **Language**: <primary language> (<file counts per language from the report>)
- **polyscan**: `<version>` (commit `<polyscan-sha>`)
- **Date**: 2026-09-05
- **LOC analyzed**: <summary.total_loc> across <summary.total_files> files (<summary.skipped_files> skipped)
- **Health**: <summary.health_score> (<summary.grade>) — penalty breakdown: <dimension> <n>pts, … ;
  note any dimension whose points are an artifact rather than real debt
- **Score-integrity differentials**: <knobs tested> — invariant held / violated (<detail>)

## Summary

| Cluster | Total | Sampled | TP | FP | Unsure | FP rate (sample) |
|---|---|---|---|---|---|---|
| clone / go / type-2-similarity-high | 42 | 5 | 4 | 1 | 0 | 20% |
| ...

## Issues filed

- `<pattern_slug>` [P1] → #N — <one-line>
- `<pattern_slug>` [P2] → deduped against #M (skipped)
- `<pattern_slug>` [P0] → rate-limited (drafted to `.polyscan/audit/issues/draft-<slug>.md`)

## Tuning drafts updated

- `.polyscan/audit/issues/draft-<pattern_slug>.md` (now N repos)

## Notable false positives

### `<analysis> / <language> / <pattern>` — `<file>:<line>`
**Verdict**: FP
**Why**: <reason from sub-agent>
**Evidence**:
```<lang>
<minimal snippet>
```

(repeat for each FP / interesting unsure)

## Notable true positives (sanity check)

(1–2 confirmed TPs to verify the tool is working)

## Raw

- `raw/analyze.json`
- `raw/analyze.stderr`
```

### 8. Update the queue

If the target came from `queue.md`:
- Flip `- [ ]` to `- [x]`, append `— audited 2026-09-05` with a relative link to the report
- Move the line from `## Pending` to `## Audited`

### 9. Clean up the clone

If the target was downloaded (not a local path), delete the source tree to save disk:

```bash
rm -rf .polyscan/audit/repos/<slug>
```

The `raw/analyze.json` and `report.md` already contain everything needed for retrospective analysis — the source can be re-fetched if needed.

### 10. Report to user

One paragraph: language, total findings, suspected FP rate per cluster, **# of issues filed (with URLs) and # of tuning drafts touched**, link to the report file. Don't paste the full report inline.

## Notes

- All work stays under `.polyscan/audit/` (gitignored) **except** auto-filed issues, which are public on GitHub. Treat the `auto-filed` label as the kill-switch: a single `gh issue list -R ludo-technologies/polyscan -l auto-filed --json number -q '.[].number' | xargs -n1 gh issue close -R ludo-technologies/polyscan` can roll everything back if the heuristics drift.
- Sub-agents are read-only on the local filesystem — they investigate and judge, they don't modify code. Step 5b is the exception and is run by you, since it re-runs polyscan under varied config.
- **"Works as designed" is not "correct".** Classifying a finding `none` because the behaviour follows from a documented default explains the *behaviour*, not its consequences — ask once more where that value flows. pyscn #785 was missed on a first pass by stopping at "CBO lists 56 of 124 classes because `show_zeros` defaults to false"; the filtered 56 was also the scoring denominator. Before dropping such a lead, grep the value's consumers.
- The `clear-bug` category is intentionally narrow. When in doubt, prefer `tuning` (which only drafts, never auto-files) — false issues are far more costly than missed ones.
- If a cluster is huge (>200 findings), sample 5 by default — the rate from a sample is usually what
  matters. The exception: when the verdict can be decided **mechanically** from the finding plus its
  own source file, check the whole cluster with a script instead. A sample of 5 yields "probably
  mostly wrong"; a scripted pass over all of them yields "91 of 91 are false", which is a far
  stronger claim in an issue and takes about as long to write. `unused_import` is the worked example
  — extract `(file_path, imported name)` from the JSON, grep each name in its own file outside the
  import lines, and bucket the hits by construct.
- Python repos are out of scope: polyscan has no Python analyzer; use `pyscn-fp-audit` in the pyscn checkout for those.
- Avoid running on this monorepo (`polyscan` itself) — that's not the audit target. `polyscan/testdata/` is a fixture tree full of intentional clones and dead code.
