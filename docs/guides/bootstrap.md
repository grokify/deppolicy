# Bootstrapping an Ecosystem

You have many repositories, real coupling, and no policy. The bootstrap flow gets you to enforcement **without** classifying every historical edge first.

## 1. Scan everything

Multiple directory roots combine into one graph with one shared component registry — required so a deep import from one org resolves to its owning module in another:

```bash
deppolicy scan --output graph.json ~/src/github.com/org-a ~/src/github.com/org-b
```

Without `--policy`, scan runs in *bootstrap mode*: nothing is excluded, because no scope exists yet. The report lists everything skipped (noise directories, parse failures) — excluded is never invisible.

## 2. Study the graph

```bash
deppolicy report --policy draft-policy.json --graph graph.json
```

The report surfaces exactly what policy authoring needs: **fan-in leaders** (your `foundation` candidates — components half the ecosystem imports), **cross-org edges** (real coupling you may not have known about), and **rename-safety candidates** (components with zero dependents, safe to rename to convention-excluded names like `*-archive`). The [HTML report](reports.md) makes the same data browsable.

## 3. Author a starter policy

Resist bulk-generating relationships for all observed edges. A good starter policy is small:

- **scope** — your orgs, plus naming-convention excludes (`sandbox-*`, `experiment-*`, `*-archive`)
- **foundation** — the few universal utility libraries the fan-in data identifies
- **tiers** — your archetypes and their rules, which absorb whole classes of edges at once
- **relationships** — only edges you actively decide to bless now

Then let `promote` drive incremental classification of the rest:

```bash
deppolicy promote --policy policy.json --graph graph.json --output proposed.json
```

Each proposal carries evidence and a placeholder reason that must be replaced before merging — `promote` discovers missing decisions; it doesn't make them.

## 4. Freeze a baseline and turn on the ratchet

Save the scanned graph (evidence can be trimmed) as your baseline, then enforce:

```bash
deppolicy diff --policy policy.json --baseline baseline-graph.json <roots...>
```

Zero *new* violations passes — every pre-existing unauthorized edge is baselined debt, reported as warnings. From this moment the ecosystem cannot get more coupled accidentally, while the debt burns down through `promote` review at whatever pace suits you.

!!! tip "Ratcheting the baseline"
    Shrinking the baseline (removing edges that no longer exist) is always safe. *Adding* edges to it forgives new debt — review such changes like policy changes.

## 5. Keep the policy honest

- `deppolicy validate --policy policy.json --graph graph.json` — schema, structure, expired exceptions, dangling references
- `deppolicy policy-diff old.json new.json --graph graph.json` — review every policy change; removals whose edges are still observed are flagged **breaking**
