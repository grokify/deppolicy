package cli

import (
	"context"
	"path/filepath"
	"testing"
	"time"
)

func TestScan_NoPolicy_BootstrapMode(t *testing.T) {
	root := t.TempDir()
	writeCliFile(t, filepath.Join(root, "consumer/go.mod"), "module github.com/orgA/consumer\n\ngo 1.26\n")
	writeCliFile(t, filepath.Join(root, "consumer/main.go"), `package main

import "github.com/orgA/target"

func main() { _ = target.X }
`)
	writeCliFile(t, filepath.Join(root, "target/go.mod"), "module github.com/orgA/target\n\ngo 1.26\n")
	writeCliFile(t, filepath.Join(root, "target/main.go"), "package main\n\nfunc main() {}\n")

	result, err := Scan(context.Background(), ScanOptions{
		RootDirs: []string{root},
		Scanner:  "test",
		AsOf:     time.Now(),
	})
	if err != nil {
		t.Fatalf("Scan() error = %v", err)
	}
	if result.Policy != nil {
		t.Error("Policy = non-nil, want nil when PolicyPath is empty (bootstrap mode)")
	}
	if len(result.Graph.Components) != 2 {
		t.Errorf("Components = %+v, want 2", result.Graph.Components)
	}
	if !result.Graph.HasEdge("go:github.com/orgA/consumer", "go:github.com/orgA/target") {
		t.Errorf("expected consumer -> target edge, deps = %+v", result.Graph.Dependencies)
	}
}

func TestScan_WithPolicy_ExcludesOutOfScope(t *testing.T) {
	root := t.TempDir()
	writeCliFile(t, filepath.Join(root, "consumer/go.mod"), "module github.com/orgA/consumer\n\ngo 1.26\n")
	writeCliFile(t, filepath.Join(root, "consumer/main.go"), "package main\n\nfunc main() {}\n")
	writeCliFile(t, filepath.Join(root, "sandbox-x/go.mod"), "module github.com/orgA/sandbox-x\n\ngo 1.26\n")
	writeCliFile(t, filepath.Join(root, "sandbox-x/main.go"), "package main\n\nfunc main() {}\n")

	policyPath := filepath.Join(root, "policy.json")
	writeCliFile(t, policyPath, `{
  "schemaVersion": "1",
  "scope": { "include": ["go:github.com/orgA/*"], "exclude": ["go:github.com/orgA/sandbox-*"] }
}`)

	result, err := Scan(context.Background(), ScanOptions{
		RootDirs:   []string{root},
		PolicyPath: policyPath,
		Scanner:    "test",
		AsOf:       time.Now(),
	})
	if err != nil {
		t.Fatalf("Scan() error = %v", err)
	}
	if result.Policy == nil {
		t.Fatal("Policy = nil, want non-nil when PolicyPath is set")
	}
	if len(result.Graph.Components) != 1 || result.Graph.Components[0].ID != "go:github.com/orgA/consumer" {
		t.Errorf("Components = %+v, want only consumer", result.Graph.Components)
	}
	if len(result.Reports) != 1 || len(result.Reports[0].Excluded) != 1 {
		t.Errorf("Reports = %+v, want 1 report with 1 excluded entry", result.Reports)
	}
}

func TestScan_MultipleRoots_CombinesIntoOneGraph(t *testing.T) {
	rootA := t.TempDir()
	rootB := t.TempDir()

	writeCliFile(t, filepath.Join(rootA, "consumer/go.mod"), "module github.com/orgA/consumer\n\ngo 1.26\n")
	writeCliFile(t, filepath.Join(rootA, "consumer/main.go"), `package main

import "github.com/orgB/provider/deep/pkg"

func main() { _ = pkg.X }
`)
	writeCliFile(t, filepath.Join(rootB, "provider/go.mod"), "module github.com/orgB/provider\n\ngo 1.26\n")
	writeCliFile(t, filepath.Join(rootB, "provider/main.go"), "package main\n\nfunc main() {}\n")

	result, err := Scan(context.Background(), ScanOptions{
		RootDirs: []string{rootA, rootB},
		Scanner:  "test",
		AsOf:     time.Now(),
	})
	if err != nil {
		t.Fatalf("Scan() error = %v", err)
	}
	if len(result.Reports) != 2 {
		t.Fatalf("Reports = %+v, want one per root", result.Reports)
	}
	if !result.Graph.HasEdge("go:github.com/orgA/consumer", "go:github.com/orgB/provider") {
		t.Errorf("expected the cross-root deep import to resolve to its owning module, deps = %+v", result.Graph.Dependencies)
	}
}

func TestScan_MissingPolicyFile(t *testing.T) {
	root := t.TempDir()
	_, err := Scan(context.Background(), ScanOptions{
		RootDirs:   []string{root},
		PolicyPath: filepath.Join(root, "no-such-policy.json"),
		Scanner:    "test",
		AsOf:       time.Now(),
	})
	if err == nil {
		t.Fatal("expected an error for a missing policy file")
	}
}

func TestScan_EmptyRoot(t *testing.T) {
	result, err := Scan(context.Background(), ScanOptions{RootDirs: []string{t.TempDir()}, Scanner: "test", AsOf: time.Now()})
	if err != nil {
		t.Fatalf("Scan() error = %v", err)
	}
	if len(result.Graph.Components) != 0 {
		t.Errorf("Components = %+v, want none", result.Graph.Components)
	}
}
