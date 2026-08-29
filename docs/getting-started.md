# Getting Started

## Install

```bash
go install github.com/grokify/deppolicy/cmd/deppolicy@latest
```

Or build from source:

```bash
git clone https://github.com/grokify/deppolicy
cd deppolicy && go build -o deppolicy ./cmd/deppolicy
```

## A minimal policy

Policy is JSON, authored and reviewed like code. Within the declared scope, every direct dependency between governed components is **denied unless authorized**; anything outside scope (the standard library, third-party modules) is unrestricted.

```json
{
  "schemaVersion": "1",
  "scope": { "include": ["go:github.com/yourorg/*"] },
  "relationships": [
    {
      "source": "go:github.com/yourorg/dashforge",
      "relation": "can_depend_on",
      "target": "go:github.com/yourorg/omniagent",
      "reason": "dashforge consumes the shared agent runtime"
    }
  ]
}
```

Every relationship must state a `reason` — a permanent architectural permission should be able to say why in one sentence.

## Your first check

Point `check` at a directory tree containing one or more Go modules:

```bash
deppolicy check --policy policy.json /path/to/yourorg
```

If `dashforge` also imports `omnillm` directly, without a matching relationship:

```text
PASS  go:github.com/yourorg/dashforge -> go:github.com/yourorg/omniagent
FAIL  go:github.com/yourorg/dashforge -> go:github.com/yourorg/omnillm
      no allow rule; default-deny
      internal/agent/client.go:12

1 conforming, 1 violation(s), 0 unused allowance(s)
```

The command exits non-zero on any violation, so it is CI-ready as-is. Violations point at the file and line of the offending import.

## What happens under the hood

1. **Scan** — `go.mod` files locate modules; source files are parsed imports-only (no type checking) and every cross-component import becomes an observed `depends_on` edge with evidence.
2. **Evaluate** — each edge is checked against the policy: explicit relationship, [foundation status](concepts/model.md#status), [tier rule](concepts/tiers.md), or unexpired exception ⇒ conforming; otherwise ⇒ violation.
3. **Report** — findings with file:line evidence, as console output, JSON, [HTML report](guides/reports.md), or CI exit codes.

## Next steps

- Have an existing codebase with pre-existing coupling? Start with the [bootstrap workflow](guides/bootstrap.md) — you don't need to classify everything before enforcement begins.
- Want violations at the import line in your editor? See [Editor and Vet Integration](guides/analyzer.md).
