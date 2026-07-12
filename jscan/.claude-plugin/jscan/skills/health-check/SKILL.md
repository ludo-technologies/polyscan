---
name: health-check
description: Get an overall JavaScript/TypeScript code quality health score using jscan. Use when user asks how healthy or good the code is, wants a quality overview, a grade, a summary of technical debt, or a before/after quality comparison.
---

# JavaScript/TypeScript Code Health Check with jscan

Run the jscan CLI to give a quick, quantified picture of code quality. No install needed:

```bash
npx jscan@latest analyze --text <path>
```

The output ends with a Health Score section: a score (0-100), a letter grade (A-F), and per-category scores.

## Commands

| User Request | Command |
|-------------|---------|
| "How healthy is this code?" | `npx jscan@latest analyze --text <path>` |
| "Give me a quality overview" | Same command; walk through the category breakdown |
| "Did my refactoring improve quality?" | Run before and after, compare scores |

For machine-readable detail use `--json`: full results go to stdout and the health-score summary is printed to stderr. **Without `--text` or `--json`, jscan writes an HTML report (`jscan-report.html`) and opens a browser** — prefer `--text` or `--json` when running as an agent.

## Interpreting Results

- Score 0-100 with letter grade; category scores cover complexity, dead code, code duplication, coupling (CBO), and dependencies.
- Lead with the grade and the weakest categories, then name the top offenders (files/functions) driving them.
- For deeper follow-up, hand off to the focused skills: refactoring targets → `refactoring`, module structure → `architecture-review`.

Always explain the score in plain terms and suggest the highest-impact next step.
