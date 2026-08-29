# DepPolicy

[![Go CI][go-ci-svg]][go-ci-url]
[![Go Lint][go-lint-svg]][go-lint-url]
[![Go SAST][go-sast-svg]][go-sast-url]
[![Docs][docs-godoc-svg]][docs-godoc-url]
[![Visualization][viz-svg]][viz-url]
[![License][license-svg]][license-url]

 [go-ci-svg]: https://github.com/grokify/deppolicy/actions/workflows/go-ci.yaml/badge.svg?branch=main
 [go-ci-url]: https://github.com/grokify/deppolicy/actions/workflows/go-ci.yaml
 [go-lint-svg]: https://github.com/grokify/deppolicy/actions/workflows/go-lint.yaml/badge.svg?branch=main
 [go-lint-url]: https://github.com/grokify/deppolicy/actions/workflows/go-lint.yaml
 [go-sast-svg]: https://github.com/grokify/deppolicy/actions/workflows/go-sast-codeql.yaml/badge.svg?branch=main
 [go-sast-url]: https://github.com/grokify/deppolicy/actions/workflows/go-sast-codeql.yaml
 [docs-godoc-svg]: https://pkg.go.dev/badge/github.com/grokify/deppolicy
 [docs-godoc-url]: https://pkg.go.dev/github.com/grokify/deppolicy
 [viz-svg]: https://img.shields.io/badge/visualization-Go-blue.svg
 [viz-url]: https://mango-dune-07a8b7110.1.azurestaticapps.net/?repo=grokify%2Fdeppolicy
 [loc-svg]: https://tokei.rs/b1/github/grokify/deppolicy
 [repo-url]: https://github.com/grokify/deppolicy
 [license-svg]: https://img.shields.io/badge/license-MIT-blue.svg
 [license-url]: https://github.com/grokify/deppolicy/blob/main/LICENSE

A statically analyzable dependency-policy engine: which **components** may directly depend on which others, enforced as a default-deny policy against what your code actually does. Think "OpenFGA for code dependencies" — a relationship-based authorization model (`source can_depend_on target`), deliberately constrained so every check is a deterministic, local, O(1)-style lookup: no policy interpreter, no network, no service.

The policy model is **ecosystem-neutral**: a component may be a Go module, a directory within one (a proto-module), an npm package, a Rust crate, a container image, or a microservice — anything addressable by a locator. Scanners discover facts per ecosystem; one shared evaluator decides. **Go is the first and reference scanner**; other ecosystems extend the same graph contract without touching policy semantics.

> **Status:** early development. Interfaces and the policy schema may still change. Implemented scanners today: Go (`go.mod` manifests, source imports, and GitHub-API fleet scans).

## Why

As an ecosystem grows across many repositories, architectural intent — which components may depend on which others, and in which direction — tends to live only in ADRs and people's heads, while adding an import stays cheap (especially with AI-assisted development). DepPolicy makes that intent executable: an authored `policy.json` declares allowed direct dependencies, a scanner produces a `graph.json` of what actually exists, and a pure evaluator reports where reality diverges from intent.

See [`SPEC.md`](SPEC.md) for the full normative model.

## Quickstart

Build the CLI:

```bash
go build -o deppolicy ./cmd/deppolicy
```

Write a minimal policy (`policy.json`):

```json
{
  "schemaVersion": "1",
  "scope": { "include": ["go:github.com/yourorg/*"] },
  "relationships": [
    {
      "source": "go:github.com/yourorg/dashforge",
      "relation": "can_depend_on",
      "target": "go:github.com/yourorg/omniagent"
    }
  ]
}
```

Check a directory tree (e.g. an org root containing several modules) against it:

```bash
./deppolicy check --policy policy.json --output findings.json /path/to/yourorg
```

If `dashforge` also imports `omnillm` directly, without a matching relationship, output looks like:

```text
PASS  go:github.com/yourorg/dashforge -> go:github.com/yourorg/omniagent
FAIL  go:github.com/yourorg/dashforge -> go:github.com/yourorg/omnillm
      no allow rule; default-deny
      internal/agent/client.go:12

1 conforming, 1 violation(s), 0 unused allowance(s)
```

The command exits non-zero when any violation is found, so it's CI-ready as-is.

