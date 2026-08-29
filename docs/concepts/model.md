# Component Model

The governed unit is a **component** — declared by policy, not implied by packaging.

```text
id       stable identity referenced by policy and graph documents
kind     packaging mechanism: go-module, go-package, npm-package, crate,
         container-image, service, ...   (metadata only)
locator  current ecosystem address, e.g. "go:github.com/yourorg/widgets"
status   governed (default) | foundation | deprecated | ungoverned
```

`kind` never affects policy semantics — that is what keeps the model portable across ecosystems.

## Locator resolution

An observed import path resolves to its owning component by **longest-declared-prefix match**: the locator either equals a declared component's locator, or begins with it followed by `/`. A Go module is the default component for its whole import path, so `yourorg/widgets/internal/db` attributes to `yourorg/widgets` without any declaration.

### Proto-modules

Declaring a sub-path as its own component carves a finer-grained unit out of its parent — governance without extracting a new module:

```json
"components": {
  "go:github.com/yourorg/widgets/database": { "kind": "go-package", "tier": "storage" }
}
```

Imports under `widgets/database/...` now resolve to that component; every other subpath of `widgets` still resolves to the module. A proto-module's identity survives later promotion to a real module, because resolution is independent of packaging mechanics.

!!! note "Use the language first"
    Go's `internal/` visibility already gives free, compiler-enforced privacy for subtrees that should simply be private. Proto-module carve-outs are for subtrees that should be *selectively* importable.

## Status

| Status | Meaning |
|---|---|
| `governed` (default) | Edges to and from it require authorization under default-deny. |
| `foundation` | Any governed component may depend on it without an explicit edge. A foundation component may itself depend **only on other foundation components** — an invariant that overrides even explicit relationships. |
| `deprecated` | Existing dependents (present in a baseline graph) are tolerated; any *new* inbound edge is a violation — "no new dependents." |
| `ungoverned` | Treated like a third-party component for evaluation, but remains visible in graphs and reports (unlike scope-excluded components, which are dropped — though never silently: scans always report what they skipped). |

## Scope

A locator is in scope if it matches at least one `scope.include` pattern and no `scope.exclude` pattern. `*` matches within one path segment; a pattern ending in `/*` matches everything beneath its prefix at any depth:

```json
"scope": {
  "include": ["go:github.com/yourorg/*"],
  "exclude": ["go:github.com/yourorg/sandbox-*", "go:github.com/*/*-archive"]
}
```

Exclude patterns pair naturally with repo naming conventions (`sandbox-*`, `experiment-*`, `*-archive`), so classification comes from names rather than per-repo declarations. Classification precedence: explicit component declaration → scope pattern → default.
