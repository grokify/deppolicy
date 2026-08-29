# Reports and Visualization

All report commands read previously scanned artifacts — none scan live, so you can iterate on views of one saved graph without re-paying the scan.

## Multi-page HTML report

```bash
deppolicy ui --policy policy.json --graph graph.json --baseline baseline-graph.json --output-dir report/
```

Five pages, each a single self-contained file — inline CSS/JS, data embedded as JSON, no server, no CDN — so the report attaches to CI runs and opens from disk:

| Page | Contents |
|---|---|
| `index.html` | Summary tiles (conforming / baselined debt / new violations), components by org, foundation list, tier counts |
| `findings.html` | Filterable findings: status filter (including "new violations"), text search, ratchet labels, file:line evidence |
| `components.html` | Component inventory with fan-in/fan-out, filterable by org/tier/status, sortable |
| `graph.html` | Interactive dependency graph: deterministic org-grouped layout (readable and stable across regenerations, not a force-directed hairball), pan/zoom, click-to-isolate a node's edges, violations-only toggle |
| `tiers.html` | Every tier's allow/deny rules and members |

With `--baseline`, violations split into *new* versus *baselined debt*, exactly as `diff` classifies them. The report is strictly read-only by design — policy edits flow through `promote` / `policy-diff` review, never a save button.

## Diagrams

```bash
deppolicy export --graph graph.json --policy policy.json --format mermaid   # or: dot
```

Components grouped into per-org subgraphs/clusters; with `--policy`, only in-scope components render and violating edges are highlighted red. DOT output feeds Graphviz for large-format rendering; Mermaid embeds in markdown.

## Ecosystem summary

```bash
deppolicy report --policy policy.json --graph graph.json --top 20
```

Console/JSON summary: totals, cross-org edge count, fan-in leaders (foundation-tier candidates), and rename-safety candidates — components with zero observed dependents (safe to rename; *not* evidence of being unused).

## Analytics tables

```bash
deppolicy tables --policy policy.json --graph graph.json --output-dir tables/ --format csv
```

Flattens Policy vs Actual vs Difference into four tables (`components`, `policy_edges`, `observed_edges`, `findings`) for BI tools, spreadsheets, or notebooks.
