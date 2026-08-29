# Editor and Vet Integration

The `go/analysis` analyzer reports violations **at the offending import line** — the enforcement point closest to where an import gets written. It runs the same pure evaluator as the CLI, so tier rules, foundation status, and explicit relationships behave identically everywhere.

## Install

```bash
go install github.com/grokify/deppolicy/cmd/deppolicy-vet@latest
```

## Run

Standalone:

```bash
deppolicy-vet -policy=policy.json ./...
```

Through `go vet` (and therefore any editor/IDE that surfaces vet diagnostics):

```bash
go vet -vettool=$(which deppolicy-vet) -policy=policy.json ./...
```

A violation looks like:

```text
internal/agent/client.go:12:2: go:github.com/yourorg/app may not depend on go:github.com/yourorg/engine: no allow rule; default-deny
```

## Behavior notes

- The module identity comes from the build system (`go.mod`); the `-module` flag overrides it for drivers that don't supply module info.
- Modules outside the policy's scope produce no diagnostics — third-party code is never policed.
- Test files are skipped, matching the scanner: test-only imports are integration surface, not architecture.
- The analyzer runs with **no baseline**, so it is strictly ratchet-tightening at the editor: an edge to a `deprecated` component always reports, even if CI's baselined `diff` would tolerate it.
- For import paths not covered by a declared component, deep paths are collapsed to `host/org/repo` heuristically; nested modules should be declared as components for exact attribution.
