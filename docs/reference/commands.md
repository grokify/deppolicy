# Commands

Every command's `--help` documents its full contract; this page is the map.

## Observing

### `scan <dir>...`

Scan one or more directory trees into a `graph.json`. Multiple roots combine into one graph with one shared component registry (so cross-root deep imports resolve correctly). Without `--policy`, runs in bootstrap mode: nothing excluded. Flags: `--policy`, `--output`, `--workers`.

### `scan-github <org>...`

Fleet scan via the GitHub contents API — one `go.mod` fetch per repository, no cloning; archived repos skipped. Produces `manifest`-level evidence. Requires `--token` or `GITHUB_TOKEN`. Flags: `--policy`, `--output`.

## Enforcing

### `check <dir>`

Scan a directory and evaluate against policy. Exits non-zero on any violation, with file:line evidence. Flags: `--policy` (required), `--output`, `--workers`.

### `diff <dir>...`

The ratchet: scan, evaluate, and classify each violation as **new** (absent from `--baseline`, fails) or **existing** (baselined debt, warns). Flags: `--policy`, `--baseline` (both required), `--output`, `--workers`.

### `policy-diff <old-policy.json> <new-policy.json>`

Review a policy change as an architecture change: added/removed relationships and component status/tier changes. With `--graph`, removals whose edges are still observed are flagged **BREAKING**. Exits zero unless `--fail-on-breaking`. Flags: `--graph`, `--output`, `--fail-on-breaking`.

### `validate`

Three-layer policy validation: JSON Schema (catches unknown fields Go's permissive unmarshal would ignore), structural semantics, and — with `--graph` — referential integrity plus expired-exception warnings. All issues reported in one pass. Flags: `--policy` (required), `--graph`.

## Authoring

### `promote`

Propose a `can_depend_on` relationship for every observed edge that currently violates default-deny, each with evidence and a placeholder reason that must be replaced by a reviewer. Writes a separate draft via `--output` — **never** modifies `--policy`. Flags: `--policy`, `--graph` (both required), `--output`.

## Reporting

### `report`

Ecosystem summary: component/edge totals, cross-org edges, fan-in leaders, rename-safety candidates. Flags: `--policy`, `--graph` (both required), `--output`, `--top`.

### `export`

Mermaid or Graphviz DOT diagram, grouped by org; with `--policy`, in-scope-only with violations highlighted. Flags: `--graph` (required), `--policy`, `--format`, `--output`.

### `tables`

Policy/Actual/Difference as four flat tables (JSON to stdout, or one CSV/JSON file per table via `--output-dir`). Flags: `--policy`, `--graph` (both required), `--format`, `--output-dir`.

### `ui`

Self-contained multi-page HTML report (summary, findings, components, interactive graph, tiers). Flags: `--policy`, `--graph`, `--output-dir` (all required), `--baseline`.

## Analyzer

`deppolicy-vet` is a separate binary — see [Editor and Vet Integration](../guides/analyzer.md).
