package golang

import (
	"os"
	"path/filepath"
	"sort"
	"testing"

	"github.com/grokify/deppolicy/component"
)

func writeTestModule(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()

	files := map[string]string{
		"go.mod": "module github.com/plexusone/dashforge\n\ngo 1.26\n",

		"internal/util/util.go": "package util\n\nfunc Foo() {}\n",

		"internal/agent/client.go": `package agent

import (
	"context"

	"github.com/google/uuid"
	"github.com/plexusone/dashforge/internal/util"
	"github.com/plexusone/omniagent/client"
)

var (
	_ = context.Background
	_ = uuid.New
	_ = util.Foo
	_ = client.New
)
`,

		"internal/agent/client_test.go": `package agent

import "github.com/plexusone/omnillm"

var _ = omnillm.Model{}
`,

		"vendor/github.com/other/pkg/pkg.go": `package pkg

import "github.com/should/not/appear"

var _ = "unused"
`,

		"testdata/sample.go": `package testdata

import "github.com/should/not/appear/either"

var _ = "unused"
`,
	}

	for rel, content := range files {
		full := filepath.Join(dir, rel)
		if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
			t.Fatalf("mkdir for %s: %v", rel, err)
		}
		if err := os.WriteFile(full, []byte(content), 0o600); err != nil {
			t.Fatalf("write %s: %v", rel, err)
		}
	}

	return dir
}

func TestImportDependencies_WithRegistry(t *testing.T) {
	dir := writeTestModule(t)

	registry, err := component.NewRegistry([]component.Component{
		{ID: "omniagent", Kind: component.KindGoModule, Locator: "go:github.com/plexusone/omniagent"},
	})
	if err != nil {
		t.Fatalf("NewRegistry() error = %v", err)
	}

	moduleLocator, deps, err := ImportDependencies(dir, registry)
	if err != nil {
		t.Fatalf("ImportDependencies() error = %v", err)
	}

	if want := "go:github.com/plexusone/dashforge"; moduleLocator != want {
		t.Errorf("moduleLocator = %q, want %q", moduleLocator, want)
	}

	var gotTargets []string
	for _, d := range deps {
		gotTargets = append(gotTargets, d.Target)
		if d.Source != moduleLocator {
			t.Errorf("dependency source = %q, want %q", d.Source, moduleLocator)
		}
	}
	sort.Strings(gotTargets)

	want := []string{"go:github.com/google/uuid", "go:github.com/plexusone/omniagent"}
	if len(gotTargets) != len(want) {
		t.Fatalf("targets = %v, want %v", gotTargets, want)
	}
	for i := range want {
		if gotTargets[i] != want[i] {
			t.Errorf("targets = %v, want %v", gotTargets, want)
			break
		}
	}
}

func TestImportDependencies_NoRegistry_UsesRawImportPath(t *testing.T) {
	dir := writeTestModule(t)

	_, deps, err := ImportDependencies(dir, nil)
	if err != nil {
		t.Fatalf("ImportDependencies() error = %v", err)
	}

	found := false
	for _, d := range deps {
		if d.Target == "go:github.com/plexusone/omniagent/client" {
			found = true
		}
	}
	if !found {
		t.Errorf("expected unresolved deep import path when registry is nil, got %+v", deps)
	}
}

func TestImportDependencies_ExcludesSelfImport(t *testing.T) {
	dir := writeTestModule(t)

	_, deps, err := ImportDependencies(dir, nil)
	if err != nil {
		t.Fatalf("ImportDependencies() error = %v", err)
	}

	for _, d := range deps {
		if d.Target == "go:github.com/plexusone/dashforge/internal/util" {
			t.Errorf("self-import must not be recorded as a dependency edge: %+v", d)
		}
	}
}

func TestImportDependencies_ExcludesTestOnlyImport(t *testing.T) {
	dir := writeTestModule(t)

	_, deps, err := ImportDependencies(dir, nil)
	if err != nil {
		t.Fatalf("ImportDependencies() error = %v", err)
	}

	for _, d := range deps {
		if d.Target == "go:github.com/plexusone/omnillm" {
			t.Errorf("test-only import must not be recorded: %+v", d)
		}
	}
}

func TestImportDependencies_ExcludesVendorAndTestdata(t *testing.T) {
	dir := writeTestModule(t)

	_, deps, err := ImportDependencies(dir, nil)
	if err != nil {
		t.Fatalf("ImportDependencies() error = %v", err)
	}

	for _, d := range deps {
		if d.Target == "go:github.com/should/not/appear" || d.Target == "go:github.com/should/not/appear/either" {
			t.Errorf("vendor/testdata imports must not be recorded: %+v", d)
		}
	}
}

func TestImportDependencies_EvidenceLocation(t *testing.T) {
	dir := writeTestModule(t)

	_, deps, err := ImportDependencies(dir, nil)
	if err != nil {
		t.Fatalf("ImportDependencies() error = %v", err)
	}

	for _, d := range deps {
		if d.Target != "go:github.com/google/uuid" {
			continue
		}
		if len(d.Evidence) != 1 {
			t.Fatalf("expected 1 evidence entry, got %d", len(d.Evidence))
		}
		ev := d.Evidence[0]
		if ev.Kind != "source-import" {
			t.Errorf("evidence kind = %q, want source-import", ev.Kind)
		}
		if ev.File != filepath.Join("internal", "agent", "client.go") {
			t.Errorf("evidence file = %q, want internal/agent/client.go", ev.File)
		}
		if ev.Line == 0 {
			t.Error("evidence line = 0, want a real line number")
		}
		return
	}
	t.Fatal("expected to find uuid dependency with evidence")
}

func TestIsStdlib(t *testing.T) {
	tests := map[string]bool{
		"context":                  true,
		"encoding/json":            true,
		"github.com/google/uuid":   false,
		"golang.org/x/mod/modfile": false,
	}
	for importPath, want := range tests {
		if got := IsStdlib(importPath); got != want {
			t.Errorf("IsStdlib(%q) = %v, want %v", importPath, got, want)
		}
	}
}