## Commands

| Command | Purpose |
|---|---|
| `scan <dir>...` | Scan directory trees into a `graph.json` (bootstrap mode without `--policy`) |
| `scan-github <org>...` | Fleet scan via the GitHub API — `go.mod` only, no cloning |
| `check <dir>` | Evaluate a scan against policy; exit non-zero on violations |
| `diff <dir>` | Ratchet CI: fail only on violations absent from `--baseline`, warn on existing debt |
| `policy-diff <old> <new>` | Review a policy change; flag removals whose edges are still observed |
| `validate` | Schema + structural + expired-exception + referential-integrity checks |
| `promote` | Propose relationships for unauthorized observed edges (never writes the policy) |
| `report` | Ecosystem summary: fan-in leaders, cross-org edges, rename-safety candidates |
| `export` | Mermaid/DOT diagram grouped by org, violations highlighted |
| `tables` | Policy/Actual/Difference as flat tables (JSON/CSV) for analytics platforms |
| `ui` | Self-contained multi-page HTML report: summary, findings, components, graph, tiers |

There is also a `go/analysis` analyzer (`cmd/deppolicy-vet`) that reports violations at the offending import line:

```bash
go vet -vettool=$(which deppolicy-vet) -policy=policy.json ./...
```

## GitHub Action

```yaml
- uses: grokify/deppolicy@main
  with:
    mode: diff              # or "check" once the ecosystem is fully classified
    policy: policy.json
    baseline: baseline-graph.json
```

`check` fails on any violation; `diff` fails only on newly introduced ones, which is the right mode while pre-existing architecture debt is still being classified.

## How it works

Three artifacts, evaluated as a pure function:

```text
policy.json     authored intent      what MAY exist
graph.json      generated reality    what DOES exist (from scanning)
findings        evaluate(policy, graph)
```

The governed unit is a **component** — declared by policy, not implied by packaging. Its `kind` (go-module, go-package, npm-package, crate, container-image, service, …) is metadata only: policy semantics never branch on it, which is what keeps the model portable across ecosystems. Within the governed scope, a direct dependency is denied by default unless it's explicitly authorized, targets a `foundation`-status component, is allowed by a tier rule, or is covered by a time-bounded exception. Transitive dependencies are not independently governed — DepPolicy governs direct boundary crossings, not reachability.

Components can also be organized into **tiers** (e.g. libraries → adapters → platform apps) with explicit, non-transitive `mayDependOn` allow-lists and `mayNotDependOn` deny patterns — the latter being the one rule that can reach third-party targets, for constraints like "an abstraction core must not import provider SDKs."

Full semantics: [`SPEC.md`](SPEC.md).

## Extending beyond Go

Scanners and policy are strictly separated: a scanner only emits normalized `depends_on` edges with evidence into the shared graph contract, and never makes authorization decisions. Adding an ecosystem (npm, Cargo, PyPI, Maven, …) means writing a scanner that maps that ecosystem's manifests/imports to component locators — the evaluator, tier system, CLI, and report tooling work unchanged. The same shape extends past source code: microservice call graphs or container-image dependencies become components and edges under other locator schemes, with typed relations (e.g. `can_call`) reserved in the model for exactly that.

## Package layout

| Package | Purpose |
|---|---|
| `component` | The ecosystem-neutral governed unit (ID/Kind/Locator/Status) and longest-prefix locator resolution |
| `policy` | Policy document types, parsing/validation, and the pure evaluator (`Evaluate`, `EvaluateWithStatus`) |
| `graph` | Generated dependency graph document types and JSON I/O |
| `scanner/golang` | Go scanners (reference implementation): `go.mod` manifests, source imports, concurrent multi-module scanning, GitHub-API fleet mode |
| `schema` | JSON Schema generated from the `policy` and `graph` Go types (`go generate ./schema/...`) |
| `cli` | Reusable CLI orchestration (loading policy, scanning, evaluating, formatting reports) |
| `cmd/deppolicy` | The Cobra-based CLI binary — a thin adapter over `cli` |

## Development

```bash
go build ./...
go test ./...
golangci-lint run ./...
```

Regenerate JSON Schemas after changing `policy.Policy` or `graph.Graph`:

```bash
go generate ./schema/...
```
