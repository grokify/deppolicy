# Extending Beyond Go

DepPolicy's policy semantics are ecosystem-neutral; only scanners are ecosystem-specific. The boundary is strict:

> **Scanners discover facts. The policy engine decides whether those facts are allowed.**

A scanner emits normalized `depends_on` edges with evidence into the shared graph contract, and never makes authorization decisions. That means a buggy scanner can miss edges, but it can never implement subtly different policy semantics than another ecosystem's scanner.

## The graph contract

```json
{
  "scan": { "level": "source-imports", "scanner": "your-scanner v1", "coverage": ["..."] },
  "components": [ { "id": "npm:@yourorg/widgets", "kind": "npm-package" } ],
  "dependencies": [
    {
      "source": "npm:@yourorg/app",
      "relation": "depends_on",
      "target": "npm:@yourorg/widgets",
      "evidence": [ { "kind": "source-import", "file": "src/main.ts", "line": 3 } ]
    }
  ]
}
```

Locators are scheme-prefixed (`go:`, and by convention `npm:`, `cargo:`, `pypi:`, `oci:`, `service:`, …) so identifiers never collide across ecosystems. Component `kind` is metadata — the evaluator never branches on it.

## Adding an ecosystem

Each ecosystem maps naturally onto the same two scan levels the Go scanner implements:

| Ecosystem | Manifest level | Source-import level |
|---|---|---|
| Go | `go.mod` | `.go` imports |
| npm / TypeScript | `package.json` | `import` / `require` |
| Rust | `Cargo.toml` | `use` / extern crates |
| Python | `pyproject.toml` | `import` statements |
| JVM | `pom.xml` / Gradle | package imports |

Write the scanner in any language: the contract is JSON, and the JSON Schemas for both policy and graph documents are generated from the Go types and embedded in the module (`schema` package). A scanner produces `graph.json`; every existing command — `check`, `diff`, `promote`, `report`, `ui` — works unchanged. In-process Go scanners can alternatively emit `graph.Dependency` values directly.

## Beyond source code

The component model extends past packages because components are *declared*, not derived from packaging:

- **Microservices** — services as components (`service:` locators), call graphs as edges discovered from OpenAPI/proto definitions, IaC, or traces. The relation vocabulary reserves typed relations (e.g. `can_call` / `calls`) for exactly this; V1 keeps a single `can_depend_on`/`depends_on` pair deliberately.
- **Containers** — images as components (`oci:` locators), base-image and bundled-artifact relationships as edges.

Evidence kinds `iac`, `llm-analysis`, and `telemetry` are reserved in the graph model for these scanners. The design rule that keeps all of it coherent: evidence provenance and confidence affect **reporting only** — authorization remains a deterministic function of policy and edges, whatever discovered them.
