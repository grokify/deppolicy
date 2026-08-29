package policy

import "testing"

func validPolicy() Policy {
	return Policy{
		SchemaVersion: "1",
		Scope: Scope{
			Include: []string{"go:github.com/plexusone/*"},
		},
		Components: map[string]ComponentDeclaration{
			"go:github.com/plexusone/omnillm": {Status: "foundation"},
		},
		Relationships: []Relationship{
			{
				Source:   "go:github.com/plexusone/dashforge",
				Relation: RelationCanDependOn,
				Target:   "go:github.com/plexusone/omniagent",
				Reason:   "dashforge consumes the shared agent runtime",
			},
		},
		Exceptions: []Exception{
			{
				Source:  "go:github.com/plexusone/a",
				Target:  "go:github.com/plexusone/b",
				Effect:  EffectAllow,
				Expires: "2099-12-31",
				Reason:  "migration",
			},
		},
	}
}

func TestPolicy_Validate_Valid(t *testing.T) {
	if err := validPolicy().Validate(); err != nil {
		t.Fatalf("Validate() error = %v, want nil", err)
	}
}

func TestPolicy_Validate_MissingSchemaVersion(t *testing.T) {
	p := validPolicy()
	p.SchemaVersion = ""
	if err := p.Validate(); err == nil {
		t.Fatal("expected error for missing schemaVersion")
	}
}

func TestPolicy_Validate_EmptyScope(t *testing.T) {
	p := validPolicy()
	p.Scope.Include = nil
	if err := p.Validate(); err == nil {
		t.Fatal("expected error for empty scope.include")
	}
}

func TestPolicy_Validate_ComponentOutOfScope(t *testing.T) {
	p := validPolicy()
	p.Components["go:github.com/other/x"] = ComponentDeclaration{Status: "foundation"}
	if err := p.Validate(); err == nil {
		t.Fatal("expected error for out-of-scope component declaration")
	}
}

func TestPolicy_Validate_RelationshipUnsupportedRelation(t *testing.T) {
	p := validPolicy()
	p.Relationships[0].Relation = "can_call"
	if err := p.Validate(); err == nil {
		t.Fatal("expected error for unsupported relation")
	}
}

func TestPolicy_Validate_RelationshipOutOfScope(t *testing.T) {
	p := validPolicy()
	p.Relationships[0].Target = "go:github.com/other/x"
	if err := p.Validate(); err == nil {
		t.Fatal("expected error for out-of-scope relationship target")
	}
}

func TestPolicy_Validate_DuplicateRelationship(t *testing.T) {
	p := validPolicy()
	p.Relationships = append(p.Relationships, p.Relationships[0])
	if err := p.Validate(); err == nil {
		t.Fatal("expected error for duplicate relationship")
	}
}

func TestPolicy_Validate_ExceptionMissingReason(t *testing.T) {
	p := validPolicy()
	p.Exceptions[0].Reason = ""
	if err := p.Validate(); err == nil {
		t.Fatal("expected error for exception missing reason")
	}
}

func TestPolicy_Validate_ExceptionUnsupportedEffect(t *testing.T) {
	p := validPolicy()
	p.Exceptions[0].Effect = "deny"
	if err := p.Validate(); err == nil {
		t.Fatal("expected error for unsupported exception effect")
	}
}

func TestPolicy_Validate_ExceptionBadExpiry(t *testing.T) {
	p := validPolicy()
	p.Exceptions[0].Expires = "not-a-date"
	if err := p.Validate(); err == nil {
		t.Fatal("expected error for malformed expires date")
	}
}

func TestPolicy_Validate_DuplicateException(t *testing.T) {
	p := validPolicy()
	p.Exceptions = append(p.Exceptions, p.Exceptions[0])
	if err := p.Validate(); err == nil {
		t.Fatal("expected error for duplicate exception")
	}
}
