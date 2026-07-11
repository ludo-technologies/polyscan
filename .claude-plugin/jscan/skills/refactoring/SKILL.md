---
name: refactoring
description: Find refactoring targets in JavaScript/TypeScript code using jscan - duplicate code (clones), overly complex functions, and dead code. Use when user asks about refactoring, code duplication, complexity hotspots, unreachable code, or cleaning up a codebase.
---

# JavaScript/TypeScript Refactoring Analysis with jscan

Run the jscan CLI to locate concrete refactoring targets. No install needed: `npx jscan@latest <command>`.

## Commands

| User Request | Command |
|-------------|---------|
| "Find complex functions" | `npx jscan@latest analyze --text --select complexity <path>` |
| "Find duplicate code" | `npx jscan@latest analyze --text --select clone <path>` |
| "Find dead code" | `npx jscan@latest analyze --text --select deadcode <path>` |
| "What should I refactor first?" | `npx jscan@latest analyze --text --select complexity,deadcode,clone <path>` |

Use `--json` instead of `--text` for line-level findings in machine-readable form (written to stdout). Without either flag, jscan writes an HTML report and opens a browser.

## Interpreting Results

- Complexity risk levels (defaults, configurable in `jscan.config.json`): Low (≤9), Medium (10-19), High (20+). Complexity counts branches plus logical operators (`&&`, `||`, `??`) and ternaries.
- Dead code severity: critical means code after return/break/continue/throw that can never execute; warning covers unreachable branches, unused imports, and orphan files; info covers unused exports.
- Clone types: Type-1 (identical), Type-2 (renamed identifiers/literals), Type-3 (modified, disabled by default), Type-4 (functionally similar), each with a similarity score.

## Prioritizing Findings

1. Critical dead code: safe deletions, do these first.
2. High-complexity functions (20+): extract functions, flatten conditionals.
3. Clone groups spanning multiple files: extract shared helpers; clones within one file are usually quicker wins.

When suggesting a refactor, cite the specific function names, files, and line ranges from the results, and re-run the same command afterward to confirm the improvement.
