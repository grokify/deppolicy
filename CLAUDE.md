# CLAUDE.md

Repo-specific guidance for `github.com/grokify/deppolicy`. Org-wide conventions (error handling, commit style, dependency verification, JSON Schema workflow) come from `~/go/src/github.com/grokify/.github/CLAUDE.md` — this file only covers what's specific to this codebase.

## Read SPEC.md first

Before changing evaluation semantics, read [`SPEC.md`](SPEC.md). It is the normative reference for default-deny, classification precedence, the foundation invariant, and the three-artifact (policy/graph/findings) contract.

## Package dependency direction

```text
component  →  graph  →  policy  →  scanner/golang  →  cli  →  cmd/deppolicy
```

`component` has no internal imports. `graph` imports only `component`. `policy` imports both (its evaluator takes a `graph.Graph`). Keep it one-directional — nothing upstream should import something downstream. If you find yourself needing `graph` to know about `policy`, that's a sign the logic belongs in `policy` or in a caller, not in `graph`.

## Evaluate vs EvaluateWithStatus

`policy.Evaluate(graph, asOf)` implements only base scope/default-deny/exceptions. `policy.EvaluateWithStatus(graph, asOf, baseline)` layers foundation-tier and deprecated-target semantics on top. They're deliberately separate functions, not one function with more parameters — `EvaluateWithStatus` is tested against `Evaluate` for identical output on edges the status rules don't touch (see `TestEvaluateWithStatus_UnaffectedGovernedEdgesMatchBaseEvaluate`). If you add a new status-driven rule, extend `EvaluateWithStatus`, don't complicate `Evaluate`.

## Purity is load-bearing, not a style preference

`Evaluate`/`EvaluateWithStatus` never read the wall clock or touch the filesystem — evaluation time (`asOf`) is always an explicit argument. This is what makes `evaluate(policy, graph)` reproducible and what will make `deppolicy diff`'s baseline comparison meaningful. Don't introduce `time.Now()` or file I/O into anything in the `policy` or `graph` packages; push it to the `cli` or `cmd` layer instead.

## Testing pattern: real fixtures, not mocks

Every scanner/evaluator test builds real files in `t.TempDir()` and runs the real `go/parser`/`golang.org/x/mod/modfile` against them — there are no mocked ASTs or fake filesystems anywhere in this repo. Follow that pattern for new tests. For CLI changes, the `cmd/deppolicy` test executes the real Cobra command in-process (`cmd.Execute()`), and changes to `cli.Check`/report formatting should additionally be spot-checked with `go build -o /tmp/deppolicy ./cmd/deppolicy` and a manual run before considering the change verified — this has already caught output-format mismatches that unit tests alone missed.

## Schema regeneration

After changing `policy.Policy` or `graph.Graph` field shapes, regenerate the embedded JSON Schemas and re-lint them:

```bash
cd schema && go run gen/main.go
schemakit lint --property-case camelCase policy.schema.json
schemakit lint --property-case camelCase graph.schema.json
```

`go generate ./schema/...` also works from the repo root. `schema/gen/main.go` is `//go:build ignore` and `schema/gen/tools.go` is `//go:build tools`; both are excluded from normal builds by design — don't remove `tools.go` or `go mod tidy` will drop the generator's only import of `invopop/jsonschema`.
