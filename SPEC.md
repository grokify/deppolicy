# DepPolicy Specification v1

This document is the normative specification of DepPolicy's data model and
evaluation semantics, describing what is true of the `policy`, `component`,
and `graph` packages in this repository.

DepPolicy governs **direct** dependency relationships between **components**
in a **governed ecosystem**, using a **default-deny** policy that is
evaluated as a pure, deterministic function of two documents: an authored
**policy** and a generated **graph**.

## 1. Components

A **component** is the ecosystem-neutral unit that dependency policy
governs. A component may be a Go module, a directory within a Go module
declared as a finer-grained component (a **proto-module**), an npm package,
a Rust crate, a service, or any other unit identified by a **locator**.

```text
id       stable identity referenced by policy and graph documents
kind     ecosystem-specific packaging mechanism (go-module, go-package, npm-package, crate, service, ...)
locator  current ecosystem address, e.g. "go:github.com/plexusone/omnillm"
status   governed (default) | foundation | deprecated | ungoverned
```

`kind` is metadata only — policy and graph semantics never branch on it.

### 1.1 Locator resolution

An observed import path or manifest requirement resolves to its governing
component by **longest-declared-prefix match**: the locator either equals a
declared component's locator, or begins with that locator followed by `/`.

This is what lets a Go module be the default component for its whole import
path while a declared sub-path carves out a proto-module: declaring
`go:github.com/grokify/mogo/database` as its own component causes imports
under that path to resolve to it instead of to the parent `mogo` module,
without changing how any other subpath of `mogo` resolves. A proto-module's
locator can later become a real module's locator with no change to policy
semantics, because resolution and identity are independent of packaging
mechanics.

Component carve-outs should be rare and deliberate. Every sub-module
component declared re-couples central policy to that repository's internal
directory layout; Go's `internal/` visibility already provides free,
zero-maintenance governance for subtrees that should simply be private, not
selectively importable.

### 1.2 Status semantics

| Status | Meaning |
|---|---|
| `governed` (default) | Edges to and from this component require explicit authorization under default-deny. |
| `foundation` | Any governed component may depend on it without an explicit allow edge. A foundation component may itself depend only on other foundation components (an invariant, not a request). |
| `deprecated` | Existing inbound edges are tolerated (baselined); any *new* inbound edge is a violation ("no new dependents"). |
| `ungoverned` | Treated exactly like a third-party component for evaluation, but — unlike a scope-excluded component — remains visible in the graph and in reports. |

### 1.3 Tiers

A component may additionally declare a **tier** — an archetype name drawn
from the policy's `tiers` section, which pairs the tier vocabulary with
per-tier auto-allow rules:

```json
"tiers": {
  "library":          { },
  "mcp-cli":          { },
  "local-app":        { },
  "omni-adapter":     { },
  "forge-foundation": { },
  "horizontal-forge": { "mayDependOn": ["forge-foundation", "omni-adapter"] },
  "vertical-app":     { "mayDependOn": ["horizontal-forge"] }
}
```

An observed edge A → B is tier-authorized when both components carry a
declared tier and A's tier lists B's tier in `mayDependOn`.

A tier may additionally declare `mayNotDependOn`: locator *patterns*
(Scope syntax) its components are prohibited from depending on. This is
the model's only rule that governs **third-party** targets — everything
else deliberately ignores out-of-scope dependencies — and it exists for
constraints like "an abstraction core must not import provider SDKs":

```json
"omni-core": {
  "mayNotDependOn": ["go:github.com/anthropics/*", "go:github.com/openai/*"]
}
```

A deny overrides every allow, explicit relationships included. The one
escape valve is an unexpired Exception for the exact edge, so a denied
dependency that must exist temporarily is visible, justified, and
time-bounded. Denied-but-unobserved pairs produce no findings.

Three further properties are deliberate and non-configurable:

- **No transitivity.** `vertical-app → horizontal-forge` plus
  `horizontal-forge → omni-adapter` does **not** authorize
  `vertical-app → omni-adapter`. Skipping a layer is an architecture
  bypass and always requires an explicit relationship.
- **No self-tier match.** A tier may never list itself (`Validate` rejects
  it): same-tier peers — cross-forge, cross-omni — always require explicit
  wiring.
- **Foundation is orthogonal.** There is no blanket "→ library" rule
  shape. Making a library universally consumable is a per-component
  `foundation` status elevation (mogo, goauth), never a side effect of
  tier membership — so adding a library to the ecosystem never silently
  widens what everything else may depend on.

Tier authorization slots into evaluation after foundation-target
auto-allow and before explicit relationships; the foundation invariant
and deprecated no-new-dependents rules both override it. Tier rules never
generate unused-allowance findings — they are blanket permissions, not
per-edge declarations.

