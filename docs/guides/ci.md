# CI Enforcement

## GitHub Action

```yaml
- uses: grokify/deppolicy@main
  with:
    mode: diff              # or "check" once the repo's edges are fully classified
    policy: policy.json
    baseline: baseline-graph.json
```

Two modes, matching two maturity stages:

| Mode | Fails on | Use when |
|---|---|---|
| `diff` | violations **absent from the baseline** (new debt only) | pre-existing coupling is still being classified — the ratchet |
| `check` | **any** violation | the scope under test is fully classified |

Optional inputs: `path` (directory to scan, default `.`), `output` (write findings JSON for artifact upload), `version` (deppolicy version to install, default `latest`).

## Exit-code semantics

- `check` exits non-zero on any violation.
- `diff` exits non-zero only on new violations; baselined debt prints as `WARN` lines.
- `policy-diff` exits zero even with breaking removals unless `--fail-on-breaking` is set.

## Reviewing policy changes in the policy repo's own CI

A policy PR is an architecture change. Gate it with:

```bash
deppolicy validate --policy policy.json
deppolicy policy-diff main-policy.json pr-policy.json --graph latest-graph.json --fail-on-breaking
```

`policy-diff` reports added relationships (newly permitted dependencies), removed ones, and component status/tier changes. With `--graph`, a removal whose edge is **still observed** is flagged `BREAKING`: by design, merging it does not break other repos' CI runs — the next fleet scan reports those edges as violations instead — so policy review is the moment to decide whether the code gets fixed first.

## Division of labor

- **Per-PR (each repo):** `diff` against the shared baseline — fast (seconds), scans only the changed repo.
- **Scheduled (fleet):** [`scan-github`](fleet.md) manifests-only across whole orgs, no cloning, feeding refreshed graphs and reports.
- **Policy repo:** `validate` + `policy-diff` on every change.
