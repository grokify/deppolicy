package golang

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	"github.com/grokify/deppolicy/policy"
)

func buildScanFixture(t *testing.T) string {
	t.Helper()
	root := t.TempDir()

	writeFile(t, filepath.Join(root, "orgA/consumer/go.mod"), "module github.com/orgA/consumer\n\ngo 1.26\n")
	writeFile(t, filepath.Join(root, "orgA/consumer/main.go"), `package main

import (
	"github.com/orgA/provider/client"
	"github.com/google/uuid"
)

func main() {
	_ = client.New
	_ = uuid.New
}
`)

	writeFile(t, filepath.Join(root, "orgA/provider/go.mod"), "module github.com/orgA/provider\n\ngo 1.26\n")
	writeFile(t, filepath.Join(root, "orgA/provider/client/client.go"), "package client\n\nfunc New() {}\n")

	writeFile(t, filepath.Join(root, "orgA/sandbox-scratch/go.mod"), "module github.com/orgA/sandbox-scratch\n\ngo 1.26\n")
	writeFile(t, filepath.Join(root, "orgA/sandbox-scratch/main.go"), "package main\n\nfunc main() {}\n")

	return root
}

func TestScan_EndToEnd(t *testing.T) {
	root := buildScanFixture(t)
	generated := time.Date(2026, 8, 27, 0, 0, 0, 0, time.UTC)

	g, report, err := Scan(context.Background(), root, generated, ScanOptions{
		Scanner: "deppolicy-go vtest",
		Workers: 2,
	})
	if err != nil {
		t.Fatalf("Scan() error = %v", err)
	}

	if !g.Generated.Equal(generated) {
		t.Errorf("Generated = %v, want %v", g.Generated, generated)
	}
	if g.Scan.Level != "source-imports" {
		t.Errorf("Scan.Level = %q, want source-imports", g.Scan.Level)
	}
	if g.Scan.Scanner != "deppolicy-go vtest" {
		t.Errorf("Scan.Scanner = %q", g.Scan.Scanner)
	}
	if len(g.Scan.Coverage) != 1 || g.Scan.Coverage[0] != root {
		t.Errorf("Scan.Coverage = %v, want [%s]", g.Scan.Coverage, root)
	}

	if len(g.Components) != 3 {
		t.Fatalf("Components = %+v, want 3 (consumer, provider, sandbox-scratch)", g.Components)
	}

	if !g.HasEdge("go:github.com/orgA/consumer", "go:github.com/orgA/provider") {
		t.Errorf("expected consumer -> provider edge (resolved via registry from deep import), deps = %+v", g.Dependencies)
	}
	if !g.HasEdge("go:github.com/orgA/consumer", "go:github.com/google/uuid") {
		t.Errorf("expected consumer -> uuid edge, deps = %+v", g.Dependencies)
	}
	if g.HasEdge("go:github.com/orgA/consumer", "go:github.com/orgA/provider/client") {
		t.Error("deep import must resolve to the owning module locator, not remain unresolved")
	}

	if err := g.Validate(); err != nil {
		t.Errorf("resulting graph fails Validate(): %v", err)
	}

	if len(report.Included) != 3 {
		t.Errorf("report.Included has %d entries, want 3", len(report.Included))
	}
}

func TestScan_ScopeExcludesModule(t *testing.T) {
	root := buildScanFixture(t)

	scope := &policy.Policy{
		SchemaVersion: "1",
		Scope: policy.Scope{
			Include: []string{"go:github.com/orgA/*"},
			Exclude: []string{"go:github.com/orgA/sandbox-*"},
		},
	}

	g, report, err := Scan(context.Background(), root, time.Now(), ScanOptions{
		Scanner: "deppolicy-go vtest",
		Workers: 2,
		Scope:   scope,
	})
	if err != nil {
		t.Fatalf("Scan() error = %v", err)
	}

	for _, c := range g.Components {
		if c.ID == "go:github.com/orgA/sandbox-scratch" {
			t.Error("excluded module must not appear in graph.Components")
		}
	}

	found := false
	for _, line := range g.Scan.Excluded {
		if line != "" {
			found = true
		}
	}
	if !found || len(report.Excluded) != 1 {
		t.Errorf("expected sandbox-scratch to be recorded as excluded, report.Excluded = %+v, g.Scan.Excluded = %v", report.Excluded, g.Scan.Excluded)
	}
}

func TestScan_EmptyRoot(t *testing.T) {
	root := t.TempDir()

	g, _, err := Scan(context.Background(), root, time.Now(), ScanOptions{Scanner: "deppolicy-go vtest"})
	if err != nil {
		t.Fatalf("Scan() error = %v", err)
	}
	if len(g.Components) != 0 || len(g.Dependencies) != 0 {
		t.Errorf("Scan(empty root) = %+v, want empty graph", g)
	}
}

func TestGitCommit_NonGitDirectory(t *testing.T) {
	dir := t.TempDir()
	if got := gitCommit(context.Background(), dir); got != "" {
		t.Errorf("gitCommit(non-git dir) = %q, want empty string", got)
	}
}
