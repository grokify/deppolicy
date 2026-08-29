package golang

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/grokify/deppolicy/policy"
)

func writeFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("mkdir for %s: %v", path, err)
	}
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatalf("write %s: %v", path, err)
	}
}

func buildDiscoveryFixture(t *testing.T) string {
	t.Helper()
	root := t.TempDir()

	writeFile(t, filepath.Join(root, "orgA/repo1/go.mod"), "module github.com/orgA/repo1\n\ngo 1.26\n")
	writeFile(t, filepath.Join(root, "orgA/repo1/main.go"), "package main\n\nfunc main() {}\n")

	// Nested module: must still be discovered.
	writeFile(t, filepath.Join(root, "orgA/repo1/tools/go.mod"), "module github.com/orgA/repo1/tools\n\ngo 1.26\n")

	// Vendor copy with its own go.mod: must never be discovered.
	writeFile(t, filepath.Join(root, "orgA/repo1/vendor/other/go.mod"), "module github.com/other/vendored\n\ngo 1.26\n")

	// Sandbox repo: discoverable, but excludable by scope.
	writeFile(t, filepath.Join(root, "orgA/sandbox-foo/go.mod"), "module github.com/orgA/sandbox-foo\n\ngo 1.26\n")

	// Malformed go.mod: must not abort the walk.
	writeFile(t, filepath.Join(root, "orgA/broken/go.mod"), "this is not { a valid go.mod")

	// Directory with no go.mod at all: silently skipped.
	writeFile(t, filepath.Join(root, "orgA/notamodule/README.md"), "# nothing here\n")

	return root
}

func TestDiscoverModules_NoScope(t *testing.T) {
	root := buildDiscoveryFixture(t)

	report, err := DiscoverModules(root, nil)
	if err != nil {
		t.Fatalf("DiscoverModules() error = %v", err)
	}

	locators := make(map[string]bool, len(report.Included))
	for _, m := range report.Included {
		locators[m.ModuleLocator] = true
	}

	for _, want := range []string{
		"go:github.com/orgA/repo1",
		"go:github.com/orgA/repo1/tools",
		"go:github.com/orgA/sandbox-foo",
	} {
		if !locators[want] {
			t.Errorf("expected %q to be discovered and included, got %+v", want, report.Included)
		}
	}

	if locators["go:github.com/other/vendored"] {
		t.Error("vendored go.mod must never be discovered")
	}

	if len(report.Errored) != 1 || report.Errored[0].Dir != filepath.Join(root, "orgA/broken") {
		t.Errorf("Errored = %+v, want exactly the broken module", report.Errored)
	}

	if len(report.Excluded) != 0 {
		t.Errorf("Excluded = %+v, want none when scope is nil", report.Excluded)
	}
}

func TestDiscoverModules_WithScope(t *testing.T) {
	root := buildDiscoveryFixture(t)

	scope := &policy.Policy{
		SchemaVersion: "1",
		Scope: policy.Scope{
			Include: []string{"go:github.com/orgA/*"},
			Exclude: []string{"go:github.com/orgA/sandbox-*"},
		},
	}

	report, err := DiscoverModules(root, scope)
	if err != nil {
		t.Fatalf("DiscoverModules() error = %v", err)
	}

	included := make(map[string]bool, len(report.Included))
	for _, m := range report.Included {
		included[m.ModuleLocator] = true
	}
	if !included["go:github.com/orgA/repo1"] {
		t.Error("expected repo1 to be included")
	}
	if included["go:github.com/orgA/sandbox-foo"] {
		t.Error("expected sandbox-foo to be excluded, not included")
	}

	if len(report.Excluded) != 1 {
		t.Fatalf("Excluded = %+v, want exactly 1 entry", report.Excluded)
	}
	if report.Excluded[0].ModuleLocator != "go:github.com/orgA/sandbox-foo" {
		t.Errorf("Excluded[0] = %+v, want sandbox-foo", report.Excluded[0])
	}
	if report.Excluded[0].Reason == "" {
		t.Error("Excluded[0].Reason is empty, want an explanation")
	}
}

func TestDiscoveryReport_ExcludedSummary(t *testing.T) {
	report := DiscoveryReport{
		Excluded: []ExcludedModule{
			{Dir: "/x/sandbox-foo", ModuleLocator: "go:github.com/orgA/sandbox-foo", Reason: "out of policy scope"},
		},
	}

	summary := report.ExcludedSummary()
	if len(summary) != 1 {
		t.Fatalf("ExcludedSummary() = %v, want 1 line", summary)
	}
	if summary[0] == "" {
		t.Error("ExcludedSummary() line is empty")
	}
}

func TestDiscoverModules_EmptyRoot(t *testing.T) {
	root := t.TempDir()
	report, err := DiscoverModules(root, nil)
	if err != nil {
		t.Fatalf("DiscoverModules() error = %v", err)
	}
	if len(report.Included) != 0 || len(report.Excluded) != 0 || len(report.Errored) != 0 {
		t.Errorf("DiscoverModules(empty root) = %+v, want all empty", report)
	}
}
