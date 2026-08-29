package policy

import "testing"

const examplePolicyJSON = `{
  "schemaVersion": "1",
  "policyVersion": "2026.08.1",
  "scope": {
    "include": ["go:github.com/plexusone/*"],
    "exclude": ["go:github.com/plexusone/sandbox-*"]
  },
  "components": {
    "go:github.com/plexusone/omnillm": { "status": "foundation" }
  },
  "relationships": [
    {
      "source": "go:github.com/plexusone/dashforge",
      "relation": "can_depend_on",
      "target": "go:github.com/plexusone/omniagent",
      "reason": "dashforge consumes the shared agent runtime",
      "decision": "ADR-0042"
    }
  ],
  "exceptions": [
    {
      "source": "go:github.com/plexusone/a",
      "target": "go:github.com/plexusone/b",
      "effect": "allow",
      "expires": "2099-12-31",
      "reason": "migration"
    }
  ]
}`

func TestParse_Valid(t *testing.T) {
	p, err := Parse([]byte(examplePolicyJSON))
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}
	if p.SchemaVersion != "1" {
		t.Errorf("SchemaVersion = %q, want %q", p.SchemaVersion, "1")
	}
	if !p.HasRelationship("go:github.com/plexusone/dashforge", "go:github.com/plexusone/omniagent") {
		t.Error("expected parsed relationship to be present")
	}
	if !p.InScope("go:github.com/plexusone/omniagent") {
		t.Error("expected omniagent to be in scope")
	}
	if p.InScope("go:github.com/plexusone/sandbox-foo") {
		t.Error("expected sandbox-foo to be excluded from scope")
	}
}

func TestParse_InvalidJSON(t *testing.T) {
	if _, err := Parse([]byte("{not json")); err == nil {
		t.Fatal("expected error for malformed JSON")
	}
}

func TestParse_StructurallyInvalid(t *testing.T) {
	if _, err := Parse([]byte(`{"schemaVersion":"1","scope":{"include":[]}}`)); err == nil {
		t.Fatal("expected error for empty scope.include")
	}
}

func TestMustParse_PanicsOnError(t *testing.T) {
	defer func() {
		if recover() == nil {
			t.Fatal("expected MustParse to panic on invalid input")
		}
	}()
	MustParse([]byte("{not json"))
}
