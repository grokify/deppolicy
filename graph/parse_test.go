package graph

import "testing"

const exampleGraphJSON = `{
  "generated": "2026-08-27T00:00:00Z",
  "scan": {
    "level": "source-imports",
    "scanner": "deppolicy-go v0.1.0",
    "coverage": ["go:github.com/plexusone/*"]
  },
  "components": [
    { "id": "go:github.com/plexusone/dashforge", "kind": "go-module" }
  ],
  "dependencies": [
    {
      "source": "go:github.com/plexusone/dashforge",
      "relation": "depends_on",
      "target": "go:github.com/plexusone/omniagent",
      "evidence": [
        { "kind": "source-import", "file": "internal/agent/client.go", "line": 12 }
      ]
    }
  ]
}`

func TestParse_Valid(t *testing.T) {
	g, err := Parse([]byte(exampleGraphJSON))
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}
	if g.Scan.Level != ScanLevelSourceImports {
		t.Errorf("Scan.Level = %q, want %q", g.Scan.Level, ScanLevelSourceImports)
	}
	if !g.HasEdge("go:github.com/plexusone/dashforge", "go:github.com/plexusone/omniagent") {
		t.Error("expected parsed edge to be present")
	}
}

func TestParse_InvalidJSON(t *testing.T) {
	if _, err := Parse([]byte("{not json")); err == nil {
		t.Fatal("expected error for malformed JSON")
	}
}

func TestParse_StructurallyInvalid(t *testing.T) {
	if _, err := Parse([]byte(`{"scan":{"level":"bogus","scanner":"x"}}`)); err == nil {
		t.Fatal("expected error for invalid scan level")
	}
}
