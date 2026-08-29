package policy

import (
	"testing"
	"time"
)

func TestException_IsExpired(t *testing.T) {
	e := Exception{Expires: "2026-12-31"}

	before := time.Date(2026, 12, 31, 0, 0, 0, 0, time.UTC)
	if e.IsExpired(before) {
		t.Error("IsExpired() on expiry date = true, want false (still valid through end of day)")
	}

	after := time.Date(2027, 1, 1, 0, 0, 0, 0, time.UTC)
	if !e.IsExpired(after) {
		t.Error("IsExpired() after expiry date = false, want true")
	}

	malformed := Exception{Expires: "not-a-date"}
	if !malformed.IsExpired(before) {
		t.Error("IsExpired() with unparseable expires = false, want true (fail closed)")
	}
}

func TestPolicy_HasRelationship(t *testing.T) {
	p := validPolicy()
	if !p.HasRelationship("go:github.com/plexusone/dashforge", "go:github.com/plexusone/omniagent") {
		t.Error("HasRelationship() = false for declared relationship, want true")
	}
	if p.HasRelationship("go:github.com/plexusone/dashforge", "go:github.com/plexusone/omnillm") {
		t.Error("HasRelationship() = true for undeclared relationship, want false")
	}
}

func TestPolicy_ActiveException(t *testing.T) {
	p := validPolicy()
	asOf := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)

	if _, ok := p.ActiveException("go:github.com/plexusone/a", "go:github.com/plexusone/b", asOf); !ok {
		t.Error("ActiveException() ok = false for unexpired exception, want true")
	}

	future := time.Date(2100, 1, 1, 0, 0, 0, 0, time.UTC)
	if _, ok := p.ActiveException("go:github.com/plexusone/a", "go:github.com/plexusone/b", future); ok {
		t.Error("ActiveException() ok = true for expired exception, want false")
	}
}

func TestPolicy_ComponentDeclaration(t *testing.T) {
	p := validPolicy()
	decl, ok := p.ComponentDeclaration("go:github.com/plexusone/omnillm")
	if !ok {
		t.Fatal("ComponentDeclaration() ok = false, want true")
	}
	if decl.Status != "foundation" {
		t.Errorf("ComponentDeclaration().Status = %q, want %q", decl.Status, "foundation")
	}

	if _, ok := p.ComponentDeclaration("go:github.com/plexusone/undeclared"); ok {
		t.Error("ComponentDeclaration() ok = true for undeclared locator, want false")
	}
}