## 2. Governed ecosystem scope

A component's locator is **in scope** (subject to policy at all) if it
matches at least one `scope.include` pattern and no `scope.exclude` pattern.

Patterns support one wildcard form: `*` matches any sequence of characters
that does not cross a `/`. A pattern ending in `/*` additionally matches
everything nested beneath its prefix, at any depth — this is what lets
`go:github.com/grokify/*` cover every component under that org, while
`go:github.com/*/*-archive` matches only a repository whose name ends in
`-archive`, without also matching that repository's own subpaths.

A locator outside scope is treated as **external**: dependencies to or from
it are never evaluated, exactly like a third-party import.

Classification precedence, most specific first:

```text
1. Explicit component declaration (policy.components[locator].status)
2. Scope pattern (scope.exclude → out of scope; unmatched by scope.include → out of scope)
3. Default (governed, if in scope)
```

## 3. Default-deny

Within the governed ecosystem, a direct dependency from component A to
component B requires one of:

- an explicit `can_depend_on` relationship authorizing A → B, or
- B has `foundation` status, or
- A's declared tier lists B's declared tier in `mayDependOn` (§1.3), or
- an unexpired exception authorizing A → B.

Absence of authorization is denial. There is no need to write `"default":
"deny"` anywhere in the policy document — it is the specification, not a
configurable setting.

**Transitive dependencies are not independently governed.** If A → B and
B → C are each individually authorized, A does not need a direct
authorization to reach C transitively. DepPolicy governs direct boundary
crossings, not reachability:

> We govern boundaries, not reachability. Every direct internal boundary
> crossing must be explicitly authorized.

## 4. The three-artifact contract

```text
policy   authored intent      what MAY exist   — reviewed, versioned, never auto-generated
graph    generated reality    what DOES exist  — produced by scanners, with evidence
findings pure comparison      evaluate(policy, graph)
```

`policy` and `graph` use parallel-but-distinct relation vocabulary
deliberately, so a reader never confuses intent with observation:

```text
policy:  A can_depend_on B     (permission)
graph:   A depends_on B        (observed fact)
```

V1 defines exactly one relation on each side. Typed relations
(`can_call`, `can_publish_to`, ...) are reserved for a future version, not
because they are hard to add, but because a narrow relation vocabulary is
what keeps evaluation a bounded, statically analyzable lookup rather than a
programming problem.

**The evaluator (`evaluate(policy, graph) -> findings`) is a pure
function**: no filesystem, network, database, or wall-clock access.
Evaluation-time-dependent facts (such as whether an exception has expired)
are passed as explicit arguments, not read from the environment.

**Policy is never generated from the graph.** The graph is a worklist for
authoring policy — reviewed and promoted deliberately — never an automatic
source. A tool that auto-writes policy from observed reality stops
governing coupling and starts ratifying it.

## 5. Findings

For every observed direct edge A → B in the graph, and every authorized
edge A → B in the policy, evaluation produces one of four states:

| | Authorized | Not authorized |
|---|---|---|
| **Observed** | conforming edge | **violation** |
| **Not observed** | unused allowance | *(no finding)* |

A `deprecated` target additionally distinguishes violations by whether the
edge was already present in a baseline graph (tolerated debt) or is newly
introduced (violation, regardless of whether an explicit relationship exists
— deprecation means "stop adding dependents," not "revoke existing
permission").

## 6. Exceptions

An exception is a centrally-declared, time-bounded authorization that
overrides default-deny for exactly one edge without adding a permanent
relationship. Unlike a `Relationship`, an `Exception` must carry an
`expires` date and a `reason`, so temporary debt stays visible instead of
quietly becoming permanent. V1 supports only `effect: "allow"` — there is no
"deny exception," because default-deny already denies anything not
explicitly authorized.

## 7. Non-goals (V1)

- Transitive-dependency policy (§3).
- Typed relations beyond `can_depend_on` / `depends_on`.
- Tier-rule transitivity, tier hierarchies/inheritance, or same-tier
  auto-allow (§1.3 declares flat, explicit, non-composing rules only).
- Local (per-repository) policy overrides of any kind — see §2 and §3: the
  central policy is authoritative; repositories may contribute component
  metadata but never authorization.
- Runtime/telemetry-derived evidence, LLM-inferred edges — the graph's
  evidence model reserves fields for these, but no V1 scanner emits them.

## Prior art

The relationship model deliberately mirrors OpenFGA's user/relation/object
tuples ("who can access what" becomes "what code may depend on what"),
constrained to remain statically and deterministically evaluable; rule
semantics also draw on dependency-cruiser (allowed/forbidden dependency
rules) and depguard (import allow/deny lists).
