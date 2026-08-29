# DepPolicy

A statically analyzable dependency-policy engine: which **components** may directly depend on which others, enforced as a **default-deny** policy against what your code actually does.

Think *"OpenFGA for code dependencies"* — a relationship-based authorization model (`source can_depend_on target`), deliberately constrained so every check is a deterministic, local, O(1)-style lookup: no policy interpreter, no network, no service.

## The three-artifact contract

```text
policy.json     authored intent      what MAY exist   — reviewed, versioned, never auto-generated
graph.json      generated reality    what DOES exist  — produced by scanners, with evidence
findings        pure comparison      evaluate(policy, graph)
```

The evaluator is a pure function — no filesystem, network, or clock access. Scanners discover facts; the evaluator decides; enforcement points (CLI, CI, `go vet`) act on decisions. Policy is never generated from the graph automatically: observation informs authoring, it never replaces it.

## Why

As an ecosystem grows across many repositories, architectural intent — which components may depend on which others, and in which direction — tends to live only in ADRs and people's heads, while adding an import stays cheap (especially with AI-assisted development). DepPolicy makes that intent executable, and gives you a **ratchet**: existing architecture debt may remain while it's classified, but no change may introduce new debt.

## Ecosystem-neutral by design

A component may be a Go module, a directory within one (a *proto-module*), an npm package, a Rust crate, a container image, or a microservice — anything addressable by a locator. Policy semantics never branch on a component's kind. **Go is the first and reference scanner**; other ecosystems extend the same graph contract. See [Extending Beyond Go](reference/extending.md).

## Where to start

- [Getting Started](getting-started.md) — a minimal policy and your first check
- [Component Model](concepts/model.md) — components, locators, status
- [Bootstrapping an Ecosystem](guides/bootstrap.md) — from zero policy to ratchet enforcement
- [Commands](reference/commands.md) — the full CLI surface
- [Specification](https://github.com/grokify/deppolicy/blob/main/SPEC.md) — the normative model
