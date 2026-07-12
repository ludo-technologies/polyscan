---
name: architecture-review
description: Analyze JavaScript/TypeScript module architecture using jscan - class coupling (CBO), instability metrics, and circular dependency detection. Use when user asks about architecture, module structure, coupling, or circular dependencies.
---

# JavaScript/TypeScript Architecture Review with jscan

Run the jscan CLI to understand module structure and coupling. No install needed: `npx jscan@latest <command>`.

## Commands

| User Request | Command |
|-------------|---------|
| "Check class coupling" | `npx jscan@latest analyze --text --select cbo <path>` |
| "Find circular dependencies" | `npx jscan@latest analyze --text --select deps <path>` |
| "Map the module dependencies" | `npx jscan@latest deps <path>` |
| "Which modules are risky to change?" | `npx jscan@latest deps --format json <path>` |
| "Draw the dependency graph" | `npx jscan@latest deps --format dot <path> \| dot -Tsvg -o deps.svg` |

`jscan deps` flags: `--format text|json|dot`, `--include-external` (include npm packages), `--no-cycles` (skip cycle detection), `--min-coupling <n>` (filter low-coupling modules in DOT output), `--max-depth <n>`.

Text output shows per-class CBO and cycle membership, but only **aggregate** coupling stats for modules. Per-module Martin metrics live in `deps --format json`: `analysis.ModuleMetrics` (Ca/Ce, instability, abstractness, distance, risk level per module) and `analysis.CouplingAnalysis` (explicit `ZoneOfPain` / `MainSequence` module lists).

## Interpreting Coupling Results

- High CBO classes depend on many others; changes ripple widely. Suggest interface extraction or dependency inversion.
- Martin metrics per module (from the JSON output above): instability I = Ce / (Ca + Ce) and distance from the main sequence. Modules in the Zone of Pain (stable but concrete) are risky to change; name them explicitly.
- Dependency cycles are the highest-priority architectural issue; name the modules in each cycle and the weakest edge to break.

Always tie findings back to concrete modules and suggest a specific structural change.
