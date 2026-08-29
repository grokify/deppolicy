# Evaluation

## A pure function

```text
evaluate(policy, graph, asOf) → findings
```

No filesystem, network, database, or clock access — evaluation time is an explicit argument (it resolves exception expiry), so the same inputs always produce the same findings. This purity is load-bearing: it makes results reproducible, ratchet comparisons meaningful, and every enforcement point (CLI, CI, `go vet`, editors) behave identically.

## Four states

Every edge that is observed, authorized, or both lands in exactly one cell:

| | Authorized | Not authorized |
|---|---|---|
| **Observed** | conforming | **violation** |
| **Not observed** | unused allowance | *(no finding)* |

Unused allowances are architecture debt of the opposite kind from violations — permissions that outlived their use and should be pruned so the policy doesn't overstate coupling.

## Evidence and scan levels

Graph edges carry **evidence** — where the dependency was seen:

- `source-import` — an actual import statement (file and line). The authoritative level for architecture decisions.
- `manifest` — a declared requirement (e.g. `go.mod`). Fast and cheap; indirect requirements are excluded, since they aren't dependencies of the module's own code.

Every graph records its **scan level**, so a fast manifest-only fleet scan is never mistaken for a full source audit. Evidence and confidence never affect authorization — only reporting.

## The ratchet

The graph is a first-class serialized artifact, which enables graph-to-graph comparison independent of policy:

```text
diff(graph@baseline, graph@now) → added / removed / unchanged edges
```

`deppolicy diff` combines this with evaluation: a violating edge **absent from the baseline** is a *new violation* (fails CI); a violating edge already in the baseline is *existing debt* (warned, not blocking). That yields the bootstrap invariant:

> Existing architecture debt may remain while it's classified — but no change may introduce new debt.

Enforcement starts on day one, before a single historical edge has been reviewed. The same baseline also drives `deprecated`'s no-new-dependents rule, so "new" means one thing everywhere.

## Findings never edit policy

`promote` proposes relationships for unauthorized observed edges — with placeholder reasons that must be replaced by a human — and writes only a separate draft document. There is deliberately no command that writes policy from observation: a tool that auto-ratifies reality stops governing coupling and starts ratifying it.
