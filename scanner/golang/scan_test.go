package golang

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

func writeModule(t *testing.T, root, name, importPath string, requires ...string) string {
	t.Helper()
	dir := filepath.Join(root, name)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatalf("mkdir %s: %v", dir, err)
	}

	content := "module " + importPath + "\n\ngo 1.26\n"
	if err := os.WriteFile(filepath.Join(dir, "go.mod"), []byte(content), 0o600); err != nil {
		t.Fatalf("write go.mod: %v", err)
	}

	var body string
	for _, req := range requires {
		body += "\t_ \"" + req + "\"\n"
	}
	main := "package main\n\nimport (\n" + body + ")\n\nfunc main() {}\n"
	if err := os.WriteFile(filepath.Join(dir, "main.go"), []byte(main), 0o600); err != nil {
		t.Fatalf("write main.go: %v", err)
	}

	return dir
}

func TestScanModules_Concurrent(t *testing.T) {
	root := t.TempDir()
	dirA := writeModule(t, root, "a", "github.com/plexusone/a", "github.com/plexusone/b")
	dirB := writeModule(t, root, "b", "github.com/plexusone/b", "github.com/plexusone/c")
	dirC := writeModule(t, root, "c", "github.com/plexusone/c")

	dirs := []string{dirA, dirB, dirC}
	results := ScanModules(context.Background(), dirs, nil, 2)

	if len(results) != len(dirs) {
		t.Fatalf("got %d results, want %d", len(results), len(dirs))
	}
	for i, r := range results {
		if r.Dir != dirs[i] {
			t.Errorf("results[%d].Dir = %q, want %q (results must preserve input order)", i, r.Dir, dirs[i])
		}
		if r.Err != nil {
			t.Errorf("results[%d].Err = %v, want nil", i, r.Err)
		}
	}

	if results[0].ModuleLocator != "go:github.com/plexusone/a" {
		t.Errorf("results[0].ModuleLocator = %q", results[0].ModuleLocator)
	}
	if len(results[0].Dependencies) != 1 || results[0].Dependencies[0].Target != "go:github.com/plexusone/b" {
		t.Errorf("results[0].Dependencies = %+v, want one edge to plexusone/b", results[0].Dependencies)
	}
}

func TestScanModules_PartialFailureDoesNotAbortOthers(t *testing.T) {
	root := t.TempDir()
	dirGood := writeModule(t, root, "good", "github.com/plexusone/good")

	dirBad := filepath.Join(root, "bad")
	if err := os.MkdirAll(dirBad, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	// No go.mod written -- ImportDependencies must fail for this one.

	results := ScanModules(context.Background(), []string{dirGood, dirBad}, nil, 2)

	if results[0].Err != nil {
		t.Errorf("results[0] (good module) Err = %v, want nil", results[0].Err)
	}
	if results[1].Err == nil {
		t.Error("results[1] (missing go.mod) Err = nil, want an error")
	}
}

func TestScanModules_EmptyInput(t *testing.T) {
	if got := ScanModules(context.Background(), nil, nil, 4); got != nil {
		t.Errorf("ScanModules(nil dirs) = %v, want nil", got)
	}
}

func TestScanModules_DefaultWorkers(t *testing.T) {
	root := t.TempDir()
	dir := writeModule(t, root, "solo", "github.com/plexusone/solo")

	// workers <= 0 must default sensibly rather than hang or panic.
	results := ScanModules(context.Background(), []string{dir}, nil, 0)
	if len(results) != 1 || results[0].Err != nil {
		t.Fatalf("ScanModules with workers=0 = %+v", results)
	}
}

func TestMergeResults(t *testing.T) {
	root := t.TempDir()
	dirA := writeModule(t, root, "a", "github.com/plexusone/a", "github.com/plexusone/shared")
	dirB := writeModule(t, root, "b", "github.com/plexusone/b", "github.com/plexusone/shared")

	results := ScanModules(context.Background(), []string{dirA, dirB}, nil, 2)
	deps, errs := MergeResults(results)

	if len(errs) != 0 {
		t.Errorf("errs = %v, want none", errs)
	}
	if len(deps) != 2 {
		t.Fatalf("deps = %+v, want 2 edges (a->shared, b->shared)", deps)
	}
}

func TestMergeResults_CollectsErrors(t *testing.T) {
	results := []ScanResult{
		{Dir: "/broken", Err: os.ErrNotExist},
	}
	deps, errs := MergeResults(results)
	if len(deps) != 0 {
		t.Errorf("deps = %v, want none", deps)
	}
	if len(errs) != 1 {
		t.Fatalf("errs = %v, want 1", errs)
	}
}
