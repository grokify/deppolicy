package component

import "testing"

func TestNewRegistry_DuplicateID(t *testing.T) {
	_, err := NewRegistry([]Component{
		{ID: "a", Locator: "go:github.com/plexusone/a"},
		{ID: "a", Locator: "go:github.com/plexusone/a2"},
	})
	if err == nil {
		t.Fatal("expected error for duplicate component id, got nil")
	}
}

func TestNewRegistry_DuplicateLocator(t *testing.T) {
	_, err := NewRegistry([]Component{
		{ID: "a", Locator: "go:github.com/plexusone/omnillm"},
		{ID: "b", Locator: "go:github.com/plexusone/omnillm"},
	})
	if err == nil {
		t.Fatal("expected error for duplicate locator, got nil")
	}
}

func TestRegistry_Resolve(t *testing.T) {
	reg, err := NewRegistry([]Component{
		{ID: "omnillm", Kind: KindGoModule, Locator: "go:github.com/plexusone/omnillm"},
		{ID: "omnillm2", Kind: KindGoModule, Locator: "go:github.com/plexusone/omnillm2"},
		{ID: "mogo-database", Kind: KindGoPackage, Locator: "go:github.com/grokify/mogo/database", Status: StatusGoverned},
		{ID: "mogo", Kind: KindGoModule, Locator: "go:github.com/grokify/mogo", Status: StatusFoundation},
	})
	if err != nil {
		t.Fatalf("NewRegistry() error = %v", err)
	}

	tests := []struct {
		name    string
		locator string
		wantID  ID
		wantOK  bool
	}{
		{"exact module match", "go:github.com/plexusone/omnillm", "omnillm", true},
		{"module subpackage matches owning module", "go:github.com/plexusone/omnillm/providers/openai", "omnillm", true},
		{"prefix collision does not false-match", "go:github.com/plexusone/omnillm2", "omnillm2", true},
		{"prefix collision subpackage resolves correctly", "go:github.com/plexusone/omnillm2/foo", "omnillm2", true},
		{"declared proto-module wins over parent module", "go:github.com/grokify/mogo/database/dolt", "mogo-database", true},
		{"parent module governs everything else under it", "go:github.com/grokify/mogo/log/slogutil", "mogo", true},
		{"unrelated locator does not resolve", "go:github.com/google/uuid", "", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, ok := reg.Resolve(tt.locator)
			if ok != tt.wantOK {
				t.Fatalf("Resolve(%q) ok = %v, want %v", tt.locator, ok, tt.wantOK)
			}
			if ok && got.ID != tt.wantID {
				t.Errorf("Resolve(%q) = %q, want %q", tt.locator, got.ID, tt.wantID)
			}
		})
	}
}

func TestRegistry_Get(t *testing.T) {
	reg, err := NewRegistry([]Component{
		{ID: "omnillm", Kind: KindGoModule, Locator: "go:github.com/plexusone/omnillm"},
	})
	if err != nil {
		t.Fatalf("NewRegistry() error = %v", err)
	}

	if _, ok := reg.Get("omnillm"); !ok {
		t.Error("Get(\"omnillm\") ok = false, want true")
	}
	if _, ok := reg.Get("missing"); ok {
		t.Error("Get(\"missing\") ok = true, want false")
	}
}

func TestRegistry_All(t *testing.T) {
	components := []Component{
		{ID: "a", Locator: "go:github.com/plexusone/a"},
		{ID: "b", Locator: "go:github.com/plexusone/b"},
	}
	reg, err := NewRegistry(components)
	if err != nil {
		t.Fatalf("NewRegistry() error = %v", err)
	}

	all := reg.All()
	if len(all) != len(components) {
		t.Fatalf("All() returned %d components, want %d", len(all), len(components))
	}

	// Mutating the returned slice must not affect the registry.
	all[0].ID = "mutated"
	if _, ok := reg.Get("a"); !ok {
		t.Error("mutating All() result affected registry internal state")
	}
}
