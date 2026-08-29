package golang

import (
	"os"
	"path/filepath"
	"testing"
)

const testGoMod = `module github.com/plexusone/dashforge

go 1.26

require (
	github.com/plexusone/omniagent v1.2.0
	github.com/google/uuid v1.6.0
	github.com/plexusone/omnillm v0.8.0 // indirect
)
`

func TestParseManifest(t *testing.T) {
	moduleLocator, deps, err := ParseManifest("go.mod", []byte(testGoMod))
	if err != nil {
		t.Fatalf("ParseManifest() error = %v", err)
	}

	if want := "go:github.com/plexusone/dashforge"; moduleLocator != want {
		t.Errorf("moduleLocator = %q, want %q", moduleLocator, want)
	}

	if len(deps) != 2 {
		t.Fatalf("got %d deps, want 2 (indirect require must be excluded): %+v", len(deps), deps)
	}

	byTarget := make(map[string]bool, len(deps))
	for _, d := range deps {
		if d.Source != moduleLocator {
			t.Errorf("dependency source = %q, want %q", d.Source, moduleLocator)
		}
		if d.Relation != "depends_on" {
			t.Errorf("dependency relation = %q, want depends_on", d.Relation)
		}
		if len(d.Evidence) != 1 || d.Evidence[0].Kind != "manifest" || d.Evidence[0].Source != "go.mod" {
			t.Errorf("unexpected evidence: %+v", d.Evidence)
		}
		byTarget[d.Target] = true
	}

	if !byTarget["go:github.com/plexusone/omniagent"] {
		t.Error("expected direct require omniagent to be present")
	}
	if !byTarget["go:github.com/google/uuid"] {
		t.Error("expected direct require uuid to be present")
	}
	if byTarget["go:github.com/plexusone/omnillm"] {
		t.Error("indirect require omnillm must be excluded")
	}
}

func TestParseManifest_NoModuleDirective(t *testing.T) {
	if _, _, err := ParseManifest("go.mod", []byte("go 1.26\n")); err == nil {
		t.Fatal("expected error for go.mod without a module directive")
	}
}

func TestParseManifest_Malformed(t *testing.T) {
	if _, _, err := ParseManifest("go.mod", []byte("not a go.mod file {{{")); err == nil {
		t.Fatal("expected error for malformed go.mod")
	}
}

func TestManifestDependencies_ReadsFile(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "go.mod"), []byte(testGoMod), 0o600); err != nil {
		t.Fatalf("write go.mod: %v", err)
	}

	moduleLocator, deps, err := ManifestDependencies(dir)
	if err != nil {
		t.Fatalf("ManifestDependencies() error = %v", err)
	}
	if want := "go:github.com/plexusone/dashforge"; moduleLocator != want {
		t.Errorf("moduleLocator = %q, want %q", moduleLocator, want)
	}
	if len(deps) != 2 {
		t.Errorf("got %d deps, want 2", len(deps))
	}
}

func TestManifestDependencies_MissingFile(t *testing.T) {
	if _, _, err := ManifestDependencies(t.TempDir()); err == nil {
		t.Fatal("expected error when go.mod is missing")
	}
}
