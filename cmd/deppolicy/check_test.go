package main

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestCheckCmd_Wiring is a thin smoke test of the cobra wiring itself
// (flag names, argument binding) using a fixture with no policy
// violations, so it never triggers os.Exit(1) inside RunE. Evaluation
// behavior (violations, findings, formatting) is covered by the cli
// package's own tests, which don't require invoking the command.
func TestCheckCmd_Wiring(t *testing.T) {
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

	cmd := newRootCmd()
	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetArgs([]string{"check", "--policy", policyPath, moduleDir})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	if !strings.Contains(out.String(), "0 conforming, 0 violation(s), 0 unused allowance(s)") {
		t.Errorf("unexpected output: %s", out.String())
	}
}

func TestCheckCmd_RequiresPolicyFlag(t *testing.T) {
	cmd := newRootCmd()
	cmd.SetArgs([]string{"check", t.TempDir()})
	cmd.SilenceUsage = true
	cmd.SilenceErrors = true

	if err := cmd.Execute(); err == nil {
		t.Fatal("expected an error when --policy is omitted")
	}
}
