package main

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestDiffCmd_Wiring is a thin smoke test of the cobra wiring itself,
// using a fixture with no violations at all, so it never triggers
// os.Exit(1) inside RunE. Ratchet classification behavior is covered by
// the cli package's own tests.
func TestDiffCmd_Wiring(t *testing.T) {
	root := t.TempDir()

	moduleDir := filepath.Join(root, "solo")
	if err := os.MkdirAll(moduleDir, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(moduleDir, "go.mod"), []byte("module github.com/orgA/solo\n\ngo 1.26\n"), 0o600); err != nil {
		t.Fatalf("write go.mod: %v", err)
	}
	if err := os.WriteFile(filepath.Join(moduleDir, "main.go"), []byte("package main\n\nfunc main() {}\n"), 0o600); err != nil {
		t.Fatalf("write main.go: %v", err)
	}

	policyPath := filepath.Join(root, "policy.json")
	policyJSON := `{"schemaVersion":"1","scope":{"include":["go:github.com/orgA/*"]}}`
	if err := os.WriteFile(policyPath, []byte(policyJSON), 0o600); err != nil {
		t.Fatalf("write policy.json: %v", err)
	}

	baselinePath := filepath.Join(root, "baseline.json")
	baselineJSON := `{"generated":"2026-01-01T00:00:00Z","scan":{"level":"source-imports","scanner":"test"}}`
	if err := os.WriteFile(baselinePath, []byte(baselineJSON), 0o600); err != nil {
		t.Fatalf("write baseline.json: %v", err)
	}

	cmd := newRootCmd()
	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetArgs([]string{"diff", "--policy", policyPath, "--baseline", baselinePath, moduleDir})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	if !strings.Contains(out.String(), "0 conforming, 0 new violation(s), 0 existing violation(s), 0 unused allowance(s)") {
		t.Errorf("unexpected output: %s", out.String())
	}
	if !strings.Contains(out.String(), "graph: 0 edge(s) added, 0 removed, 0 unchanged") {
		t.Errorf("unexpected output: %s", out.String())
	}
}

func TestDiffCmd_RequiresBaselineFlag(t *testing.T) {
	cmd := newRootCmd()
	cmd.SetArgs([]string{"diff", "--policy", "policy.json", t.TempDir()})
	cmd.SilenceUsage = true
	cmd.SilenceErrors = true

	if err := cmd.Execute(); err == nil {
		t.Fatal("expected an error when --baseline is omitted")
	}
}
