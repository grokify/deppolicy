# Fleet Scanning

Two ways to observe a large ecosystem, with different cost/assurance trade-offs.

## Local: pre-cloned trees

If you keep repositories cloned under a workspace (e.g. a `~/src/github.com/<org>` layout), source-level scanning across hundreds of modules takes seconds to minutes:

```bash
deppolicy scan --output graph.json ~/src/github.com/org-a ~/src/github.com/org-b
```

- Imports are parsed header-only (`parser.ImportsOnly`) — no type checking, cost scales with file count, not code size.
- A bounded worker pool (default `GOMAXPROCS`) scans modules concurrently; one broken repository never aborts the rest — failures are reported per module.
- All roots share one component registry, so cross-org deep imports resolve to their owning modules.
- Duplicate module paths (backup copies, unedited template scaffolds) are deduplicated deterministically and reported, never silently dropped.

This produces `source-imports`-level evidence — the authoritative kind, with file:line for every edge.

## Remote: GitHub API, no cloning

```bash
export GITHUB_TOKEN=...
deppolicy scan-github --output graph.json org-a org-b
```

Fetches exactly one file per repository (`go.mod`) via the contents API — no cloning, so it scales to orgs of any size and runs fine on a schedule. Trade-offs, stated in the output's scan metadata:

- Evidence level is `manifest` (declared dependencies), not observed source imports.
- Archived repositories are skipped without even a fetch.
- Repositories without `go.mod` are excluded (not errors); fetch/parse failures are reported per repo without aborting the org.

`--policy` applies scope exclusion in both modes; omit it for bootstrap discovery.

## Choosing

| | Local scan | scan-github |
|---|---|---|
| Evidence | source imports (file:line) | go.mod manifests |
| Cost | minutes for hundreds of repos | one API call per repo |
| Needs | clones on disk | a token |
| Good for | authoritative graphs, baselines, reports | scheduled drift detection, orgs you don't clone |
